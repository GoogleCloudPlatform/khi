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
	"log/slog"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
)

type resourceRevisionHistoryModifierState struct {
	// WasCompletelyRemoved is true if the resource was completely removed.
	WasCompletelyRemoved bool
	// DeletionStarted is true if the deletion started.
	DeletionStarted bool
	// PrevUID is the previous UID of the resource.
	PrevUID string
}

// ResourceRevisionHistoryModifierTaskSetting is the setting for the resource revision history modifier task.
type ResourceRevisionHistoryModifierTaskSetting struct {
	// minimumDeltaTimeToCreateInferredCreationRevision is a threshold of a duration that controls if KHI should create an inferred cretion revision from creationTimestamp.
	minimumDeltaTimeToCreateInferredCreationRevision time.Duration
	// kindsToWaitExactDeletionToDeterminDeletion is the map of kinds to wait exact deletion to determine deletion.
	kindsToWaitExactDeletionToDeterminDeletion map[string]struct{}
}

// Dependencies implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (r *ResourceRevisionHistoryModifierTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (r *ResourceRevisionHistoryModifierTaskSetting) GroupedLogTask() taskid.TaskReference[commonlogk8sauditv2_contract.ResourceManifestLogGroupMap] {
	return commonlogk8sauditv2_contract.ResourceLifetimeTrackerTaskID.Ref()
}

// LogSerializerTask implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (r *ResourceRevisionHistoryModifierTaskSetting) LogSerializerTask() taskid.TaskReference[[]*log.Log] {
	return commonlogk8sauditv2_contract.K8sAuditLogSerializerTaskID.Ref()
}

// Process implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (r *ResourceRevisionHistoryModifierTaskSetting) Process(ctx context.Context, passIndex int, event commonlogk8sauditv2_contract.ResourceChangeEvent, cs *history.ChangeSet, builder *history.Builder, prevGroupData *resourceRevisionHistoryModifierState) (*resourceRevisionHistoryModifierState, error) {
	switch event.EventType {
	case commonlogk8sauditv2_contract.ChangeEventTypeSourceDeletion:
		return &resourceRevisionHistoryModifierState{}, r.handleParentChangeForSubresource(ctx, event, cs)
	case commonlogk8sauditv2_contract.ChangeEventTypeSourceModification:
		return &resourceRevisionHistoryModifierState{}, r.handleParentChangeForSubresource(ctx, event, cs)
	case commonlogk8sauditv2_contract.ChangeEventTypeSourceCreation:
		return &resourceRevisionHistoryModifierState{}, r.handleParentChangeForSubresource(ctx, event, cs)
	default:
		return r.handleTargetChange(ctx, event, cs, prevGroupData)
	}
}

// PassCount implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (r *ResourceRevisionHistoryModifierTaskSetting) PassCount() int {
	return 1
}

// TaskID implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (r *ResourceRevisionHistoryModifierTaskSetting) TaskID() taskid.TaskImplementationID[struct{}] {
	return commonlogk8sauditv2_contract.ResourceRevisionHistoryModifierTaskID
}

// ResourcePairs implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (r *ResourceRevisionHistoryModifierTaskSetting) ResourcePairs(ctx context.Context, groupedLogs commonlogk8sauditv2_contract.ResourceManifestLogGroupMap) ([]commonlogk8sauditv2_contract.ResourcePair, error) {
	result := []commonlogk8sauditv2_contract.ResourcePair{}
	for _, group := range groupedLogs {
		switch group.Resource.Type() {
		case commonlogk8sauditv2_contract.Namespace:
			continue
		case commonlogk8sauditv2_contract.Resource:
			result = append(result, commonlogk8sauditv2_contract.ResourcePair{
				TargetGroup: group.Resource,
			})
			continue
		case commonlogk8sauditv2_contract.Subresource:
			result = append(result, commonlogk8sauditv2_contract.ResourcePair{
				SourceGroup: group.Resource.ParentIdentity(),
				TargetGroup: group.Resource,
			})
		default:
			panic(fmt.Sprintf("unknown resource type: %v", group.Resource.Type()))
		}
	}
	return result, nil
}

