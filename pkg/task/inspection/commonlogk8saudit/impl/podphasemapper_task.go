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
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var phaseToState = map[string]*pb.RevisionState{
	"Pending":   commonlogk8saudit_contract.RevisionStatePodPhasePending,
	"Running":   commonlogk8saudit_contract.RevisionStatePodPhaseRunning,
	"Succeeded": commonlogk8saudit_contract.RevisionStatePodPhaseSucceeded,
	"Failed":    commonlogk8saudit_contract.RevisionStatePodPhaseFailed,
	"Unknown":   commonlogk8saudit_contract.RevisionStatePodPhaseUnknown,
}

type podPhaseTaskState struct {
	// lastPhase is the last phase of the pod.
	lastPhase string
	// lastNode is the last node of the pod.
	lastNode string
	// uidToNodeNameMap is the map of UID to node name.
	uidToNodeNameMap map[string]string
}

type podPhaseLogToTimelineMapperTaskSettingV2 struct {
	// minimumDeltaTimeToCreateInferredCreationRevision is a threshold of a duration that controls if KHI should create an inferred creation revision from creationTimestamp.
	minimumDeltaTimeToCreateInferredCreationRevision time.Duration
}

// Dependencies implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *podPhaseLogToTimelineMapperTaskSettingV2) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// PassCount implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *podPhaseLogToTimelineMapperTaskSettingV2) PassCount() int {
	return 1
}

// GroupedLogTask implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *podPhaseLogToTimelineMapperTaskSettingV2) GroupedLogTask() taskid.TaskReference[commonlogk8saudit_contract.ResourceManifestLogGroupMap] {
	return commonlogk8saudit_contract.ResourceLifetimeTrackerTaskID.Ref()
}

// LogIngesterTask implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *podPhaseLogToTimelineMapperTaskSettingV2) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return commonlogk8saudit_contract.K8sAuditLogIngesterTaskID.Ref()
}

// TaskID implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *podPhaseLogToTimelineMapperTaskSettingV2) TaskID() taskid.TaskImplementationID[struct{}] {
	return commonlogk8saudit_contract.PodPhaseLogToTimelineMapperTaskID
}

// ResolveRelatedGroupSets implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *podPhaseLogToTimelineMapperTaskSettingV2) ResolveRelatedGroupSets(ctx context.Context, groupedLogs commonlogk8saudit_contract.ResourceManifestLogGroupMap) ([]commonlogk8saudit_contract.RelatedGroupSet, error) {
	result := []commonlogk8saudit_contract.RelatedGroupSet{}
	for _, group := range groupedLogs {
		if group.Resource.Type() == commonlogk8saudit_contract.Resource && group.Resource.APIVersion == "core/v1" && group.Resource.Kind == "pod" {
			bindingGroup := groupedLogs[group.Resource.SubresourceIdentity("binding").ResourcePathString()]
			result = append(result, commonlogk8saudit_contract.RelatedGroupSet{
				Roles: map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{
					"pod":     group,
					"binding": bindingGroup,
				},
			})
		}
	}
	return result, nil
}

// PreProcessLog implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *podPhaseLogToTimelineMapperTaskSettingV2) PreProcessLog(ctx context.Context, passIndex int, event commonlogk8saudit_contract.MultiGroupLogEvent, prevGroupData *podPhaseTaskState) (*podPhaseTaskState, error) {
	if prevGroupData == nil {
		prevGroupData = &podPhaseTaskState{
			uidToNodeNameMap: map[string]string{},
		}
	}
	if event.GroupRole != "pod" {
		return prevGroupData, nil
	}
	bodyReader, ok := event.GetLastBodyReader("pod")
	if !ok || bodyReader == nil {
		return prevGroupData, nil
	}
	nodeName, found := GetNodeNameOfPod(bodyReader)
	if !found {
		return prevGroupData, nil
	}
	uid, _ := GetUID(bodyReader)

	if nodeName != "" && uid != "" {
		prevGroupData.uidToNodeNameMap[uid] = nodeName
	}
	return prevGroupData, nil
}

