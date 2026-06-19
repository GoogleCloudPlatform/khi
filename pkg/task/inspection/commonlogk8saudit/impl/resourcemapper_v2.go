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
	"log/slog"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
)

// resourceRevisionLogToTimelineMapperStateV2 tracks the status of a resource during V2 timeline generation.
type resourceRevisionLogToTimelineMapperStateV2 struct {
	// WasCompletelyRemoved is true if the resource was completely removed.
	WasCompletelyRemoved bool
	// DeletionStarted is true if the deletion started.
	DeletionStarted bool
	// PrevUID is the previous UID of the resource.
	PrevUID string
}

// ResourceRevisionLogToTimelineMapperTaskSettingV2 is the setting for the V2 resource revision timeline mapper task.
type ResourceRevisionLogToTimelineMapperTaskSettingV2 struct {
	commonlogk8saudit_contract.ManifestSinglePassMapperBaseV2[*resourceRevisionLogToTimelineMapperStateV2]

	// minimumDeltaTimeToCreateInferredCreationRevision is a threshold of a duration that controls if KHI should create an inferred creation revision from creationTimestamp.
	minimumDeltaTimeToCreateInferredCreationRevision time.Duration
	// kindsToWaitExactDeletionToDeterminDeletion is the map of kinds to wait exact deletion to determine deletion.
	kindsToWaitExactDeletionToDeterminDeletion map[string]struct{}
}

// Dependencies implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (r *ResourceRevisionLogToTimelineMapperTaskSettingV2) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (r *ResourceRevisionLogToTimelineMapperTaskSettingV2) GroupedLogTask() taskid.TaskReference[commonlogk8saudit_contract.ResourceManifestLogGroupMap] {
	return commonlogk8saudit_contract.ResourceLifetimeTrackerTaskID.Ref()
}

// LogIngesterTask implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (r *ResourceRevisionLogToTimelineMapperTaskSettingV2) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return commonlogk8saudit_contract.K8sAuditLogIngesterTaskID.Ref()
}

// TaskID implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (r *ResourceRevisionLogToTimelineMapperTaskSettingV2) TaskID() taskid.TaskImplementationID[struct{}] {
	return commonlogk8saudit_contract.ResourceRevisionLogToTimelineMapperTaskID
}

// ResolveRelatedGroupSets implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (r *ResourceRevisionLogToTimelineMapperTaskSettingV2) ResolveRelatedGroupSets(ctx context.Context, groupedLogs commonlogk8saudit_contract.ResourceManifestLogGroupMap) ([]commonlogk8saudit_contract.RelatedGroupSet, error) {
	result := []commonlogk8saudit_contract.RelatedGroupSet{}
	for _, group := range groupedLogs {
		switch group.Resource.Type() {
		case commonlogk8saudit_contract.Namespace:
			continue
		case commonlogk8saudit_contract.Resource:
			result = append(result, commonlogk8saudit_contract.RelatedGroupSet{
				Roles: map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{
					"target": group,
				},
			})
			continue
		case commonlogk8saudit_contract.Subresource:
			parentGroup := groupedLogs[group.Resource.ParentIdentity().String()]
			result = append(result, commonlogk8saudit_contract.RelatedGroupSet{
				Roles: map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{
					"source": parentGroup,
					"target": group,
				},
			})
		default:
			panic(fmt.Sprintf("unknown resource type: %v", group.Resource.Type()))
		}
	}
	return result, nil
}

// ProcessLog implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (r *ResourceRevisionLogToTimelineMapperTaskSettingV2) ProcessLog(ctx context.Context, event commonlogk8saudit_contract.MultiGroupLogEvent, prevGroupData *resourceRevisionLogToTimelineMapperStateV2) (*khifilev6.TimelineChangeSet, *resourceRevisionLogToTimelineMapperStateV2, error) {
	if prevGroupData == nil {
		prevGroupData = &resourceRevisionLogToTimelineMapperStateV2{}
	}

	cs := khifilev6.NewTimelineChangeSet(event.Log)

	switch event.GroupRole {
	case "source":
		err := r.handleParentChangeForSubresource(ctx, event, cs)
		return cs, prevGroupData, err
	default:
		nextState, err := r.handleTargetChange(ctx, event, cs, prevGroupData)
		return cs, nextState, err
	}
}

