// Copyright 2026 Google LLC
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

package commonlogk8saudit_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

type podIdentity struct {
	// uid is the UID of the pod.
	uid string
	// name is the name of the pod.
	name string
	// namespace is the namespace of the pod.
	namespace string
}

// endpointResourceLogToTimelineMapperStateV2 tracks the status of an EndpointSlice resource during V2 timeline generation.
type endpointResourceLogToTimelineMapperStateV2 struct {
	// serviceNames is the set of service names.
	serviceNames map[string]struct{}
	// foundPods is the map of found pods.
	foundPods map[string]*podIdentity
	// lastStates is the map of last states.
	lastStates map[string]*pb.RevisionState
}

// EndpointResourceLogToTimelineMapperTask is the V2 task to generate endpoint resource history.
var EndpointResourceLogToTimelineMapperTask = commonlogk8saudit_contract.NewManifestLogToTimelineMapperV2[*endpointResourceLogToTimelineMapperStateV2](&endpointResourceLogToTimelineMapperTaskSettingV2{})

type endpointResourceLogToTimelineMapperTaskSettingV2 struct {
}

// PassCount implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (e *endpointResourceLogToTimelineMapperTaskSettingV2) PassCount() int {
	return 1
}

// Dependencies implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (e *endpointResourceLogToTimelineMapperTaskSettingV2) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (e *endpointResourceLogToTimelineMapperTaskSettingV2) GroupedLogTask() taskid.TaskReference[commonlogk8saudit_contract.ResourceManifestLogGroupMap] {
	return commonlogk8saudit_contract.ResourceLifetimeTrackerTaskID.Ref()
}

// LogIngesterTask implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (e *endpointResourceLogToTimelineMapperTaskSettingV2) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return commonlogk8saudit_contract.K8sAuditLogIngesterTaskID.Ref()
}

// TaskID implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (e *endpointResourceLogToTimelineMapperTaskSettingV2) TaskID() taskid.TaskImplementationID[inspectiontaskbase.TimelineMapperResult] {
	return commonlogk8saudit_contract.EndpointResourceLogToTimelineMapperTaskID
}

// ResolveRelatedGroupSets implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (e *endpointResourceLogToTimelineMapperTaskSettingV2) ResolveRelatedGroupSets(ctx context.Context, groupedLogs commonlogk8saudit_contract.ResourceManifestLogGroupMap) ([]commonlogk8saudit_contract.RelatedGroupSet, error) {
	result := []commonlogk8saudit_contract.RelatedGroupSet{}
	for _, group := range groupedLogs {
		if group.Resource.APIVersion == "discovery.k8s.io/v1" && group.Resource.Kind == "endpointslice" {
			result = append(result, commonlogk8saudit_contract.RelatedGroupSet{
				Roles: map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{
					"target": group,
				},
			})
		}
	}
	return result, nil
}

// PreProcessLog implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (e *endpointResourceLogToTimelineMapperTaskSettingV2) PreProcessLog(ctx context.Context, passIndex int, event commonlogk8saudit_contract.MultiGroupLogEvent, state *endpointResourceLogToTimelineMapperStateV2) (*endpointResourceLogToTimelineMapperStateV2, error) {
	if state == nil {
		state = &endpointResourceLogToTimelineMapperStateV2{
			serviceNames: map[string]struct{}{},
			foundPods:    map[string]*podIdentity{},
			lastStates:   map[string]*pb.RevisionState{},
		}
	}
	if event.GroupRole != "target" {
		return state, nil
	}
	bodyReader, hasBody := event.GetLastBodyReader("target")
	if !hasBody || bodyReader == nil {
		return state, nil
	}

	ownerReferences, err := bodyReader.GetReader("metadata.ownerReferences")
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
	endpoints, err := bodyReader.GetReader("endpoints")
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

