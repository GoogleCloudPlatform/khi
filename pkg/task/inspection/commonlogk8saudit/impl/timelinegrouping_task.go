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

package commonlogk8saudit_impl

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/grouper"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var TimelineGroupingTask = inspectiontaskbase.NewProgressReportableInspectionTask(commonlogk8saudit_contract.TimelineGroupingTaskID, []taskid.UntypedTaskReference{
	commonlogk8saudit_contract.CommonLogParseTaskID.Ref(),
}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, tp *inspectionmetadata.TaskProgressMetadata) ([]*commonlogk8saudit_contract.TimelineGrouperResult, error) {
	if taskMode == inspectioncore_contract.TaskModeDryRun {
		return nil, nil
	}
	preStepParseResult := coretask.GetTaskResult(ctx, commonlogk8saudit_contract.CommonLogParseTaskID.Ref())
	progressUpdater := progressutil.NewIndeterminateUpdator(tp, time.Second)
	err := progressUpdater.Start("Grouping logs by timeline")
	if err != nil {
		return nil, err
	}
	defer progressUpdater.Done()

	timelineGrouper := grouper.NewBasicGrouper(func(input *commonlogk8saudit_contract.AuditLogParserInput) string {
		return input.Operation.CovertToResourcePath()
	})
	groups := timelineGrouper.Group(preStepParseResult)
	result := []*commonlogk8saudit_contract.TimelineGrouperResult{}
	for key, group := range groups {
		result = append(result, &commonlogk8saudit_contract.TimelineGrouperResult{
			TimelineResourcePath: key,
			PreParsedLogs:        group,
		})
	}
	createDeletionRequestsByDeleteColection(result)
	return result, nil
})

func createDeletionRequestsByDeleteColection(groups []*commonlogk8saudit_contract.TimelineGrouperResult) {
	requireSortTimelinePaths := map[string]struct{}{}
	for _, group := range groups {
		// delete collection only happens for namespace layer
		if strings.HasSuffix(group.TimelineResourcePath, "#") {
			for _, l := range group.PreParsedLogs {
				if l.Operation.Verb == enum.RevisionVerbDeleteCollection {
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
							refLogCommonField := log.MustGetFieldSet(refLog.Log, &log.CommonFieldSet{})
							logCommonField := log.MustGetFieldSet(l.Log, &log.CommonFieldSet{})
							if refLogCommonField.Timestamp.Sub(logCommonField.Timestamp) > 0 {
								// This delete collection happened before the resource existing. ignore the delete collection request.
								continue
							}
							childGroup.PreParsedLogs = append(childGroup.PreParsedLogs, &commonlogk8saudit_contract.AuditLogParserInput{
								Log:                                    l.Log,
								Requestor:                              l.Requestor,
								Operation:                              &k8sOp,
								ResponseErrorCode:                      l.ResponseErrorCode,
								ResponseErrorMessage:                   l.ResponseErrorMessage,
								IsErrorResponse:                        l.IsErrorResponse,
								Request:                                nil,
								RequestType:                            commonlogk8saudit_contract.RTypeUnknown,
								Response:                               nil,
								ResponseType:                           commonlogk8saudit_contract.RTypeUnknown,
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
				logICommonField := log.MustGetFieldSet(group.PreParsedLogs[i].Log, &log.CommonFieldSet{})
				logJCommonField := log.MustGetFieldSet(group.PreParsedLogs[j].Log, &log.CommonFieldSet{})
				return logICommonField.Timestamp.Sub(logJCommonField.Timestamp) <= 0
			})
		}
	}
}
