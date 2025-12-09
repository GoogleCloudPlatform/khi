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
	"slices"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
)

// ConditionHistoryModifierTask is a ManifestHistoryModifier task that tracks and records the history of Kubernetes resource conditions.
// It analyzes status.conditions fields in audit logs to generate revisions for each condition type (e.g., Ready, Scheduled).
var ConditionHistoryModifierTask = commonlogk8sauditv2_contract.NewManifestHistoryModifier[*conditionHistoryModifierTaskState](&conditionHistoryModifierTaskSetting{
	minimumDeltaTimeToCreateInferredCreationRevision: 10 * time.Second,
})

type conditionHistoryModifierTaskState struct {
	// AvailableTypes is the set of available condition types.
	AvailableTypes map[string]struct{}
	// ConditionWalkers is the map of condition walkers.
	ConditionWalkers map[string]*conditionWalker
}

type conditionHistoryModifierTaskSetting struct {
	// minimumDeltaTimeToCreateInferredCreationRevision is the minimum delta time to create inferred creation revision.
	minimumDeltaTimeToCreateInferredCreationRevision time.Duration
}

// Process implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (c *conditionHistoryModifierTaskSetting) Process(ctx context.Context, passIndex int, event commonlogk8sauditv2_contract.ResourceChangeEvent, cs *history.ChangeSet, builder *history.Builder, prevResource *conditionHistoryModifierTaskState) (*conditionHistoryModifierTaskState, error) {
	if event.EventTargetBodyReader == nil {
		return prevResource, nil
	}
	switch passIndex {
	case 0:
		return c.processFirstPass(ctx, event, cs, builder, prevResource)
	case 1:
		return c.processSecondPass(ctx, event, cs, builder, prevResource)
	default:
		panic("unreachable. passIndex should be 0 or 1 for condition history modifier")
	}
}

// TaskID implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (c *conditionHistoryModifierTaskSetting) TaskID() taskid.TaskImplementationID[struct{}] {
	return commonlogk8sauditv2_contract.ConditionHistoryModifierTaskID
}

// ResourcePairs implements commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting.
func (c *conditionHistoryModifierTaskSetting) ResourcePairs(ctx context.Context, groupedLogs commonlogk8sauditv2_contract.ResourceManifestLogGroupMap) ([]commonlogk8sauditv2_contract.ResourcePair, error) {
	result := []commonlogk8sauditv2_contract.ResourcePair{}
	for _, group := range groupedLogs {
		if group.Resource.Type() == commonlogk8sauditv2_contract.Resource {
			result = append(result, commonlogk8sauditv2_contract.ResourcePair{
				TargetGroup: group.Resource,
			})
		}
	}
	return result, nil
}

// processFirstPass collects all available condition types from the log.
// This is necessary because some conditions might appear later in the history, and we need to know about them upfront to track their state correctly.
func (c *conditionHistoryModifierTaskSetting) processFirstPass(ctx context.Context, event commonlogk8sauditv2_contract.ResourceChangeEvent, cs *history.ChangeSet, builder *history.Builder, prevResource *conditionHistoryModifierTaskState) (*conditionHistoryModifierTaskState, error) {
	if prevResource == nil {
		prevResource = &conditionHistoryModifierTaskState{
			AvailableTypes:   map[string]struct{}{},
			ConditionWalkers: map[string]*conditionWalker{},
		}
	}
	if event.EventTargetBodyReader != nil {
		conditionsReader, err := event.EventTargetBodyReader.GetReader("status.conditions")
		if err != nil {
			return prevResource, nil
		}
		for _, child := range conditionsReader.Children() {
			conditionType, err := child.ReadString("type")
			if err == nil {
				prevResource.AvailableTypes[conditionType] = struct{}{}
			}
		}
	}
	return prevResource, nil
}

