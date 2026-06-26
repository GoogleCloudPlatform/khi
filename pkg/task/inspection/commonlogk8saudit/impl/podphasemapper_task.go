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

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

const (
	podPhasePassCollectUID      = 0
	podPhasePassCollectNodeName = 1
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
	// parentPathToUIDMap is the map of parent pod path to UID history.
	parentPathToUIDMap map[string]*common.TimeSeries[string]
	// uidToCreationTimestampMap maps pod UID to its creationTimestamp.
	uidToCreationTimestampMap map[string]time.Time
}

func newPodPhaseTaskState() *podPhaseTaskState {
	return &podPhaseTaskState{
		uidToNodeNameMap:          map[string]string{},
		parentPathToUIDMap:        map[string]*common.TimeSeries[string]{},
		uidToCreationTimestampMap: map[string]time.Time{},
	}
}

// podPhaseLogToTimelineMapperTaskSettingV2 handles pod phase timeline generation.
// This mapper runs in 2 passes during PreProcessLog to associate pod UIDs with node names,
// even when scheduling (binding) logs and pod status logs appear out of order or when pod logs are missing:
//   - Pass 0 (podPhasePassCollectUID): Collects the mapping from parent pod path to its UID from "pod" role logs.
//   - Pass 1 (podPhasePassCollectNodeName): Resolves the node name from either "pod" or "binding" logs and
//     stores the mapping from UID to node name. A 2-pass approach is necessary because a chronological log stream
//     might process "binding" logs (which lack pod UID) before "pod" logs (where the UID is first observed).
type podPhaseLogToTimelineMapperTaskSettingV2 struct {
}

// Dependencies implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *podPhaseLogToTimelineMapperTaskSettingV2) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// PassCount implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *podPhaseLogToTimelineMapperTaskSettingV2) PassCount() int {
	return 2
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
	processedPods := map[string]struct{}{}

	for _, group := range groupedLogs {
		var podPath string
		var podGroup *commonlogk8saudit_contract.ResourceManifestLogGroup
		var bindingGroup *commonlogk8saudit_contract.ResourceManifestLogGroup

		if group.Resource.APIVersion == "core/v1" && group.Resource.Kind == "pod" {
			if group.Resource.Type() == commonlogk8saudit_contract.Resource {
				podPath = group.Resource.String()
				podGroup = group
				bindingGroup = groupedLogs[group.Resource.SubresourceIdentity("binding").String()]
			} else if group.Resource.Type() == commonlogk8saudit_contract.Subresource && group.Resource.SubresourceName == "binding" {
				parent := *group.Resource
				parent.SubresourceName = ""
				podPath = parent.String()
				podGroup = groupedLogs[podPath]
				bindingGroup = group
			}
		}

		if podPath != "" {
			if _, processed := processedPods[podPath]; !processed {
				processedPods[podPath] = struct{}{}
				result = append(result, commonlogk8saudit_contract.RelatedGroupSet{
					Roles: map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{
						"pod":     podGroup,
						"binding": bindingGroup,
					},
				})
			}
		}
	}
	return result, nil
}

// resolveUID returns the active pod UID at time t.
func (c *podPhaseLogToTimelineMapperTaskSettingV2) resolveUID(ts *common.TimeSeries[string], t time.Time) string {
	uid, ok := ts.Get(t)
	if ok && uid != "" {
		return uid
	}
	return ""
}

