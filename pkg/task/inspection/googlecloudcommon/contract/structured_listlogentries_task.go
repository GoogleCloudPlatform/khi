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

package googlecloudcommon_contract

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"cloud.google.com/go/logging/apiv2/loggingpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// StructuredListLogEntriesTaskSetting defines the settings for a Cloud Logging task driven by StructuredLogQuery.
type StructuredListLogEntriesTaskSetting interface {
	// TaskID returns the task ID for the structured list log entries task.
	TaskID() taskid.TaskImplementationID[[]*log.Log]

	// Dependencies returns the list of dependencies for the task.
	Dependencies() []taskid.UntypedTaskReference

	// DefaultResourceNames returns default resource names (e.g. ["projects/<project-id>"]).
	DefaultResourceNames(ctx context.Context) ([]string, error)

	// Queries returns the list of structured log queries for estimation and execution.
	Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error)

	// TimePartitionCount returns the number of time partitions to gather logs in parallel.
	TimePartitionCount(ctx context.Context) (int, error)

	// QueryName returns human-readable name of the query.
	QueryName() string
}

// NewStructuredListLogEntriesTask creates a new task that queries logs from Cloud Logging using StructuredLogQuery.
// In DryRun mode, it estimates log volumes and populates QueryMetadata with estimated counts.
func NewStructuredListLogEntriesTask(taskSetting StructuredListLogEntriesTaskSetting) coretask.Task[[]*log.Log] {
	taskID := taskSetting.TaskID()
	dependencies := taskSetting.Dependencies()
	dependencies = append(dependencies,
		InputStartTimeTaskID.Ref(),
		InputEndTimeTaskID.Ref(),
		InputLoggingFilterResourceNameTaskID.Ref(),
		LoggingFetcherTaskID.Ref(),
		APIClientFactoryTaskID.Ref(),
	)
	queryName := taskSetting.QueryName()

	return inspectiontaskbase.NewProgressReportableInspectionTask(
		taskID,
		dependencies,
		func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) ([]*log.Log, error) {
			startTime := coretask.GetTaskResult(ctx, InputStartTimeTaskID.Ref())
			endTime := coretask.GetTaskResult(ctx, InputEndTimeTaskID.Ref())
			resourceNames, err := handleResourceNames(ctx, taskID, &resourceNamesSettingAdapter{taskSetting: taskSetting})
			if err != nil {
				return nil, fmt.Errorf("failed to determine resource names list for structured log query: %w", err)
			}

			queries, err := taskSetting.Queries(ctx)
			if err != nil {
				return nil, fmt.Errorf("Queries returned an error: %w", err)
			}
			if len(queries) == 0 {
				slog.DebugContext(ctx, "Queries returned an empty list. Skipping fetching logs for this task.")
				return []*log.Log{}, nil
			}

			groups, err := groupResourceNamesByContainer(resourceNames)
			if err != nil {
				return nil, err
			}

			// In DryRun: perform volume estimation across all container groups and record query metadata.
			if taskMode != inspectioncore_contract.TaskModeRun {
				clientFactory := coretask.GetTaskResult(ctx, APIClientFactoryTaskID.Ref())
				return nil, estimateAndRecordQueries(ctx, taskID.String(), clientFactory, groups, queries, startTime, endTime, queryName)
			}

			// In Run mode: fetch logs across partitions.
			timePartitionCount, err := taskSetting.TimePartitionCount(ctx)
			if err != nil {
				return nil, fmt.Errorf("TimePartitionCount returned an error: %w", err)
			}
			if timePartitionCount < 1 {
				return nil, fmt.Errorf("TimePartitionCount returned an invalid value %d, it must be bigger than 0", timePartitionCount)
			}

			logFetcher := coretask.GetTaskResult(ctx, LoggingFetcherTaskID.Ref())
			return fetchLogsForStructuredQueries(ctx, taskID.String(), logFetcher, groups, queries, startTime, endTime, queryName, timePartitionCount, progress)
		},
		coretask.WithLabelValue(RequestOptionalInputResourceNameTaskLabel, taskID.ReferenceIDString()),
	)
}

