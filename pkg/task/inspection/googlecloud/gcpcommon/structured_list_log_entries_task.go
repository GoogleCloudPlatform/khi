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

package gcpcommon

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/common/kwaymerge"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progress"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// StructuredLogQuerySource provides the resource names, queries, and time partitions of a Cloud Logging task driven by StructuredLogQuery.
type StructuredLogQuerySource interface {
	// DefaultResourceNames returns default resource names (e.g. ["projects/<project-id>"]).
	DefaultResourceNames(ctx context.Context) ([]string, error)

	// Queries returns the list of structured log queries for estimation and execution.
	Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error)

	// TimePartitionCount returns the number of time partitions to gather logs in parallel.
	TimePartitionCount(ctx context.Context) (int, error)
}

// DefineStructuredListLogEntriesTask defines a task that queries logs from Cloud Logging using the StructuredLogQuery list of the source returned by bind.
// It declares the time range, resource names, log fetcher, and API client inputs before it calls bind, so bind only declares the inputs of the source.
// In DryRun mode, it estimates log volumes and populates QueryMetadata with estimated counts.
func DefineStructuredListLogEntriesTask(taskID taskid.TaskImplementationID[[]*log.Log], queryName string, bind func(b *coretask.Binder) StructuredLogQuerySource) coretask.DefinedTask[[]*log.Log] {
	return inspectiontaskbase.DefineInspectionTask(
		taskID,
		func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[[]*log.Log] {
			startTime := coretask.Use(b, InputStartTimeTaskID.Ref())
			endTime := coretask.Use(b, InputEndTimeTaskID.Ref())
			resourceNamesInput := coretask.Use(b, InputLoggingFilterResourceNameTaskID.Ref())
			logFetcher := coretask.Use(b, LoggingFetcherTaskID.Ref())
			clientFactory := coretask.Use(b, APIClientFactoryTaskID.Ref())
			callOptionInjector := coretask.Use(b, APIClientCallOptionsInjectorTaskID.Ref())
			source := bind(b)
			return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]*log.Log, error) {
				groups, queries, err := resolveStructuredQueries(ctx, taskID, resourceNamesInput.Get(ctx), source)
				if err != nil {
					return nil, err
				}
				if len(queries) == 0 {
					return []*log.Log{}, nil
				}

				if taskMode != inspectioncore.TaskModeRun {
					return nil, estimateAndRecordQueries(ctx, taskID.String(), clientFactory.Get(ctx), callOptionInjector.Get(ctx), groups, queries, startTime.Get(ctx), endTime.Get(ctx), queryName)
				}

				timePartitionCount, err := resolveTimePartitionCount(ctx, source)
				if err != nil {
					return nil, err
				}
				return fetchLogsForStructuredQueries(ctx, taskID.String(), logFetcher.Get(ctx), groups, queries, startTime.Get(ctx), endTime.Get(ctx), queryName, timePartitionCount)
			}
		},
		coretask.WithLabelValue(RequestOptionalInputResourceNameTaskLabel, taskID.ReferenceIDString()),
		progress.WithTitle(fmt.Sprintf("Fetch %s", queryName)),
	)
}

// resolveStructuredQueries determines the resource names and queries of a structured list log entries task and groups the resource names by container.
// It returns no queries when the source has none, so that the caller can skip fetching logs.
func resolveStructuredQueries(ctx context.Context, taskID taskid.TaskImplementationID[[]*log.Log], resourceNamesInput *ResourceNamesInput, source StructuredLogQuerySource) ([]*resourceContainerLogQueryGroup, []*logestimator.StructuredLogQuery, error) {
	resourceNames, err := handleResourceNames(ctx, taskID, resourceNamesInput, source.DefaultResourceNames)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to determine resource names list for structured log query: %w", err)
	}

	queries, err := source.Queries(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("Queries returned an error: %w", err)
	}
	if len(queries) == 0 {
		slog.DebugContext(ctx, "Queries returned an empty list. Skipping fetching logs for this task.")
		return nil, nil, nil
	}

	groups, err := groupResourceNamesByContainer(resourceNames)
	if err != nil {
		return nil, nil, err
	}
	return groups, queries, nil
}

