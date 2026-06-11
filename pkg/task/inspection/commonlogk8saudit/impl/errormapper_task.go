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

// NonSuccessLogLogToTimelineMapperTask is the V2 task to generate history from non-success logs.
var NonSuccessLogLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2[struct{}](
	commonlogk8saudit_contract.NonSuccessLogLogToTimelineMapperTaskID,
	&nonSuccessLogLogToTimelineMapperTaskSettingV2{
		subresourceMapToWriteToParent: map[string]struct{}{
			"status":   {},
			"finalize": {},
			"approve":  {},
		},
	},
)

type nonSuccessLogLogToTimelineMapperTaskSettingV2 struct {
	inspectiontaskbase.StatelessMapperBase

	// subresourceMapToWriteToParent is the map of subresources to write to the parent resource.
	subresourceMapToWriteToParent map[string]struct{}
}

// Dependencies implements inspectiontaskbase.LogToTimelineMapperV2.
func (e *nonSuccessLogLogToTimelineMapperTaskSettingV2) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapperV2.
func (e *nonSuccessLogLogToTimelineMapperTaskSettingV2) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return commonlogk8saudit_contract.NonSuccessLogGrouperTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapperV2.
func (e *nonSuccessLogLogToTimelineMapperTaskSettingV2) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return commonlogk8saudit_contract.K8sAuditLogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapperV2.
func (e *nonSuccessLogLogToTimelineMapperTaskSettingV2) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	fieldSet := log.MustGetFieldSet(l, &commonlogk8saudit_contract.K8sAuditLogFieldSet{})

	subresource := fieldSet.SubresourceName
	if _, ok := e.subresourceMapToWriteToParent[subresource]; subresource != "" && ok {
		subresource = ""
	}

	// Resolve TimelinePath hierarchically.
	cluster := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, fieldSet.ClusterName)
	api := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, cluster, fieldSet.APIVersion)
	kind := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, api, strings.ToLower(commonlogk8saudit_contract.GetSingularKindName(fieldSet.PluralKind)))

	var resPath *khifilev6.TimelinePath
	if fieldSet.Namespace != "cluster-scope" && fieldSet.Namespace != "" {
		ns := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kind, fieldSet.Namespace)
		resPath = commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, ns, fieldSet.ResourceName)
	} else {
		resPath = commonlogk8saudit_contract.MustK8sClusterScopeResourceTimeline(ctx, kind, fieldSet.ResourceName)
	}

	if subresource != "" {
		resPath = commonlogk8saudit_contract.MustK8sSubresourceTimeline(ctx, resPath, subresource)
	}

	cs := khifilev6.NewTimelineChangeSet(l)
	cs.AddEvent(resPath)

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapperV2[struct{}] = (*nonSuccessLogLogToTimelineMapperTaskSettingV2)(nil)
