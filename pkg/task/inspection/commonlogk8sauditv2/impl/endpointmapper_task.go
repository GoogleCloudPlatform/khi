// Copyright 2025 Google LLC
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

package commonlogk8sauditv2_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
)

type podIdentity struct {
	// uid is the UID of the pod.
	uid string
	// name is the name of the pod.
	name string
	// namespace is the namespace of the pod.
	namespace string
}

type endpointResourceLogToTimelineMapperState struct {
	// serviceNames is the set of service names.
	serviceNames map[string]struct{}
	// foundPods is the map of found pods.
	foundPods map[string]*podIdentity
	// lastStates is the map of last states.
	lastStates map[string]enum.RevisionState
}

// EndpointResourceLogToTimelineMapperTask is the task to generate endpoint resource history.
var EndpointResourceLogToTimelineMapperTask = commonlogk8sauditv2_contract.NewManifestLogToTimelineMapper[*endpointResourceLogToTimelineMapperState](&endpointResourceLogToTimelineMapperTaskSetting{})

type endpointResourceLogToTimelineMapperTaskSetting struct {
}

// PassCount implements commonlogk8sauditv2_contract.ManifestLogToTimelineMapperTaskSetting.
func (e *endpointResourceLogToTimelineMapperTaskSetting) PassCount() int {
	return 2
}

// Process implements commonlogk8sauditv2_contract.ManifestLogToTimelineMapperTaskSetting.
func (e *endpointResourceLogToTimelineMapperTaskSetting) Process(ctx context.Context, passIndex int, event commonlogk8sauditv2_contract.ResourceChangeEvent, cs *history.ChangeSet, builder *history.Builder, state *endpointResourceLogToTimelineMapperState) (*endpointResourceLogToTimelineMapperState, error) {
	if state == nil {
		state = &endpointResourceLogToTimelineMapperState{
			serviceNames: map[string]struct{}{},
			foundPods:    map[string]*podIdentity{},
			lastStates:   map[string]enum.RevisionState{},
		}
	}
	if event.EventTargetBodyReader == nil {
		return state, nil
	}
	switch passIndex {
	case 0:
		return e.processFirstPass(ctx, event, cs, builder, state)
	case 1:
		return e.processSecondPass(ctx, event, cs, builder, state)
	default:
		return nil, fmt.Errorf("invalid pass index: %d", passIndex)
	}
}

// processFirstPass collects all service names and pod identities from the log.
func (e *endpointResourceLogToTimelineMapperTaskSetting) processFirstPass(ctx context.Context, event commonlogk8sauditv2_contract.ResourceChangeEvent, cs *history.ChangeSet, builder *history.Builder, state *endpointResourceLogToTimelineMapperState) (*endpointResourceLogToTimelineMapperState, error) {
	ownerReferences, err := event.EventTargetBodyReader.GetReader("metadata.ownerReferences")
	if err == nil {
		// Scan all owner references to collect service names.
		for _, ownerReference := range ownerReferences.Children() {
			kind, err := ownerReference.ReadString("kind")
			if err != nil {
				continue
			}
			name, err := ownerReference.ReadString("name")
			if err != nil {
				continue
			}
			if strings.ToLower(kind) == "service" {
				state.serviceNames[name] = struct{}{}
			}
		}
	}

	// Scan all endpoints to collect pod names.
	endpoints, err := event.EventTargetBodyReader.GetReader("endpoints")
	if err == nil {
		for _, endpoint := range endpoints.Children() {
			kind, err := endpoint.ReadString("targetRef.kind")
			if err != nil {
				continue
			}
			name, err := endpoint.ReadString("targetRef.name")
			if err != nil {
				continue
			}
			namespace, err := endpoint.ReadString("targetRef.namespace")
			if err != nil {
				continue
			}
			uid, err := endpoint.ReadString("targetRef.uid")
			if err != nil {
				continue
			}
			if strings.ToLower(kind) == "pod" {
				state.foundPods[uid] = &podIdentity{
					uid:       uid,
					name:      name,
					namespace: namespace,
				}
			}
		}
	}

	return state, nil
}

