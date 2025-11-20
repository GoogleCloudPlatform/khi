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

package googlecloudcommon_contract

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

	"cloud.google.com/go/logging/apiv2/loggingpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// maxResourceNameCountPerRequest is the maximum allowed count of resource names per single entries.list. The default quota is 100.
var maxResourceNameCountPerRequest = 100

// ListLogEntriesTaskDescription holds descriptive information for a task to list log entries from CloudLogging.
type ListLogEntriesTaskDescription struct {
	DefaultLogType enum.LogType
	QueryName      string
	ExampleQuery   string
}

// ListLogEntriesTaskSetting defines the settings for a Cloud Logging list log entries task.
type ListLogEntriesTaskSetting interface {
	// TaskID returns the task ID for the Cloud Logging list log entries task.
	TaskID() taskid.TaskImplementationID[[]*log.Log]

	// Dependencies returns the list of dependencies for the Cloud Logging list log entries task.
	// Return the dependency task reference IDs when the result is used in DefaultResourceNames(), LogFilters() or TimePartitionCount().
	Dependencies() []taskid.UntypedTaskReference

	// DefaultResourceNames returns the list of resource names for the Cloud Logging list log entries task.
	// This is just a default value for the resource name. Users can override this value with the form field.
	// Return the list of resource names. ref: https://cloud.google.com/logging/docs/reference/v2/rest/v2/entries/list
	DefaultResourceNames(ctx context.Context) ([]string, error)

	// LogFilters returns the list of log filters for the Cloud Logging list log entries task.
	// When generated logging filter can exceed the 20,000 character maximum limit in Cloud Logging, return multiple subset query.
	// Result includes the logs for all log filters.
	LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error)

	// TimePartitionCount returns the number of time partitions for the Cloud Logging list log entries task.
	// ListLogEntriesTask split the duration into the number of partition count to gather logs in parallel.
	// Return 1 - 16 values depending on the expected log volume by the log filter.
	TimePartitionCount(ctx context.Context) (int, error)

	// Description returns the description for the Cloud Logging filter task.
	Description() *ListLogEntriesTaskDescription
}

// NewListLogEntriesTask creates a new task that lists log entries from Cloud Logging based on the provided settings.
func NewListLogEntriesTask(taskSetting ListLogEntriesTaskSetting) coretask.Task[[]*log.Log] {
	taskID := taskSetting.TaskID()
	dependencies := taskSetting.Dependencies()
	dependencies = append(dependencies, InputStartTimeTaskID.Ref(), InputEndTimeTaskID.Ref(), InputLoggingFilterResourceNameTaskID.Ref(), LoggingFetcherTaskID.Ref())
	description := taskSetting.Description()

	return inspectiontaskbase.NewProgressReportableInspectionTask(
		taskID,
		dependencies,
		func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) ([]*log.Log, error) {
			startTime := coretask.GetTaskResult(ctx, InputStartTimeTaskID.Ref())
			endTime := coretask.GetTaskResult(ctx, InputEndTimeTaskID.Ref())
			resourceNames, err := handleResourceNames(ctx, taskID, taskSetting)
			if err != nil {
				return nil, fmt.Errorf("failed to determine the resource names list for log filter: %w", err)
			}

			filters, err := taskSetting.LogFilters(ctx, taskMode)
			if err != nil {
				return nil, fmt.Errorf("LogFilters returned an error: %w", err)
			}
			if len(filters) == 0 {
				slog.DebugContext(ctx, "LogFilters returned an emptry list. Skipping fetching logs for this task")
				return []*log.Log{}, nil
			}
			timePartitionCount, err := taskSetting.TimePartitionCount(ctx)
			if err != nil {
				return nil, fmt.Errorf("TimePartitionCount returned an error: %w", err)
			}
			if timePartitionCount < 1 {
				return nil, fmt.Errorf("TimePartitionCount returned an invalid value %d, it must be bigger than 0", timePartitionCount)
			}

			allLogs := make([]*log.Log, 0)
			for filterIndex, filter := range filters {
				err := setQueryInfo(ctx, taskID.String(), filter, filterIndex, len(filters), startTime, endTime, description)
				if err != nil {
					return nil, err
				}

				// Don't run logging filter except the run mode
				if taskMode != inspectioncore_contract.TaskModeRun {
					continue
				}

				groups, err := groupResourceNamesByContainer(resourceNames)
				if err != nil {
					return nil, err
				}
				groups = divideGroupByMaximumResourceName(groups, maxResourceNameCountPerRequest)

				logFetcher := coretask.GetTaskResult(ctx, LoggingFetcherTaskID.Ref())
				progressReportableLogFetcher := NewTimePartitioningProgressReportableLogFetcher(logFetcher, 500*time.Millisecond, timePartitionCount, runtime.GOMAXPROCS(0))

				for groupIndex, group := range groups {
					var wg sync.WaitGroup
					var logChan = make(chan *loggingpb.LogEntry)
					var progressChan = make(chan LogFetchProgress)
					listCallIndex := filterIndex*len(groups) + groupIndex
					allListCalls := len(filters) * len(groups)
					monitorProgress(ctx, &wg, progressChan, progress, listCallIndex, allListCalls)
					convertLogsArray(ctx, &wg, logChan, &allLogs, description.DefaultLogType)
					err = progressReportableLogFetcher.FetchLogsWithProgress(logChan, progressChan, ctx, startTime, endTime, filter, group.container, group.resourceNames)
					wg.Wait()

					if err != nil {
						err := setErrorMetadataForFetchLogError(ctx, err)
						return nil, err
					}
				}
			}

			// GCPCommonFieldSet is always required for any logs retrieved from Cloud Logging.
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
		}, inspectioncore_contract.NewQueryTaskLabelOpt(description.DefaultLogType, description.ExampleQuery),
	)
}

