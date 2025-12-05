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
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
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

// String returns the string representation of the container identity.
func (c *containerStatusIdentity) String() string {
	switch c.containerType {
	case ContainerTypeContainer:
		return c.containerName
	case ContainerTypeInitContainer:
		return fmt.Sprintf("%s(init)", c.containerName)
	case ContainerTypeEphemeral:
		return fmt.Sprintf("%s(ephemeral)", c.containerName)
	default:
		panic(fmt.Sprintf("unknown container type:%s", c.containerType))
	}
}

// ContainerHistoryModifierTask is the task to generate container history.
var ContainerHistoryModifierTask = commonlogk8sauditv2_contract.NewManifestHistoryModifier[*containerHistoryModifierTaskState](&containerHistoryModifierTaskSetting{})

type containerHistoryModifierTaskState struct {
	// containerIdentities is the map of container identities.
	containerIdentities map[string]*containerStatusIdentity
	// containerStateWalkers is the map of container state walkers.
	containerStateWalkers map[string]*containerStateWalker
}

type containerHistoryModifierTaskSetting struct {
}

// Dependencies implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (c *containerHistoryModifierTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (c *containerHistoryModifierTaskSetting) GroupedLogTask() taskid.TaskReference[commonlogk8sauditv2_contract.ResourceManifestLogGroupMap] {
	return commonlogk8sauditv2_contract.ResourceLifetimeTrackerTaskID.Ref()
}

// LogSerializerTask implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (c *containerHistoryModifierTaskSetting) LogSerializerTask() taskid.TaskReference[[]*log.Log] {
	return commonlogk8sauditv2_contract.K8sAuditLogSerializerTaskID.Ref()
}

// PassCount implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (c *containerHistoryModifierTaskSetting) PassCount() int {
	return 2
}

// Process implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (c *containerHistoryModifierTaskSetting) Process(ctx context.Context, passIndex int, event commonlogk8sauditv2_contract.ResourceChangeEvent, cs *history.ChangeSet, builder *history.Builder, state *containerHistoryModifierTaskState) (*containerHistoryModifierTaskState, error) {
	if state == nil {
		state = &containerHistoryModifierTaskState{
			containerIdentities:   map[string]*containerStatusIdentity{},
			containerStateWalkers: map[string]*containerStateWalker{},
		}
	}
	if event.EventTargetBodyReader == nil {
		return state, nil
	}

	switch passIndex {
	case 0:
		return c.processFirstPass(ctx, event, cs, builder, state)
	case 1:
		return c.processSecondPass(ctx, event, cs, builder, state)
	default:
		return nil, fmt.Errorf("invalid pass index: %d", passIndex)
	}
}

// processFirstPass collects all container identities from the log.
func (c *containerHistoryModifierTaskSetting) processFirstPass(ctx context.Context, event commonlogk8sauditv2_contract.ResourceChangeEvent, cs *history.ChangeSet, builder *history.Builder, state *containerHistoryModifierTaskState) (*containerHistoryModifierTaskState, error) {
	findContainers := func(containerType containerType, fieldName string) {
		statuses, err := event.EventTargetBodyReader.GetReader(fieldName)
		if err == nil {
			for _, status := range statuses.Children() {
				name, err := status.ReadString("name")
				if err == nil {
					identity := &containerStatusIdentity{
						containerName: name,
						containerType: containerType,
					}
					state.containerIdentities[identity.String()] = identity
				}
			}
		}
	}
	findContainers(ContainerTypeContainer, "status.containerStatuses")
	findContainers(ContainerTypeInitContainer, "status.initContainerStatuses")
	findContainers(ContainerTypeEphemeral, "status.ephemeralContainerStatuses")
	return state, nil
}

// processSecondPass generates revisions for each container.
func (c *containerHistoryModifierTaskSetting) processSecondPass(ctx context.Context, event commonlogk8sauditv2_contract.ResourceChangeEvent, cs *history.ChangeSet, builder *history.Builder, state *containerHistoryModifierTaskState) (*containerHistoryModifierTaskState, error) {
	currentStateReaders := map[string]*structured.NodeReader{}
	findContainerStateReaders := func(containerType containerType, fieldName string) {
		statuses, err := event.EventTargetBodyReader.GetReader(fieldName)
		if err == nil {
			for _, status := range statuses.Children() {
				name, err := status.ReadString("name")
				if err == nil {
					identity := containerStatusIdentity{
						containerName: name,
						containerType: containerType,
					}
					currentStateReaders[identity.String()] = &status
				}
			}
		}
	}
	findContainerStateReaders(ContainerTypeContainer, "status.containerStatuses")
	findContainerStateReaders(ContainerTypeInitContainer, "status.initContainerStatuses")
	findContainerStateReaders(ContainerTypeEphemeral, "status.ephemeralContainerStatuses")

	commonLogFieldSet := log.MustGetFieldSet(event.Log, &log.CommonFieldSet{})
	k8sAuditLogFieldSet := log.MustGetFieldSet(event.Log, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})

	for _, identity := range state.containerIdentities {
		if _, found := state.containerStateWalkers[identity.String()]; !found {
			state.containerStateWalkers[identity.String()] = &containerStateWalker{
				containerIdentity: identity,
				podNamespace:      event.EventTargetResource.Namespace,
				podName:           event.EventTargetResource.Name,
			}
		}
		walker := state.containerStateWalkers[identity.String()]
		walker.CheckAndRecord(currentStateReaders[identity.String()], cs, commonLogFieldSet, k8sAuditLogFieldSet)
	}
	return state, nil
}