// resourceNamesSettingAdapter adapts StructuredListLogEntriesTaskSetting to ListLogEntriesTaskSetting for resource name handling.
type resourceNamesSettingAdapter struct {
	taskSetting StructuredListLogEntriesTaskSetting
}

func (a *resourceNamesSettingAdapter) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return a.taskSetting.TaskID()
}

func (a *resourceNamesSettingAdapter) Dependencies() []taskid.UntypedTaskReference {
	return a.taskSetting.Dependencies()
}

func (a *resourceNamesSettingAdapter) DefaultResourceNames(ctx context.Context) ([]string, error) {
	return a.taskSetting.DefaultResourceNames(ctx)
}

func (a *resourceNamesSettingAdapter) LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	return nil, nil
}

func (a *resourceNamesSettingAdapter) TimePartitionCount(ctx context.Context) (int, error) {
	return a.taskSetting.TimePartitionCount(ctx)
}

func (a *resourceNamesSettingAdapter) Description() *ListLogEntriesTaskDescription {
	return &ListLogEntriesTaskDescription{
		QueryName: a.taskSetting.QueryName(),
	}
}

var _ ListLogEntriesTaskSetting = (*resourceNamesSettingAdapter)(nil)

// LogEstimatorCacheKey is the key to retrieve or store CachedStructuredLogEstimator in the InspectionSharedMap.
var LogEstimatorCacheKey = typedmap.NewTypedKey[*logestimator.CachedStructuredLogEstimator]("googlecloud.logestimator.cache")

// getOrInitLogEstimatorCache gets the cached estimator from InspectionSharedMap or initializes a new one.
func getOrInitLogEstimatorCache(ctx context.Context) *logestimator.CachedStructuredLogEstimator {
	sharedMap, err := khictx.GetValue(ctx, inspectioncore_contract.InspectionSharedMap)
	if err != nil || sharedMap == nil {
		return logestimator.NewCachedStructuredLogEstimator()
	}

	cachedEstimator, found := typedmap.Get(sharedMap, LogEstimatorCacheKey)
	if found && cachedEstimator != nil {
		return cachedEstimator
	}

	newEstimator := logestimator.NewCachedStructuredLogEstimator()
	typedmap.Set(sharedMap, LogEstimatorCacheKey, newEstimator)

	inspectionContext, err := khictx.GetValue(ctx, inspectioncore_contract.InspectionContext)
	if err == nil && inspectionContext != nil {
		context.AfterFunc(inspectionContext, func() {
			newEstimator.Close()
		})
	}
	return newEstimator
}

// estimateAndRecordQueries performs volume estimation across container groups and writes query metadata during dryrun.
func estimateAndRecordQueries(
	ctx context.Context,
	taskID string,
	clientFactory *googlecloud.ClientFactory,
	groups []*resourceContainerLogQueryGroup,
	queries []*logestimator.StructuredLogQuery,
	startTime, endTime time.Time,
	queryName string,
) error {
	estimatorCache := getOrInitLogEstimatorCache(ctx)

	for queryIndex, q := range queries {
		if q.Incomplete {
			filterString := q.GenerateCloudLoggingQuery()
			if err := setStructuredQueryInfo(ctx, taskID, filterString, queryIndex, len(queries), startTime, endTime, queryName, nil, true); err != nil {
				return err
			}
			continue
		}

		var totalEstimatedCount int64
		var estimated *int64
		for _, group := range groups {
			taskSlotKey := fmt.Sprintf("%s/%s/%d", taskID, group.container.Identifier(), queryIndex)
			res, estErr := estimatorCache.EstimateWithTaskSlot(
				ctx,
				taskSlotKey,
				group.container,
				q,
				startTime,
				endTime,
				func(queryCtx context.Context, container googlecloud.ResourceContainer) (*logestimator.StructuredLogEstimator, error) {
					loggingClient, logErr := clientFactory.LoggingClient(queryCtx, container)
					metricClient, monErr := clientFactory.MonitoringMetricClient(queryCtx, container)
					if logErr != nil || monErr != nil {
						return nil, fmt.Errorf("failed to initialize clients for container %s: loggingErr=%v, metricErr=%v", container.Identifier(), logErr, monErr)
					}
					return logestimator.NewStructuredLogEstimatorFromClients(loggingClient, metricClient), nil
				},
			)
			if estErr != nil {
				slog.WarnContext(ctx, fmt.Sprintf("log estimation failed for query %s in %s: %v", queryName, group.container.Identifier(), estErr))
			} else {
				totalEstimatedCount += res.EstimatedCount
				estimated = &totalEstimatedCount
			}
		}
		filterString := q.GenerateCloudLoggingQuery()
		if err := setStructuredQueryInfo(ctx, taskID, filterString, queryIndex, len(queries), startTime, endTime, queryName, estimated, false); err != nil {
			return err
		}
	}
	return nil
}

