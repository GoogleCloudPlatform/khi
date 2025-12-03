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
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
)

var phaseToState = map[string]enum.RevisionState{
	"Pending":   enum.RevisionStatePodPhasePending,
	"Running":   enum.RevisionStatePodPhaseRunning,
	"Succeeded": enum.RevisionStatePodPhaseSucceeded,
	"Failed":    enum.RevisionStatePodPhaseFailed,
	"Unknown":   enum.RevisionStatePodPhaseUnknown,
}

type podPhaseTaskState struct {
	// lastPhase is the last phase of the pod.
	lastPhase string
	// lastNode is the last node of the pod.
	lastNode string
	// uidToNodeNameMap is the map of UID to node name.
	uidToNodeNameMap map[string]string
}

type podPhaseTaskSetting struct {
	minimumDeltaTimeToCreateInferredCreationRevision time.Duration
}

// Process processes the log to generate pod phase history.
func (c *podPhaseTaskSetting) Process(ctx context.Context, passIndex int, event commonlogk8sauditv2_contract.ResourceChangeEvent, cs *history.ChangeSet, builder *history.Builder, state *podPhaseTaskState) (*podPhaseTaskState, error) {
	if state == nil {
		state = &podPhaseTaskState{
			uidToNodeNameMap: map[string]string{},
		}
	}
	if event.EventTargetBodyReader == nil {
		return state, nil
	}

	switch passIndex {
	case 0:
		return c.firstPass(ctx, event, cs, builder, state)
	case 1:
		return c.secondPass(ctx, event, cs, builder, state)
	default:
		return state, nil
	}
}

// firstPass collects the node name of the pod.
func (c *podPhaseTaskSetting) firstPass(ctx context.Context, event commonlogk8sauditv2_contract.ResourceChangeEvent, cs *history.ChangeSet, builder *history.Builder, state *podPhaseTaskState) (*podPhaseTaskState, error) {
	nodeName, found := GetNodeNameOfPod(event.EventTargetBodyReader)
	if !found {
		return state, nil
	}
	uid, _ := GetUID(event.EventTargetBodyReader)

	if nodeName != "" && uid != "" {
		state.uidToNodeNameMap[uid] = nodeName
	}
	return state, nil
}

// secondPass generates revisions for pod phase.
func (c *podPhaseTaskSetting) secondPass(ctx context.Context, event commonlogk8sauditv2_contract.ResourceChangeEvent, cs *history.ChangeSet, builder *history.Builder, state *podPhaseTaskState) (*podPhaseTaskState, error) {
	commonLogFieldSet := log.MustGetFieldSet(event.Log, &log.CommonFieldSet{})
	k8sFieldSet := log.MustGetFieldSet(event.Log, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})
	uid, found := GetUID(event.EventTargetBodyReader)
	if !found {
		return state, nil
	}
	nodeName, found := state.uidToNodeNameMap[uid]
	if !found {
		return state, nil
	}
	if event.EventType == commonlogk8sauditv2_contract.ChangeEventTypeTargetCreation {
		creationTime, found := GetCreationTimestamp(event.EventTargetBodyReader)
		if found && commonLogFieldSet.Timestamp.Sub(creationTime) > c.minimumDeltaTimeToCreateInferredCreationRevision {
			cs.AddRevision(resourcepath.PodPhase(nodeName, event.EventTargetResource.Namespace, event.EventTargetResource.Name, uid), &history.StagingResourceRevision{
				Verb:       enum.RevisionVerbUnknown,
				Body:       "# Pod exists during this period but no body information available",
				Partial:    false,
				Requestor:  "N/A",
				ChangeTime: creationTime,
				State:      enum.RevisionStatePodPhaseUnknown,
			})
		}
	}
	if event.EventType == commonlogk8sauditv2_contract.ChangeEventTypeSourceCreation {
		cs.AddRevision(resourcepath.PodPhase(nodeName, event.EventTargetResource.Namespace, event.EventTargetResource.Name, uid), &history.StagingResourceRevision{
			Verb:       k8sFieldSet.K8sOperation.Verb,
			Body:       event.EventTargetBodyYAML,
			Partial:    false,
			Requestor:  k8sFieldSet.Principal,
			ChangeTime: commonLogFieldSet.Timestamp,
			State:      enum.RevisionStatePodPhaseScheduled,
		})
		return state, nil
	}

	phase, found := GetPodPhase(event.EventTargetBodyReader)
	if !found {
		return state, nil
	}
	if state.lastPhase != phase || state.lastNode != nodeName {
		cs.AddRevision(resourcepath.PodPhase(nodeName, event.EventTargetResource.Namespace, event.EventTargetResource.Name, uid), &history.StagingResourceRevision{
			Verb:       k8sFieldSet.K8sOperation.Verb,
			Body:       event.EventTargetBodyYAML,
			Partial:    false,
			Requestor:  k8sFieldSet.Principal,
			ChangeTime: commonLogFieldSet.Timestamp,
			State:      phaseToState[phase],
		})
	}
	state.lastPhase = phase
	state.lastNode = nodeName
	return state, nil
}

func (c *podPhaseTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

func (c *podPhaseTaskSetting) PassCount() int {
	return 2
}

func (c *podPhaseTaskSetting) GroupedLogTask() taskid.TaskReference[commonlogk8sauditv2_contract.ResourceManifestLogGroupMap] {
	return commonlogk8sauditv2_contract.ResourceLifetimeTrackerTaskID.Ref()
}

func (c *podPhaseTaskSetting) LogSerializerTask() taskid.TaskReference[[]*log.Log] {
	return commonlogk8sauditv2_contract.K8sAuditLogSerializerTaskID.Ref()
}

func (c *podPhaseTaskSetting) TaskID() taskid.TaskImplementationID[struct{}] {
	return commonlogk8sauditv2_contract.PodPhaseHistoryModifierTaskID
}

func (c *podPhaseTaskSetting) ResourcePairs(ctx context.Context, groupedLogs commonlogk8sauditv2_contract.ResourceManifestLogGroupMap) ([]commonlogk8sauditv2_contract.ResourcePair, error) {
	result := []commonlogk8sauditv2_contract.ResourcePair{}
	for _, group := range groupedLogs {
		// core/v1#pod#namespace#podnanme
		if group.Resource.Type() != commonlogk8sauditv2_contract.Resource || group.Resource.APIVersion != "core/v1" || group.Resource.Kind != "pod" {
			continue
		}
		result = append(result, commonlogk8sauditv2_contract.ResourcePair{
			TargetGroup: group.Resource,
			SourceGroup: group.Resource.SubresourceIdentity("binding"),
		})
	}
	return result, nil
}

var _ commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting[*podPhaseTaskState] = (*podPhaseTaskSetting)(nil)

// PodPhaseHistoryModifierTask is the task to generate pod phase history.
var PodPhaseHistoryModifierTask = commonlogk8sauditv2_contract.NewManifestHistoryModifier[*podPhaseTaskState](&podPhaseTaskSetting{
	minimumDeltaTimeToCreateInferredCreationRevision: 5 * time.Second,
})
