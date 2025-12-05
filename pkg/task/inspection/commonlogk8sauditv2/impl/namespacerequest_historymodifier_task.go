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

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// NamespaceRequestHistoryModifierTask is a task to generate events of requests against namespace wide by deletecollection.
// TODO: This must be reimplemented with the HistoryModifierTask once it supports receiving group path.
var NamespaceRequestHistoryModifierTask = inspectiontaskbase.NewProgressReportableInspectionTask(commonlogk8sauditv2_contract.NamespaceRequestHistoryModifierTaskID, []taskid.UntypedTaskReference{
	commonlogk8sauditv2_contract.K8sAuditLogSerializerTaskID.Ref(),
	commonlogk8sauditv2_contract.ChangeTargetGrouperTaskID.Ref(),
}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, tp *inspectionmetadata.TaskProgressMetadata) (struct{}, error) {
	if taskMode == inspectioncore_contract.TaskModeDryRun {
		return struct{}{}, nil
	}

	builder := khictx.MustGetValue(ctx, inspectioncore_contract.CurrentHistoryBuilder)
	logs := coretask.GetTaskResult(ctx, commonlogk8sauditv2_contract.ChangeTargetGrouperTaskID.Ref())

	changedPaths := map[string]struct{}{}
	for _, group := range logs {
		if group.Resource.Type() == commonlogk8sauditv2_contract.Namespace {
			for _, l := range group.Logs {
				cs := history.NewChangeSet(l)
				cs.AddEvent(resourcepath.ResourcePath{
					Path:               group.Resource.ResourcePathString(),
					ParentRelationship: enum.RelationshipChild,
				})
				cp, err := cs.FlushToHistory(builder)
				if err != nil {
					return struct{}{}, err
				}
				for _, path := range cp {
					changedPaths[path] = struct{}{}
				}
			}
		}
	}
	for path := range changedPaths {
		tb := builder.GetTimelineBuilder(path)
		tb.Sort()
	}
	return struct{}{}, nil
})
