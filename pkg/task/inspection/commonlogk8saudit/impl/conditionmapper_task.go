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
	"slices"
	"sort"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// ConditionLogToTimelineMapperTask is a ManifestLogToTimelineMapperV2 task that tracks and records the history of Kubernetes resource conditions.
// It analyzes status.conditions fields in audit logs to generate revisions for each condition type (e.g., Ready, Scheduled).
var ConditionLogToTimelineMapperTask = commonlogk8saudit_contract.NewManifestLogToTimelineMapperV2[*conditionLogToTimelineMapperTaskStateV2](&conditionLogToTimelineMapperTaskSettingV2{
	minimumDeltaTimeToCreateInferredCreationRevision: 10 * time.Second,
})

// conditionLogToTimelineMapperTaskStateV2 tracks the status of all conditions of a resource during V2 timeline generation.
type conditionLogToTimelineMapperTaskStateV2 struct {
	// AvailableTypes is the set of available condition types.
	AvailableTypes map[string]struct{}
	// ConditionWalkers is the map of condition walkers.
	ConditionWalkers map[string]*conditionWalkerV2
}

// conditionLogToTimelineMapperTaskSettingV2 maps resource status conditions to timeline revisions under the V2 model.
type conditionLogToTimelineMapperTaskSettingV2 struct {
	// minimumDeltaTimeToCreateInferredCreationRevision is the minimum duration to controls if KHI should create an inferred creation revision.
	minimumDeltaTimeToCreateInferredCreationRevision time.Duration
}

// Dependencies implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *conditionLogToTimelineMapperTaskSettingV2) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// PassCount implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *conditionLogToTimelineMapperTaskSettingV2) PassCount() int {
	return 1
}

// GroupedLogTask implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *conditionLogToTimelineMapperTaskSettingV2) GroupedLogTask() taskid.TaskReference[commonlogk8saudit_contract.ResourceManifestLogGroupMap] {
	return commonlogk8saudit_contract.ResourceLifetimeTrackerTaskID.Ref()
}

// LogIngesterTask implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *conditionLogToTimelineMapperTaskSettingV2) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return commonlogk8saudit_contract.K8sAuditLogIngesterTaskID.Ref()
}

// TaskID implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *conditionLogToTimelineMapperTaskSettingV2) TaskID() taskid.TaskImplementationID[struct{}] {
	return commonlogk8saudit_contract.ConditionLogToTimelineMapperTaskID
}

