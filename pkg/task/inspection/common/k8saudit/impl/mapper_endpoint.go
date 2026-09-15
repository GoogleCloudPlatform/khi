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

package k8saudit_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

type podIdentity struct {
	// uid is the UID of the pod.
	uid string
	// name is the name of the pod.
	name string
	// namespace is the namespace of the pod.
	namespace string
}

// endpointResourceLogToTimelineMapperState tracks the status of an EndpointSlice resource during timeline generation.
type endpointResourceLogToTimelineMapperState struct {
	// serviceNames is the set of service names.
	serviceNames map[string]struct{}
	// foundPods is the map of found pods.
	foundPods map[string]*podIdentity
	// lastStates is the map of last states.
	lastStates map[string]*pb.RevisionState
}

var (
	pathEndpointOwnerReferences       = structured.CompileFieldPath("metadata.ownerReferences")
	pathEndpointOwnerKind             = structured.CompileFieldPath("kind")
	pathEndpointOwnerName             = structured.CompileFieldPath("name")
	pathEndpoints                     = structured.CompileFieldPath("endpoints")
	pathEndpointTargetRefKind         = structured.CompileFieldPath("targetRef.kind")
	pathEndpointTargetRefName         = structured.CompileFieldPath("targetRef.name")
	pathEndpointTargetRefNamespace    = structured.CompileFieldPath("targetRef.namespace")
	pathEndpointTargetRefUID          = structured.CompileFieldPath("targetRef.uid")
	pathEndpointConditionsTerminating = structured.CompileFieldPath("conditions.terminating")
	pathEndpointConditionsReady       = structured.CompileFieldPath("conditions.ready")
)

// EndpointResourceLogToTimelineMapperTask is the task to generate endpoint resource history.
var EndpointResourceLogToTimelineMapperTask = k8saudit.NewManifestLogToTimelineMapper[*endpointResourceLogToTimelineMapperState](&endpointResourceLogToTimelineMapperTaskSetting{})

type endpointResourceLogToTimelineMapperTaskSetting struct {
}

// PassCount implements k8saudit.ManifestLogToTimelineMapper.
func (e *endpointResourceLogToTimelineMapperTaskSetting) PassCount() int {
	return 1
}

// Dependencies implements k8saudit.ManifestLogToTimelineMapper.
func (e *endpointResourceLogToTimelineMapperTaskSetting) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{}
}

// GroupedLogTask implements k8saudit.ManifestLogToTimelineMapper.
func (e *endpointResourceLogToTimelineMapperTaskSetting) GroupedLogTask() taskid.TaskReference[k8saudit.ResourceManifestLogGroupMap] {
	return k8saudit.ResourceLifetimeTrackerTaskID.Ref()
}

// LogIngesterTask implements k8saudit.ManifestLogToTimelineMapper.
func (e *endpointResourceLogToTimelineMapperTaskSetting) LogIngesterTask() taskid.TaskReference[struct{}] {
	return k8saudit.K8sAuditLogIngesterTaskID.Ref()
}

// TaskID implements k8saudit.ManifestLogToTimelineMapper.
func (e *endpointResourceLogToTimelineMapperTaskSetting) TaskID() taskid.TaskImplementationID[struct{}] {
	return k8saudit.EndpointResourceLogToTimelineMapperTaskID
}

// ResolveRelatedGroupSets implements k8saudit.ManifestLogToTimelineMapper.
func (e *endpointResourceLogToTimelineMapperTaskSetting) ResolveRelatedGroupSets(ctx context.Context, groupedLogs k8saudit.ResourceManifestLogGroupMap) ([]k8saudit.RelatedGroupSet, error) {
	result := []k8saudit.RelatedGroupSet{}
	for _, group := range groupedLogs {
		if group.Resource.APIVersion == "discovery.k8s.io/v1" && group.Resource.Kind == "endpointslice" {
			result = append(result, k8saudit.RelatedGroupSet{
				Roles: map[string]*k8saudit.ResourceManifestLogGroup{
					"target": group,
				},
			})
		}
	}
	return result, nil
}