// ProcessLog implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *podPhaseLogToTimelineMapperTaskSettingV2) ProcessLog(ctx context.Context, event commonlogk8saudit_contract.MultiGroupLogEvent, prevGroupData *podPhaseTaskState) (*khifilev6.TimelineChangeSet, *podPhaseTaskState, error) {
	if prevGroupData == nil {
		prevGroupData = &podPhaseTaskState{
			uidToNodeNameMap: map[string]string{},
		}
	}

	cs := khifilev6.NewTimelineChangeSet(event.Log)

	commonLogFieldSet := log.MustGetFieldSet(event.Log, &log.CommonFieldSet{})
	k8sFieldSet := log.MustGetFieldSet(event.Log, &commonlogk8saudit_contract.K8sAuditLogFieldSet{})

	var targetBodyReader *structured.NodeReader
	if reader, ok := event.GetLastBodyReader("pod"); ok {
		targetBodyReader = reader
	}

	uid, found := GetUID(targetBodyReader)
	if !found {
		return cs, prevGroupData, nil
	}

	nodeName, found := prevGroupData.uidToNodeNameMap[uid]
	if !found {
		return cs, prevGroupData, nil
	}

	if event.GroupRole == "binding" {
		if event.EventType == commonlogk8saudit_contract.ChangeEventTypeV2Creation {
			targetPath := MustPodPhaseTimelinePath(ctx, k8sFieldSet.ClusterName, nodeName, event.GroupSet.Roles["pod"].Resource.Namespace, event.GroupSet.Roles["pod"].Resource.Name, uid)
			var bodyNode structured.Node
			if targetBodyReader != nil {
				bodyNode = targetBodyReader.Node
			}
			cs.AddRevision(targetPath, &khifilev6.StagingRevision{
				ChangedTime:  commonLogFieldSet.Timestamp,
				ResourceBody: bodyNode,
				Principal:    k8sFieldSet.Principal,
				VerbType:     k8sFieldSet.Verb,
				StateType:    commonlogk8saudit_contract.RevisionStatePodPhaseScheduled,
			})
		}
		return cs, prevGroupData, nil
	}

	if event.GroupRole == "pod" {
		if event.EventType == commonlogk8saudit_contract.ChangeEventTypeV2Creation {
			creationTime, found := GetCreationTimestamp(targetBodyReader)
			if found && commonLogFieldSet.Timestamp.Sub(creationTime) > c.minimumDeltaTimeToCreateInferredCreationRevision {
				targetPath := MustPodPhaseTimelinePath(ctx, k8sFieldSet.ClusterName, nodeName, event.ResourceIdentity.Namespace, event.ResourceIdentity.Name, uid)
				cs.AddRevision(targetPath, &khifilev6.StagingRevision{
					ChangedTime:  creationTime,
					ResourceBody: nil,
					Principal:    "N/A",
					VerbType:     commonlogk8saudit_contract.VerbCreate,
					StateType:    commonlogk8saudit_contract.RevisionStatePodPhaseUnknown,
				})
			}
		}

		phase, found := GetPodPhase(targetBodyReader)
		if !found {
			return cs, prevGroupData, nil
		}

		if prevGroupData.lastPhase != phase || prevGroupData.lastNode != nodeName {
			targetPath := MustPodPhaseTimelinePath(ctx, k8sFieldSet.ClusterName, nodeName, event.ResourceIdentity.Namespace, event.ResourceIdentity.Name, uid)
			var bodyNode structured.Node
			if targetBodyReader != nil {
				bodyNode = targetBodyReader.Node
			}
			cs.AddRevision(targetPath, &khifilev6.StagingRevision{
				ChangedTime:  commonLogFieldSet.Timestamp,
				ResourceBody: bodyNode,
				Principal:    k8sFieldSet.Principal,
				VerbType:     k8sFieldSet.Verb,
				StateType:    phaseToState[phase],
			})
		}
		prevGroupData.lastPhase = phase
		prevGroupData.lastNode = nodeName
	}

	return cs, prevGroupData, nil
}

// MustPodPhaseTimelinePath resolves TimelinePath for PodPhase under Node timeline.
func MustPodPhaseTimelinePath(ctx context.Context, clusterName, nodeName, namespace, podName, uid string) *khifilev6.TimelinePath {
	cluster := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	api := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, cluster, "core/v1")
	kind := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, api, "node")
	nodePath := commonlogk8saudit_contract.MustK8sClusterScopeResourceTimeline(ctx, kind, nodeName)

	builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
	return builder.TimelineAccumulator.GetPath(nodePath, khifilev6.PathSegment{
		Name: fmt.Sprintf("%s/%s[%s]", namespace, podName, uid),
		Type: commonlogk8saudit_contract.TimelineTypePodPhase,
	})
}

// Explicit interface compliance assertion.
var _ commonlogk8saudit_contract.ManifestLogToTimelineMapperV2[*podPhaseTaskState] = (*podPhaseLogToTimelineMapperTaskSettingV2)(nil)

// PodPhaseLogToTimelineMapperTask is the V2 task to generate pod phase history.
var PodPhaseLogToTimelineMapperTask = commonlogk8saudit_contract.NewManifestLogToTimelineMapperV2[*podPhaseTaskState](&podPhaseLogToTimelineMapperTaskSettingV2{
	minimumDeltaTimeToCreateInferredCreationRevision: 5 * time.Second,
})
