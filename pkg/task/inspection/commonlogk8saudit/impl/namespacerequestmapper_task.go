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
	"strings"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
)

// namespaceRequestLogToTimelineMapperTaskSetting maps namespace-wide requests to namespace timelines under the model.
type namespaceRequestLogToTimelineMapperTaskSetting struct {
	commonlogk8saudit_contract.ManifestStatelessMapperBase
}

// Dependencies implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (n *namespaceRequestLogToTimelineMapperTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (n *namespaceRequestLogToTimelineMapperTaskSetting) GroupedLogTask() taskid.TaskReference[commonlogk8saudit_contract.ResourceManifestLogGroupMap] {
	return commonlogk8saudit_contract.ResourceLifetimeTrackerTaskID.Ref()
}

// LogIngesterTask implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (n *namespaceRequestLogToTimelineMapperTaskSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return commonlogk8saudit_contract.K8sAuditLogIngesterTaskID.Ref()
}

// TaskID implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (n *namespaceRequestLogToTimelineMapperTaskSetting) TaskID() taskid.TaskImplementationID[inspectiontaskbase.TimelineMapperResult] {
	return commonlogk8saudit_contract.NamespaceRequestLogToTimelineMapperTaskID
}

// ResolveRelatedGroupSets implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (n *namespaceRequestLogToTimelineMapperTaskSetting) ResolveRelatedGroupSets(ctx context.Context, groupedLogs commonlogk8saudit_contract.ResourceManifestLogGroupMap) ([]commonlogk8saudit_contract.RelatedGroupSet, error) {
	result := []commonlogk8saudit_contract.RelatedGroupSet{}
	for _, group := range groupedLogs {
		if group.Resource.Type() == commonlogk8saudit_contract.Namespace {
			result = append(result, commonlogk8saudit_contract.RelatedGroupSet{
				Roles: map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{
					"target": group,
				},
			})
		}
	}
	return result, nil
}

// ProcessLog implements commonlogk8saudit_contract.ManifestLogToTimelineMapper.
func (n *namespaceRequestLogToTimelineMapperTaskSetting) ProcessLog(ctx context.Context, event commonlogk8saudit_contract.MultiGroupLogEvent, state struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	cs := khifilev6.NewTimelineChangeSet(event.Log)

	k8sFieldSet, err := log.GetFieldSet(event.Log, &commonlogk8saudit_contract.K8sAuditLogFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}

	cluster := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, k8sFieldSet.ClusterName)
	api := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, cluster, event.ResourceIdentity.APIVersion)
	kind := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, api, strings.ToLower(event.ResourceIdentity.Kind))
	nsPath := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kind, event.ResourceIdentity.Namespace)

	cs.AddEvent(nsPath)

	return cs, struct{}{}, nil
}

// Explicit interface compliance assertion.
var _ commonlogk8saudit_contract.ManifestLogToTimelineMapper[struct{}] = (*namespaceRequestLogToTimelineMapperTaskSetting)(nil)

// NamespaceRequestLogToTimelineMapperTask is the task to generate events of requests against namespace wide by deletecollection.
var NamespaceRequestLogToTimelineMapperTask = commonlogk8saudit_contract.NewManifestLogToTimelineMapper[struct{}](&namespaceRequestLogToTimelineMapperTaskSetting{})