// resolveTimePartitionCount returns the time partition count of the source after checking that it is positive.
func resolveTimePartitionCount(ctx context.Context, source StructuredLogQuerySource) (int, error) {
	timePartitionCount, err := source.TimePartitionCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("TimePartitionCount returned an error: %w", err)
	}
	if timePartitionCount < 1 {
		return 0, fmt.Errorf("TimePartitionCount returned an invalid value %d, it must be bigger than 0", timePartitionCount)
	}
	return timePartitionCount, nil
}

// LogEstimatorCacheKey is the key to retrieve or store CachedStructuredLogEstimator in the InspectionSharedMap.
var LogEstimatorCacheKey = typedmap.NewTypedKey[*logestimator.CachedStructuredLogEstimator]("googlecloud.logestimator.cache")

// getOrInitLogEstimatorCache gets the cached estimator from InspectionSharedMap or initializes a new one.
func getOrInitLogEstimatorCache(ctx context.Context) *logestimator.CachedStructuredLogEstimator {
	sharedMap, err := khictx.GetValue(ctx, inspectioncore.InspectionSharedMap)
	if err != nil || sharedMap == nil {
		return logestimator.NewCachedStructuredLogEstimator()
	}

	cachedEstimator, found := typedmap.Get(sharedMap, LogEstimatorCacheKey)
	if found && cachedEstimator != nil {
		return cachedEstimator
	}

	newEstimator := logestimator.NewCachedStructuredLogEstimator()
	typedmap.Set(sharedMap, LogEstimatorCacheKey, newEstimator)

	inspectionContext, err := khictx.GetValue(ctx, inspectioncore.InspectionContext)
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
	callOptionInjector *googlecloud.CallOptionInjector,
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

		if q.Preset != logestimator.EstimatedCountPresetNone {
			filterString := q.GenerateCloudLoggingQuery()
			if err := setStructuredQueryInfoWithPendingAndPreset(ctx, taskID, filterString, queryIndex, len(queries), startTime, endTime, queryName, nil, false, false, q.Preset); err != nil {
				return err
			}
			continue
		}

		var totalEstimatedCount int64
		var estimated *int64
		preset := q.Preset
		anyPending := false
		for _, group := range groups {
			taskSlotKey := fmt.Sprintf("%s/%s/%d", taskID, group.container.Identifier(), queryIndex)
			res, isPending, estErr := estimatorCache.EstimateWithTaskSlotNonBlocking(
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
					return logestimator.NewStructuredLogEstimatorFromClients(loggingClient, metricClient, callOptionInjector), nil
				},
			)
			switch {
			case isPending:
				anyPending = true
			case estErr != nil:
				slog.WarnContext(ctx, fmt.Sprintf("log estimation failed for query %s in %s: %v", queryName, group.container.Identifier(), estErr))
			case res != nil:
				if res.Preset != logestimator.EstimatedCountPresetNone {
					preset = res.Preset
				} else {
					totalEstimatedCount += res.EstimatedCount
					estimated = &totalEstimatedCount
				}
			}
		}
		filterString := q.GenerateCloudLoggingQuery()
		if err := setStructuredQueryInfoWithPendingAndPreset(ctx, taskID, filterString, queryIndex, len(queries), startTime, endTime, queryName, estimated, false, anyPending, preset); err != nil {
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
) ([]*log.Log, error) {
	groups = divideGroupByMaximumResourceName(groups, maxResourceNameCountPerRequest)
	progressReportableLogFetcher := NewTimePartitioningProgressReportableLogFetcher(logFetcher, 500*time.Millisecond, timePartitionCount, runtime.GOMAXPROCS(0))

	tracker := progress.NewRatioTracker(ctx, progress.WithUnit("logs"))
	defer tracker.Done()
	totalLogsFetched := 0
	allLogSlices := make([][]*log.Log, 0, len(queries)*len(groups))
	for queryIndex, q := range queries {
		filterString := q.GenerateCloudLoggingQuery()
		if err := setStructuredQueryInfo(ctx, taskID, filterString, queryIndex, len(queries), startTime, endTime, queryName, nil, false); err != nil {
			return nil, err
		}

		for groupIndex, group := range groups {
			var wg sync.WaitGroup
			var progressChan = make(chan LogFetchProgress)
			listCallIndex := queryIndex*len(groups) + groupIndex
			totalListCalls := len(queries) * len(groups)
			monitorProgress(ctx, &wg, progressChan, tracker, totalLogsFetched, listCallIndex, totalListCalls)
			logs, err := progressReportableLogFetcher.FetchLogsWithProgress(progressChan, ctx, startTime, endTime, filterString, group.container, group.resourceNames)
			wg.Wait()

			if err != nil {
				return nil, setErrorMetadataForFetchLogError(ctx, err)
			}
			totalLogsFetched += len(logs)
			allLogSlices = append(allLogSlices, logs)
		}
	}

	allLogs := kwaymerge.Merge(allLogSlices, func(a, b *log.Log) int {
		return a.Timestamp.Compare(b.Timestamp)
	})

	tracingActive, _ := khictx.GetValue(ctx, inspectioncore.TracingActive)
	if tracingActive {
		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("log_count", fmt.Sprintf("%d", len(allLogs))),
		)
	}

	return allLogs, nil
}

