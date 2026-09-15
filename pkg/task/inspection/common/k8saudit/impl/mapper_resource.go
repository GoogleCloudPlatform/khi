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
	"log/slog"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
)

// resourceRevisionLogToTimelineMapperState tracks the status of a resource during timeline generation.
type resourceRevisionLogToTimelineMapperState struct {
	// WasCompletelyRemoved is true if the resource was completely removed.
	WasCompletelyRemoved bool
	// DeletionStarted is true if the deletion started.
	DeletionStarted bool
	// PrevUID is the previous UID of the resource.
	PrevUID string
	// creationTimePerUID maps resource UID to its creationTimestamp collected during PreProcessLog pass.
	creationTimePerUID map[string]time.Time
	// fallbackCreationTime is the first creationTimestamp found in the log group during PreProcessLog pass.
	fallbackCreationTime time.Time
	// hasFallbackCreationTime is true if fallbackCreationTime was recorded.
	hasFallbackCreationTime bool
}

// newResourceRevisionLogToTimelineMapperState returns a new instance of resourceRevisionLogToTimelineMapperState.
func newResourceRevisionLogToTimelineMapperState() *resourceRevisionLogToTimelineMapperState {
	return &resourceRevisionLogToTimelineMapperState{
		creationTimePerUID: make(map[string]time.Time),
	}
}

// ResourceRevisionLogToTimelineMapperTaskSetting is the setting for the resource revision timeline mapper task.
type ResourceRevisionLogToTimelineMapperTaskSetting struct {
	// kindsToWaitExactDeletionToDeterminDeletion is the map of kinds to wait exact deletion to determine deletion.
	kindsToWaitExactDeletionToDeterminDeletion map[string]struct{}
}

// Dependencies implements k8saudit.ManifestLogToTimelineMapper.
func (r *ResourceRevisionLogToTimelineMapperTaskSetting) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		k8saudit.InitialResourceStateProviderRef,
	}
}

// PassCount implements k8saudit.ManifestLogToTimelineMapper.
func (r *ResourceRevisionLogToTimelineMapperTaskSetting) PassCount() int {
	return 1
}

// PreProcessLog implements k8saudit.ManifestLogToTimelineMapper.
func (r *ResourceRevisionLogToTimelineMapperTaskSetting) PreProcessLog(ctx context.Context, passIndex int, event k8saudit.MultiGroupLogEvent, prevGroupData *resourceRevisionLogToTimelineMapperState) (*resourceRevisionLogToTimelineMapperState, error) {
	if prevGroupData == nil {
		prevGroupData = newResourceRevisionLogToTimelineMapperState()
	}
	if event.GroupRole != "target" {
		return prevGroupData, nil
	}

	bodyReader, hasBody := event.GetLastBodyReader(event.GroupRole)
	if hasBody && bodyReader != nil {
		creationTime, found := GetCreationTimestamp(bodyReader)
		if found {
			if !prevGroupData.hasFallbackCreationTime {
				prevGroupData.fallbackCreationTime = creationTime
				prevGroupData.hasFallbackCreationTime = true
			}
			uid, ok := GetUID(bodyReader)
			if ok && uid != "" {
				if _, exists := prevGroupData.creationTimePerUID[uid]; !exists {
					prevGroupData.creationTimePerUID[uid] = creationTime
				}
			}
		}
	}
	return prevGroupData, nil
}

// GroupedLogTask implements k8saudit.ManifestLogToTimelineMapper.
func (r *ResourceRevisionLogToTimelineMapperTaskSetting) GroupedLogTask() taskid.TaskReference[k8saudit.ResourceManifestLogGroupMap] {
	return k8saudit.ResourceLifetimeTrackerTaskID.Ref()
}

// LogIngesterTask implements k8saudit.ManifestLogToTimelineMapper.
func (r *ResourceRevisionLogToTimelineMapperTaskSetting) LogIngesterTask() taskid.TaskReference[struct{}] {
	return k8saudit.K8sAuditLogIngesterTaskID.Ref()
}

// TaskID implements k8saudit.ManifestLogToTimelineMapper.
func (r *ResourceRevisionLogToTimelineMapperTaskSetting) TaskID() taskid.TaskImplementationID[struct{}] {
	return k8saudit.ResourceRevisionLogToTimelineMapperTaskID
}