// ResourceRevisionLogToTimelineMapperTask is the V2 task to generate resource revision history.
var ResourceRevisionLogToTimelineMapperTask = commonlogk8saudit_contract.NewManifestLogToTimelineMapperV2[*resourceRevisionLogToTimelineMapperStateV2](&ResourceRevisionLogToTimelineMapperTaskSettingV2{
	minimumDeltaTimeToCreateInferredCreationRevision: 5 * time.Second,
	kindsToWaitExactDeletionToDeterminDeletion: map[string]struct{}{
		"core/v1#pod": {},
	},
})

// handleParentChangeForSubresource handles the parent change for subresource V2.
func (r *ResourceRevisionLogToTimelineMapperTaskSettingV2) handleParentChangeForSubresource(ctx context.Context, event commonlogk8saudit_contract.MultiGroupLogEvent, cs *khifilev6.TimelineChangeSet) error {
	switch event.EventType {
	case commonlogk8saudit_contract.ChangeEventTypeV2Deletion:
		targetGroup, found := event.GroupSet.Roles["target"]
		if !found || targetGroup == nil {
			return nil
		}
		k8sFieldSet := log.MustGetFieldSet(event.Log, &commonlogk8saudit_contract.K8sAuditLogFieldSet{})
		targetPath := MustResolveTimelinePath(ctx, k8sFieldSet.ClusterName, targetGroup.Resource)

		commonLogFieldSet := log.MustGetFieldSet(event.Log, &log.CommonFieldSet{})

		var bodyNode structured.Node
		if bodyReader, ok := event.GetLastBodyReader("target"); ok && bodyReader != nil {
			bodyNode = bodyReader.Node
		}

		cs.AddRevision(targetPath, &khifilev6.StagingRevision{
			ChangedTime:  commonLogFieldSet.Timestamp,
			ResourceBody: bodyNode,
			Principal:    k8sFieldSet.Principal,
			VerbType:     commonlogk8saudit_contract.VerbDelete,
			StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceIsDeleted,
		})
		return nil
	case commonlogk8saudit_contract.ChangeEventTypeV2Modification:
		return nil
	case commonlogk8saudit_contract.ChangeEventTypeV2Creation:
		return nil
	default:
		slog.WarnContext(ctx, "unknown event type", "eventType", event.EventType)
		return nil
	}
}