// setStructuredQueryInfo records the generated Cloud Logging query details and estimated count into the inspection run metadata.
func setStructuredQueryInfo(ctx context.Context, taskID, baseLogFilter string, logFilterIndex, totalLogFilterCount int, startTime, endTime time.Time, queryName string, estimatedCount *int64, incomplete bool) error {
	return setStructuredQueryInfoWithPendingAndPreset(ctx, taskID, baseLogFilter, logFilterIndex, totalLogFilterCount, startTime, endTime, queryName, estimatedCount, incomplete, false, logestimator.EstimatedCountPresetNone)
}

// setStructuredQueryInfoWithPendingAndPreset records the generated Cloud Logging query details, estimated count, pending status, and preset into the inspection run metadata.
func setStructuredQueryInfoWithPendingAndPreset(ctx context.Context, taskID, baseLogFilter string, logFilterIndex, totalLogFilterCount int, startTime, endTime time.Time, queryName string, estimatedCount *int64, incomplete bool, pending bool, preset logestimator.EstimatedCountPreset) error {
	metadata := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
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
	case pending:
		queryInfo.SetPendingQuery(taskID, logFilterName, finalFilter)
	case preset != logestimator.EstimatedCountPresetNone:
		queryInfo.SetQueryWithPreset(taskID, logFilterName, finalFilter, preset)
	case estimatedCount != nil:
		queryInfo.SetQueryWithEstimate(taskID, logFilterName, finalFilter, *estimatedCount)
	default:
		queryInfo.SetQuery(taskID, logFilterName, finalFilter)
	}
	return nil
}

// maxResourceNameCountPerRequest is the maximum allowed count of resource names per single entries.list. The default quota is 100.
var maxResourceNameCountPerRequest = 100

func monitorProgress(ctx context.Context, wg *sync.WaitGroup, source <-chan LogFetchProgress, tracker *progress.RatioTracker, baseLogCount int, listCallIndex int, totalListCalls int) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case p, ok := <-source:
				if !ok {
					return
				}
				totalLogCount := baseLogCount + p.LogCount
				completeRatio := (float32(listCallIndex) + p.Progress) / float32(totalListCalls)
				tracker.Update(completeRatio, totalLogCount, progress.WithStep(listCallIndex+1, totalListCalls))
			}
		}
	}()
}