// ResolveRelatedGroupSets implements k8saudit.ManifestLogToTimelineMapper.
func (r *ResourceRevisionLogToTimelineMapperTaskSetting) ResolveRelatedGroupSets(ctx context.Context, groupedLogs k8saudit.ResourceManifestLogGroupMap) ([]k8saudit.RelatedGroupSet, error) {
	result := []k8saudit.RelatedGroupSet{}
	for _, group := range groupedLogs {
		switch group.Resource.Type() {
		case k8saudit.Namespace:
			continue
		case k8saudit.Resource:
			result = append(result, k8saudit.RelatedGroupSet{
				Roles: map[string]*k8saudit.ResourceManifestLogGroup{
					"target": group,
				},
			})
			continue
		case k8saudit.Subresource:
			parentGroup := groupedLogs[group.Resource.ParentIdentity().String()]
			result = append(result, k8saudit.RelatedGroupSet{
				Roles: map[string]*k8saudit.ResourceManifestLogGroup{
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

// ProcessLog implements k8saudit.ManifestLogToTimelineMapper.
func (r *ResourceRevisionLogToTimelineMapperTaskSetting) ProcessLog(ctx context.Context, event k8saudit.MultiGroupLogEvent, prevGroupData *resourceRevisionLogToTimelineMapperState) (*khifilev6.TimelineChangeSet, *resourceRevisionLogToTimelineMapperState, error) {
	if prevGroupData == nil {
		prevGroupData = newResourceRevisionLogToTimelineMapperState()
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

// ResourceRevisionLogToTimelineMapperTask is the task to generate resource revision history.
var ResourceRevisionLogToTimelineMapperTask = k8saudit.NewManifestLogToTimelineMapper[*resourceRevisionLogToTimelineMapperState](&ResourceRevisionLogToTimelineMapperTaskSetting{
	kindsToWaitExactDeletionToDeterminDeletion: map[string]struct{}{
		"core/v1#pod": {},
	},
})

// handleParentChangeForSubresource handles the parent change for subresource.
func (r *ResourceRevisionLogToTimelineMapperTaskSetting) handleParentChangeForSubresource(ctx context.Context, event k8saudit.MultiGroupLogEvent, cs *khifilev6.TimelineChangeSet) error {
	switch event.EventType {
	case k8saudit.ChangeEventTypeDeletion:
		targetGroup, found := event.GroupSet.Roles["target"]
		if !found || targetGroup == nil {
			return nil
		}
		k8sFieldSet, _ := k8saudit.ExtractK8sAuditLog(ctx, event.Log.NodeReader)
		if k8sFieldSet.IsDryRun {
			return nil
		}
		targetPath := MustResolveTimelinePath(ctx, k8sFieldSet.ClusterName, targetGroup.Resource)

		var bodyNode structured.Node
		if bodyReader, ok := event.GetLastBodyReader("target"); ok && bodyReader != nil {
			bodyNode = bodyReader.Node
		}

		cs.AddRevision(targetPath, &khifilev6.StagingRevision{
			ChangedTime:  event.Log.Timestamp,
			ResourceBody: bodyNode,
			Principal:    k8sFieldSet.Principal,
			VerbType:     k8saudit.VerbDelete,
			StateType:    k8saudit.RevisionStateK8sResourceDeleted,
		})
		return nil
	case k8saudit.ChangeEventTypeModification:
		return nil
	case k8saudit.ChangeEventTypeCreation:
		return nil
	default:
		slog.WarnContext(ctx, "unknown event type", "eventType", event.EventType)
		return nil
	}
}

// handleTargetChange handles the target change.
func (r *ResourceRevisionLogToTimelineMapperTaskSetting) handleTargetChange(ctx context.Context, event k8saudit.MultiGroupLogEvent, cs *khifilev6.TimelineChangeSet, prevGroupData *resourceRevisionLogToTimelineMapperState) (*resourceRevisionLogToTimelineMapperState, error) {
	k8sFieldSet, _ := k8saudit.ExtractK8sAuditLog(ctx, event.Log.NodeReader)
	targetPath := MustResolveTimelinePath(ctx, k8sFieldSet.ClusterName, event.ResourceIdentity)

	if prevGroupData == nil {
		prevGroupData = newResourceRevisionLogToTimelineMapperState()
	}

	if k8sFieldSet.IsDryRun {
		cs.AddEvent(targetPath)
		return prevGroupData, nil
	}

	if k8sFieldSet.Verb == k8saudit.VerbDeleteCollection && prevGroupData.WasCompletelyRemoved {
		return prevGroupData, nil
	}

	state := k8saudit.RevisionStateK8sResourceExisting
	bodyReader, hasBody := event.GetLastBodyReader(event.GroupRole)

	switch {
	case k8sFieldSet.IsTruncated:
		if isDeletiveVerb(k8sFieldSet.Verb) {
			prevGroupData.DeletionStarted = true
			state = k8saudit.RevisionStateK8sResourceDeleted
		} else {
			state = k8saudit.RevisionStateK8sResourceTruncated
		}
	case !hasBody || bodyReader == nil:
		if isDeletiveVerb(k8sFieldSet.Verb) {
			prevGroupData.DeletionStarted = true
			state = k8saudit.RevisionStateK8sResourceDeleted
		}
	default:
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

		if k8sFieldSet.Verb == k8saudit.VerbPatch && state == k8saudit.RevisionStateK8sResourceExisting {
			if prevGroupData.DeletionStarted {
				state = k8saudit.RevisionStateK8sResourceDeleting
			}
			if prevGroupData.WasCompletelyRemoved {
				state = k8saudit.RevisionStateK8sResourceDeleted
			}
		}

		switch {
		case deletionCompleted:
			prevGroupData.WasCompletelyRemoved = true
			prevGroupData.DeletionStarted = false
			state = k8saudit.RevisionStateK8sResourceDeleted
		case underGracefulPeriod:
			prevGroupData.WasCompletelyRemoved = false
			prevGroupData.DeletionStarted = true
			state = k8saudit.RevisionStateK8sResourceDeleting
		case deletionStarted:
			prevGroupData.WasCompletelyRemoved = false
			prevGroupData.DeletionStarted = true
			apiVersionKind := fmt.Sprintf("%s#%s", k8sFieldSet.APIVersion, k8saudit.GetSingularKindName(k8sFieldSet.PluralKind))
			if _, found := r.kindsToWaitExactDeletionToDeterminDeletion[apiVersionKind]; !found {
				state = k8saudit.RevisionStateK8sResourceDeleted
			}
		default:
			prevGroupData.WasCompletelyRemoved = false
			prevGroupData.DeletionStarted = false
		}
	}

	// Resolve resource creation time with fallbacks.
	// 1. Try resolving creationTimestamp directly from the current log body.
	// 2. Fallback to pre-processed creationTimestamp mapped to the resource UID.
	// 3. Fallback to the first creationTimestamp observed in the entire log group.
	var creationTime time.Time
	var hasCreationTime bool
	if bodyReader != nil {
		creationTime, hasCreationTime = GetCreationTimestamp(bodyReader)
		if !hasCreationTime {
			uid, ok := GetUID(bodyReader)
			if ok && uid != "" {
				if t, exists := prevGroupData.creationTimePerUID[uid]; exists {
					creationTime = t
					hasCreationTime = true
				}
			}
		}
	}
	if !hasCreationTime && prevGroupData.hasFallbackCreationTime {
		creationTime = prevGroupData.fallbackCreationTime
		hasCreationTime = true
	}

	// For the initial observation of a resource without an explicit creation log (e.g. starting with patch),
	// prepend an inferred creation revision indicating that the resource already existed prior to the logs.
	if event.EventType == k8saudit.ChangeEventTypeCreation && k8sFieldSet.Verb != k8saudit.VerbCreate {
		// The provider side renders both the unknown period before the observed manifest and the manifest
		// itself, so this body-less revision only covers the resources the inventory does not know.
		initialStateProvider := coretask.GetTaskResult(ctx, k8saudit.InitialResourceStateProviderRef)
		if _, hasInitialState := initialStateProvider.InitialResourceState(event.ResourceIdentity); !hasInitialState {
			existenceStartTime := time.Unix(0, 0)
			if hasCreationTime {
				existenceStartTime = creationTime
			}
			cs.AddRevision(targetPath, &khifilev6.StagingRevision{
				ChangedTime:  existenceStartTime,
				ResourceBody: nil,
				Principal:    "N/A",
				VerbType:     k8saudit.VerbCreate,
				StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
			})
		}
	}

	var bodyNode structured.Node
	if bodyReader != nil {
		bodyNode = bodyReader.Node
	}

	var fieldAnnotations []*khifilev6.StagingFieldAnnotation
	for _, hook := range k8sFieldSet.MutatingWebhookResults {
		if hook.Mutated {
			for _, p := range hook.Patch {
				fieldAnnotations = append(fieldAnnotations, &khifilev6.StagingFieldAnnotation{
					FieldPath: p.Path,
					MutatingWebhook: &khifilev6.StagingMutatingWebhook{
						Configuration: hook.Configuration,
						Webhook:       hook.Webhook,
						Round:         int32(hook.Round),
						Index:         int32(hook.Index),
					},
				})
			}
		}
	}

	cs.AddRevision(targetPath, &khifilev6.StagingRevision{
		ChangedTime:      event.Log.Timestamp,
		ResourceBody:     bodyNode,
		Principal:        k8sFieldSet.Principal,
		VerbType:         k8sFieldSet.Verb,
		StateType:        state,
		FieldAnnotations: fieldAnnotations,
	})
	return prevGroupData, nil
}

// MustResolveTimelinePath resolves TimelinePath from ResourceIdentity using K6 core helpers.
func MustResolveTimelinePath(ctx context.Context, clusterName string, identity *k8saudit.ResourceIdentity) *khifilev6.TimelinePath {
	cluster := k8saudit.MustK8sClusterTimeline(ctx, clusterName)
	api := k8saudit.MustK8sAPIVersionTimeline(ctx, cluster, identity.APIVersion)
	kind := k8saudit.MustK8sKindTimeline(ctx, api, strings.ToLower(identity.Kind))

	var resPath *khifilev6.TimelinePath
	if identity.Namespace != "" {
		ns := k8saudit.MustK8sNamespaceTimeline(ctx, kind, identity.Namespace)
		resPath = k8saudit.MustK8sNamespacedResourceTimeline(ctx, ns, identity.Name)
	} else {
		resPath = k8saudit.MustK8sClusterScopeResourceTimeline(ctx, kind, identity.Name)
	}

	if identity.SubresourceName != "" {
		return k8saudit.MustK8sSubresourceTimeline(ctx, resPath, identity.SubresourceName)
	}

	return resPath
}

// Explicit interface compliance assertion.
var _ k8saudit.ManifestLogToTimelineMapper[*resourceRevisionLogToTimelineMapperState] = (*ResourceRevisionLogToTimelineMapperTaskSetting)(nil)