// processSecondPass generates revisions for each endpoint.
func (e *endpointResourceLogToTimelineMapperTaskSetting) processSecondPass(ctx context.Context, event commonlogk8sauditv2_contract.ResourceChangeEvent, cs *history.ChangeSet, builder *history.Builder, state *endpointResourceLogToTimelineMapperState) (*endpointResourceLogToTimelineMapperState, error) {
	commonLogFieldSet := log.MustGetFieldSet(event.Log, &log.CommonFieldSet{})
	k8sFieldSet := log.MustGetFieldSet(event.Log, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})
	if event.EventType == commonlogk8sauditv2_contract.ChangeEventTypeTargetCreation && k8sFieldSet.K8sOperation.Verb != enum.RevisionVerbCreate {
		creationTime, found := GetCreationTimestamp(event.EventTargetBodyReader)
		if found {

			// TODO: this must track the minimum of service creation time and endpoint creation time
			for service := range state.serviceNames {
				rp := resourcepath.ServiceEndpointSlice(event.EventTargetResource.Namespace, event.EventTargetResource.Name, service)
				cs.AddRevision(rp, &history.StagingResourceRevision{
					Verb:       enum.RevisionVerbUnknown,
					Body:       "# No available information during log collection period. Resource may exists but no status information during this period.",
					Partial:    false,
					Requestor:  "N/A",
					ChangeTime: creationTime,
					State:      enum.RevisionStateInferred,
				})
			}
			// TODO: this must track the minimum of pod creation time and endpoint creation time
			for _, podIdentity := range state.foundPods {
				rp := resourcepath.PodEndpointSlice(event.EventTargetResource.Namespace, event.EventTargetResource.Name, podIdentity.namespace, podIdentity.name)
				cs.AddRevision(rp, &history.StagingResourceRevision{
					Verb:       enum.RevisionVerbUnknown,
					Body:       "# No available information during log collection period. Resource may exists but no status information during this period.",
					Partial:    false,
					Requestor:  "N/A",
					ChangeTime: creationTime,
					State:      enum.RevisionStateInferred,
				})
				rp = resourcepath.EndpointSliceChildPod(event.EventTargetResource.Namespace, event.EventTargetResource.Name, podIdentity.namespace, podIdentity.name)
				cs.AddRevision(rp, &history.StagingResourceRevision{
					Verb:       enum.RevisionVerbUnknown,
					Body:       "# No available information during log collection period. Resource may exists but no status information during this period.",
					Partial:    false,
					Requestor:  "N/A",
					ChangeTime: creationTime,
					State:      enum.RevisionStateInferred,
				})
			}
		}
	}
	endpointCount := 0
	readyEndpointCount := 0
	terminatingEndpointCount := 0
	endpoints, err := event.EventTargetBodyReader.GetReader("endpoints")
	foundUIDs := map[string]struct{}{}
	removedEndpoints := []string{}
	if err == nil {
		endpointCount = endpoints.Len()
		for _, endpoint := range endpoints.Children() {
			terminating, err := endpoint.ReadBool("conditions.terminating")
			if err == nil && terminating {
				terminatingEndpointCount++
			}
			ready, err := endpoint.ReadBool("conditions.ready")
			if err == nil && ready {
				readyEndpointCount++
			}

			// Add a revision under pod when it was changed from the last state
			currentState := endpointConditionToPodEndpointState(ready, terminating)
			uid, err := endpoint.ReadString("targetRef.uid")
			if err == nil {
				foundUIDs[uid] = struct{}{}
				if podIdentity, found := state.foundPods[uid]; found {
					if lastState, found := state.lastStates[uid]; !found || lastState != currentState {
						var endpointBody string
						endPointRaw, err := endpoint.Serialize("", &structured.YAMLNodeSerializer{})
						if err == nil {
							endpointBody = string(endPointRaw)
						}
						rp := resourcepath.PodEndpointSlice(event.EventTargetResource.Namespace, event.EventTargetResource.Name, podIdentity.namespace, podIdentity.name)
						cs.AddRevision(rp, &history.StagingResourceRevision{
							Verb:       k8sFieldSet.K8sOperation.Verb,
							Body:       endpointBody,
							Partial:    false,
							Requestor:  k8sFieldSet.Principal,
							ChangeTime: commonLogFieldSet.Timestamp,
							State:      currentState,
						})
						rp = resourcepath.EndpointSliceChildPod(event.EventTargetResource.Namespace, event.EventTargetResource.Name, podIdentity.namespace, podIdentity.name)
						cs.AddRevision(rp, &history.StagingResourceRevision{
							Verb:       k8sFieldSet.K8sOperation.Verb,
							Body:       endpointBody,
							Partial:    false,
							Requestor:  k8sFieldSet.Principal,
							ChangeTime: commonLogFieldSet.Timestamp,
							State:      currentState,
						})
						state.lastStates[uid] = currentState
					}
				}
			}
		}

		for touchedUID := range state.lastStates {
			if _, found := foundUIDs[touchedUID]; !found {
				// the resource associated with the touched UID was found in the previous log, but not found in the current log.
				if podIdentity, found := state.foundPods[touchedUID]; found {
					rp := resourcepath.PodEndpointSlice(event.EventTargetResource.Namespace, event.EventTargetResource.Name, podIdentity.namespace, podIdentity.name)
					cs.AddRevision(rp, &history.StagingResourceRevision{
						Verb:       k8sFieldSet.K8sOperation.Verb,
						Body:       "",
						Partial:    false,
						Requestor:  k8sFieldSet.Principal,
						ChangeTime: commonLogFieldSet.Timestamp,
						State:      enum.RevisionStateDeleted,
					})
					rp = resourcepath.EndpointSliceChildPod(event.EventTargetResource.Namespace, event.EventTargetResource.Name, podIdentity.namespace, podIdentity.name)
					cs.AddRevision(rp, &history.StagingResourceRevision{
						Verb:       k8sFieldSet.K8sOperation.Verb,
						Body:       "",
						Partial:    false,
						Requestor:  k8sFieldSet.Principal,
						ChangeTime: commonLogFieldSet.Timestamp,
						State:      enum.RevisionStateDeleted,
					})
					removedEndpoints = append(removedEndpoints, touchedUID)
				}
			}
		}

		// determine service state
		var serviceState enum.RevisionState
		switch {
		case terminatingEndpointCount == endpointCount:
			serviceState = enum.RevisionStateEndpointTerminating
		case readyEndpointCount == 0:
			serviceState = enum.RevisionStateEndpointUnready
		default:
			serviceState = enum.RevisionStateEndpointReady
		}

		for service := range state.serviceNames {
			rp := resourcepath.ServiceEndpointSlice(event.EventTargetResource.Namespace, event.EventTargetResource.Name, service)
			cs.AddRevision(rp, &history.StagingResourceRevision{
				Verb:       k8sFieldSet.K8sOperation.Verb,
				Body:       event.EventTargetBodyYAML,
				Partial:    false,
				Requestor:  k8sFieldSet.Principal,
				ChangeTime: commonLogFieldSet.Timestamp,
				State:      serviceState,
			})
		}
	}
	if event.EventType == commonlogk8sauditv2_contract.ChangeEventTypeTargetDeletion {
		for touchedUID := range state.lastStates {
			if podIdentity, found := state.foundPods[touchedUID]; found {
				rp := resourcepath.PodEndpointSlice(event.EventTargetResource.Namespace, event.EventTargetResource.Name, podIdentity.namespace, podIdentity.name)
				cs.AddRevision(rp, &history.StagingResourceRevision{
					Verb:       k8sFieldSet.K8sOperation.Verb,
					Body:       "",
					Partial:    false,
					Requestor:  k8sFieldSet.Principal,
					ChangeTime: commonLogFieldSet.Timestamp,
					State:      enum.RevisionStateDeleted,
				})
				rp = resourcepath.EndpointSliceChildPod(event.EventTargetResource.Namespace, event.EventTargetResource.Name, podIdentity.namespace, podIdentity.name)
				cs.AddRevision(rp, &history.StagingResourceRevision{
					Verb:       k8sFieldSet.K8sOperation.Verb,
					Body:       "",
					Partial:    false,
					Requestor:  k8sFieldSet.Principal,
					ChangeTime: commonLogFieldSet.Timestamp,
					State:      enum.RevisionStateDeleted,
				})
				removedEndpoints = append(removedEndpoints, touchedUID)
			}
		}
		for service := range state.serviceNames {
			rp := resourcepath.ServiceEndpointSlice(event.EventTargetResource.Namespace, event.EventTargetResource.Name, service)
			cs.AddRevision(rp, &history.StagingResourceRevision{
				Verb:       k8sFieldSet.K8sOperation.Verb,
				Body:       "",
				Partial:    false,
				Requestor:  k8sFieldSet.Principal,
				ChangeTime: commonLogFieldSet.Timestamp,
				State:      enum.RevisionStateDeleted,
			})
		}
	}
	for _, uid := range removedEndpoints {
		delete(state.lastStates, uid)
	}
	return state, nil
}

