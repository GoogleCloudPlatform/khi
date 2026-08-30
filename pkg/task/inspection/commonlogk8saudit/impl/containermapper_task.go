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

// ContainerLogToTimelineMapperTask is the task to generate container history.
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

var (
	pathStateWaiting    = structured.CompileFieldPath("state.waiting")
	pathReason          = structured.CompileFieldPath("reason")
	pathStateRunning    = structured.CompileFieldPath("state.running")
	pathStartedAt       = structured.CompileFieldPath("startedAt")
	pathReady           = structured.CompileFieldPath("ready")
	pathStateTerminated = structured.CompileFieldPath("state.terminated")
	pathFinishedAt      = structured.CompileFieldPath("finishedAt")
	pathExitCode        = structured.CompileFieldPath("exitCode")
)

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

	findContainers := func(containerType containerType, fieldPath structured.FieldPath) {
		statuses, err := bodyReader.GetReader(fieldPath)
		if err == nil {
			statuses.Children()(func(key structured.NodeChildrenKey, status structured.NodeReader) bool {
				name, err := status.ReadString(pathContainerName)
				if err == nil {
					identity := &containerStatusIdentity{
						containerName: name,
						containerType: containerType,
					}
					state.containerIdentities[identity.containerName] = identity
				}
				return true
			})
		}
	}
	findContainers(ContainerTypeContainer, pathContainerStatuses)
	findContainers(ContainerTypeInitContainer, pathInitContainerStatuses)
	findContainers(ContainerTypeEphemeral, pathEphemeralContainerStatuses)

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
	cs := khifilev6.NewTimelineChangeSet(event.Log)
	if event.GroupRole != "pod" {
		return cs, state, nil
	}
	k8sFieldSet, _ := commonlogk8saudit_contract.ExtractK8sAuditLog(ctx, event.Log.NodeReader)
	if k8sFieldSet.IsDryRun {
		return cs, state, nil
	}
	bodyReader, hasBody := event.GetLastBodyReader("pod")

	currentStateReaders := map[string]*structured.NodeReader{}
	if hasBody && bodyReader != nil {
		findContainerStateReaders := func(containerType containerType, fieldPath structured.FieldPath) {
			statuses, err := bodyReader.GetReader(fieldPath)
			if err == nil {
				statuses.Children()(func(key structured.NodeChildrenKey, status structured.NodeReader) bool {
					name, err := status.ReadString(pathContainerName)
					if err == nil {
						s := status
						currentStateReaders[name] = &s
					}
					return true
				})
			}
		}
		findContainerStateReaders(ContainerTypeContainer, pathContainerStatuses)
		findContainerStateReaders(ContainerTypeInitContainer, pathInitContainerStatuses)
		findContainerStateReaders(ContainerTypeEphemeral, pathEphemeralContainerStatuses)
	}

	for _, identity := range state.containerIdentities {
		if _, found := state.containerStateWalkers[identity.containerName]; !found {
			state.containerStateWalkers[identity.containerName] = &containerStateWalker{
				containerIdentity: identity,
				podNamespace:      event.ResourceIdentity.Namespace,
				podName:           event.ResourceIdentity.Name,
			}
		}
		walker := state.containerStateWalkers[identity.containerName]
		walker.CheckAndRecord(ctx, currentStateReaders[identity.containerName], cs, event.Log.Timestamp, k8sFieldSet)

		if event.EventType == commonlogk8saudit_contract.ChangeEventTypeDeletion {
			containerPath := MustResolveContainerTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity.Namespace, event.ResourceIdentity.Name, identity.containerName)
			cs.AddRevision(containerPath, &khifilev6.StagingRevision{
				VerbType:     k8sFieldSet.Verb,
				ResourceBody: nil,
				Principal:    k8sFieldSet.Principal,
				ChangedTime:  event.Log.Timestamp,
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
func (w *containerStateWalker) CheckAndRecord(ctx context.Context, stateReader *structured.NodeReader, cs *khifilev6.TimelineChangeSet, changedTime time.Time, k8sAuditLog commonlogk8saudit_contract.K8sAuditLogFieldSet) {
	containerPath := MustResolveContainerTimelinePath(ctx, k8sAuditLog.ClusterName, w.podNamespace, w.podName, w.containerIdentity.containerName)
	if stateReader == nil {
		if w.lastState != "no state" {
			cs.AddRevision(containerPath, &khifilev6.StagingRevision{
				Principal:    k8sAuditLog.Principal,
				VerbType:     k8sAuditLog.Verb,
				ResourceBody: nil,
				ChangedTime:  changedTime,
				StateType:    commonlogk8saudit_contract.RevisionStateContainerStatusNotAvailable,
			})
			w.lastState = "no state"
		}
	} else {
		containerBody := stateReader.Node

		// Get the reason from waiting state
		waiting, err := stateReader.GetReader(pathStateWaiting)
		if err == nil {
			reason, err := waiting.ReadString(pathReason)
			state := fmt.Sprintf("waiting-%s", reason)
			if err == nil && w.lastState != state {
				cs.AddRevision(containerPath, &khifilev6.StagingRevision{
					Principal:    k8sAuditLog.Principal,
					VerbType:     k8sAuditLog.Verb,
					ResourceBody: containerBody,
					ChangedTime:  changedTime,
					StateType:    commonlogk8saudit_contract.RevisionStateContainerWaiting,
				})
				w.lastState = state
			}
		}

		// Get the reason from running state
		running, err := stateReader.GetReader(pathStateRunning)
		if err == nil {
			startTime, err := running.ReadString(pathStartedAt)
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
			ready, err := stateReader.ReadBool(pathReady)
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
						ChangedTime:  changedTime,
						StateType:    revisionState,
					})
					w.lastState = currentState
				}
			}
		}

		// Get the reason from terminated state
		terminated, err := stateReader.GetReader(pathStateTerminated)
		if err == nil {
			startTime, err := terminated.ReadString(pathStartedAt)
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

			finishTime, err := terminated.ReadString(pathFinishedAt)
			if err == nil && w.lastFinishTime != finishTime {
				finishTimeParsed, err := time.Parse(time.RFC3339, finishTime)
				if err == nil {
					exitCode := terminated.ReadIntOrDefault(pathExitCode, -1)
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