// ResolveRelatedGroupSets implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *conditionLogToTimelineMapperTaskSettingV2) ResolveRelatedGroupSets(ctx context.Context, groupedLogs commonlogk8saudit_contract.ResourceManifestLogGroupMap) ([]commonlogk8saudit_contract.RelatedGroupSet, error) {
	result := []commonlogk8saudit_contract.RelatedGroupSet{}
	for _, group := range groupedLogs {
		if group.Resource.Type() == commonlogk8saudit_contract.Resource {
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
func (c *conditionLogToTimelineMapperTaskSettingV2) PreProcessLog(ctx context.Context, passIndex int, event commonlogk8saudit_contract.MultiGroupLogEvent, state *conditionLogToTimelineMapperTaskStateV2) (*conditionLogToTimelineMapperTaskStateV2, error) {
	if state == nil {
		state = &conditionLogToTimelineMapperTaskStateV2{
			AvailableTypes:   map[string]struct{}{},
			ConditionWalkers: map[string]*conditionWalkerV2{},
		}
	}

	bodyReader, hasBody := event.GetLastBodyReader("target")
	if !hasBody || bodyReader == nil {
		return state, nil
	}

	commonFieldSet := log.MustGetFieldSet(event.Log, &log.CommonFieldSet{})
	k8sFieldSet := log.MustGetFieldSet(event.Log, &commonlogk8saudit_contract.K8sAuditLogFieldSet{})

	conditionsReader, err := bodyReader.GetReader("status.conditions")
	if err != nil {
		return state, nil
	}

	ownerPath := MustResolveTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity)

	for _, child := range conditionsReader.Children() {
		conditionType, err := child.ReadString("type")
		if err == nil {
			state.AvailableTypes[conditionType] = struct{}{}
			walker := state.ConditionWalkers[conditionType]
			if walker == nil {
				conditionPath := MustK8sConditionTimeline(ctx, ownerPath, conditionType)
				walker = newConditionWalkerV2(conditionPath, conditionType)
				state.ConditionWalkers[conditionType] = walker
			}
			var condition model.K8sResourceStatusCondition
			if err := structured.ReadReflect(&child, "", &condition); err != nil {
				continue
			}
			walker.checkLastTransitionTimes(commonFieldSet, k8sFieldSet, &condition)
		}
	}

	return state, nil
}

// ProcessLog implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *conditionLogToTimelineMapperTaskSettingV2) ProcessLog(ctx context.Context, event commonlogk8saudit_contract.MultiGroupLogEvent, state *conditionLogToTimelineMapperTaskStateV2) (*khifilev6.TimelineChangeSet, *conditionLogToTimelineMapperTaskStateV2, error) {
	if state == nil {
		state = &conditionLogToTimelineMapperTaskStateV2{
			AvailableTypes:   map[string]struct{}{},
			ConditionWalkers: map[string]*conditionWalkerV2{},
		}
	}

	cs := khifilev6.NewTimelineChangeSet(event.Log)

	commonFieldSet := log.MustGetFieldSet(event.Log, &log.CommonFieldSet{})
	k8sFieldSet := log.MustGetFieldSet(event.Log, &commonlogk8saudit_contract.K8sAuditLogFieldSet{})
	ownerPath := MustResolveTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity)

	bodyReader, hasBody := event.GetLastBodyReader("target")

	var resourceContainingStatus model.K8sResourceContainingStatus
	if hasBody && bodyReader != nil {
		err := structured.ReadReflect(bodyReader, "", &resourceContainingStatus)
		if err != nil {
			return nil, nil, err
		}
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

	if event.EventType == commonlogk8saudit_contract.ChangeEventTypeV2Creation {
		creationTime, found := GetCreationTimestamp(bodyReader)
		if found {
			if commonFieldSet.Timestamp.Sub(creationTime) > c.minimumDeltaTimeToCreateInferredCreationRevision {
				// The creation time is not included in the log range.
				for _, key := range sortedKeys {
					walker := state.ConditionWalkers[key]
					if walker == nil {
						conditionPath := MustK8sConditionTimeline(ctx, ownerPath, key)
						walker = newConditionWalkerV2(conditionPath, key)
						state.ConditionWalkers[key] = walker
					}
					cs.AddRevision(walker.conditionPath, &khifilev6.StagingRevision{
						VerbType:     k8sFieldSet.Verb,
						ResourceBody: nil,
						Principal:    k8sFieldSet.Principal,
						ChangedTime:  creationTime,
						StateType:    commonlogk8saudit_contract.RevisionStateConditionNoAvailableInfo,
					})
				}
			}
		}
	}

	for _, key := range sortedKeys {
		walker := state.ConditionWalkers[key]
		if walker == nil {
			conditionPath := MustK8sConditionTimeline(ctx, ownerPath, key)
			walker = newConditionWalkerV2(conditionPath, key)
			state.ConditionWalkers[key] = walker
		}
		walker.CheckAndRecord(ctx, commonFieldSet, k8sFieldSet, currentConditions[key], cs)
	}

	if event.EventType == commonlogk8saudit_contract.ChangeEventTypeV2Deletion {
		for _, key := range sortedKeys {
			walker := state.ConditionWalkers[key]
			if walker == nil {
				conditionPath := MustK8sConditionTimeline(ctx, ownerPath, key)
				walker = newConditionWalkerV2(conditionPath, key)
				state.ConditionWalkers[key] = walker
			}
			walker.RecordDeletion(commonFieldSet.Timestamp.Add(time.Nanosecond))
			cs.AddRevision(walker.conditionPath, &khifilev6.StagingRevision{
				VerbType:     k8sFieldSet.Verb,
				ResourceBody: nil,
				Principal:    k8sFieldSet.Principal,
				ChangedTime:  commonFieldSet.Timestamp,
				StateType:    commonlogk8saudit_contract.RevisionStateConditionNotGiven,
			})
		}
	}

	return cs, state, nil
}