// processSecondPass generates revisions for each condition type based on the collected available types.
// It handles standard updates, inferred creations (when creation time is missing from the log), and deletions.
func (c *conditionHistoryModifierTaskSetting) processSecondPass(ctx context.Context, event commonlogk8sauditv2_contract.ResourceChangeEvent, cs *history.ChangeSet, builder *history.Builder, state *conditionHistoryModifierTaskState) (*conditionHistoryModifierTaskState, error) {
	commonFieldSet := log.MustGetFieldSet(event.Log, &log.CommonFieldSet{})
	k8sFieldSet := log.MustGetFieldSet(event.Log, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})
	ownerPath := resourcepath.ResourcePath{
		Path:               event.EventTargetResource.ResourcePathString(),
		ParentRelationship: enum.RelationshipChild,
	}
	var resourceContainingStatus model.K8sResourceContainingStatus
	err := structured.ReadReflect(event.EventTargetBodyReader, "", &resourceContainingStatus)
	if err != nil {
		return nil, err
	}
	currentConditions := map[string]*model.K8sResourceStatusCondition{}
	if resourceContainingStatus.Status != nil {
		for _, condition := range resourceContainingStatus.Status.Conditions {
			currentConditions[condition.Type] = condition
		}
	}
	sortedKeys := make([]string, 0, len(state.AvailableTypes))
	for key := range state.AvailableTypes {
		sortedKeys = append(sortedKeys, key)
	}
	slices.Sort(sortedKeys)

	if event.EventType == commonlogk8sauditv2_contract.ChangeEventTypeTargetCreation {
		creationTime, found := GetCreationTimestamp(event.EventTargetBodyReader)
		if found {
			if commonFieldSet.Timestamp.Sub(creationTime) > c.minimumDeltaTimeToCreateInferredCreationRevision {
				// The creation time is not included in the log range.
				for _, key := range sortedKeys {
					statePath := resourcepath.Condition(ownerPath, key)
					cs.AddRevision(statePath, &history.StagingResourceRevision{
						Verb:       k8sFieldSet.K8sOperation.Verb,
						Body:       "# Status information is not available. The creation time is not included in the log range.",
						Partial:    false,
						Requestor:  k8sFieldSet.Principal,
						ChangeTime: creationTime,
						State:      enum.RevisionStateConditionNoAvailableInfo,
					})
				}
			}
		}
	}
	for _, key := range sortedKeys {
		walker := state.ConditionWalkers[key]
		if walker == nil {
			walker = newConditionWalker(ownerPath, key)
			state.ConditionWalkers[key] = walker
		}
		walker.CheckAndRecord(commonFieldSet, k8sFieldSet, currentConditions[key], cs)
	}

	if event.EventType == commonlogk8sauditv2_contract.ChangeEventTypeTargetDeletion {
		for _, key := range sortedKeys {
			walker := state.ConditionWalkers[key]
			if walker == nil {
				walker = newConditionWalker(ownerPath, key)
				state.ConditionWalkers[key] = walker
			}
			walker.RecordDeletion(commonFieldSet.Timestamp.Add(time.Nanosecond))
			statePath := resourcepath.Condition(ownerPath, key)
			cs.AddRevision(statePath, &history.StagingResourceRevision{
				Verb:       k8sFieldSet.K8sOperation.Verb,
				Body:       "",
				Partial:    false,
				Requestor:  k8sFieldSet.Principal,
				ChangeTime: commonFieldSet.Timestamp,
				State:      enum.RevisionStateDeleted,
			})
		}
	}

	return state, nil
}