// handleResourceNames retrieves and validates resource names for a given task, updating default values if necessary.
func handleResourceNames(ctx context.Context, taskID taskid.TaskImplementationID[[]*log.Log], resourceNamesInput *ResourceNamesInput, getDefaultResourceNames func(ctx context.Context) ([]string, error)) ([]string, error) {
	queryResourceNamePair := resourceNamesInput.GetResourceNamesForQuery(ctx, taskID.ReferenceIDString())

	defaultResourceNames, err := getDefaultResourceNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("ResourceNames returned an error: %w", err)
	}

	resourceNamesInput.UpdateDefaultResourceNamesForQuery(taskID.ReferenceIDString(), defaultResourceNames)

	return queryResourceNamePair.CurrentResourceNames, nil
}

// setErrorMetadataForFetchLogError extracts error information from a log fetching operation and adds it to the inspection run's error message set metadata.
func setErrorMetadataForFetchLogError(ctx context.Context, err error) error {
	metadata := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
	errorMessageSet, found := typedmap.Get(metadata, inspectionmetadata.ErrorMessageSetMetadataKey)
	if !found {
		return fmt.Errorf("error message set metadata was not found. originalError=%w", err)
	}
	errorMessageSet.AddErrorMessage(&inspectionmetadata.ErrorMessage{
		ErrorId: 0,
		Message: err.Error(),
	})
	return err
}

// resourceContainerLogQueryGroup groups resource names under a common Google Cloud resource container.
type resourceContainerLogQueryGroup struct {
	container     googlecloud.ResourceContainer
	resourceNames []string
}

// groupResourceNamesByContainer groups a list of resource names by their Google Cloud resource container.
// It returns a slice of resourceContainerLogQueryGroup, where each group contains resource names
// belonging to the same container (e.g., project).
func groupResourceNamesByContainer(resourceNames []string) ([]*resourceContainerLogQueryGroup, error) {
	groups := make(map[string]*resourceContainerLogQueryGroup)

	for _, resourceName := range resourceNames {
		var container googlecloud.ResourceContainer
		switch {
		case strings.HasPrefix(resourceName, "projects/"):
			projectID := resourceName[len("projects/"):]
			slashIndex := strings.Index(projectID, "/")
			if slashIndex != -1 {
				projectID = projectID[:slashIndex]
			}
			container = googlecloud.Project(projectID)
		default:
			// TODO: Add support for other resource containers like organizations, folders, and billingAccounts.
			// Unsupported resource container types.
		}
		if container == nil {
			return nil, fmt.Errorf("unsupported resource name %q : %w", resourceName, khierrors.ErrInvalidInput)
		}
		containerIdentifier := container.Identifier()
		if _, ok := groups[containerIdentifier]; !ok {
			groups[containerIdentifier] = &resourceContainerLogQueryGroup{
				container: container,
			}
		}

		group := groups[containerIdentifier]
		group.resourceNames = append(group.resourceNames, resourceName)
	}

	result := slices.Collect(maps.Values(groups))
	slices.SortFunc(result, func(a, b *resourceContainerLogQueryGroup) int {
		return strings.Compare(a.container.Identifier(), b.container.Identifier())
	})
	return result, nil
}

// divideGroupByMaximumResourceName divides resourceContainerLogQueryGroup instances into smaller groups if their resourceNames slice exceeds maxResourceNamePerGroup.
func divideGroupByMaximumResourceName(groups []*resourceContainerLogQueryGroup, maxResourceNamePerGroup int) []*resourceContainerLogQueryGroup {
	var dividedGroups []*resourceContainerLogQueryGroup
	for _, group := range groups {
		for len(group.resourceNames) > maxResourceNamePerGroup {
			dividedGroups = append(dividedGroups, &resourceContainerLogQueryGroup{
				container:     group.container,
				resourceNames: group.resourceNames[:maxResourceNamePerGroup],
			})
			group.resourceNames = group.resourceNames[maxResourceNamePerGroup:]
		}
		dividedGroups = append(dividedGroups, group)
	}
	return dividedGroups
}