// TaskID implements commonlogk8sauditv2_contract.ManifestLogToTimelineMapperTaskSetting.
func (e *endpointResourceLogToTimelineMapperTaskSetting) TaskID() taskid.TaskImplementationID[struct{}] {
	return commonlogk8sauditv2_contract.EndpointResourceLogToTimelineMapperTaskID
}

// ResourcePairs implements commonlogk8sauditv2_contract.ManifestLogToTimelineMapperTaskSetting.
func (e *endpointResourceLogToTimelineMapperTaskSetting) ResourcePairs(ctx context.Context, groupedLogs commonlogk8sauditv2_contract.ResourceManifestLogGroupMap) ([]commonlogk8sauditv2_contract.ResourcePair, error) {
	result := []commonlogk8sauditv2_contract.ResourcePair{}
	for _, group := range groupedLogs {
		if group.Resource.APIVersion == "discovery.k8s.io/v1" && group.Resource.Kind == "endpointslice" {
			result = append(result, commonlogk8sauditv2_contract.ResourcePair{
				TargetGroup: group.Resource,
			})
		}
	}
	return result, nil
}

// Dependencies implements commonlogk8sauditv2_contract.ManifestLogToTimelineMapperTaskSetting.
func (e *endpointResourceLogToTimelineMapperTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements commonlogk8sauditv2_contract.ManifestLogToTimelineMapperTaskSetting.
func (e *endpointResourceLogToTimelineMapperTaskSetting) GroupedLogTask() taskid.TaskReference[commonlogk8sauditv2_contract.ResourceManifestLogGroupMap] {
	return commonlogk8sauditv2_contract.ResourceLifetimeTrackerTaskID.Ref()
}

// LogIngesterTask implements commonlogk8sauditv2_contract.ManifestLogToTimelineMapperTaskSetting.
func (e *endpointResourceLogToTimelineMapperTaskSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return commonlogk8sauditv2_contract.K8sAuditLogIngesterTaskID.Ref()
}

// endpointConditionToPodEndpointState converts endpoint conditions to revision state.
func endpointConditionToPodEndpointState(ready bool, terminating bool) enum.RevisionState {
	switch {
	case ready:
		return enum.RevisionStateEndpointReady
	case terminating:
		return enum.RevisionStateEndpointTerminating
	default:
		return enum.RevisionStateEndpointUnready
	}
}