var _ commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting[*resourceRevisionHistoryModifierState] = (*ResourceRevisionHistoryModifierTaskSetting)(nil)

// ResourceRevisionHistoryModifierTask is the task to generate resource revision history.
var ResourceRevisionHistoryModifierTask = commonlogk8sauditv2_contract.NewManifestHistoryModifier(&ResourceRevisionHistoryModifierTaskSetting{
	minimumDeltaTimeToCreateInferredCreationRevision: 5 * time.Second,
	kindsToWaitExactDeletionToDeterminDeletion: map[string]struct{}{
		"core/v1#pod": {},
	},
})

// handleParentChangeForSubresource handles the parent change for subresource.
func (r *ResourceRevisionHistoryModifierTaskSetting) handleParentChangeForSubresource(ctx context.Context, event commonlogk8sauditv2_contract.ResourceChangeEvent, cs *history.ChangeSet) error {
	switch event.EventType {
	case commonlogk8sauditv2_contract.ChangeEventTypeSourceDeletion:
		path := resourcepath.ResourcePath{
			Path:               event.EventTargetResource.ResourcePathString(),
			ParentRelationship: enum.RelationshipChild,
		}
		commonLogFieldSet := log.MustGetFieldSet(event.Log, &log.CommonFieldSet{})
		k8sFieldSet := log.MustGetFieldSet(event.Log, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})
		cs.AddRevision(path, &history.StagingResourceRevision{
			Verb:       enum.RevisionVerbDelete,
			Requestor:  k8sFieldSet.Principal,
			ChangeTime: commonLogFieldSet.Timestamp,
			Body:       event.EventTargetBodyYAML,
			State:      enum.RevisionStateDeleted,
		})
		return nil
	case commonlogk8sauditv2_contract.ChangeEventTypeSourceModification:
		return nil
	case commonlogk8sauditv2_contract.ChangeEventTypeSourceCreation:
		return nil
	default:
		slog.WarnContext(ctx, "unknown event type", "eventType", event.EventType)
		return nil
	}
}