// TaskID implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (c *containerHistoryModifierTaskSetting) TaskID() taskid.TaskImplementationID[struct{}] {
	return commonlogk8sauditv2_contract.ContainerHistoryModifierTaskID
}

// ResourcePairs implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (c *containerHistoryModifierTaskSetting) ResourcePairs(ctx context.Context, groupedLogs commonlogk8sauditv2_contract.ResourceManifestLogGroupMap) ([]commonlogk8sauditv2_contract.ResourcePair, error) {
	results := []commonlogk8sauditv2_contract.ResourcePair{}
	for _, group := range groupedLogs {
		// core/v1#pod#namespace#podnanme
		if group.Resource.APIVersion == "core/v1" && group.Resource.Kind == "pod" {
			results = append(results, commonlogk8sauditv2_contract.ResourcePair{
				TargetGroup: group.Resource,
			})
		}
	}
	return results, nil
}

var _ commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting[*containerHistoryModifierTaskState] = (*containerHistoryModifierTaskSetting)(nil)

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
func (w *containerStateWalker) CheckAndRecord(stateReader *structured.NodeReader, cs *history.ChangeSet, commonLog *log.CommonFieldSet, k8sAuditLog *commonlogk8sauditv2_contract.K8sAuditLogFieldSet) {
	rp := resourcepath.Container(w.podNamespace, w.podName, w.containerIdentity.String())
	if stateReader == nil {
		if w.lastState != "no state" {
			cs.AddRevision(rp, &history.StagingResourceRevision{
				Requestor:  k8sAuditLog.Principal,
				Verb:       k8sAuditLog.K8sOperation.Verb,
				Body:       "# No state for this container is recorded yet",
				ChangeTime: commonLog.Timestamp,
				State:      enum.RevisionStateContainerStatusNotAvailable,
			})
			w.lastState = "no state"
		}
	} else {
		var containerBody string
		containerBodyRaw, err := stateReader.Serialize("", &structured.YAMLNodeSerializer{})
		if err == nil {
			containerBody = string(containerBodyRaw)
		}

		// Get the reason from waiting state
		waiting, err := stateReader.GetReader("state.waiting")
		if err == nil {
			reason, err := waiting.ReadString("reason")
			state := fmt.Sprintf("waiting-%s", reason)
			if err == nil && w.lastState != state {
				cs.AddRevision(rp, &history.StagingResourceRevision{
					Requestor:  k8sAuditLog.Principal,
					Verb:       k8sAuditLog.K8sOperation.Verb,
					Body:       containerBody,
					ChangeTime: commonLog.Timestamp,
					State:      enum.RevisionStateContainerWaiting,
				})
				w.lastState = state
			}
		}

		// Ge the reason from running state
		running, err := stateReader.GetReader("state.running")
		if err == nil {
			startTime, err := running.ReadString("startedAt")
			if err == nil && w.lastStartTime != startTime {
				startTimeParsed, err := time.Parse(time.RFC3339, startTime)
				if err == nil {
					cs.AddRevision(rp, &history.StagingResourceRevision{
						Requestor:  k8sAuditLog.Principal,
						Verb:       k8sAuditLog.K8sOperation.Verb,
						Body:       containerBody,
						ChangeTime: startTimeParsed,
						State:      enum.RevisionStateContainerStarted,
					})
					w.lastStartTime = startTime
					w.lastState = "started"
				}
			}
			ready, err := stateReader.ReadBool("ready")
			if err == nil {
				currentState := "ready"
				revisionState := enum.RevisionStateContainerRunningReady
				if !ready {
					currentState = "not ready"
					revisionState = enum.RevisionStateContainerRunningNonReady
				}
				if w.lastState != currentState {

					cs.AddRevision(rp, &history.StagingResourceRevision{
						Requestor:  k8sAuditLog.Principal,
						Verb:       k8sAuditLog.K8sOperation.Verb,
						Body:       containerBody,
						ChangeTime: commonLog.Timestamp,
						State:      revisionState,
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
					cs.AddRevision(rp, &history.StagingResourceRevision{
						Requestor:  k8sAuditLog.Principal,
						Verb:       k8sAuditLog.K8sOperation.Verb,
						Body:       containerBody,
						ChangeTime: startTimeParsed,
						State:      enum.RevisionStateContainerStarted,
					})
					w.lastStartTime = startTime
				}
			}

			finishTime, err := terminated.ReadString("finishedAt")
			if err == nil && w.lastFinishTime != finishTime {
				finishTimeParsed, err := time.Parse(time.RFC3339, finishTime)
				if err == nil {
					exitCode := terminated.ReadIntOrDefault("exitCode", -1)
					revState := enum.RevisionStateContainerTerminatedWithSuccess
					if exitCode != 0 {
						revState = enum.RevisionStateContainerTerminatedWithError
					}
					cs.AddRevision(rp, &history.StagingResourceRevision{
						Requestor:  k8sAuditLog.Principal,
						Verb:       k8sAuditLog.K8sOperation.Verb,
						Body:       containerBody,
						ChangeTime: finishTimeParsed,
						State:      revState,
					})
					w.lastFinishTime = finishTime
				}
			}
			w.lastState = "terminated"
		}
	}
}
