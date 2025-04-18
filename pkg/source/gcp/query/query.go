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

package query

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	inspection_task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/inspection/contextkey"
	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	error_metadata "github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/error"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/query"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/task/label"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/api"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

const GKEQueryPrefix = gcp_task.GCPPrefix + "query/gke/"

// Query task will return @Skip when query builder decided to skip.
const SkipQueryBody = "@Skip"

type QueryGeneratorFunc = func(context.Context, inspection_task_interface.InspectionTaskMode) ([]string, error)

var queryThreadPool = worker.NewPool(16)

func NewQueryGeneratorTask(taskId taskid.TaskImplementationID[[]*log.LogEntity], readableQueryName string, logType enum.LogType, dependencies []taskid.UntypedTaskReference, generator QueryGeneratorFunc, sampleQuery string) task.Task[[]*log.LogEntity] {
	return inspection_task.NewProgressReportableInspectionTask(taskId, append(dependencies, gcp_task.InputProjectIdTaskID, gcp_task.InputStartTimeTaskID, gcp_task.InputEndTimeTaskID, inspection_task.ReaderFactoryGeneratorTaskID), func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode, progress *progress.TaskProgress) ([]*log.LogEntity, error) {
		client, err := api.DefaultGCPClientFactory.NewClient()
		if err != nil {
			return nil, err
		}

		metadata := khictx.MustGetValue(ctx, inspection_task_contextkey.InspectionRunMetadata)

		projectId := task.GetTaskResult(ctx, gcp_task.InputProjectIdTaskID.GetTaskReference())
		readerFactory := task.GetTaskResult(ctx, inspection_task.ReaderFactoryGeneratorTaskID.GetTaskReference())
		startTime := task.GetTaskResult(ctx, gcp_task.InputStartTimeTaskID.GetTaskReference())
		endTime := task.GetTaskResult(ctx, gcp_task.InputEndTimeTaskID.GetTaskReference())

		queryStrings, err := generator(ctx, taskMode)
		if err != nil {
			return nil, err
		}
		if len(queryStrings) == 0 {
			slog.InfoContext(ctx, fmt.Sprintf("Query generator `%s` decided to skip.", taskId))
			return []*log.LogEntity{}, nil
		}
		queryInfo, found := typedmap.Get(metadata, query.QueryMetadataKey)
		if !found {
			return nil, fmt.Errorf("query metadata was not found")
		}

		allLogs := []*log.LogEntity{}
		for queryIndex, queryString := range queryStrings {
			// Record query information in metadat a
			readableQueryNameForQueryIndex := readableQueryName
			if len(queryStrings) > 1 {
				readableQueryNameForQueryIndex = fmt.Sprintf("%s-%d", readableQueryName, queryIndex)
			}
			finalQuery := fmt.Sprintf("%s\n%s", queryString, queryutil.TimeRangeQuerySection(startTime, endTime, true))
			if len(finalQuery) > 20000 {
				slog.WarnContext(ctx, fmt.Sprintf("Logging filter is exceeding Cloud Logging limitation 20000 charactors\n%s", finalQuery))
			}
			queryInfo.SetQuery(taskId.String(), readableQueryNameForQueryIndex, finalQuery)
			// TODO: not to store whole logs on memory to avoid OOM
			// Run query only when thetask mode is for running
			if taskMode == inspection_task_interface.TaskModeRun {
				worker := queryutil.NewParallelQueryWorker(queryThreadPool, client, queryString, startTime, endTime, 5)
				queryLogs, queryErr := worker.Query(ctx, readerFactory, projectId, progress)
				if queryErr != nil {
					errorMessageSet, found := typedmap.Get(metadata, error_metadata.ErrorMessageSetMetadataKey)
					if !found {
						return nil, fmt.Errorf("error message set metadata was not found")
					}
					if strings.HasPrefix(queryErr.Error(), "401:") {
						errorMessageSet.AddErrorMessage(error_metadata.NewUnauthorizedErrorMessage())
					}
					if strings.HasPrefix(queryErr.Error(), "403:") {
						errorMessageSet.AddErrorMessage(error_metadata.NewPermissionErrorMessage(projectId))
					}
					if strings.HasPrefix(queryErr.Error(), "404:") {
						errorMessageSet.AddErrorMessage(error_metadata.NewNotFoundErrorMessage(projectId))
					}
					return nil, queryErr
				}
				allLogs = append(allLogs, queryLogs...)
			}
		}
		if taskMode == inspection_task_interface.TaskModeRun {
			slices.SortFunc(allLogs, func(a, b *log.LogEntity) int {
				return int(a.Timestamp().Sub(b.Timestamp()))
			})
			for _, l := range allLogs {
				l.LogType = logType
			}
			return allLogs, err
		}

		return []*log.LogEntity{}, err
	}, label.NewQueryTaskLabelOpt(logType, sampleQuery))
}