// PreProcessLog implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *podPhaseLogToTimelineMapperTaskSettingV2) PreProcessLog(ctx context.Context, passIndex int, event commonlogk8saudit_contract.MultiGroupLogEvent, prevGroupData *podPhaseTaskState) (*podPhaseTaskState, error) {
	if prevGroupData == nil {
		prevGroupData = newPodPhaseTaskState()
	}

	commonLogFieldSet := log.MustGetFieldSet(event.Log, &log.CommonFieldSet{})
	eventTime := commonLogFieldSet.Timestamp

	switch passIndex {
	// The first pass to collect UIDs & creationTimestamps from Pods.
	case podPhasePassCollectUID:
		if event.GroupRole == "pod" {
			path := event.ResourceIdentity.String()
			ts, found := prevGroupData.parentPathToUIDMap[path]
			if !found {
				ts = common.NewTimeSeries[string]()
				prevGroupData.parentPathToUIDMap[path] = ts
			}

			if event.EventType == commonlogk8saudit_contract.ChangeEventTypeV2Deletion {
				ts.Set(eventTime, "")
			} else {
				bodyReader, ok := event.GetLastBodyReader("pod")
				if ok && bodyReader != nil {
					uid, _ := GetUID(bodyReader)
					if uid != "" {
						ts.Set(eventTime, uid)
						creationTime, found := GetCreationTimestamp(bodyReader)
						if found {
							prevGroupData.uidToCreationTimestampMap[uid] = creationTime
						}
					}
				}
			}
		}
	// Secound pass to gather node names for Pods for each uids.
	// This step can't be merged with the first pass for the case when the first log is binding and it has no Pod uid in it.
	case podPhasePassCollectNodeName:
		switch event.GroupRole {
		case "pod":
			bodyReader, ok := event.GetLastBodyReader("pod")
			if ok && bodyReader != nil {
				uid, found := GetUID(bodyReader)
				if found && uid != "" {
					nodeName, found := GetNodeNameOfPod(bodyReader)
					if found && nodeName != "" {
						prevGroupData.uidToNodeNameMap[uid] = nodeName
					}
				}
			}
		case "binding":
			bodyReader, ok := event.GetLastBodyReader("binding")
			if ok && bodyReader != nil {
				parent := *event.ResourceIdentity
				parent.SubresourceName = ""
				parentPath := parent.String()
				uid := ""
				if ts, found := prevGroupData.parentPathToUIDMap[parentPath]; found {
					uid = c.resolveUID(ts, eventTime)
				}
				if uid != "" {
					nodeName, found := GetNodeNameOfBinding(bodyReader)
					if found && nodeName != "" {
						prevGroupData.uidToNodeNameMap[uid] = nodeName
					}
				}
			}
		}
	}
	return prevGroupData, nil
}

// ProcessLog implements commonlogk8saudit_contract.ManifestLogToTimelineMapperV2.
func (c *podPhaseLogToTimelineMapperTaskSettingV2) ProcessLog(ctx context.Context, event commonlogk8saudit_contract.MultiGroupLogEvent, prevGroupData *podPhaseTaskState) (*khifilev6.TimelineChangeSet, *podPhaseTaskState, error) {
	cs := khifilev6.NewTimelineChangeSet(event.Log)

	commonLogFieldSet := log.MustGetFieldSet(event.Log, &log.CommonFieldSet{})
	k8sFieldSet := log.MustGetFieldSet(event.Log, &commonlogk8saudit_contract.K8sAuditLogFieldSet{})
	eventTime := commonLogFieldSet.Timestamp

	var targetBodyReader *structured.NodeReader
	if reader, ok := event.GetLastBodyReader("pod"); ok {
		targetBodyReader = reader
	}

	var uid string
	var found bool
	if targetBodyReader != nil {
		uid, found = GetUID(targetBodyReader)
	}
	if !found {
		parent := *event.ResourceIdentity
		parent.SubresourceName = ""
		if ts, found := prevGroupData.parentPathToUIDMap[parent.String()]; found {
			uid = c.resolveUID(ts, eventTime)
		}
	}
	if uid == "" {
		return cs, prevGroupData, nil
	}

	nodeName, found := prevGroupData.uidToNodeNameMap[uid]
	if !found {
		return cs, prevGroupData, nil
	}

	// Place the first revision into the changeset.
	if event.EventType == commonlogk8saudit_contract.ChangeEventTypeV2Creation {
		nonCreationForPod := event.GroupRole == "pod" && k8sFieldSet.Verb != commonlogk8saudit_contract.VerbCreate
		if nonCreationForPod {
			creationTime, found := prevGroupData.uidToCreationTimestampMap[uid]
			var changedTime time.Time
			if found {
				changedTime = creationTime
			} else {
				changedTime = time.Unix(0, 0).UTC()
			}
			targetPath := MustPodPhaseTimelinePath(ctx, k8sFieldSet.ClusterName, nodeName, event.ResourceIdentity.Namespace, event.ResourceIdentity.Name, uid)
			cs.AddRevision(targetPath, &khifilev6.StagingRevision{
				ChangedTime:  changedTime,
				ResourceBody: nil,
				Principal:    "N/A",
				VerbType:     commonlogk8saudit_contract.VerbCreate,
				StateType:    commonlogk8saudit_contract.RevisionStatePodPhaseUnknown,
			})
		}
	}

	switch event.GroupRole {
	case "binding":
		if event.EventType == commonlogk8saudit_contract.ChangeEventTypeV2Creation {
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
				StateType:    commonlogk8saudit_contract.RevisionStatePodPhaseScheduled,
			})
		}
		return cs, prevGroupData, nil
	case "pod":
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
		return cs, prevGroupData, nil
	default:
		panic("unreachable: unexpected group role: " + event.GroupRole)
	}
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
var PodPhaseLogToTimelineMapperTask = commonlogk8saudit_contract.NewManifestLogToTimelineMapperV2[*podPhaseTaskState](&podPhaseLogToTimelineMapperTaskSettingV2{})
