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
	"strings"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
)

// namespaceRequestLogToTimelineMapperTaskSetting maps namespace-wide requests to namespace timelines under the model.
type namespaceRequestLogToTimelineMapperTaskSetting struct {
	k8saudit.ManifestStatelessMapperBase
}

// Dependencies implements k8saudit.ManifestLogToTimelineMapper.
func (n *namespaceRequestLogToTimelineMapperTaskSetting) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{}
}

// GroupedLogTask implements k8saudit.ManifestLogToTimelineMapper.
func (n *namespaceRequestLogToTimelineMapperTaskSetting) GroupedLogTask() taskid.TaskReference[k8saudit.ResourceManifestLogGroupMap] {
	return k8saudit.ResourceLifetimeTrackerTaskID.Ref()
}

// LogIngesterTask implements k8saudit.ManifestLogToTimelineMapper.
func (n *namespaceRequestLogToTimelineMapperTaskSetting) LogIngesterTask() taskid.TaskReference[struct{}] {
	return k8saudit.K8sAuditLogIngesterTaskID.Ref()
}

// TaskID implements k8saudit.ManifestLogToTimelineMapper.
func (n *namespaceRequestLogToTimelineMapperTaskSetting) TaskID() taskid.TaskImplementationID[struct{}] {
	return k8saudit.NamespaceRequestLogToTimelineMapperTaskID
}

// ResolveRelatedGroupSets implements k8saudit.ManifestLogToTimelineMapper.
func (n *namespaceRequestLogToTimelineMapperTaskSetting) ResolveRelatedGroupSets(ctx context.Context, groupedLogs k8saudit.ResourceManifestLogGroupMap) ([]k8saudit.RelatedGroupSet, error) {
	result := []k8saudit.RelatedGroupSet{}
	for _, group := range groupedLogs {
		if group.Resource.Type() == k8saudit.Namespace {
			result = append(result, k8saudit.RelatedGroupSet{
				Roles: map[string]*k8saudit.ResourceManifestLogGroup{
					"target": group,
				},
			})
		}
	}
	return result, nil
}

// ProcessLog implements k8saudit.ManifestLogToTimelineMapper.
func (n *namespaceRequestLogToTimelineMapperTaskSetting) ProcessLog(ctx context.Context, event k8saudit.MultiGroupLogEvent, state struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	cs := khifilev6.NewTimelineChangeSet(event.Log)

	k8sFieldSet, err := k8saudit.ExtractK8sAuditLog(ctx, event.Log.NodeReader)
	if err != nil {
		return nil, struct{}{}, err
	}

	cluster := k8saudit.MustK8sClusterTimeline(ctx, k8sFieldSet.ClusterName)
	api := k8saudit.MustK8sAPIVersionTimeline(ctx, cluster, event.ResourceIdentity.APIVersion)
	kind := k8saudit.MustK8sKindTimeline(ctx, api, strings.ToLower(event.ResourceIdentity.Kind))
	nsPath := k8saudit.MustK8sNamespaceTimeline(ctx, kind, event.ResourceIdentity.Namespace)

	cs.AddEvent(nsPath)

	return cs, struct{}{}, nil
}

// Explicit interface compliance assertion.
var _ k8saudit.ManifestLogToTimelineMapper[struct{}] = (*namespaceRequestLogToTimelineMapperTaskSetting)(nil)

// NamespaceRequestLogToTimelineMapperTask is the task to generate events of requests against namespace wide by deletecollection.
var NamespaceRequestLogToTimelineMapperTask = k8saudit.NewManifestLogToTimelineMapper[struct{}](&namespaceRequestLogToTimelineMapperTaskSetting{})