// Explicit interface compliance assertion.
var _ commonlogk8saudit_contract.ManifestLogToTimelineMapperV2[*conditionLogToTimelineMapperTaskStateV2] = (*conditionLogToTimelineMapperTaskSettingV2)(nil)

// conditionStateToRevisionState converts a Kubernetes condition status string ("True", "False", etc.) to a KHI RevisionState enum.
func conditionStateToRevisionState(conditionState string) *pb.RevisionState {
	switch conditionState {
	case "True":
		return commonlogk8saudit_contract.RevisionStateConditionTrue
	case "False":
		return commonlogk8saudit_contract.RevisionStateConditionFalse
	case "":
		return commonlogk8saudit_contract.RevisionStateConditionNoAvailableInfo
	default:
		return commonlogk8saudit_contract.RevisionStateConditionUnknown
	}
}

// conditionWalkerV2 tracks revision generation for a single condition type.
type conditionWalkerV2 struct {
	// conditionPath is the timeline path of the condition.
	conditionPath *khifilev6.TimelinePath
	// conditionType is the `type` field of the condition.
	conditionType string
	// lastStatus is the last status of the condition.
	lastStatus string
	// lastTransitionTime is the last transition time of the condition.
	lastTransitionTime string
	// lastProbeLikeTime is the last probe like time of the condition.
	lastProbeLikeTime string
	// minChangeTime is the minimum change time.
	// This is used not to create a revision too early for the resource retaining the condition after recreation.
	minChangeTime *time.Time

	lastTransitionStates map[string]*model.K8sResourceStatusCondition

	lastTransitionTimeSorted []*time.Time
}

// newConditionWalkerV2 creates a new conditionWalkerV2 for a specific condition type.
func newConditionWalkerV2(conditionPath *khifilev6.TimelinePath, conditionType string) *conditionWalkerV2 {
	return &conditionWalkerV2{
		conditionPath:            conditionPath,
		conditionType:            conditionType,
		lastStatus:               "",
		lastTransitionTime:       "",
		lastProbeLikeTime:        "",
		lastTransitionStates:     map[string]*model.K8sResourceStatusCondition{},
		lastTransitionTimeSorted: []*time.Time{},
	}
}

// checkLastTransitionTimes memorizes the last transition time of the condition. This value is used for complementing values for logs without the full status information.
func (c *conditionWalkerV2) checkLastTransitionTimes(commonLog *log.CommonFieldSet, k8sAuditLog *commonlogk8saudit_contract.K8sAuditLogFieldSet, condition *model.K8sResourceStatusCondition) {
	if condition != nil && condition.Status != "" && condition.LastTransitionTime != "" {
		c.lastTransitionStates[condition.LastTransitionTime] = condition
	}
}