// PreProcessLog implements k8saudit.ManifestLogToTimelineMapper.
func (e *endpointResourceLogToTimelineMapperTaskSetting) PreProcessLog(ctx context.Context, passIndex int, event k8saudit.MultiGroupLogEvent, state *endpointResourceLogToTimelineMapperState) (*endpointResourceLogToTimelineMapperState, error) {
	if state == nil {
		state = &endpointResourceLogToTimelineMapperState{
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

	ownerReferences, err := bodyReader.GetReader(pathEndpointOwnerReferences)
	if err == nil {
		// Scan all owner references to collect service names.
		for _, ownerReference := range ownerReferences.Children() {
			kind, err := ownerReference.ReadString(pathEndpointOwnerKind)
			if err != nil {
				continue
			}
			name, err := ownerReference.ReadString(pathEndpointOwnerName)
			if err != nil {
				continue
			}
			if strings.ToLower(kind) == "service" {
				state.serviceNames[name] = struct{}{}
			}
		}
	}

	// Scan all endpoints to collect pod names.
	endpoints, err := bodyReader.GetReader(pathEndpoints)
	if err == nil {
		for _, endpoint := range endpoints.Children() {
			kind, err := endpoint.ReadString(pathEndpointTargetRefKind)
			if err != nil {
				continue
			}
			name, err := endpoint.ReadString(pathEndpointTargetRefName)
			if err != nil {
				continue
			}
			namespace, err := endpoint.ReadString(pathEndpointTargetRefNamespace)
			if err != nil {
				continue
			}
			uid, err := endpoint.ReadString(pathEndpointTargetRefUID)
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

// ProcessLog implements k8saudit.ManifestLogToTimelineMapper.
func (e *endpointResourceLogToTimelineMapperTaskSetting) ProcessLog(ctx context.Context, event k8saudit.MultiGroupLogEvent, state *endpointResourceLogToTimelineMapperState) (*khifilev6.TimelineChangeSet, *endpointResourceLogToTimelineMapperState, error) {
	if state == nil {
		state = &endpointResourceLogToTimelineMapperState{
			serviceNames: map[string]struct{}{},
			foundPods:    map[string]*podIdentity{},
			lastStates:   map[string]*pb.RevisionState{},
		}
	}

	cs := khifilev6.NewTimelineChangeSet(event.Log)
	eventTime := event.Log.Timestamp
	k8sFieldSet, _ := k8saudit.ExtractK8sAuditLog(ctx, event.Log.NodeReader)
	if k8sFieldSet.IsDryRun {
		return cs, state, nil
	}

	bodyReader, _ := event.GetLastBodyReader("target")

	if event.GroupRole == "target" && event.EventType == k8saudit.ChangeEventTypeCreation && k8sFieldSet.Verb != k8saudit.VerbCreate {
		creationTime, found := GetCreationTimestamp(bodyReader)
		if found {
			for service := range state.serviceNames {
				rp := MustResolveServiceEndpointSliceTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity.Namespace, event.ResourceIdentity.Name, service)
				cs.AddRevision(rp, &khifilev6.StagingRevision{
					VerbType:     k8saudit.VerbUnknown,
					ResourceBody: nil,
					Principal:    "N/A",
					ChangedTime:  creationTime,
					StateType:    k8saudit.RevisionStateConditionNoAvailableInfo,
				})
			}
			for _, podIdentity := range state.foundPods {
				rp1 := MustResolvePodEndpointSliceTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity.Namespace, event.ResourceIdentity.Name, podIdentity.namespace, podIdentity.name)
				cs.AddRevision(rp1, &khifilev6.StagingRevision{
					VerbType:     k8saudit.VerbUnknown,
					ResourceBody: nil,
					Principal:    "N/A",
					ChangedTime:  creationTime,
					StateType:    k8saudit.RevisionStateConditionNoAvailableInfo,
				})

				rp2 := MustResolveEndpointSliceChildPodTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity, podIdentity.namespace, podIdentity.name)
				cs.AddRevision(rp2, &khifilev6.StagingRevision{
					VerbType:     k8saudit.VerbUnknown,
					ResourceBody: nil,
					Principal:    "N/A",
					ChangedTime:  creationTime,
					StateType:    k8saudit.RevisionStateConditionNoAvailableInfo,
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
		endpoints, err := bodyReader.GetReader(pathEndpoints)
		if err == nil {
			endpointCount = endpoints.Len()
			for _, endpoint := range endpoints.Children() {
				terminating, err := endpoint.ReadBool(pathEndpointConditionsTerminating)
				if err == nil && terminating {
					terminatingEndpointCount++
				}
				ready, err := endpoint.ReadBool(pathEndpointConditionsReady)
				if err == nil && ready {
					readyEndpointCount++
				}

				currentState := endpointConditionToPodEndpointState(ready, terminating)
				uid, err := endpoint.ReadString(pathEndpointTargetRefUID)
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
								ChangedTime:  eventTime,
								StateType:    currentState,
							})

							rp2 := MustResolveEndpointSliceChildPodTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity, podIdentity.namespace, podIdentity.name)
							cs.AddRevision(rp2, &khifilev6.StagingRevision{
								VerbType:     k8sFieldSet.Verb,
								ResourceBody: endpointBody,
								Principal:    k8sFieldSet.Principal,
								ChangedTime:  eventTime,
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
							ChangedTime:  eventTime,
							StateType:    k8saudit.RevisionStateK8sResourceDeleted,
						})

						rp2 := MustResolveEndpointSliceChildPodTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity, podIdentity.namespace, podIdentity.name)
						cs.AddRevision(rp2, &khifilev6.StagingRevision{
							VerbType:     k8sFieldSet.Verb,
							ResourceBody: nil,
							Principal:    k8sFieldSet.Principal,
							ChangedTime:  eventTime,
							StateType:    k8saudit.RevisionStateK8sResourceDeleted,
						})
						removedEndpoints = append(removedEndpoints, touchedUID)
					}
				}
			}

			var serviceState *pb.RevisionState
			switch {
			case terminatingEndpointCount == endpointCount:
				serviceState = k8saudit.RevisionStateEndpointTerminating
			case readyEndpointCount == 0:
				serviceState = k8saudit.RevisionStateEndpointUnready
			default:
				serviceState = k8saudit.RevisionStateEndpointReady
			}

			bodyNode := bodyReader.Node

			for service := range state.serviceNames {
				rp := MustResolveServiceEndpointSliceTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity.Namespace, event.ResourceIdentity.Name, service)
				cs.AddRevision(rp, &khifilev6.StagingRevision{
					VerbType:     k8sFieldSet.Verb,
					ResourceBody: bodyNode,
					Principal:    k8sFieldSet.Principal,
					ChangedTime:  eventTime,
					StateType:    serviceState,
				})
			}
		}
	}

	if event.EventType == k8saudit.ChangeEventTypeDeletion {
		for touchedUID := range state.lastStates {
			if podIdentity, found := state.foundPods[touchedUID]; found {
				rp1 := MustResolvePodEndpointSliceTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity.Namespace, event.ResourceIdentity.Name, podIdentity.namespace, podIdentity.name)
				cs.AddRevision(rp1, &khifilev6.StagingRevision{
					VerbType:     k8sFieldSet.Verb,
					ResourceBody: nil,
					Principal:    k8sFieldSet.Principal,
					ChangedTime:  eventTime,
					StateType:    k8saudit.RevisionStateK8sResourceDeleted,
				})

				rp2 := MustResolveEndpointSliceChildPodTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity, podIdentity.namespace, podIdentity.name)
				cs.AddRevision(rp2, &khifilev6.StagingRevision{
					VerbType:     k8sFieldSet.Verb,
					ResourceBody: nil,
					Principal:    k8sFieldSet.Principal,
					ChangedTime:  eventTime,
					StateType:    k8saudit.RevisionStateK8sResourceDeleted,
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
				ChangedTime:  eventTime,
				StateType:    k8saudit.RevisionStateK8sResourceDeleted,
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
		return k8saudit.RevisionStateEndpointReady
	case terminating:
		return k8saudit.RevisionStateEndpointTerminating
	default:
		return k8saudit.RevisionStateEndpointUnready
	}
}

// MustResolveServiceEndpointSliceTimelinePath resolves ServiceEndpointSlice timeline path.
func MustResolveServiceEndpointSliceTimelinePath(ctx context.Context, clusterName, namespace, endpointSliceName, serviceName string) *khifilev6.TimelinePath {
	cluster := k8saudit.MustK8sClusterTimeline(ctx, clusterName)
	api := k8saudit.MustK8sAPIVersionTimeline(ctx, cluster, "core/v1")
	kind := k8saudit.MustK8sKindTimeline(ctx, api, "service")
	var servicePath *khifilev6.TimelinePath
	if namespace != "" {
		ns := k8saudit.MustK8sNamespaceTimeline(ctx, kind, namespace)
		servicePath = k8saudit.MustK8sNamespacedResourceTimeline(ctx, ns, serviceName)
	} else {
		servicePath = k8saudit.MustK8sClusterScopeResourceTimeline(ctx, kind, serviceName)
	}
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)
	return builder.TimelineAccumulator.GetPath(servicePath, khifilev6.PathSegment{
		Name: endpointSliceName,
		Type: k8saudit.TimelineTypeEndpointSlice,
	})
}

