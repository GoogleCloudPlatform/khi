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

package googlecloudcommon_contract

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"slices"
	"sync"
	"time"

	"cloud.google.com/go/logging/apiv2/loggingpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logconvert"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

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
	projectID := coretask.GetTaskResult(ctx, InputProjectIdTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", projectID)}, nil
}

// GetDependentTasks implements DefaultResourceNamesGenerator.
func (p *ProjectIDDefaultResourceNamesGenerator) GetDependentTasks() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		InputProjectIdTaskID.Ref(),
	}
}

func monitorProgress(ctx context.Context, wg *sync.WaitGroup, source <-chan LogFetchProgress, progressDest *inspectionmetadata.TaskProgressMetadata, listCallIndex int, allListCalls int) {
	wg.Add(1)
	startingTime := time.Now()
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case progress, ok := <-source:
				if !ok {
					return
				}
				current := time.Now()
				elapsed := current.Sub(startingTime).Seconds()
				var lps float64
				if elapsed > 0 {
					lps = float64(progress.LogCount) / elapsed
				}
				completeRatio := (float32(listCallIndex) + progress.Progress) / float32(allListCalls)
				progressDest.Update(completeRatio, fmt.Sprintf("%d logs fetched(%.2f lps)[%d/%d]", progress.LogCount, lps, listCallIndex, allListCalls))
			}
		}
	}()
}

func convertLogsArray(ctx context.Context, wg *sync.WaitGroup, source <-chan *loggingpb.LogEntry, dest *[]*log.Log, logType enum.LogType) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case l, ok := <-source:
				if !ok {
					return
				}
				node, err := logconvert.LogEntryToNode(l)
				if err != nil {
					slog.WarnContext(ctx, fmt.Sprintf("failed to convert loggingpb.LogEntry (insertId: %s, timestamp: %v) to structured.Node %v", l.InsertId, l.Timestamp, err))
					continue
				}
				khiLog := log.NewLog(structured.NewNodeReader(node))
				khiLog.LogType = logType
				*dest = append(*dest, khiLog)
			}
		}
	}()
}

var _ DefaultResourceNamesGenerator = (*ProjectIDDefaultResourceNamesGenerator)(nil)

// NewLegacyCloudLoggingListLogTask creates a new task that lists log entries from Cloud Logging.
func NewLegacyCloudLoggingListLogTask(taskId taskid.TaskImplementationID[[]*log.Log], readableQueryName string, logType enum.LogType, dependencies []taskid.UntypedTaskReference, resourceNamesGenerator DefaultResourceNamesGenerator, generator QueryGeneratorFunc, sampleQuery string) coretask.Task[[]*log.Log] {
	return inspectiontaskbase.NewProgressReportableInspectionTask(taskId, append(
		append(dependencies, resourceNamesGenerator.GetDependentTasks()...),
		InputStartTimeTaskID.Ref(),
		InputEndTimeTaskID.Ref(),
		InputProjectIdTaskID.Ref(), // TODO: To support mulit project query, this task needs to provide a way to change this dependency
		InputLoggingFilterResourceNameTaskID.Ref(),
		LoggingFetcherTaskID.Ref(),
	), func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) ([]*log.Log, error) {

		metadata := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)
		resourceNames := coretask.GetTaskResult(ctx, InputLoggingFilterResourceNameTaskID.Ref())

		defaultResourceNames, err := resourceNamesGenerator.GenerateResourceNames(ctx)
		if err != nil {
			return nil, err
		}

		resourceNames.UpdateDefaultResourceNamesForQuery(taskId.ReferenceIDString(), defaultResourceNames)
		queryResourceNamePair := resourceNames.GetResourceNamesForQuery(ctx, taskId.ReferenceIDString())
		resourceNamesFromInput := queryResourceNamePair.CurrentResourceNames

		startTime := coretask.GetTaskResult(ctx, InputStartTimeTaskID.Ref())
		endTime := coretask.GetTaskResult(ctx, InputEndTimeTaskID.Ref())
		projectID := coretask.GetTaskResult(ctx, InputProjectIdTaskID.Ref())

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
			finalQuery := fmt.Sprintf("%s\n%s", queryString, gcpqueryutil.TimeRangeQuerySection(startTime, endTime, true))
			if len(finalQuery) > 20000 {
				slog.WarnContext(ctx, fmt.Sprintf("Logging filter is exceeding Cloud Logging limitation 20000 charactors\n%s", finalQuery))
			}
			queryInfo.SetQuery(taskId.String(), readableQueryNameForQueryIndex, finalQuery)
			// TODO: not to store whole logs on memory to avoid OOM
			// Run query only when thetask mode is for running
			if taskMode == inspectioncore_contract.TaskModeRun {

				logFetcher := coretask.GetTaskResult(ctx, LoggingFetcherTaskID.Ref())
				progressReportableLogFetcher := NewTimePartitioningProgressReportableLogFetcher(logFetcher, 500*time.Millisecond, 10, runtime.GOMAXPROCS(0))

				var wg sync.WaitGroup
				var logChan = make(chan *loggingpb.LogEntry)
				var progressChan = make(chan LogFetchProgress)
				monitorProgress(ctx, &wg, progressChan, progress, queryIndex, len(queryStrings))
				convertLogsArray(ctx, &wg, logChan, &allLogs, logType)
				err := progressReportableLogFetcher.FetchLogsWithProgress(logChan, progressChan, ctx, startTime, endTime, queryString, googlecloud.Project(projectID), resourceNamesFromInput)
				wg.Wait()

				if err != nil {
					errorMessageSet, found := typedmap.Get(metadata, inspectionmetadata.ErrorMessageSetMetadataKey)
					if !found {
						return nil, fmt.Errorf("error message set metadata was not found")
					}
					st, ok := status.FromError(err)
					if !ok {
						switch st.Code() {
						case codes.Unauthenticated:
							errorMessageSet.AddErrorMessage(inspectionmetadata.NewUnauthorizedErrorMessage())
							return nil, err
						default:
						}
					}
					errorMessageSet.AddErrorMessage(&inspectionmetadata.ErrorMessage{
						ErrorId: 0,
						Message: err.Error(),
					})
					return nil, err
				}
			}
		}

		for _, l := range allLogs {
			l.SetFieldSetReader(&gcpqueryutil.GCPCommonFieldSetReader{})
			l.SetFieldSetReader(&gcpqueryutil.GCPMainMessageFieldSetReader{})
		}

		if taskMode == inspectioncore_contract.TaskModeRun {
			slices.SortFunc(allLogs, func(a, b *log.Log) int {
				commonFieldSetForA, _ := log.GetFieldSet(a, &log.CommonFieldSet{}) // errors are safely ignored because this field set is required in previous steps
				commonFieldSetForB, _ := log.GetFieldSet(b, &log.CommonFieldSet{})
				return int(commonFieldSetForA.Timestamp.Sub(commonFieldSetForB.Timestamp))
			})
			return allLogs, err
		}

		return []*log.Log{}, err
	}, inspectioncore_contract.NewQueryTaskLabelOpt(logType, sampleQuery))
}
