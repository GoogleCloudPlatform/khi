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
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// containerType is the type of the container.
type containerType string

const (
	// ContainerTypeContainer is the container type for standard containers.
	ContainerTypeContainer containerType = "container"
	// ContainerTypeInitContainer is the container type for init containers.
	ContainerTypeInitContainer containerType = "initContainer"
	// ContainerTypeEphemeral is the container type for ephemeral containers.
	ContainerTypeEphemeral containerType = "ephemeral"
)

type containerStatusIdentity struct {
	// containerName is the name of the container.
	containerName string
	// containerType is the type of the container.
	containerType containerType
}

// ContainerLogToTimelineMapperTask is the V2 task to generate container history.
var ContainerLogToTimelineMapperTask = commonlogk8saudit_contract.NewManifestLogToTimelineMapper[*containerLogToTimelineMapperTaskState](&containerLogToTimelineMapperTaskSetting{})

type containerLogToTimelineMapperTaskState struct {
	// containerIdentities is the map of container identities.
	containerIdentities map[string]*containerStatusIdentity
	// containerStateWalkers is the map of container state walkers.
	containerStateWalkers map[string]*containerStateWalker
}

type containerLogToTimelineMapperTaskSetting struct {
}

// Dependencies implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (c *containerLogToTimelineMapperTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (c *containerLogToTimelineMapperTaskSetting) GroupedLogTask() taskid.TaskReference[commonlogk8saudit_contract.ResourceManifestLogGroupMap] {
	return commonlogk8saudit_contract.ResourceLifetimeTrackerTaskID.Ref()
}

// LogIngesterTask implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (c *containerLogToTimelineMapperTaskSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return commonlogk8saudit_contract.K8sAuditLogIngesterTaskID.Ref()
}

// PassCount implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (c *containerLogToTimelineMapperTaskSetting) PassCount() int {
	return 1
}

// TaskID implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (c *containerLogToTimelineMapperTaskSetting) TaskID() taskid.TaskImplementationID[inspectiontaskbase.TimelineMapperResult] {
	return commonlogk8saudit_contract.ContainerLogToTimelineMapperTaskID
}

// ResolveRelatedGroupSets implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (c *containerLogToTimelineMapperTaskSetting) ResolveRelatedGroupSets(ctx context.Context, groupedLogs commonlogk8saudit_contract.ResourceManifestLogGroupMap) ([]commonlogk8saudit_contract.RelatedGroupSet, error) {
	result := []commonlogk8saudit_contract.RelatedGroupSet{}
	for _, group := range groupedLogs {
		if group.Resource.Type() == commonlogk8saudit_contract.Resource && group.Resource.APIVersion == "core/v1" && group.Resource.Kind == "pod" {
			result = append(result, commonlogk8saudit_contract.RelatedGroupSet{
				Roles: map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{
					"pod": group,
				},
			})
		}
	}
	return result, nil
}

// PreProcessLog implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (c *containerLogToTimelineMapperTaskSetting) PreProcessLog(ctx context.Context, passIndex int, event commonlogk8saudit_contract.MultiGroupLogEvent, state *containerLogToTimelineMapperTaskState) (*containerLogToTimelineMapperTaskState, error) {
	if state == nil {
		state = &containerLogToTimelineMapperTaskState{
			containerIdentities:   map[string]*containerStatusIdentity{},
			containerStateWalkers: map[string]*containerStateWalker{},
		}
	}
	if event.GroupRole != "pod" {
		return state, nil
	}
	bodyReader, ok := event.GetLastBodyReader("pod")
	if !ok || bodyReader == nil {
		return state, nil
	}

	findContainers := func(containerType containerType, fieldName string) {
		statuses, err := bodyReader.GetReader(fieldName)
		if err == nil {
			for _, status := range statuses.Children() {
				name, err := status.ReadString("name")
				if err == nil {
					identity := &containerStatusIdentity{
						containerName: name,
						containerType: containerType,
					}
					state.containerIdentities[identity.containerName] = identity
				}
			}
		}
	}
	findContainers(ContainerTypeContainer, "status.containerStatuses")
	findContainers(ContainerTypeInitContainer, "status.initContainerStatuses")
	findContainers(ContainerTypeEphemeral, "status.ephemeralContainerStatuses")

	return state, nil
}

