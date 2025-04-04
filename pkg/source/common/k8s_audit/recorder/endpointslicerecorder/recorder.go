// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package endpointslicerecorder

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourceinfo/resourcelease"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"

	goyaml "gopkg.in/yaml.v3"
)

// singleEndpointParseResult is a type used in the middle of parsing an endpoint of entire EndpointSlice resource.
type singleEndpointParseResult struct {
	verb               enum.RevisionVerb
	state              enum.RevisionState
	manifest           string
	isEndpointForPod   bool
	hasCoditionChanged bool
}

// endpointSliceParseResult is a type used in the middle of parsing entire EndpointSlice resource.
type endpointsParseResult struct {
	verb             enum.RevisionVerb
	state            enum.RevisionState
	manifest         string
	hasConditionInfo bool
}

func Register(manager *recorder.RecorderTaskManager) error {
	manager.AddRecorder("endpointslices", []taskid.UntypedTaskReference{}, func(ctx context.Context, resourcePath string, currentLog *types.AuditLogParserInput, prevStateInGroup any, cs *history.ChangeSet, builder *history.Builder) (any, error) {
		var prevEndpointSlice *model.EndpointSlice
		if prevStateInGroup != nil {
			prevEndpointSlice = prevStateInGroup.(*model.EndpointSlice)
		}
		return recordChangeSetForLog(ctx, currentLog, prevEndpointSlice, cs, builder)
	}, recorder.ResourceKindLogGroupFilter("endpointslice"), recorder.AndLogFilter(recorder.OnlySucceedLogs(), recorder.OnlyWithResourceBody()))
	return nil
}

func recordChangeSetForLog(ctx context.Context, log *types.AuditLogParserInput, prevEndpointSlices *model.EndpointSlice, cs *history.ChangeSet, builder *history.Builder) (*model.EndpointSlice, error) {
	var endpointSlice model.EndpointSlice
	err := log.ResourceBodyReader.ReadReflect("", &endpointSlice)
	if err != nil {
		return nil, err
	}
	relatedServiceName := ""
	if endpointSlice.Metadata != nil && endpointSlice.Metadata.OwnerReferences != nil {
		for _, owner := range endpointSlice.Metadata.OwnerReferences {
			if strings.ToLower(owner.Kind) == "service" {
				if relatedServiceName != "" {
					slog.WarnContext(ctx, fmt.Sprintf("multiple owners found for a single endpoint slice. ignoreing service %s", relatedServiceName))
				}
				relatedServiceName = owner.Name
			}
		}
	}
	isOwnedByService := relatedServiceName != ""

	if endpointSlice.Endpoints == nil {
		endpointSlice.Endpoints = make([]*model.EndpointSliceEndpoint, 0)
	}
	for _, endpoint := range endpointSlice.Endpoints {
		endpointParseResult, err := parseSingleEndpoint(ctx, endpoint, prevEndpointSlices)
		if err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("failed to parse an endpoint\n%s", err.Error()))
		}

		// records Ips used in Pods. IP can be read from Pod manifest, but it can be ignored when users didn't turn on DATA_WRITE audit log, but endpoint slice update will be recorded always.
		if endpointParseResult.isEndpointForPod {
			for _, address := range endpoint.Addresses {
				builder.ClusterResource.IPs.TouchResourceLease(address, log.Log.Timestamp(), resourcelease.NewK8sResourceLeaseHolder(endpoint.TargetRef.Kind, endpoint.TargetRef.Namespace, endpoint.TargetRef.Name))
			}
		}

		// record conditions as subresource of pod.
		if endpointParseResult.hasCoditionChanged && endpointParseResult.isEndpointForPod {
			podEndpointSliceResourcePath := resourcepath.PodEndpointSlice(log.Operation.Namespace, log.Operation.Name, endpoint.TargetRef.Namespace, endpoint.TargetRef.Name)
			cs.RecordRevision(podEndpointSliceResourcePath, &history.StagingResourceRevision{
				Body:       endpointParseResult.manifest,
				State:      endpointParseResult.state,
				Verb:       endpointParseResult.verb,
				ChangeTime: log.Log.Timestamp(),
			})
		}
	}
	// find endpoints included in the previous revision but not in in the current revision
	if prevEndpointSlices != nil {
		for _, prevEndpoint := range prevEndpointSlices.Endpoints {
			var current *model.EndpointSliceEndpoint
			if prevEndpoint.TargetRef != nil {
				current = lookupEndpointFromUid(&endpointSlice, prevEndpoint.TargetRef.Uid)
			}
			if current != nil {
				continue
			}
			if prevEndpoint.TargetRef != nil {
				podEndpointSliceResourcePath := resourcepath.PodEndpointSlice(log.Operation.Namespace, log.Operation.Name, prevEndpoint.TargetRef.Namespace, prevEndpoint.TargetRef.Name)
				// Only process endpoints not included in current endpoint slices
				cs.RecordRevision(podEndpointSliceResourcePath, &history.StagingResourceRevision{
					Body:       "# This endpoint removed from endpoint list of the EndpointSlice",
					State:      enum.RevisionStateDeleted,
					Verb:       enum.RevisionVerbDelete,
					ChangeTime: log.Log.Timestamp(),
				})
			}
		}
	}

	if isOwnedByService {
		endpointsParseResults, err := parseEndpointsOfEndpointSlice(ctx, &endpointSlice)
		if err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("failed to parse an endpoint\n%s", err.Error()))
		} else {
			serviceEndpointSliceResourcePath := resourcepath.ServiceEndpointSlice(log.Operation.Namespace, log.Operation.Name, relatedServiceName)
			cs.RecordRevision(serviceEndpointSliceResourcePath, &history.StagingResourceRevision{
				Body:       endpointsParseResults.manifest,
				State:      endpointsParseResults.state,
				Verb:       endpointsParseResults.verb,
				ChangeTime: log.Log.Timestamp(),
			})
		}
	}
	return &endpointSlice, nil
}