// Dependencies implements commonlogk8sauditv2_contract.ResourceBasedHistoryModifer.
func (c *conditionHistoryModifierTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// PassCount implements commonlogk8sauditv2_contract.ResourceBasedHistoryModifer.
func (c *conditionHistoryModifierTaskSetting) PassCount() int {
	return 2
}

// GroupedLogTask implements commonlogk8sauditv2_contract.ResourceBasedHistoryModifer.
func (c *conditionHistoryModifierTaskSetting) GroupedLogTask() taskid.TaskReference[commonlogk8sauditv2_contract.ResourceManifestLogGroupMap] {
	return commonlogk8sauditv2_contract.ResourceLifetimeTrackerTaskID.Ref()
}

// LogSerializerTask implements commonlogk8sauditv2_contract.ResourceBasedHistoryModifer.
func (c *conditionHistoryModifierTaskSetting) LogSerializerTask() taskid.TaskReference[[]*log.Log] {
	return commonlogk8sauditv2_contract.K8sAuditLogSerializerTaskID.Ref()
}

var _ commonlogk8sauditv2_contract.ManifestHistoryModifierTaskSetting[*conditionHistoryModifierTaskState] = (*conditionHistoryModifierTaskSetting)(nil)

// conditionStateToRevisionState converts a Kubernetes condition status string ("True", "False", etc.) to a KHI RevisionState enum.
func conditionStateToRevisionState(conditionState string) enum.RevisionState {
	switch conditionState {
	case "True":
		return enum.RevisionStateConditionTrue
	case "False":
		return enum.RevisionStateConditionFalse
	case "":
		return enum.RevisionStateConditionNoAvailableInfo
	default:
		return enum.RevisionStateConditionUnknown
	}
}

type conditionWalker struct {
	// parentResource is the parent resource path.
	parentResource resourcepath.ResourcePath
	// conditionType is the `type` field of the condition.
	conditionType string
	// lastStatus is the last status of the condition.
	lastStatus string
	// lastTransitionTime is the last transition time of the condition.
	lastTransitionTime string
	// lastProbeLikeTime is the last probe like time of the condition.
	lastProbeLikeTime string
	// minChangeTime is the minimum change time.
	// This is used not to create a revision too ealier for the resource retaining the condition after recreation.
	minChangeTime *time.Time
}

// newConditionWalker creates a new conditionWalker for a specific condition type.
func newConditionWalker(parentResource resourcepath.ResourcePath, stateType string) *conditionWalker {
	return &conditionWalker{
		parentResource:     parentResource,
		conditionType:      stateType,
		lastStatus:         "",
		lastTransitionTime: "",
		lastProbeLikeTime:  "",
	}
}

// CheckAndRecord compares the current condition with the previous state and records a revision if there is a significant change.
// It tracks changes in Status, LastTransitionTime, and LastHeartbeatTime (ProbeLikeTime).
func (c *conditionWalker) CheckAndRecord(commonLog *log.CommonFieldSet, k8sAuditLog *commonlogk8sauditv2_contract.K8sAuditLogFieldSet, condition *model.K8sResourceStatusCondition, cs *history.ChangeSet) {
	if condition == nil {
		if c.lastStatus != "n/a" {
			cs.AddRevision(c.conditionPath(), &history.StagingResourceRevision{
				Verb:       k8sAuditLog.K8sOperation.Verb,
				Body:       "",
				Partial:    false,
				Requestor:  k8sAuditLog.Principal,
				ChangeTime: commonLog.Timestamp,
				State:      enum.RevisionStateConditionNotGiven,
			})
			c.minChangeTime = &commonLog.Timestamp
			c.lastStatus = "n/a"
		}
	} else {
		c.lastStatus = condition.Status
		if condition.LastTransitionTime != "" && c.lastTransitionTime != condition.LastTransitionTime {
			transitionTime, err := time.Parse(time.RFC3339, condition.LastTransitionTime)
			if err == nil {
				state := conditionStateToRevisionState(condition.Status)
				body := c.serializeCondition(condition)
				cs.AddRevision(c.conditionPath(), &history.StagingResourceRevision{
					Verb:       k8sAuditLog.K8sOperation.Verb,
					Body:       body,
					Partial:    false,
					Requestor:  k8sAuditLog.Principal,
					ChangeTime: c.clampMinChangeTime(transitionTime),
					State:      state,
				})
				c.lastTransitionTime = condition.LastTransitionTime
			}
		}
		probeLikeTime, err := condition.ProbeLikeTime()
		if err == nil {
			if c.lastProbeLikeTime != probeLikeTime.Format(time.RFC3339) {
				state := conditionStateToRevisionState(condition.Status)
				body := c.serializeCondition(condition)
				cs.AddRevision(c.conditionPath(), &history.StagingResourceRevision{
					Verb:       k8sAuditLog.K8sOperation.Verb,
					Body:       body,
					Partial:    false,
					Requestor:  k8sAuditLog.Principal,
					ChangeTime: c.clampMinChangeTime(probeLikeTime),
					State:      state,
				})
				c.lastProbeLikeTime = probeLikeTime.Format(time.RFC3339)
			}
		}
	}
}

// RecordDeletion records the deletion of the condition.
func (c *conditionWalker) RecordDeletion(deletionTime time.Time) {
	c.lastStatus = ""
	c.lastTransitionTime = ""
	c.lastProbeLikeTime = ""
}

// conditionPath returns the ResourcePath for the specific condition type tracked by this walker.
func (c *conditionWalker) conditionPath() resourcepath.ResourcePath {
	return resourcepath.Condition(c.parentResource, c.conditionType)
}

// serializeCondition serializes the K8sResourceStatusCondition to a YAML string for storage in the revision body.
func (c *conditionWalker) serializeCondition(condition *model.K8sResourceStatusCondition) string {
	var conditionBody string
	conditionNode, err := structured.FromGoValue(condition.ToMap(), &structured.AlphabeticalGoMapKeyOrderProvider{})
	if err == nil {
		reader := structured.NewNodeReader(conditionNode)
		conditionBodyBytes, err := reader.Serialize("", &structured.YAMLNodeSerializer{})
		if err == nil {
			conditionBody = string(conditionBodyBytes)
		}
	}
	return conditionBody
}

// clampMinChangeTime clamps the change time to the minimum change time if it is before the minimum change time.
// This is needed not to write a revision overraps the previous revisions before deletion because some conditions are kept used again after recreation.
// This happens especially in static Pods.
func (c *conditionWalker) clampMinChangeTime(changeTime time.Time) time.Time {
	if c.minChangeTime != nil && changeTime.Before(*c.minChangeTime) {
		return *c.minChangeTime
	}
	return changeTime
}
