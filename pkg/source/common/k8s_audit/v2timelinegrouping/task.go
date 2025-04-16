// Copyright 2024 Google LLC
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

package v2timelinegrouping

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/grouper"
	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/rtype"
	common_k8saudit_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var Task = inspection_task.NewProgressReportableInspectionTask(common_k8saudit_taskid.TimelineGroupingTaskID, []taskid.UntypedTaskReference{
	common_k8saudit_taskid.CommonLogParseTaskID,
}, func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode, tp *progress.TaskProgress) ([]*types.TimelineGrouperResult, error) {
	if taskMode == inspection_task_interface.TaskModeDryRun {
		return nil, nil
	}
	preStepParseResult := task.GetTaskResult(ctx, common_k8saudit_taskid.CommonLogParseTaskID.GetTaskReference())
	progressUpdater := progress.NewIndeterminateUpdator(tp, time.Second)
	err := progressUpdater.Start("Grouping logs by timeline")
	if err != nil {
		return nil, err
	}
	defer progressUpdater.Done()

	timelineGrouper := grouper.NewBasicGrouper(func(input *types.AuditLogParserInput) string { return input.Operation.CovertToResourcePath() })
	groups := timelineGrouper.Group(preStepParseResult)
	result := []*types.TimelineGrouperResult{}
	for key, group := range groups {
		result = append(result, &types.TimelineGrouperResult{
			TimelineResourcePath: key,
			PreParsedLogs:        group,
		})
	}
	createDeletionRequestsByDeleteColection(result)
	return result, nil
})

func createDeletionRequestsByDeleteColection(groups []*types.TimelineGrouperResult) {
	requireSortTimelinePaths := map[string]struct{}{}
	for _, group := range groups {
		// delete collection only happens for namespace layer
		if strings.HasSuffix(group.TimelineResourcePath, "#") {
			for _, log := range group.PreParsedLogs {
				if log.Operation.Verb == enum.RevisionVerbDeleteCollection {
					for _, childGroup := range groups {
						// find any timelines under current timeline
						if childGroup.TimelineResourcePath != group.TimelineResourcePath && strings.HasPrefix(childGroup.TimelineResourcePath, group.TimelineResourcePath) {
							refLog := childGroup.PreParsedLogs[0]
							k8sOp := model.KubernetesObjectOperation{
								APIVersion: refLog.Operation.APIVersion,
								PluralKind: refLog.Operation.PluralKind,
								Namespace:  refLog.Operation.Namespace,
								Name:       refLog.Operation.Name,
								Verb:       enum.RevisionVerbDelete,
							}
							if refLog.Log.Timestamp().Sub(log.Log.Timestamp()) > 0 {
								// This delete collection happened before the resource existing. ignore the delete collection request.
								continue
							}
							childGroup.PreParsedLogs = append(childGroup.PreParsedLogs, &types.AuditLogParserInput{
								Log:                                    log.Log,
								Requestor:                              log.Requestor,
								Operation:                              &k8sOp,
								ResponseErrorCode:                      log.ResponseErrorCode,
								ResponseErrorMessage:                   log.ResponseErrorMessage,
								IsErrorResponse:                        log.IsErrorResponse,
								Request:                                nil,
								RequestType:                            rtype.RTypeUnknown,
								Response:                               nil,
								ResponseType:                           rtype.RTypeUnknown,
								GeneratedFromDeleteCollectionOperation: true,
							})
							requireSortTimelinePaths[childGroup.TimelineResourcePath] = struct{}{}
						}
					}
				}
			}
		}
	}
	// sort logs with additional deletion logs in timeline
	for _, group := range groups {
		if _, found := requireSortTimelinePaths[group.TimelineResourcePath]; found {
			sort.Slice(group.PreParsedLogs, func(i, j int) bool {
				return group.PreParsedLogs[i].Log.Timestamp().Sub(group.PreParsedLogs[j].Log.Timestamp()) <= 0
			})
		}
	}
}