// handleTargetChange handles the target change.
func (r *ResourceRevisionHistoryModifierTaskSetting) handleTargetChange(ctx context.Context, event commonlogk8sauditv2_contract.ResourceChangeEvent, cs *history.ChangeSet, prevGroupData *resourceRevisionHistoryModifierState) (*resourceRevisionHistoryModifierState, error) {
	commonFieldSet := log.MustGetFieldSet(event.Log, &log.CommonFieldSet{})
	k8sFieldSet := log.MustGetFieldSet(event.Log, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})
	resourcePath := resourcepath.ResourcePath{
		Path:               event.EventTargetResource.ResourcePathString(),
		ParentRelationship: enum.RelationshipChild,
	}
	if prevGroupData == nil {
		prevGroupData = &resourceRevisionHistoryModifierState{}
	}

	if k8sFieldSet.K8sOperation.Verb == enum.RevisionVerbDeleteCollection && prevGroupData.WasCompletelyRemoved {
		return prevGroupData, nil
	}

	state := enum.RevisionStateExisting
	if event.EventTargetBodyReader == nil {
		if isDeletiveVerb(k8sFieldSet.K8sOperation.Verb) {
			prevGroupData.DeletionStarted = true
			state = enum.RevisionStateDeleted
		}
	} else {
		deletionStarted := false
		underGracefulPeriod := false
		deletionCompleted := false
		uid, _ := GetUID(event.EventTargetBodyReader)
		if uid != prevGroupData.PrevUID {
			prevGroupData.PrevUID = uid
			prevGroupData.DeletionStarted = false
			prevGroupData.WasCompletelyRemoved = false
		} else {
			deletionStarted = prevGroupData.DeletionStarted
			deletionCompleted = prevGroupData.WasCompletelyRemoved
		}

		if isDeletiveVerb(k8sFieldSet.K8sOperation.Verb) {
			prevGroupData.DeletionStarted = true
			deletionStarted = true
			if isPod(k8sFieldSet.K8sOperation) {
				phase, _ := GetPodPhase(event.EventTargetBodyReader)
				switch phase {
				case "Failed", "Succeeded":
					deletionCompleted = true
				default:
					underGracefulPeriod = true
				}
			}
		}
		deletionGracefulPeriods, found := GetDeletionGracePeriodSeconds(event.EventTargetBodyReader)
		if found {
			if deletionGracefulPeriods > 0 {
				underGracefulPeriod = true
			}
			if deletionGracefulPeriods == 0 {
				deletionCompleted = true
			}
			deletionStarted = true
		}

		finalizers, found := GetFinalizers(event.EventTargetBodyReader)
		if found && len(finalizers) > 0 && deletionStarted {
			deletionCompleted = false
			underGracefulPeriod = true
		}

		_, found = GetDeletionTimestamp(event.EventTargetBodyReader)
		if found {
			deletionStarted = true
			if !underGracefulPeriod { // if the graceful period seconds wasn't found and become zero, then the resource is deleted.
				deletionCompleted = true
			}
		}

		if k8sFieldSet.K8sOperation.Verb == enum.RevisionVerbPatch && state == enum.RevisionStateExisting {
			if prevGroupData.DeletionStarted {
				state = enum.RevisionStateDeleting
			}
			if prevGroupData.WasCompletelyRemoved {
				state = enum.RevisionStateDeleted
			}
		}

		switch {
		case deletionCompleted:
			prevGroupData.WasCompletelyRemoved = true
			prevGroupData.DeletionStarted = false
			state = enum.RevisionStateDeleted
		case underGracefulPeriod:
			prevGroupData.WasCompletelyRemoved = false
			prevGroupData.DeletionStarted = true
			state = enum.RevisionStateDeleting
		case deletionStarted: // if the resource is deleting but gracefulPeriod wasn't found, then it's a resource not supporting graceful deletion.
			prevGroupData.WasCompletelyRemoved = false
			prevGroupData.DeletionStarted = true
			apiVersionKind := fmt.Sprintf("%s#%s", k8sFieldSet.K8sOperation.APIVersion, k8sFieldSet.K8sOperation.GetSingularKindName())
			if _, found := r.kindsToWaitExactDeletionToDeterminDeletion[apiVersionKind]; !found {
				state = enum.RevisionStateDeleted
			}
		default:
			prevGroupData.WasCompletelyRemoved = false
			prevGroupData.DeletionStarted = false
		}
	}
	creationTime, found := GetCreationTimestamp(event.EventTargetBodyReader)
	if !found {
		creationTime = commonFieldSet.Timestamp
	}
	if event.EventType == commonlogk8sauditv2_contract.ChangeEventTypeTargetCreation && commonFieldSet.Timestamp.Sub(creationTime) > r.minimumDeltaTimeToCreateInferredCreationRevision {
		cs.AddRevision(resourcePath, &history.StagingResourceRevision{
			Verb:       enum.RevisionVerbCreate,
			Requestor:  "N/A",
			ChangeTime: creationTime,
			Body:       "# Resource creation seems to happen at the creationTime written in the later log, but the creation request wasn't found during the queried log period.",
			State:      enum.RevisionStateInferred,
		})
	}

	cs.AddRevision(resourcePath, &history.StagingResourceRevision{
		Verb:       k8sFieldSet.K8sOperation.Verb,
		Requestor:  k8sFieldSet.Principal,
		ChangeTime: commonFieldSet.Timestamp,
		Body:       event.EventTargetBodyYAML,
		State:      state,
	})
	return prevGroupData, nil
}
