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
	gcp_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

const GKEQueryPrefix = gcp_task.GCPPrefix + "query/gke/"

// Query task will return @Skip when query builder decided to skip.
const SkipQueryBody = "@Skip"

type QueryGeneratorFunc = func(context.Context, inspection_task_interface.InspectionTaskMode) ([]string, error)

// DefaultResourceNamesGenerator returns the default resource names used for querying Cloud Logging.
type DefaultResourceNamesGenerator interface {
	// GetDependentTasks returns the list of taks references needed for generating resource names.
	GetDependentTasks() []taskid.UntypedTaskReference
	// GenerateResourceNames returns the list of resource names.
	GenerateResourceNames(ctx context.Context) ([]string, error)
}

type ProjectIDDefaultResourceNamesGenerator struct{}

// GenerateResourceNames implements DefaultResourceNamesGenerator.
func (p *ProjectIDDefaultResourceNamesGenerator) GenerateResourceNames(ctx context.Context) ([]string, error) {
	projectID := task.GetTaskResult(ctx, gcp_task.InputProjectIdTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", projectID)}, nil
}

// GetDependentTasks implements DefaultResourceNamesGenerator.
func (p *ProjectIDDefaultResourceNamesGenerator) GetDependentTasks() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		gcp_task.InputProjectIdTaskID.Ref(),
	}
}

var _ DefaultResourceNamesGenerator = (*ProjectIDDefaultResourceNamesGenerator)(nil)

var queryThreadPool = worker.NewPool(16)

func NewQueryGeneratorTask(taskId taskid.TaskImplementationID[[]*log.LogEntity], readableQueryName string, logType enum.LogType, dependencies []taskid.UntypedTaskReference, resourceNamesGenerator DefaultResourceNamesGenerator, generator QueryGeneratorFunc, sampleQuery string) task.Task[[]*log.LogEntity] {
	return inspection_task.NewProgressReportableInspectionTask(taskId, append(
		append(dependencies, resourceNamesGenerator.GetDependentTasks()...),
		gcp_task.InputStartTimeTaskID.Ref(),
		gcp_task.InputEndTimeTaskID.Ref(),
		inspection_task.ReaderFactoryGeneratorTaskID.Ref(),
		gcp_taskid.LoggingFilterResourceNameInputTaskID.Ref(),
	), func(ctx context.Context, taskMode inspection_task_interface.InspectionTaskMode, progress *progress.TaskProgress) ([]*log.LogEntity, error) {
		client, err := api.DefaultGCPClientFactory.NewClient()
		if err != nil {
			return nil, err
		}

		metadata := khictx.MustGetValue(ctx, inspection_task_contextkey.InspectionRunMetadata)
		resourceNames := task.GetTaskResult(ctx, gcp_taskid.LoggingFilterResourceNameInputTaskID.Ref())
		taskInput := khictx.MustGetValue(ctx, inspection_task_contextkey.InspectionTaskInput)

		defaultResourceNames, err := resourceNamesGenerator.GenerateResourceNames(ctx)
		if err != nil {
			return nil, err
		}

		resourceNames.UpdateDefaultResourceNamesForQuery(taskId.ReferenceIDString(), defaultResourceNames)
		queryResourceNamePair := resourceNames.GetResourceNamesForQuery(taskId.ReferenceIDString())
		resourceNamesFromInput := defaultResourceNames
		inputStr, found := taskInput[queryResourceNamePair.GetInputID()]
		if found {
			resourceNamesFromInput = strings.Split(inputStr.(string), " ")
			resourceNamesList := []string{}
			hadError := false
			for _, resourceNameFromInput := range resourceNamesFromInput {
				resourceNameWithoutSurroundingSpace := strings.TrimSpace(resourceNameFromInput)
				err := api.ValidateResourceNameOnLogEntriesList(resourceNameWithoutSurroundingSpace)
				if err != nil {
					hadError = true
					break
				}
				resourceNamesList = append(resourceNamesList, resourceNameWithoutSurroundingSpace)
			}
			if !hadError {
				resourceNamesFromInput = resourceNamesList
			}
		}

		readerFactory := task.GetTaskResult(ctx, inspection_task.ReaderFactoryGeneratorTaskID.Ref())
		startTime := task.GetTaskResult(ctx, gcp_task.InputStartTimeTaskID.Ref())
		endTime := task.GetTaskResult(ctx, gcp_task.InputEndTimeTaskID.Ref())

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
				queryLogs, queryErr := worker.Query(ctx, readerFactory, resourceNamesFromInput, progress)
				if queryErr != nil {
					errorMessageSet, found := typedmap.Get(metadata, error_metadata.ErrorMessageSetMetadataKey)
					if !found {
						return nil, fmt.Errorf("error message set metadata was not found")
					}
					if strings.HasPrefix(queryErr.Error(), "401:") {
						errorMessageSet.AddErrorMessage(error_metadata.NewUnauthorizedErrorMessage())
					}
					// TODO: these errors are shown to frontend but it's not well implemented.
					if strings.HasPrefix(queryErr.Error(), "403:") {
						errorMessageSet.AddErrorMessage(&error_metadata.ErrorMessage{
							ErrorId: 0,
							Message: queryErr.Error(),
						})
					}
					if strings.HasPrefix(queryErr.Error(), "404:") {
						errorMessageSet.AddErrorMessage(&error_metadata.ErrorMessage{
							ErrorId: 0,
							Message: queryErr.Error(),
						})
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