// ProcessLog implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (c *containerLogToTimelineMapperTaskSetting) ProcessLog(ctx context.Context, event commonlogk8saudit_contract.MultiGroupLogEvent, state *containerLogToTimelineMapperTaskState) (*khifilev6.TimelineChangeSet, *containerLogToTimelineMapperTaskState, error) {
	if state == nil {
		state = &containerLogToTimelineMapperTaskState{
			containerIdentities:   map[string]*containerStatusIdentity{},
			containerStateWalkers: map[string]*containerStateWalker{},
		}
	}
	if event.GroupRole != "pod" {
		return nil, state, nil
	}

	cs := khifilev6.NewTimelineChangeSet(event.Log)
	bodyReader, hasBody := event.GetLastBodyReader("pod")

	currentStateReaders := map[string]*structured.NodeReader{}
	if hasBody && bodyReader != nil {
		findContainerStateReaders := func(containerType containerType, fieldName string) {
			statuses, err := bodyReader.GetReader(fieldName)
			if err == nil {
				for _, status := range statuses.Children() {
					name, err := status.ReadString("name")
					if err == nil {
						currentStateReaders[name] = &status
					}
				}
			}
		}
		findContainerStateReaders(ContainerTypeContainer, "status.containerStatuses")
		findContainerStateReaders(ContainerTypeInitContainer, "status.initContainerStatuses")
		findContainerStateReaders(ContainerTypeEphemeral, "status.ephemeralContainerStatuses")
	}

	commonLogFieldSet := log.MustGetFieldSet(event.Log, &log.CommonFieldSet{})
	k8sAuditLogFieldSet := log.MustGetFieldSet(event.Log, &commonlogk8saudit_contract.K8sAuditLogFieldSet{})

	for _, identity := range state.containerIdentities {
		if _, found := state.containerStateWalkers[identity.containerName]; !found {
			state.containerStateWalkers[identity.containerName] = &containerStateWalker{
				containerIdentity: identity,
				podNamespace:      event.ResourceIdentity.Namespace,
				podName:           event.ResourceIdentity.Name,
			}
		}
		walker := state.containerStateWalkers[identity.containerName]
		walker.CheckAndRecord(ctx, currentStateReaders[identity.containerName], cs, commonLogFieldSet, k8sAuditLogFieldSet)

		if event.EventType == commonlogk8saudit_contract.ChangeEventTypeV2Deletion {
			containerPath := MustResolveContainerTimelinePath(ctx, k8sAuditLogFieldSet.ClusterName, event.ResourceIdentity.Namespace, event.ResourceIdentity.Name, identity.containerName)
			cs.AddRevision(containerPath, &khifilev6.StagingRevision{
				VerbType:     k8sAuditLogFieldSet.Verb,
				ResourceBody: nil,
				Principal:    k8sAuditLogFieldSet.Principal,
				ChangedTime:  commonLogFieldSet.Timestamp,
				StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceDeleted,
			})
		}
	}

	return cs, state, nil
}

var _ commonlogk8saudit_contract.ManifestLogToTimelineMapper[*containerLogToTimelineMapperTaskState] = (*containerLogToTimelineMapperTaskSetting)(nil)

type containerStateWalker struct {
	// containerIdentity is the identity of the container.
	containerIdentity *containerStatusIdentity
	// podNamespace is the namespace of the pod.
	podNamespace string
	// podName is the name of the pod.
	podName string
	// lastState is the last state of the container.
	lastState string
	// lastStartTime is the last start time of the container.
	lastStartTime string
	// lastFinishTime is the last finish time of the container.
	lastFinishTime string
}

