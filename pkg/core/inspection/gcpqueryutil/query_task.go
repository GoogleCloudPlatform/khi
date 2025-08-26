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

package gcpqueryutil

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	googlecloudapi "github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// SkipQueryBody is a special query body that indicates the query should be skipped.
// Query task will return @Skip when query builder decided to skip.
const SkipQueryBody = "@Skip"

// QueryGeneratorFunc is a function type that generates Cloud Logging queries.
// A query task may return multiple logging filters because a logging filter has a maximum length,
// and some query tasks need to split a long filter into multiple smaller ones.
type QueryGeneratorFunc = func(context.Context, inspectioncore_contract.InspectionTaskModeType) ([]string, error)

// DefaultResourceNamesGenerator is an interface for generating the default resource names
// used for querying Cloud Logging.
type DefaultResourceNamesGenerator interface {
	// GetDependentTasks returns the list of task references needed for generating resource names.
	GetDependentTasks() []taskid.UntypedTaskReference
	// GenerateResourceNames returns the list of resource names.
	GenerateResourceNames(ctx context.Context) ([]string, error)
}

// ProjectIDDefaultResourceNamesGenerator generates resource names from the project ID.
type ProjectIDDefaultResourceNamesGenerator struct{}

// GenerateResourceNames implements DefaultResourceNamesGenerator.
func (p *ProjectIDDefaultResourceNamesGenerator) GenerateResourceNames(ctx context.Context) ([]string, error) {
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", projectID)}, nil
}

// GetDependentTasks implements DefaultResourceNamesGenerator.
func (p *ProjectIDDefaultResourceNamesGenerator) GetDependentTasks() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
	}
}

var _ DefaultResourceNamesGenerator = (*ProjectIDDefaultResourceNamesGenerator)(nil)

var queryThreadPool = worker.NewPool(16)

// NewCloudLoggingListLogTask creates a new task that lists log entries from Cloud Logging.
func NewCloudLoggingListLogTask(taskId taskid.TaskImplementationID[[]*log.Log], readableQueryName string, logType enum.LogType, dependencies []taskid.UntypedTaskReference, resourceNamesGenerator DefaultResourceNamesGenerator, generator QueryGeneratorFunc, sampleQuery string) coretask.Task[[]*log.Log] {
	return inspectiontaskbase.NewProgressReportableInspectionTask(taskId, append(
		append(dependencies, resourceNamesGenerator.GetDependentTasks()...),
		googlecloudcommon_contract.InputStartTimeTaskID.Ref(),
		googlecloudcommon_contract.InputEndTimeTaskID.Ref(),
		googlecloudcommon_contract.InputLoggingFilterResourceNameTaskID.Ref(),
	), func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) ([]*log.Log, error) {
		client, err := googlecloudapi.DefaultGCPClientFactory.NewClient()
		if err != nil {
			return nil, err
		}

		metadata := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)
		resourceNames := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputLoggingFilterResourceNameTaskID.Ref())
		taskInput := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionTaskInput)

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
				err := googlecloudapi.ValidateResourceNameOnLogEntriesList(resourceNameWithoutSurroundingSpace)
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

		startTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputStartTimeTaskID.Ref())
		endTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputEndTimeTaskID.Ref())

		queryStrings, err := generator(ctx, taskMode)
		if err != nil {
			return nil, err
		}
		if len(queryStrings) == 0 {
			slog.InfoContext(ctx, fmt.Sprintf("Query generator `%s` decided to skip.", taskId))
			return []*log.Log{}, nil
		}
		queryInfo, found := typedmap.Get(metadata, inspectionmetadata.QueryMetadataKey)
		if !found {
			return nil, fmt.Errorf("query metadata was not found")
		}

		allLogs := []*log.Log{}
		for queryIndex, queryString := range queryStrings {
			// Record query information in metadat a
			readableQueryNameForQueryIndex := readableQueryName
			if len(queryStrings) > 1 {
				readableQueryNameForQueryIndex = fmt.Sprintf("%s-%d", readableQueryName, queryIndex)
			}
			finalQuery := fmt.Sprintf("%s\n%s", queryString, TimeRangeQuerySection(startTime, endTime, true))
			if len(finalQuery) > 20000 {
				slog.WarnContext(ctx, fmt.Sprintf("Logging filter is exceeding Cloud Logging limitation 20000 charactors\n%s", finalQuery))
			}
			queryInfo.SetQuery(taskId.String(), readableQueryNameForQueryIndex, finalQuery)
			// TODO: not to store whole logs on memory to avoid OOM
			// Run query only when thetask mode is for running
			if taskMode == inspectioncore_contract.TaskModeRun {
				worker := NewParallelCloudLoggingListWorker(queryThreadPool, client, queryString, startTime, endTime, 5)
				queryLogs, queryErr := worker.Query(ctx, resourceNamesFromInput, progress)
				if queryErr != nil {
					errorMessageSet, found := typedmap.Get(metadata, inspectionmetadata.ErrorMessageSetMetadataKey)
					if !found {
						return nil, fmt.Errorf("error message set metadata was not found")
					}
					if strings.HasPrefix(queryErr.Error(), "401:") {
						errorMessageSet.AddErrorMessage(inspectionmetadata.NewUnauthorizedErrorMessage())
					}
					// TODO: these errors are shown to frontend but it's not well implemented.
					if strings.HasPrefix(queryErr.Error(), "403:") {
						errorMessageSet.AddErrorMessage(&inspectionmetadata.ErrorMessage{
							ErrorId: 0,
							Message: queryErr.Error(),
						})
					}
					if strings.HasPrefix(queryErr.Error(), "404:") {
						errorMessageSet.AddErrorMessage(&inspectionmetadata.ErrorMessage{
							ErrorId: 0,
							Message: queryErr.Error(),
						})
					}
					return nil, queryErr
				}
				allLogs = append(allLogs, queryLogs...)
			}
		}

		for _, l := range allLogs {
			l.SetFieldSetReader(&GCPCommonFieldSetReader{})
			l.SetFieldSetReader(&GCPMainMessageFieldSetReader{})
		}

		if taskMode == inspectioncore_contract.TaskModeRun {
			slices.SortFunc(allLogs, func(a, b *log.Log) int {
				commonFieldSetForA, _ := log.GetFieldSet(a, &log.CommonFieldSet{}) // errors are safely ignored because this field set is required in previous steps
				commonFieldSetForB, _ := log.GetFieldSet(b, &log.CommonFieldSet{})
				return int(commonFieldSetForA.Timestamp.Sub(commonFieldSetForB.Timestamp))
			})
			for _, l := range allLogs {
				l.LogType = logType
			}
			return allLogs, err
		}

		return []*log.Log{}, err
	}, inspectioncore_contract.NewQueryTaskLabelOpt(logType, sampleQuery))
}