// handleTargetChange handles the target change V2.
func (r *ResourceRevisionLogToTimelineMapperTaskSettingV2) handleTargetChange(ctx context.Context, event commonlogk8saudit_contract.MultiGroupLogEvent, cs *khifilev6.TimelineChangeSet, prevGroupData *resourceRevisionLogToTimelineMapperStateV2) (*resourceRevisionLogToTimelineMapperStateV2, error) {
	commonFieldSet := log.MustGetFieldSet(event.Log, &log.CommonFieldSet{})
	k8sFieldSet := log.MustGetFieldSet(event.Log, &commonlogk8saudit_contract.K8sAuditLogFieldSet{})
	targetPath := MustResolveTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity)

	if prevGroupData == nil {
		prevGroupData = &resourceRevisionLogToTimelineMapperStateV2{}
	}

	if k8sFieldSet.Verb == commonlogk8saudit_contract.VerbDeleteCollection && prevGroupData.WasCompletelyRemoved {
		return prevGroupData, nil
	}

	state := commonlogk8saudit_contract.RevisionStateK8sResourceExisting
	bodyReader, hasBody := event.GetLastBodyReader(event.GroupRole)

	if !hasBody || bodyReader == nil {
		if isDeletiveVerb(k8sFieldSet.Verb) {
			prevGroupData.DeletionStarted = true
			state = commonlogk8saudit_contract.RevisionStateK8sResourceIsDeleted
		}
	} else {
		deletionStarted := false
		underGracefulPeriod := false
		deletionCompleted := false
		uid, _ := GetUID(bodyReader)
		if uid != prevGroupData.PrevUID {
			prevGroupData.PrevUID = uid
			prevGroupData.DeletionStarted = false
			prevGroupData.WasCompletelyRemoved = false
		} else {
			deletionStarted = prevGroupData.DeletionStarted
			deletionCompleted = prevGroupData.WasCompletelyRemoved
		}

		if isDeletiveVerb(k8sFieldSet.Verb) {
			prevGroupData.DeletionStarted = true
			deletionStarted = true
			if isPod(k8sFieldSet.APIVersion, k8sFieldSet.PluralKind) {
				phase, _ := GetPodPhase(bodyReader)
				switch phase {
				case "Failed", "Succeeded":
					deletionCompleted = true
				default:
					underGracefulPeriod = true
				}
			}
		}
		deletionGracefulPeriods, found := GetDeletionGracePeriodSeconds(bodyReader)
		if found {
			if deletionGracefulPeriods > 0 {
				underGracefulPeriod = true
			}
			if deletionGracefulPeriods == 0 {
				deletionCompleted = true
			}
			deletionStarted = true
		}

		finalizers, found := GetFinalizers(bodyReader)
		if found && len(finalizers) > 0 && deletionStarted {
			deletionCompleted = false
			underGracefulPeriod = true
		}

		_, found = GetDeletionTimestamp(bodyReader)
		if found {
			deletionStarted = true
			if !underGracefulPeriod {
				deletionCompleted = true
			}
		}

		if k8sFieldSet.Verb == commonlogk8saudit_contract.VerbPatch && state == commonlogk8saudit_contract.RevisionStateK8sResourceExisting {
			if prevGroupData.DeletionStarted {
				state = commonlogk8saudit_contract.RevisionStateK8sResourceDeleting
			}
			if prevGroupData.WasCompletelyRemoved {
				state = commonlogk8saudit_contract.RevisionStateK8sResourceIsDeleted
			}
		}

		switch {
		case deletionCompleted:
			prevGroupData.WasCompletelyRemoved = true
			prevGroupData.DeletionStarted = false
			state = commonlogk8saudit_contract.RevisionStateK8sResourceIsDeleted
		case underGracefulPeriod:
			prevGroupData.WasCompletelyRemoved = false
			prevGroupData.DeletionStarted = true
			state = commonlogk8saudit_contract.RevisionStateK8sResourceDeleting
		case deletionStarted:
			prevGroupData.WasCompletelyRemoved = false
			prevGroupData.DeletionStarted = true
			apiVersionKind := fmt.Sprintf("%s#%s", k8sFieldSet.APIVersion, commonlogk8saudit_contract.GetSingularKindName(k8sFieldSet.PluralKind))
			if _, found := r.kindsToWaitExactDeletionToDeterminDeletion[apiVersionKind]; !found {
				state = commonlogk8saudit_contract.RevisionStateK8sResourceIsDeleted
			}
		default:
			prevGroupData.WasCompletelyRemoved = false
			prevGroupData.DeletionStarted = false
		}
	}

	var creationTime time.Time
	var found bool
	if bodyReader != nil {
		creationTime, found = GetCreationTimestamp(bodyReader)
	}
	if !found {
		creationTime = commonFieldSet.Timestamp
	}

	if event.EventType == commonlogk8saudit_contract.ChangeEventTypeV2Creation && commonFieldSet.Timestamp.Sub(creationTime) > r.minimumDeltaTimeToCreateInferredCreationRevision {
		cs.AddRevision(targetPath, &khifilev6.StagingRevision{
			ChangedTime:  creationTime,
			ResourceBody: nil,
			Principal:    "N/A",
			VerbType:     commonlogk8saudit_contract.VerbCreate,
			StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExisting,
		})
	}

	var bodyNode structured.Node
	if bodyReader != nil {
		bodyNode = bodyReader.Node
	}

	cs.AddRevision(targetPath, &khifilev6.StagingRevision{
		ChangedTime:  commonFieldSet.Timestamp,
		ResourceBody: bodyNode,
		Principal:    k8sFieldSet.Principal,
		VerbType:     k8sFieldSet.Verb,
		StateType:    state,
	})
	return prevGroupData, nil
}

// MustResolveTimelinePath resolves TimelinePath from ResourceIdentity using K6 core helpers.
func MustResolveTimelinePath(ctx context.Context, clusterName string, identity *commonlogk8saudit_contract.ResourceIdentity) *khifilev6.TimelinePath {
	cluster := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	api := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, cluster, identity.APIVersion)
	kind := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, api, strings.ToLower(identity.Kind))

	var resPath *khifilev6.TimelinePath
	if identity.Namespace != "" {
		ns := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kind, identity.Namespace)
		resPath = commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, ns, identity.Name)
	} else {
		resPath = commonlogk8saudit_contract.MustK8sClusterScopeResourceTimeline(ctx, kind, identity.Name)
	}

	if identity.SubresourceName != "" {
		return commonlogk8saudit_contract.MustK8sSubresourceTimeline(ctx, resPath, identity.SubresourceName)
	}

	return resPath
}

// Explicit interface compliance assertion.
var _ commonlogk8saudit_contract.ManifestLogToTimelineMapperV2[*resourceRevisionLogToTimelineMapperStateV2] = (*ResourceRevisionLogToTimelineMapperTaskSettingV2)(nil)