// parseSingleEndpoint parses specific endpoint of a EndpointSlices.
func parseSingleEndpoint(ctx context.Context, endpoint *model.EndpointSliceEndpoint, prevEndpointSlice *model.EndpointSlice) (*singleEndpointParseResult, error) {
	var prev *model.EndpointSliceEndpoint
	if endpoint.TargetRef != nil {
		prev = lookupEndpointFromUid(prevEndpointSlice, endpoint.TargetRef.Uid)
	}
	state := enum.RevisionStateConditionUnknown
	verb := enum.RevisionVerbUnknown

	isEndpointForPod := endpoint.TargetRef != nil && endpoint.TargetRef.Kind == "Pod"                                    // target of this endpoint is a Pod.
	hasConditionChanged := endpoint.Conditions != nil && (prev == nil || !endpoint.Conditions.SameWith(prev.Conditions)) // the condition state is changed.
	if hasConditionChanged {
		switch {
		case endpoint.Conditions.Ready:
			state = enum.RevisionStateEndpointReady
			verb = enum.RevisionVerbReady
		case endpoint.Conditions.Terminating:
			state = enum.RevisionStateEndpointTerminating
			verb = enum.RevisionVerbTerminating
		default:
			state = enum.RevisionStateEndpointUnready
			verb = enum.RevisionVerbNonReady
		}
	}

	conditionManifest, err := goyaml.Marshal(endpoint)
	if err != nil {
		slog.WarnContext(ctx, fmt.Sprintf("failed to marshal endpoint data to yaml\n%s", err.Error()))
		return nil, err
	}

	return &singleEndpointParseResult{
		isEndpointForPod:   isEndpointForPod,
		hasCoditionChanged: hasConditionChanged,
		state:              state,
		verb:               verb,
		manifest:           string(conditionManifest),
	}, nil
}

// parseEndpointsOfEndpointSlice parses the summarized endpoint slice status over all endpoints.
func parseEndpointsOfEndpointSlice(ctx context.Context, endpointSlice *model.EndpointSlice) (*endpointsParseResult, error) {
	hasCondition := false
	hasReadyEndpoint := false
	hasTerminatingEndpoint := false
	for _, endpoint := range endpointSlice.Endpoints {
		if endpoint.Conditions == nil {
			continue
		}
		hasCondition = true
		if endpoint.Conditions.Ready {
			hasReadyEndpoint = true
		}
		if endpoint.Conditions.Terminating {
			hasTerminatingEndpoint = true
		}
	}

	state := enum.RevisionStateEndpointUnready
	verb := enum.RevisionVerbUnknown
	if hasCondition {
		if hasReadyEndpoint {
			state = enum.RevisionStateEndpointReady
			verb = enum.RevisionVerbReady
		} else if hasTerminatingEndpoint {
			state = enum.RevisionStateEndpointTerminating
			verb = enum.RevisionVerbTerminating
		}
	}
	if len(endpointSlice.Endpoints) == 0 {
		state = enum.RevisionStateEndpointUnready
		verb = enum.RevisionVerbNonReady
	}

	endpointsYaml, err := goyaml.Marshal(endpointSlice)
	if err != nil {
		slog.WarnContext(ctx, fmt.Sprintf("failed to marshal endpoint slice data to yaml\n%s", err.Error()))
		return nil, err
	}

	return &endpointsParseResult{
		state:            state,
		verb:             verb,
		hasConditionInfo: hasCondition,
		manifest:         string(endpointsYaml),
	}, nil
}

func lookupEndpointFromUid(endpointSlices *model.EndpointSlice, uid string) *model.EndpointSliceEndpoint {
	if endpointSlices == nil || endpointSlices.Endpoints == nil {
		return nil
	}
	for _, endpoint := range endpointSlices.Endpoints {
		if endpoint.TargetRef != nil && endpoint.TargetRef.Uid == uid {
			return endpoint
		}
	}
	return nil
}