// CheckAndRecord compares the current container state with the previous state and records a revision if there is a significant change.
func (w *containerStateWalker) CheckAndRecord(ctx context.Context, stateReader *structured.NodeReader, cs *khifilev6.TimelineChangeSet, commonLog *log.CommonFieldSet, k8sAuditLog *commonlogk8saudit_contract.K8sAuditLogFieldSet) {
	containerPath := MustResolveContainerTimelinePath(ctx, k8sAuditLog.ClusterName, w.podNamespace, w.podName, w.containerIdentity.containerName)
	if stateReader == nil {
		if w.lastState != "no state" {
			cs.AddRevision(containerPath, &khifilev6.StagingRevision{
				Principal:    k8sAuditLog.Principal,
				VerbType:     k8sAuditLog.Verb,
				ResourceBody: nil,
				ChangedTime:  commonLog.Timestamp,
				StateType:    commonlogk8saudit_contract.RevisionStateContainerStatusNotAvailable,
			})
			w.lastState = "no state"
		}
	} else {
		containerBody := stateReader.Node

		// Get the reason from waiting state
		waiting, err := stateReader.GetReader("state.waiting")
		if err == nil {
			reason, err := waiting.ReadString("reason")
			state := fmt.Sprintf("waiting-%s", reason)
			if err == nil && w.lastState != state {
				cs.AddRevision(containerPath, &khifilev6.StagingRevision{
					Principal:    k8sAuditLog.Principal,
					VerbType:     k8sAuditLog.Verb,
					ResourceBody: containerBody,
					ChangedTime:  commonLog.Timestamp,
					StateType:    commonlogk8saudit_contract.RevisionStateContainerWaiting,
				})
				w.lastState = state
			}
		}

		// Get the reason from running state
		running, err := stateReader.GetReader("state.running")
		if err == nil {
			startTime, err := running.ReadString("startedAt")
			if err == nil && w.lastStartTime != startTime {
				startTimeParsed, err := time.Parse(time.RFC3339, startTime)
				if err == nil {
					cs.AddRevision(containerPath, &khifilev6.StagingRevision{
						Principal:    k8sAuditLog.Principal,
						VerbType:     k8sAuditLog.Verb,
						ResourceBody: containerBody,
						ChangedTime:  startTimeParsed,
						StateType:    commonlogk8saudit_contract.RevisionStateContainerStarted,
					})
					w.lastStartTime = startTime
					w.lastState = "started"
				}
			}
			ready, err := stateReader.ReadBool("ready")
			if err == nil {
				currentState := "ready"
				revisionState := commonlogk8saudit_contract.RevisionStateContainerRunningReady
				if !ready {
					currentState = "not ready"
					revisionState = commonlogk8saudit_contract.RevisionStateContainerRunningNonReady
				}
				if w.lastState != currentState {
					cs.AddRevision(containerPath, &khifilev6.StagingRevision{
						Principal:    k8sAuditLog.Principal,
						VerbType:     k8sAuditLog.Verb,
						ResourceBody: containerBody,
						ChangedTime:  commonLog.Timestamp,
						StateType:    revisionState,
					})
					w.lastState = currentState
				}
			}
		}

		// Get the reason from terminated state
		terminated, err := stateReader.GetReader("state.terminated")
		if err == nil {
			startTime, err := terminated.ReadString("startedAt")
			if err == nil && w.lastStartTime != startTime {
				startTimeParsed, err := time.Parse(time.RFC3339, startTime)
				if err == nil {
					cs.AddRevision(containerPath, &khifilev6.StagingRevision{
						Principal:    k8sAuditLog.Principal,
						VerbType:     k8sAuditLog.Verb,
						ResourceBody: containerBody,
						ChangedTime:  startTimeParsed,
						StateType:    commonlogk8saudit_contract.RevisionStateContainerStarted,
					})
					w.lastStartTime = startTime
				}
			}

			finishTime, err := terminated.ReadString("finishedAt")
			if err == nil && w.lastFinishTime != finishTime {
				finishTimeParsed, err := time.Parse(time.RFC3339, finishTime)
				if err == nil {
					exitCode := terminated.ReadIntOrDefault("exitCode", -1)
					revState := commonlogk8saudit_contract.RevisionStateContainerTerminatedWithSuccess
					if exitCode != 0 {
						revState = commonlogk8saudit_contract.RevisionStateContainerTerminatedWithError
					}
					cs.AddRevision(containerPath, &khifilev6.StagingRevision{
						Principal:    k8sAuditLog.Principal,
						VerbType:     k8sAuditLog.Verb,
						ResourceBody: containerBody,
						ChangedTime:  finishTimeParsed,
						StateType:    revState,
					})
					w.lastFinishTime = finishTime
				}
			}
			w.lastState = "terminated"
		}
	}
}

// MustResolveContainerTimelinePath resolves the timeline path of a container within a pod.
func MustResolveContainerTimelinePath(ctx context.Context, clusterName, namespace, podName, containerName string) *khifilev6.TimelinePath {
	cluster := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	api := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, cluster, "core/v1")
	kind := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, api, "pod")

	var podPath *khifilev6.TimelinePath
	if namespace != "" {
		ns := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kind, namespace)
		podPath = commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, ns, podName)
	} else {
		podPath = commonlogk8saudit_contract.MustK8sClusterScopeResourceTimeline(ctx, kind, podName)
	}

	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(podPath, khifilev6.PathSegment{
		Name: containerName,
		Type: commonlogk8saudit_contract.TimelineTypeContainer,
	})
}