// ProcessLog implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (e *endpointResourceLogToTimelineMapperTaskSettingV2) ProcessLog(ctx context.Context, event commonlogk8saudit_contract.MultiGroupLogEvent, state *endpointResourceLogToTimelineMapperStateV2) (*khifilev6.TimelineChangeSet, *endpointResourceLogToTimelineMapperStateV2, error) {
	if state == nil {
		state = &endpointResourceLogToTimelineMapperStateV2{
			serviceNames: map[string]struct{}{},
			foundPods:    map[string]*podIdentity{},
			lastStates:   map[string]*pb.RevisionState{},
		}
	}

	cs := khifilev6.NewTimelineChangeSet(event.Log)
	commonLogFieldSet := log.MustGetFieldSet(event.Log, &log.CommonFieldSet{})
	k8sFieldSet := log.MustGetFieldSet(event.Log, &commonlogk8saudit_contract.K8sAuditLogFieldSet{})

	bodyReader, _ := event.GetLastBodyReader("target")

	if event.GroupRole == "target" && event.EventType == commonlogk8saudit_contract.ChangeEventTypeV2Creation && k8sFieldSet.Verb != commonlogk8saudit_contract.VerbCreate {
		creationTime, found := GetCreationTimestamp(bodyReader)
		if found {
			for service := range state.serviceNames {
				rp := MustResolveServiceEndpointSliceTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity.Namespace, event.ResourceIdentity.Name, service)
				cs.AddRevision(rp, &khifilev6.StagingRevision{
					VerbType:     commonlogk8saudit_contract.VerbUnknown,
					ResourceBody: nil,
					Principal:    "N/A",
					ChangedTime:  creationTime,
					StateType:    commonlogk8saudit_contract.RevisionStateConditionNoAvailableInfo,
				})
			}
			for _, podIdentity := range state.foundPods {
				rp1 := MustResolvePodEndpointSliceTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity.Namespace, event.ResourceIdentity.Name, podIdentity.namespace, podIdentity.name)
				cs.AddRevision(rp1, &khifilev6.StagingRevision{
					VerbType:     commonlogk8saudit_contract.VerbUnknown,
					ResourceBody: nil,
					Principal:    "N/A",
					ChangedTime:  creationTime,
					StateType:    commonlogk8saudit_contract.RevisionStateConditionNoAvailableInfo,
				})

				rp2 := MustResolveEndpointSliceChildPodTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity, podIdentity.namespace, podIdentity.name)
				cs.AddRevision(rp2, &khifilev6.StagingRevision{
					VerbType:     commonlogk8saudit_contract.VerbUnknown,
					ResourceBody: nil,
					Principal:    "N/A",
					ChangedTime:  creationTime,
					StateType:    commonlogk8saudit_contract.RevisionStateConditionNoAvailableInfo,
				})
			}
		}
	}

	endpointCount := 0
	readyEndpointCount := 0
	terminatingEndpointCount := 0
	foundUIDs := map[string]struct{}{}
	removedEndpoints := []string{}

	if bodyReader != nil {
		endpoints, err := bodyReader.GetReader("endpoints")
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

				currentState := endpointConditionToPodEndpointState(ready, terminating)
				uid, err := endpoint.ReadString("targetRef.uid")
				if err == nil {
					foundUIDs[uid] = struct{}{}
					if podIdentity, found := state.foundPods[uid]; found {
						if lastState, found := state.lastStates[uid]; !found || lastState.GetId() != currentState.GetId() {
							var endpointBody structured.Node
							endpointBody = endpoint.Node

							rp1 := MustResolvePodEndpointSliceTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity.Namespace, event.ResourceIdentity.Name, podIdentity.namespace, podIdentity.name)
							cs.AddRevision(rp1, &khifilev6.StagingRevision{
								VerbType:     k8sFieldSet.Verb,
								ResourceBody: endpointBody,
								Principal:    k8sFieldSet.Principal,
								ChangedTime:  commonLogFieldSet.Timestamp,
								StateType:    currentState,
							})

							rp2 := MustResolveEndpointSliceChildPodTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity, podIdentity.namespace, podIdentity.name)
							cs.AddRevision(rp2, &khifilev6.StagingRevision{
								VerbType:     k8sFieldSet.Verb,
								ResourceBody: endpointBody,
								Principal:    k8sFieldSet.Principal,
								ChangedTime:  commonLogFieldSet.Timestamp,
								StateType:    currentState,
							})
							state.lastStates[uid] = currentState
						}
					}
				}
			}

			for touchedUID := range state.lastStates {
				if _, found := foundUIDs[touchedUID]; !found {
					if podIdentity, found := state.foundPods[touchedUID]; found {
						rp1 := MustResolvePodEndpointSliceTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity.Namespace, event.ResourceIdentity.Name, podIdentity.namespace, podIdentity.name)
						cs.AddRevision(rp1, &khifilev6.StagingRevision{
							VerbType:     k8sFieldSet.Verb,
							ResourceBody: nil,
							Principal:    k8sFieldSet.Principal,
							ChangedTime:  commonLogFieldSet.Timestamp,
							StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceDeleted,
						})

						rp2 := MustResolveEndpointSliceChildPodTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity, podIdentity.namespace, podIdentity.name)
						cs.AddRevision(rp2, &khifilev6.StagingRevision{
							VerbType:     k8sFieldSet.Verb,
							ResourceBody: nil,
							Principal:    k8sFieldSet.Principal,
							ChangedTime:  commonLogFieldSet.Timestamp,
							StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceDeleted,
						})
						removedEndpoints = append(removedEndpoints, touchedUID)
					}
				}
			}

			var serviceState *pb.RevisionState
			switch {
			case terminatingEndpointCount == endpointCount:
				serviceState = commonlogk8saudit_contract.RevisionStateEndpointTerminating
			case readyEndpointCount == 0:
				serviceState = commonlogk8saudit_contract.RevisionStateEndpointUnready
			default:
				serviceState = commonlogk8saudit_contract.RevisionStateEndpointReady
			}

			bodyNode := bodyReader.Node

			for service := range state.serviceNames {
				rp := MustResolveServiceEndpointSliceTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity.Namespace, event.ResourceIdentity.Name, service)
				cs.AddRevision(rp, &khifilev6.StagingRevision{
					VerbType:     k8sFieldSet.Verb,
					ResourceBody: bodyNode,
					Principal:    k8sFieldSet.Principal,
					ChangedTime:  commonLogFieldSet.Timestamp,
					StateType:    serviceState,
				})
			}
		}
	}

	if event.EventType == commonlogk8saudit_contract.ChangeEventTypeV2Deletion {
		for touchedUID := range state.lastStates {
			if podIdentity, found := state.foundPods[touchedUID]; found {
				rp1 := MustResolvePodEndpointSliceTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity.Namespace, event.ResourceIdentity.Name, podIdentity.namespace, podIdentity.name)
				cs.AddRevision(rp1, &khifilev6.StagingRevision{
					VerbType:     k8sFieldSet.Verb,
					ResourceBody: nil,
					Principal:    k8sFieldSet.Principal,
					ChangedTime:  commonLogFieldSet.Timestamp,
					StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceDeleted,
				})

				rp2 := MustResolveEndpointSliceChildPodTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity, podIdentity.namespace, podIdentity.name)
				cs.AddRevision(rp2, &khifilev6.StagingRevision{
					VerbType:     k8sFieldSet.Verb,
					ResourceBody: nil,
					Principal:    k8sFieldSet.Principal,
					ChangedTime:  commonLogFieldSet.Timestamp,
					StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceDeleted,
				})
				removedEndpoints = append(removedEndpoints, touchedUID)
			}
		}
		for service := range state.serviceNames {
			rp := MustResolveServiceEndpointSliceTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity.Namespace, event.ResourceIdentity.Name, service)
			cs.AddRevision(rp, &khifilev6.StagingRevision{
				VerbType:     k8sFieldSet.Verb,
				ResourceBody: nil,
				Principal:    k8sFieldSet.Principal,
				ChangedTime:  commonLogFieldSet.Timestamp,
				StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceDeleted,
			})
		}
	}

	for _, uid := range removedEndpoints {
		delete(state.lastStates, uid)
	}

	return cs, state, nil
}