// CheckAndRecord compares the current condition with the previous state and records a revision if there is a significant change.
// It tracks changes in Status, LastTransitionTime, and LastHeartbeatTime (ProbeLikeTime).
func (c *conditionWalkerV2) CheckAndRecord(ctx context.Context, commonLog *log.CommonFieldSet, k8sAuditLog *commonlogk8saudit_contract.K8sAuditLogFieldSet, condition *model.K8sResourceStatusCondition, cs *khifilev6.TimelineChangeSet) {
	if condition == nil {
		if c.lastStatus != "n/a" {
			cs.AddRevision(c.conditionPath, &khifilev6.StagingRevision{
				VerbType:     k8sAuditLog.Verb,
				ResourceBody: nil,
				Principal:    k8sAuditLog.Principal,
				ChangedTime:  commonLog.Timestamp,
				StateType:    commonlogk8saudit_contract.RevisionStateConditionNotGiven,
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
				cs.AddRevision(c.conditionPath, &khifilev6.StagingRevision{
					VerbType:     k8sAuditLog.Verb,
					ResourceBody: body,
					Principal:    k8sAuditLog.Principal,
					ChangedTime:  c.clampMinChangeTime(transitionTime),
					StateType:    state,
				})
				c.lastTransitionTime = condition.LastTransitionTime
			}
		}
		probeLikeTime, err := condition.ProbeLikeTime()
		if err == nil {
			if c.lastProbeLikeTime != probeLikeTime.Format(time.RFC3339) {
				if condition.Status == "" {
					referenceCondition := c.getLastCondition(probeLikeTime)
					if referenceCondition != nil {
						condition.Status = referenceCondition.Status
						if condition.LastTransitionTime == "" {
							condition.LastTransitionTime = referenceCondition.LastTransitionTime
						}
						if condition.Message == "" {
							condition.Message = referenceCondition.Message
						}
						if condition.Reason == "" {
							condition.Reason = referenceCondition.Reason
						}
					}
				}
				state := conditionStateToRevisionState(condition.Status)
				body := c.serializeCondition(condition)
				cs.AddRevision(c.conditionPath, &khifilev6.StagingRevision{
					VerbType:     k8sAuditLog.Verb,
					ResourceBody: body,
					Principal:    k8sAuditLog.Principal,
					ChangedTime:  c.clampMinChangeTime(probeLikeTime),
					StateType:    state,
				})
				c.lastProbeLikeTime = probeLikeTime.Format(time.RFC3339)
			}
		}
	}
}

// RecordDeletion records the deletion of the condition.
func (c *conditionWalkerV2) RecordDeletion(deletionTime time.Time) {
	c.lastStatus = ""
	c.lastTransitionTime = ""
	c.lastProbeLikeTime = ""
}

func (c *conditionWalkerV2) getLastCondition(beforeThan time.Time) *model.K8sResourceStatusCondition {
	if len(c.lastTransitionTimeSorted) != len(c.lastTransitionStates) {
		times := make([]*time.Time, 0, len(c.lastTransitionStates))
		for k := range c.lastTransitionStates {
			t, err := time.Parse(time.RFC3339, k)
			if err != nil {
				continue
			}
			times = append(times, &t)
		}
		sort.Slice(times, func(i, j int) bool {
			return times[i].Before(*times[j])
		})
		c.lastTransitionTimeSorted = times
	}
	if len(c.lastTransitionTimeSorted) == 0 {
		return nil
	}

	if c.lastTransitionTimeSorted[0].After(beforeThan) {
		return nil
	}
	idx := sort.Search(len(c.lastTransitionTimeSorted), func(i int) bool {
		return c.lastTransitionTimeSorted[i].After(beforeThan)
	})
	if idx > 0 {
		return c.lastTransitionStates[c.lastTransitionTimeSorted[idx-1].Format(time.RFC3339)]
	}
	return nil
}

// serializeCondition serializes the K8sResourceStatusCondition to a structured.Node for storage in the revision body.
func (c *conditionWalkerV2) serializeCondition(condition *model.K8sResourceStatusCondition) structured.Node {
	conditionNode, err := structured.FromGoValue(condition.ToMap(), &structured.AlphabeticalGoMapKeyOrderProvider{})
	if err == nil {
		return conditionNode
	}
	return nil
}

// clampMinChangeTime clamps the change time to the minimum change time if it is before the minimum change time.
// This is needed not to write a revision overlaps the previous revisions before deletion because some conditions are kept used again after recreation.
// This happens especially in static Pods.
func (c *conditionWalkerV2) clampMinChangeTime(changeTime time.Time) time.Time {
	if c.minChangeTime != nil && changeTime.Before(*c.minChangeTime) {
		return *c.minChangeTime
	}
	return changeTime
}

// MustK8sConditionTimeline resolves the timeline path of a resource condition.
func MustK8sConditionTimeline(ctx context.Context, ownerPath *khifilev6.TimelinePath, conditionType string) *khifilev6.TimelinePath {
	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(ownerPath, khifilev6.PathSegment{
		Name: conditionType,
		Type: commonlogk8saudit_contract.TimelineTypeResourceCondition,
	})
}