// MustResolvePodEndpointSliceTimelinePath resolves PodEndpointSlice timeline path.
func MustResolvePodEndpointSliceTimelinePath(ctx context.Context, clusterName, endpointSliceNamespace, endpointSliceName, podNamespace, podName string) *khifilev6.TimelinePath {
	cluster := k8saudit.MustK8sClusterTimeline(ctx, clusterName)
	api := k8saudit.MustK8sAPIVersionTimeline(ctx, cluster, "core/v1")
	kind := k8saudit.MustK8sKindTimeline(ctx, api, "pod")
	var podPath *khifilev6.TimelinePath
	if podNamespace != "" {
		ns := k8saudit.MustK8sNamespaceTimeline(ctx, kind, podNamespace)
		podPath = k8saudit.MustK8sNamespacedResourceTimeline(ctx, ns, podName)
	} else {
		podPath = k8saudit.MustK8sClusterScopeResourceTimeline(ctx, kind, podName)
	}
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)
	return builder.TimelineAccumulator.GetPath(podPath, khifilev6.PathSegment{
		Name: fmt.Sprintf("%s(%s)", endpointSliceName, endpointSliceNamespace),
		Type: k8saudit.TimelineTypeEndpointSlice,
	})
}

// MustResolveEndpointSliceChildPodTimelinePath resolves EndpointSliceChildPod timeline path.
func MustResolveEndpointSliceChildPodTimelinePath(ctx context.Context, clusterName string, endpointSliceResource *k8saudit.ResourceIdentity, podNamespace, podName string) *khifilev6.TimelinePath {
	endpointSlicePath := MustResolveTimelinePath(ctx, clusterName, endpointSliceResource)

	var segmentName string
	if podNamespace != endpointSliceResource.Namespace {
		segmentName = fmt.Sprintf("%s(%s)", podName, podNamespace)
	} else {
		segmentName = podName
	}

	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)
	return builder.TimelineAccumulator.GetPath(endpointSlicePath, khifilev6.PathSegment{
		Name: segmentName,
		Type: k8saudit.TimelineTypeEndpointSlice,
	})
}

var _ k8saudit.ManifestLogToTimelineMapper[*endpointResourceLogToTimelineMapperState] = (*endpointResourceLogToTimelineMapperTaskSetting)(nil)