// endpointConditionToPodEndpointState converts endpoint conditions to revision state.
func endpointConditionToPodEndpointState(ready bool, terminating bool) *pb.RevisionState {
	switch {
	case ready:
		return commonlogk8saudit_contract.RevisionStateEndpointReady
	case terminating:
		return commonlogk8saudit_contract.RevisionStateEndpointTerminating
	default:
		return commonlogk8saudit_contract.RevisionStateEndpointUnready
	}
}

// MustResolveServiceEndpointSliceTimelinePath resolves ServiceEndpointSlice timeline path.
func MustResolveServiceEndpointSliceTimelinePath(ctx context.Context, clusterName, namespace, endpointSliceName, serviceName string) *khifilev6.TimelinePath {
	cluster := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	api := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, cluster, "core/v1")
	kind := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, api, "service")
	var servicePath *khifilev6.TimelinePath
	if namespace != "" {
		ns := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kind, namespace)
		servicePath = commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, ns, serviceName)
	} else {
		servicePath = commonlogk8saudit_contract.MustK8sClusterScopeResourceTimeline(ctx, kind, serviceName)
	}
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(servicePath, khifilev6.PathSegment{
		Name: endpointSliceName,
		Type: commonlogk8saudit_contract.TimelineTypeEndpointSlice,
	})
}

// MustResolvePodEndpointSliceTimelinePath resolves PodEndpointSlice timeline path.
func MustResolvePodEndpointSliceTimelinePath(ctx context.Context, clusterName, endpointSliceNamespace, endpointSliceName, podNamespace, podName string) *khifilev6.TimelinePath {
	cluster := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	api := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, cluster, "core/v1")
	kind := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, api, "pod")
	var podPath *khifilev6.TimelinePath
	if podNamespace != "" {
		ns := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kind, podNamespace)
		podPath = commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, ns, podName)
	} else {
		podPath = commonlogk8saudit_contract.MustK8sClusterScopeResourceTimeline(ctx, kind, podName)
	}
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(podPath, khifilev6.PathSegment{
		Name: fmt.Sprintf("%s(%s)", endpointSliceName, endpointSliceNamespace),
		Type: commonlogk8saudit_contract.TimelineTypeEndpointSlice,
	})
}

// MustResolveEndpointSliceChildPodTimelinePath resolves EndpointSliceChildPod timeline path.
func MustResolveEndpointSliceChildPodTimelinePath(ctx context.Context, clusterName string, endpointSliceResource *commonlogk8saudit_contract.ResourceIdentity, podNamespace, podName string) *khifilev6.TimelinePath {
	endpointSlicePath := MustResolveTimelinePath(ctx, clusterName, endpointSliceResource)

	var segmentName string
	if podNamespace != endpointSliceResource.Namespace {
		segmentName = fmt.Sprintf("%s(%s)", podName, podNamespace)
	} else {
		segmentName = podName
	}

	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(endpointSlicePath, khifilev6.PathSegment{
		Name: segmentName,
		Type: commonlogk8saudit_contract.TimelineTypeEndpointSlice,
	})
}

var _ commonlogk8saudit_contract.ManifestLogToTimelineMapperV2[*endpointResourceLogToTimelineMapperStateV2] = (*endpointResourceLogToTimelineMapperTaskSettingV2)(nil)