// handleResourceNames retrieves and validates resource names for a given task, updating default values if necessary.
func handleResourceNames(ctx context.Context, taskID taskid.TaskImplementationID[[]*log.Log], taskSetting ListLogEntriesTaskSetting) ([]string, error) {
	resourceNamesInput := coretask.GetTaskResult(ctx, InputLoggingFilterResourceNameTaskID.Ref())
	queryResourceNamePair := resourceNamesInput.GetResourceNamesForQuery(ctx, taskID.ReferenceIDString())

	defaultResourceNames, err := taskSetting.DefaultResourceNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("ResourceNames returned an error: %w", err)
	}

	resourceNamesInput.UpdateDefaultResourceNamesForQuery(taskID.ReferenceIDString(), defaultResourceNames)

	return queryResourceNamePair.CurrentResourceNames, nil
}

// setQueryInfo records the generated Cloud Logging query details into the inspection run metadata.
func setQueryInfo(ctx context.Context, taskID, baseLogFilter string, logFilterIndex, totalLogFilterCount int, startTime, endTime time.Time, description *ListLogEntriesTaskDescription) error {
	metadata := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)
	queryInfo, found := typedmap.Get(metadata, inspectionmetadata.QueryMetadataKey)
	if !found {
		return fmt.Errorf("query metadata was not found")
	}

	// Record query information in metadata
	logFilterName := description.QueryName
	if totalLogFilterCount > 1 {
		logFilterName = fmt.Sprintf("%s-%d", description.QueryName, logFilterIndex)
	}
	finalFilter := fmt.Sprintf("%s\n%s", baseLogFilter, gcpqueryutil.TimeRangeQuerySection(startTime, endTime, true))
	if len(finalFilter) > 20000 {
		slog.WarnContext(ctx, fmt.Sprintf("Logging filter is exceeding Cloud Logging limitation 20000 charactors\n%s", finalFilter))
	}
	queryInfo.SetQuery(taskID, logFilterName, finalFilter)
	return nil
}

// setErrorMetadataForFetchLogError extracts error information from a log fetching operation and adds it to the inspection run's error message set metadata.
func setErrorMetadataForFetchLogError(ctx context.Context, err error) error {
	metadata := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)
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