// fetchLogsForStructuredQueries retrieves logs in parallel across time partitions and container groups.
func fetchLogsForStructuredQueries(
	ctx context.Context,
	taskID string,
	logFetcher LogFetcher,
	groups []*resourceContainerLogQueryGroup,
	queries []*logestimator.StructuredLogQuery,
	startTime, endTime time.Time,
	queryName string,
	timePartitionCount int,
	progress *inspectionmetadata.TaskProgressMetadata,
) ([]*log.Log, error) {
	groups = divideGroupByMaximumResourceName(groups, maxResourceNameCountPerRequest)
	progressReportableLogFetcher := NewTimePartitioningProgressReportableLogFetcher(logFetcher, 500*time.Millisecond, timePartitionCount, runtime.GOMAXPROCS(0))

	allLogs := make([]*log.Log, 0)
	for queryIndex, q := range queries {
		filterString := q.GenerateCloudLoggingQuery()
		if err := setStructuredQueryInfo(ctx, taskID, filterString, queryIndex, len(queries), startTime, endTime, queryName, nil, false); err != nil {
			return nil, err
		}

		for groupIndex, group := range groups {
			var wg sync.WaitGroup
			var logChan = make(chan *loggingpb.LogEntry)
			var progressChan = make(chan LogFetchProgress)
			listCallIndex := queryIndex*len(groups) + groupIndex
			allListCalls := len(queries) * len(groups)
			monitorProgress(ctx, &wg, progressChan, progress, listCallIndex, allListCalls)
			convertLogsArray(ctx, &wg, logChan, &allLogs)
			err := progressReportableLogFetcher.FetchLogsWithProgress(logChan, progressChan, ctx, startTime, endTime, filterString, group.container, group.resourceNames)
			wg.Wait()

			if err != nil {
				return nil, setErrorMetadataForFetchLogError(ctx, err)
			}
		}
	}

	for _, l := range allLogs {
		l.SetFieldSetReader(&gcpqueryutil.GCPCommonFieldSetReader{})
	}

	tracingActive, _ := khictx.GetValue(ctx, inspectioncore_contract.TracingActive)
	if tracingActive {
		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("log_count", fmt.Sprintf("%d", len(allLogs))),
		)
	}

	return allLogs, nil
}

// setStructuredQueryInfo records the generated Cloud Logging query details and estimated count into the inspection run metadata.
func setStructuredQueryInfo(ctx context.Context, taskID, baseLogFilter string, logFilterIndex, totalLogFilterCount int, startTime, endTime time.Time, queryName string, estimatedCount *int64, incomplete bool) error {
	metadata := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)
	queryInfo, found := typedmap.Get(metadata, inspectionmetadata.QueryMetadataKey)
	if !found {
		return fmt.Errorf("query metadata was not found")
	}

	logFilterName := queryName
	if totalLogFilterCount > 1 {
		logFilterName = fmt.Sprintf("%s-%d", queryName, logFilterIndex)
	}
	finalFilter := fmt.Sprintf("%s\n%s", baseLogFilter, gcpqueryutil.TimeRangeQuerySection(startTime, endTime, true))
	if len(finalFilter) > 20000 {
		slog.WarnContext(ctx, fmt.Sprintf("Logging filter is exceeding Cloud Logging limitation 20000 characters\n%s", finalFilter))
	}
	switch {
	case incomplete:
		queryInfo.SetIncompleteQuery(taskID, logFilterName, finalFilter)
	case estimatedCount != nil:
		queryInfo.SetQueryWithEstimate(taskID, logFilterName, finalFilter, *estimatedCount)
	default:
		queryInfo.SetQuery(taskID, logFilterName, finalFilter)
	}
	return nil
}
