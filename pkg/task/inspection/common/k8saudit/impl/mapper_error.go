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

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
)

// NonSuccessLogLogToTimelineMapperTask is the task to generate history from non-success logs.
var NonSuccessLogLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[struct{}](
	k8saudit.NonSuccessLogLogToTimelineMapperTaskID,
	&nonSuccessLogLogToTimelineMapperTaskSetting{
		subresourceMapToWriteToParent: map[string]struct{}{
			"status":   {},
			"finalize": {},
			"approve":  {},
		},
	},
)

type nonSuccessLogLogToTimelineMapperTaskSetting struct {
	inspectiontaskbase.StatelessMapperBase

	// subresourceMapToWriteToParent is the map of subresources to write to the parent resource.
	subresourceMapToWriteToParent map[string]struct{}
}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (e *nonSuccessLogLogToTimelineMapperTaskSetting) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		k8saudit.K8sAuditLogExtractorRef.Ref(coretask.FromActiveGraph),
	}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (e *nonSuccessLogLogToTimelineMapperTaskSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return k8saudit.NonSuccessLogGrouperTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (e *nonSuccessLogLogToTimelineMapperTaskSetting) LogIngesterTask() taskid.TaskReference[struct{}] {
	return k8saudit.K8sAuditLogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (e *nonSuccessLogLogToTimelineMapperTaskSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	fieldSet, _ := k8saudit.ExtractK8sAuditLog(ctx, l.NodeReader)

	subresource := fieldSet.SubresourceName
	if _, ok := e.subresourceMapToWriteToParent[subresource]; subresource != "" && ok {
		subresource = ""
	}

	// Resolve TimelinePath hierarchically.
	cluster := k8saudit.MustK8sClusterTimeline(ctx, fieldSet.ClusterName)
	api := k8saudit.MustK8sAPIVersionTimeline(ctx, cluster, fieldSet.APIVersion)
	kind := k8saudit.MustK8sKindTimeline(ctx, api, strings.ToLower(k8saudit.GetSingularKindName(fieldSet.PluralKind)))

	var resPath *khifilev6.TimelinePath
	if fieldSet.Namespace != k8saudit.ClusterScopeNamespace && fieldSet.Namespace != "" {
		ns := k8saudit.MustK8sNamespaceTimeline(ctx, kind, fieldSet.Namespace)
		resPath = k8saudit.MustK8sNamespacedResourceTimeline(ctx, ns, fieldSet.ResourceName)
	} else {
		resPath = k8saudit.MustK8sClusterScopeResourceTimeline(ctx, kind, fieldSet.ResourceName)
	}

	if subresource != "" {
		resPath = k8saudit.MustK8sSubresourceTimeline(ctx, resPath, subresource)
	}

	cs := khifilev6.NewTimelineChangeSet(l)
	cs.AddEvent(resPath)

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*nonSuccessLogLogToTimelineMapperTaskSetting)(nil)
