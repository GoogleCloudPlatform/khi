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
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/logging/apiv2/loggingpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typeddict"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progress"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockStructuredLogQuerySource is a StructuredLogQuerySource that returns fixed resource names, queries and time partition count.
type mockStructuredLogQuerySource struct {
	resourceNames      []string
	queries            []*logestimator.StructuredLogQuery
	timePartitionCount int
}

func (s *mockStructuredLogQuerySource) DefaultResourceNames(ctx context.Context) ([]string, error) {
	return s.resourceNames, nil
}

func (s *mockStructuredLogQuerySource) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	return s.queries, nil
}

func (s *mockStructuredLogQuerySource) TimePartitionCount(ctx context.Context) (int, error) {
	return s.timePartitionCount, nil
}

var _ StructuredLogQuerySource = (*mockStructuredLogQuerySource)(nil)

// defineMockStructuredListLogEntriesTask defines a structured list log entries task with the task ID "structured-test" that queries with source.
func defineMockStructuredListLogEntriesTask(queryName string, source *mockStructuredLogQuerySource) coretask.Task[[]*log.Log] {
	return DefineStructuredListLogEntriesTask(taskid.NewDefaultImplementationID[[]*log.Log]("structured-test"), queryName, func(b *coretask.Binder) StructuredLogQuerySource {
		return source
	})
}

func TestStructuredListLogEntriesTask_DryRun_FallbackWhenNoClient(t *testing.T) {
	t.Parallel()
	startTime := time.Date(2025, time.January, 1, 1, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, time.January, 1, 1, 1, 0, 0, time.UTC)

	source := &mockStructuredLogQuerySource{
		resourceNames: []string{"projects/test-project"},
		queries: []*logestimator.StructuredLogQuery{
			{
				ResourceTypes: []string{"k8s_container"},
				Filters: []logestimator.LoggingMonitoringMatcher{
					logestimator.ResourceLabel("project_id", logestimator.Exact("test-project")),
					logestimator.LogID(logestimator.Exact("events")),
				},
			},
		},
		timePartitionCount: 1,
	}

	task := defineMockStructuredListLogEntriesTask("container-logs", source)
	resourceNamesInput := NewResourceNamesInput()
	clientFactory, err := googlecloud.NewClientFactory()
	if err != nil {
		t.Fatalf("failed to create ClientFactory: %v", err)
	}

	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	gotLogs, _, err := inspectiontest.Run(t, ctx, task, inspectioncore.TaskModeDryRun, map[string]any{},
		tasktest.Given(InputStartTimeTaskID.Ref(), startTime),
		tasktest.Given(InputEndTimeTaskID.Ref(), endTime),
		tasktest.Given(InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
		// Dry run records the query without fetching logs.
		tasktest.Given[LogFetcher](LoggingFetcherTaskID.Ref(), nil),
		tasktest.Given(APIClientFactoryTaskID.Ref(), clientFactory),
		tasktest.Given(APIClientCallOptionsInjectorTaskID.Ref(), googlecloud.NewCallOptionInjector()),
	)
	if err != nil {
		t.Fatalf("DryRun returned unexpected error: %v", err)
	}
	if len(gotLogs) != 0 {
		t.Errorf("DryRun should return empty logs, got %d", len(gotLogs))
	}

	// Verify QueryMetadata
	metadata := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
	queryMetadata, found := typedmap.Get(metadata, inspectionmetadata.QueryMetadataKey)
	if !found {
		t.Fatalf("QueryMetadata not found in run metadata")
	}

	serialized := queryMetadata.ToSerializable().([]*inspectionmetadata.QueryItem)
	if len(serialized) != 1 {
		t.Fatalf("expected 1 QueryItem, got %d", len(serialized))
	}

	wantQuery := `resource.type="k8s_container"
resource.labels.project_id="test-project"
LOG_ID("events")
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`

	if diff := cmp.Diff(wantQuery, serialized[0].Query); diff != "" {
		t.Errorf("Query mismatch (-want +got):\n%s", diff)
	}
	if serialized[0].Name != "container-logs" {
		t.Errorf("Query Name mismatch: got %q, want %q", serialized[0].Name, "container-logs")
	}
}

func TestStructuredListLogEntriesTask_Run_FetchLogs(t *testing.T) {
	t.Parallel()
	startTime := time.Date(2025, time.January, 1, 1, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, time.January, 1, 1, 1, 0, 0, time.UTC)
	testErr := fmt.Errorf("fetch error")

	testCases := []struct {
		desc           string
		source         *mockStructuredLogQuerySource
		fetcherFactory func(t *testing.T) *mockLogFetcher
		wantLogsString []string
		wantError      error
	}{
		{
			desc: "successful log fetch with single query",
			source: &mockStructuredLogQuerySource{
				resourceNames: []string{"projects/test-project"},
				queries: []*logestimator.StructuredLogQuery{
					{
						ResourceTypes: []string{"k8s_container"},
						Filters: []logestimator.LoggingMonitoringMatcher{
							logestimator.ResourceLabel("project_id", logestimator.Exact("test-project")),
						},
					},
				},
				timePartitionCount: 1,
			},
			fetcherFactory: func(t *testing.T) *mockLogFetcher {
				return getMockFetcherFromFakeLogUpstreamPairs(t, []fakeLogUpstreamPair{
					newFakeLogUpstreamPair(`resource.type="k8s_container"
resource.labels.project_id="test-project"
timestamp >= "2025-01-01T01:00:00+0000"
timestamp < "2025-01-01T01:01:00+0000"`, func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
						logSource <- &loggingpb.LogEntry{InsertId: "log-1", LogName: "container-log"}
						logSource <- &loggingpb.LogEntry{InsertId: "log-2", LogName: "container-log"}
					}),
				})
			},
			wantLogsString: []string{
				"insertId: log-1\nlogName: container-log\n",
				"insertId: log-2\nlogName: container-log\n",
			},
		},
		{
			desc: "fetch error propagation",
			source: &mockStructuredLogQuerySource{
				resourceNames: []string{"projects/test-project"},
				queries: []*logestimator.StructuredLogQuery{
					{
						ResourceTypes: []string{"k8s_container"},
						Filters: []logestimator.LoggingMonitoringMatcher{
							logestimator.ResourceLabel("project_id", logestimator.Exact("test-project")),
						},
					},
				},
				timePartitionCount: 1,
			},
			fetcherFactory: func(t *testing.T) *mockLogFetcher {
				return getMockFetcherFromFakeLogUpstreamPairs(t, []fakeLogUpstreamPair{
					newFakeLogUpstreamPair(`resource.type="k8s_container"
resource.labels.project_id="test-project"
timestamp >= "2025-01-01T01:00:00+0000"
timestamp < "2025-01-01T01:01:00+0000"`, func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
						errSource <- testErr
					}),
				})
			},
			wantError: testErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			task := defineMockStructuredListLogEntriesTask("container-logs", tc.source)
			fetcher := tc.fetcherFactory(t)
			resourceNamesInput := NewResourceNamesInput()
			clientFactory, err := googlecloud.NewClientFactory()
			if err != nil {
				t.Fatalf("failed to create ClientFactory: %v", err)
			}

			firstCtx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			_, _, err = inspectiontest.Run(t, firstCtx, task, inspectioncore.TaskModeDryRun, map[string]any{},
				tasktest.Given(InputStartTimeTaskID.Ref(), startTime),
				tasktest.Given(InputEndTimeTaskID.Ref(), endTime),
				tasktest.Given(InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
				tasktest.Given[LogFetcher](LoggingFetcherTaskID.Ref(), fetcher),
				tasktest.Given(APIClientFactoryTaskID.Ref(), clientFactory),
				tasktest.Given(APIClientCallOptionsInjectorTaskID.Ref(), googlecloud.NewCallOptionInjector()),
			)
			if err != nil {
				t.Fatalf("dry run failed: %v", err)
			}

			nextCtx := inspectiontest.NextRunTaskContext(t.Context(), firstCtx)
			gotLogs, _, err := inspectiontest.Run(t, nextCtx, task, inspectioncore.TaskModeRun, map[string]any{},
				tasktest.Given(InputStartTimeTaskID.Ref(), startTime),
				tasktest.Given(InputEndTimeTaskID.Ref(), endTime),
				tasktest.Given(InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
				tasktest.Given[LogFetcher](LoggingFetcherTaskID.Ref(), fetcher),
				tasktest.Given(APIClientFactoryTaskID.Ref(), clientFactory),
				tasktest.Given(APIClientCallOptionsInjectorTaskID.Ref(), googlecloud.NewCallOptionInjector()),
			)

			if tc.wantError != nil {
				if !errors.Is(err, tc.wantError) {
					t.Errorf("error mismatch: got %v, want %v", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotLogsString := []string{}
			for _, l := range gotLogs {
				yaml, err := l.Serialize(structured.EmptyFieldPath, &structured.YAMLNodeSerializer{})
				if err != nil {
					t.Fatalf("failed to serialize to yaml: %v", err)
				}
				gotLogsString = append(gotLogsString, string(yaml))
			}

			if diff := cmp.Diff(tc.wantLogsString, gotLogsString); diff != "" {
				t.Errorf("Logs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestSetStructuredQueryInfo(t *testing.T) {
	t.Parallel()
	taskID := "task-structured"
	startTime := time.Date(2025, time.January, 1, 1, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, time.January, 1, 1, 1, 0, 0, time.UTC)
	queryName := "k8s-logs"
	baseFilter := "resource.type=k8s_container"

	testCases := []struct {
		desc            string
		queryIndex      int
		totalQueryCount int
		estimatedCount  *int64
		incomplete      bool
		wantQuery       *inspectionmetadata.QueryItem
	}{
		{
			desc:            "single query with estimate",
			queryIndex:      0,
			totalQueryCount: 1,
			estimatedCount:  ptr(int64(12500)),
			wantQuery: &inspectionmetadata.QueryItem{
				Id:   taskID,
				Name: "k8s-logs",
				Query: `resource.type=k8s_container
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`,
				EstimatedCount: ptr(int64(12500)),
			},
		},
		{
			desc:            "multi query index 1 with estimate",
			queryIndex:      1,
			totalQueryCount: 2,
			estimatedCount:  ptr(int64(450)),
			wantQuery: &inspectionmetadata.QueryItem{
				Id:   taskID,
				Name: "k8s-logs-1",
				Query: `resource.type=k8s_container
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`,
				EstimatedCount: ptr(int64(450)),
			},
		},
		{
			desc:            "query with 0 estimate is preserved as 0",
			queryIndex:      0,
			totalQueryCount: 1,
			estimatedCount:  ptr(int64(0)),
			wantQuery: &inspectionmetadata.QueryItem{
				Id:   taskID,
				Name: "k8s-logs",
				Query: `resource.type=k8s_container
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`,
				EstimatedCount: ptr(int64(0)),
			},
		},
		{
			desc:            "query with nil estimate has nil EstimatedCount",
			queryIndex:      0,
			totalQueryCount: 1,
			estimatedCount:  nil,
			incomplete:      false,
			wantQuery: &inspectionmetadata.QueryItem{
				Id:   taskID,
				Name: "k8s-logs",
				Query: `resource.type=k8s_container
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`,
				EstimatedCount: nil,
				Incomplete:     false,
			},
		},
		{
			desc:            "query marked as incomplete",
			queryIndex:      0,
			totalQueryCount: 1,
			estimatedCount:  nil,
			incomplete:      true,
			wantQuery: &inspectionmetadata.QueryItem{
				Id:   taskID,
				Name: "k8s-logs",
				Query: `resource.type=k8s_container
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`,
				EstimatedCount: nil,
				Incomplete:     true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			err := setStructuredQueryInfo(ctx, taskID, baseFilter, tc.queryIndex, tc.totalQueryCount, startTime, endTime, queryName, tc.estimatedCount, tc.incomplete)
			if err != nil {
				t.Fatalf("setStructuredQueryInfo returned error: %v", err)
			}

			metadata := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
			queryMetadata, found := typedmap.Get(metadata, inspectionmetadata.QueryMetadataKey)
			if !found {
				t.Fatalf("QueryMetadata not found")
			}

			serialized := queryMetadata.ToSerializable().([]*inspectionmetadata.QueryItem)
			if len(serialized) != 1 {
				t.Fatalf("expected 1 query item, got %d", len(serialized))
			}

			if diff := cmp.Diff(tc.wantQuery, serialized[0]); diff != "" {
				t.Errorf("QueryItem mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestStructuredListLogEntriesTask_DryRun_EstimationCache(t *testing.T) {
	source := &mockStructuredLogQuerySource{
		resourceNames: []string{"projects/test-project"},
		queries: []*logestimator.StructuredLogQuery{
			{
				ResourceTypes: []string{"k8s_cluster"},
				Filters:       []logestimator.LoggingMonitoringMatcher{logestimator.ResourceLabel("cluster_name", logestimator.Exact("test-cluster"))},
			},
		},
	}
	task := defineMockStructuredListLogEntriesTask("test-cache-query", source)
	clientFactory, err := googlecloud.NewClientFactory()
	if err != nil {
		t.Fatalf("failed to create clientFactory: %v", err)
	}

	startTime := time.Date(2025, time.January, 1, 1, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, time.January, 1, 1, 1, 0, 0, time.UTC)
	resourceNamesInput := NewResourceNamesInput()

	// Run DryRun 1
	firstCtx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	_, _, err = inspectiontest.Run(t, firstCtx, task, inspectioncore.TaskModeDryRun, map[string]any{},
		tasktest.Given(InputStartTimeTaskID.Ref(), startTime),
		tasktest.Given(InputEndTimeTaskID.Ref(), endTime),
		tasktest.Given(InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
		// Dry run records the query without fetching logs.
		tasktest.Given[LogFetcher](LoggingFetcherTaskID.Ref(), nil),
		tasktest.Given(APIClientFactoryTaskID.Ref(), clientFactory),
		tasktest.Given(APIClientCallOptionsInjectorTaskID.Ref(), googlecloud.NewCallOptionInjector()),
	)
	if err != nil {
		t.Fatalf("first dryrun failed: %v", err)
	}

	sharedMap := khictx.MustGetValue(firstCtx, inspectioncore.InspectionSharedMap)
	cachedEstimator, found := typedmap.Get(sharedMap, LogEstimatorCacheKey)
	if !found || cachedEstimator == nil {
		t.Fatalf("expected CachedStructuredLogEstimator to be stored in InspectionSharedMap")
	}

	// Run DryRun 2 in the same inspection session
	nextCtx := inspectiontest.NextRunTaskContext(t.Context(), firstCtx)
	_, _, err = inspectiontest.Run(t, nextCtx, task, inspectioncore.TaskModeDryRun, map[string]any{},
		tasktest.Given(InputStartTimeTaskID.Ref(), startTime),
		tasktest.Given(InputEndTimeTaskID.Ref(), endTime),
		tasktest.Given(InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
		// Dry run records the query without fetching logs.
		tasktest.Given[LogFetcher](LoggingFetcherTaskID.Ref(), nil),
		tasktest.Given(APIClientFactoryTaskID.Ref(), clientFactory),
		tasktest.Given(APIClientCallOptionsInjectorTaskID.Ref(), googlecloud.NewCallOptionInjector()),
	)
	if err != nil {
		t.Fatalf("second dryrun failed: %v", err)
	}

	// Verify that the cached estimator instance was reused
	nextSharedMap := khictx.MustGetValue(nextCtx, inspectioncore.InspectionSharedMap)
	reusedEstimator, found := typedmap.Get(nextSharedMap, LogEstimatorCacheKey)
	if !found || reusedEstimator != cachedEstimator {
		t.Errorf("expected same CachedStructuredLogEstimator instance to be reused across dryruns")
	}
}

func TestStructuredListLogEntriesTask_DryRun_Incomplete(t *testing.T) {
	testCases := []struct {
		name          string
		resourceNames []string
		queries       []*logestimator.StructuredLogQuery
		want          *inspectionmetadata.QueryItem
	}{
		{
			name:          "query explicitly marked as incomplete",
			resourceNames: []string{"projects/test-project"},
			queries: []*logestimator.StructuredLogQuery{
				{
					Incomplete:    true,
					ResourceTypes: []string{"k8s_cluster"},
					Filters:       []logestimator.LoggingMonitoringMatcher{logestimator.ResourceLabel("cluster_name", logestimator.Exact(""))},
				},
			},
			want: &inspectionmetadata.QueryItem{
				Id:             "structured-test#default",
				Name:           "test-query",
				Query:          "resource.type=\"k8s_cluster\"\nresource.labels.cluster_name=\"\"\ntimestamp >= \"2025-01-01T01:00:00+0000\"\ntimestamp <= \"2025-01-01T01:01:00+0000\"",
				EstimatedCount: nil,
				Incomplete:     true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source := &mockStructuredLogQuerySource{
				resourceNames: tc.resourceNames,
				queries:       tc.queries,
			}
			task := defineMockStructuredListLogEntriesTask("test-query", source)
			clientFactory, err := googlecloud.NewClientFactory()
			if err != nil {
				t.Fatalf("failed to create clientFactory: %v", err)
			}

			startTime := time.Date(2025, time.January, 1, 1, 0, 0, 0, time.UTC)
			endTime := time.Date(2025, time.January, 1, 1, 1, 0, 0, time.UTC)
			resourceNamesInput := NewResourceNamesInput()
			resourceNamesInput.UpdateDefaultResourceNamesForQuery("structured-test", tc.resourceNames)

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			_, _, err = inspectiontest.Run(t, ctx, task, inspectioncore.TaskModeDryRun, map[string]any{},
				tasktest.Given(InputStartTimeTaskID.Ref(), startTime),
				tasktest.Given(InputEndTimeTaskID.Ref(), endTime),
				tasktest.Given(InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
				// Dry run records the query without fetching logs.
				tasktest.Given[LogFetcher](LoggingFetcherTaskID.Ref(), nil),
				tasktest.Given(APIClientFactoryTaskID.Ref(), clientFactory),
				tasktest.Given(APIClientCallOptionsInjectorTaskID.Ref(), googlecloud.NewCallOptionInjector()),
			)
			if err != nil {
				t.Fatalf("dryrun failed: %v", err)
			}

			metadata := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
			queryMetadata, found := typedmap.Get(metadata, inspectionmetadata.QueryMetadataKey)
			if !found {
				t.Fatalf("QueryMetadata not found")
			}

			serialized := queryMetadata.ToSerializable().([]*inspectionmetadata.QueryItem)
			if len(serialized) != 1 {
				t.Fatalf("expected 1 query item, got %d", len(serialized))
			}

			if diff := cmp.Diff(tc.want, serialized[0]); diff != "" {
				t.Errorf("QueryItem mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestStructuredListLogEntriesTask_DryRun_CallOptionInjector(t *testing.T) {
	source := &mockStructuredLogQuerySource{
		resourceNames: []string{"projects/test-project"},
		queries: []*logestimator.StructuredLogQuery{
			{
				Incomplete:    true,
				ResourceTypes: []string{"k8s_cluster"},
				Filters:       []logestimator.LoggingMonitoringMatcher{logestimator.ResourceLabel("cluster_name", logestimator.Exact(""))},
			},
		},
	}
	task := defineMockStructuredListLogEntriesTask("test-query", source)
	clientFactory, err := googlecloud.NewClientFactory()
	if err != nil {
		t.Fatalf("failed to create clientFactory: %v", err)
	}

	startTime := time.Date(2025, time.January, 1, 1, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, time.January, 1, 1, 1, 0, 0, time.UTC)
	resourceNamesInput := NewResourceNamesInput()
	resourceNamesInput.UpdateDefaultResourceNamesForQuery("structured-test", []string{"projects/test-project"})

	inputs := []tasktest.InputValue{
		tasktest.Given(InputStartTimeTaskID.Ref(), startTime),
		tasktest.Given(InputEndTimeTaskID.Ref(), endTime),
		tasktest.Given(InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
		// Dry run records the query without fetching logs.
		tasktest.Given[LogFetcher](LoggingFetcherTaskID.Ref(), nil),
		tasktest.Given(APIClientFactoryTaskID.Ref(), clientFactory),
		tasktest.Given(APIClientCallOptionsInjectorTaskID.Ref(), googlecloud.NewCallOptionInjector()),
	}

	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	_, _, err = inspectiontest.Run(t, ctx, task, inspectioncore.TaskModeDryRun, map[string]any{}, inputs...)
	if err != nil {
		t.Fatalf("dryrun failed: %v", err)
	}

	metadata := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
	_, found := typedmap.Get(metadata, inspectionmetadata.QueryMetadataKey)
	if !found {
		t.Fatalf("QueryMetadata not found")
	}
}

// describeInputs converts point-to-point input specs to comparable strings, because dependency descriptors hold unexported fields.
func describeInputs(specs []coretask.InputSpec) []string {
	result := make([]string, 0, len(specs))
	for _, spec := range specs {
		result = append(result, fmt.Sprintf("%s %s", spec.Kind, spec.Dependency.(taskid.PointToPointDescriptor).ReferenceID()))
	}
	return result
}

// projectQuerySource is a StructuredLogQuerySource whose resource names and query use a project ID read through an input handle.
type projectQuerySource struct {
	projectID          coretask.Input[string]
	noQueries          bool
	timePartitionCount int
}

func (s *projectQuerySource) DefaultResourceNames(ctx context.Context) ([]string, error) {
	return []string{"projects/" + s.projectID.Get(ctx)}, nil
}

func (s *projectQuerySource) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	if s.noQueries {
		return nil, nil
	}
	// The query is marked incomplete so that dry run records it without calling the log estimation APIs.
	return []*logestimator.StructuredLogQuery{{
		Incomplete:    true,
		ResourceTypes: []string{"k8s_container"},
		Filters: []logestimator.LoggingMonitoringMatcher{
			logestimator.ResourceLabel("project_id", logestimator.Exact(s.projectID.Get(ctx))),
		},
	}}, nil
}

func (s *projectQuerySource) TimePartitionCount(ctx context.Context) (int, error) {
	return s.timePartitionCount, nil
}

var _ StructuredLogQuerySource = (*projectQuerySource)(nil)

func TestDefineStructuredListLogEntriesTask(t *testing.T) {
	taskID := taskid.NewDefaultImplementationID[[]*log.Log]("structured-test")
	projectIDTaskID := taskid.NewDefaultImplementationID[string]("structured-test-project-id")
	startTime := time.Date(2025, time.January, 1, 1, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, time.January, 1, 1, 1, 0, 0, time.UTC)
	wantInputs := []string{
		"required " + InputStartTimeTaskID.ReferenceIDString(),
		"required " + InputEndTimeTaskID.ReferenceIDString(),
		"required " + InputLoggingFilterResourceNameTaskID.ReferenceIDString(),
		"required " + LoggingFetcherTaskID.ReferenceIDString(),
		"required " + APIClientFactoryTaskID.ReferenceIDString(),
		"required " + APIClientCallOptionsInjectorTaskID.ReferenceIDString(),
		"required structured-test-project-id",
	}
	recordedQuery := `resource.type="k8s_container"
resource.labels.project_id="test-project"
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`
	fetchFilter := `resource.type="k8s_container"
resource.labels.project_id="test-project"
timestamp >= "2025-01-01T01:00:00+0000"
timestamp < "2025-01-01T01:01:00+0000"`

	testCases := []struct {
		desc               string
		taskMode           inspectioncore.InspectionTaskModeType
		noQueries          bool
		timePartitionCount int
		fetch              func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error)
		wantLogs           []string
		wantQueries        []*inspectionmetadata.QueryItem
		wantErrSubstr      string
	}{
		{
			desc:               "dry run records the query with the query name",
			taskMode:           inspectioncore.TaskModeDryRun,
			timePartitionCount: 1,
			wantQueries: []*inspectionmetadata.QueryItem{
				{Id: taskID.String(), Name: "container-logs", Query: recordedQuery, Incomplete: true},
			},
		},
		{
			desc:               "run fetches the logs of the query",
			taskMode:           inspectioncore.TaskModeRun,
			timePartitionCount: 1,
			fetch: func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
				logSource <- &loggingpb.LogEntry{InsertId: "log-1", LogName: "container-log"}
				logSource <- &loggingpb.LogEntry{InsertId: "log-2", LogName: "container-log"}
			},
			wantLogs: []string{
				"insertId: log-1\nlogName: container-log\n",
				"insertId: log-2\nlogName: container-log\n",
			},
			wantQueries: []*inspectionmetadata.QueryItem{
				{Id: taskID.String(), Name: "container-logs", Query: recordedQuery},
			},
		},
		{
			desc:               "returns no logs when the source has no queries",
			taskMode:           inspectioncore.TaskModeRun,
			noQueries:          true,
			timePartitionCount: 1,
		},
		{
			desc:               "returns the fetch error",
			taskMode:           inspectioncore.TaskModeRun,
			timePartitionCount: 1,
			fetch: func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
				errSource <- errors.New("fetch error")
			},
			wantErrSubstr: "fetch error",
		},
		{
			desc:               "rejects a time partition count less than 1",
			taskMode:           inspectioncore.TaskModeRun,
			timePartitionCount: 0,
			wantErrSubstr:      "TimePartitionCount returned an invalid value 0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			task := DefineStructuredListLogEntriesTask(taskID, "container-logs", func(b *coretask.Binder) StructuredLogQuerySource {
				return &projectQuerySource{
					projectID:          coretask.Use(b, projectIDTaskID.Ref()),
					noQueries:          tc.noQueries,
					timePartitionCount: tc.timePartitionCount,
				}
			})
			if diff := cmp.Diff(wantInputs, describeInputs(task.Inputs())); diff != "" {
				t.Errorf("Inputs() mismatch (-want +got):\n%s", diff)
			}

			var upstreams []fakeLogUpstreamPair
			if tc.fetch != nil {
				upstreams = append(upstreams, newFakeLogUpstreamPair(fetchFilter, tc.fetch))
			}
			fetcher := getMockFetcherFromFakeLogUpstreamPairs(t, upstreams)
			clientFactory, err := googlecloud.NewClientFactory()
			if err != nil {
				t.Fatalf("failed to create ClientFactory: %v", err)
			}
			// The resource names to query come from the previous default, and the task replaces the default with the one of the source.
			resourceNamesInput := NewResourceNamesInput()
			resourceNamesInput.UpdateDefaultResourceNamesForQuery(taskID.ReferenceIDString(), []string{"projects/previous-project"})

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			gotLogs, _, err := inspectiontest.Run(t, ctx, task, tc.taskMode, map[string]any{},
				tasktest.Given(InputStartTimeTaskID.Ref(), startTime),
				tasktest.Given(InputEndTimeTaskID.Ref(), endTime),
				tasktest.Given(InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
				tasktest.Given[LogFetcher](LoggingFetcherTaskID.Ref(), fetcher),
				tasktest.Given(APIClientFactoryTaskID.Ref(), clientFactory),
				tasktest.Given(APIClientCallOptionsInjectorTaskID.Ref(), googlecloud.NewCallOptionInjector()),
				tasktest.Given(projectIDTaskID.Ref(), "test-project"),
			)
			if tc.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("Run() error = %v, want error containing %q", err, tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() returned an unexpected error: %v", err)
			}

			var gotLogStrings []string
			for _, l := range gotLogs {
				yaml, err := l.Serialize(structured.EmptyFieldPath, &structured.YAMLNodeSerializer{})
				if err != nil {
					t.Fatalf("failed to serialize to yaml: %v", err)
				}
				gotLogStrings = append(gotLogStrings, string(yaml))
			}
			if diff := cmp.Diff(tc.wantLogs, gotLogStrings, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("logs mismatch (-want +got):\n%s", diff)
			}

			metadata := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
			queryMetadata, found := typedmap.Get(metadata, inspectionmetadata.QueryMetadataKey)
			if !found {
				t.Fatalf("QueryMetadata not found")
			}
			gotQueries := queryMetadata.ToSerializable().([]*inspectionmetadata.QueryItem)
			if diff := cmp.Diff(tc.wantQueries, gotQueries, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("QueryItem mismatch (-want +got):\n%s", diff)
			}

			queryResourceNames, found := typeddict.Get(resourceNamesInput.resourceNames, taskID.ReferenceIDString())
			if !found {
				t.Fatalf("resource names for %q not found", taskID.ReferenceIDString())
			}
			if diff := cmp.Diff([]string{"projects/test-project"}, queryResourceNames.DefaultResourceNames); diff != "" {
				t.Errorf("default resource names mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSetErrorMetadataForFetchLogError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		desc             string
		err              error
		wantErrorMessage *inspectionmetadata.ErrorMessage
	}{
		{
			desc: "unauthenticated error",
			err:  status.Error(codes.Unauthenticated, "permission denied"),
			wantErrorMessage: &inspectionmetadata.ErrorMessage{
				ErrorId: 0,
				Message: "rpc error: code = Unauthenticated desc = permission denied",
			},
		},
		{
			desc: "non-grpc error",
			err:  khierrors.ErrInvalidInput,
			wantErrorMessage: &inspectionmetadata.ErrorMessage{
				ErrorId: 0,
				Message: "invalid input",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			setErrorMetadataForFetchLogError(ctx, tt.err)

			metadata := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
			errorMessageSet, found := typedmap.Get(metadata, inspectionmetadata.ErrorMessageSetMetadataKey)
			if !found {
				t.Fatalf("error message set metadata not found")
			}
			if diff := cmp.Diff(tt.wantErrorMessage, errorMessageSet.ErrorMessages[0]); diff != "" {
				t.Errorf("setErrorMetadataForFetchLogError() mismatch (-want +got):\n%s", diff)
			}

		})
	}
}

func TestGroupResourceNamesByContainer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		resourceNames []string
		want          []*resourceContainerLogQueryGroup
		wantErr       bool
	}{
		{
			name: "valid project-based resource names",
			resourceNames: []string{
				"projects/project-1/locations/us-central1/buckets/bucket-1/views/view-1",
				"projects/project-2/locations/us-west1/buckets/bucket-2/views/view-2",
				"projects/project-1/locations/asia-northeast1/buckets/bucket-3/views/view-3",
			},
			want: []*resourceContainerLogQueryGroup{
				{
					container:     googlecloud.Project("project-1"),
					resourceNames: []string{"projects/project-1/locations/us-central1/buckets/bucket-1/views/view-1", "projects/project-1/locations/asia-northeast1/buckets/bucket-3/views/view-3"},
				},
				{
					container:     googlecloud.Project("project-2"),
					resourceNames: []string{"projects/project-2/locations/us-west1/buckets/bucket-2/views/view-2"},
				},
			},
		},
		{
			name: "unsupported resource name format",
			resourceNames: []string{
				"folders/12345",
			},
			wantErr: true,
		},
		{
			name:          "empty resource names",
			resourceNames: []string{},
			want:          nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := groupResourceNamesByContainer(tt.resourceNames)
			if (err != nil) != tt.wantErr {
				t.Errorf("groupResourceNamesByContainer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(resourceContainerLogQueryGroup{}), cmpopts.AcyclicTransformer("container", func(c googlecloud.ResourceContainer) string { return c.Identifier() })); diff != "" {
				t.Errorf("groupResourceNamesByContainer() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDivideGroupByMaximumResourceName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                    string
		groups                  []*resourceContainerLogQueryGroup
		maxResourceNamePerGroup int
		want                    []*resourceContainerLogQueryGroup
	}{
		{
			name: "group smaller than max",
			groups: []*resourceContainerLogQueryGroup{
				{container: googlecloud.Project("project-1"), resourceNames: []string{"r1", "r2"}},
			},
			maxResourceNamePerGroup: 3,
			want: []*resourceContainerLogQueryGroup{
				{container: googlecloud.Project("project-1"), resourceNames: []string{"r1", "r2"}},
			},
		},
		{
			name: "group equal to max",
			groups: []*resourceContainerLogQueryGroup{
				{container: googlecloud.Project("project-1"), resourceNames: []string{"r1", "r2", "r3"}},
			},
			maxResourceNamePerGroup: 3,
			want: []*resourceContainerLogQueryGroup{
				{container: googlecloud.Project("project-1"), resourceNames: []string{"r1", "r2", "r3"}},
			},
		},
		{
			name: "group needs multiple splits",
			groups: []*resourceContainerLogQueryGroup{
				{container: googlecloud.Project("project-1"), resourceNames: []string{"r1", "r2", "r3", "r4", "r5", "r6", "r7"}},
			},
			maxResourceNamePerGroup: 3,
			want: []*resourceContainerLogQueryGroup{
				{container: googlecloud.Project("project-1"), resourceNames: []string{"r1", "r2", "r3"}},
				{container: googlecloud.Project("project-1"), resourceNames: []string{"r4", "r5", "r6"}},
				{container: googlecloud.Project("project-1"), resourceNames: []string{"r7"}},
			},
		},
		{
			name: "multiple groups, some need splitting",
			groups: []*resourceContainerLogQueryGroup{
				{container: googlecloud.Project("project-1"), resourceNames: []string{"p1r1", "p1r2", "p1r3", "p1r4"}},
				{container: googlecloud.Project("project-2"), resourceNames: []string{"p2r1", "p2r2"}},
			},
			maxResourceNamePerGroup: 2,
			want: []*resourceContainerLogQueryGroup{
				{container: googlecloud.Project("project-1"), resourceNames: []string{"p1r1", "p1r2"}},
				{container: googlecloud.Project("project-1"), resourceNames: []string{"p1r3", "p1r4"}},
				{container: googlecloud.Project("project-2"), resourceNames: []string{"p2r1", "p2r2"}},
			},
		},
		{
			name:                    "empty input",
			groups:                  nil,
			maxResourceNamePerGroup: 5,
			want:                    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := divideGroupByMaximumResourceName(tt.groups, tt.maxResourceNamePerGroup)
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(resourceContainerLogQueryGroup{}), cmpopts.AcyclicTransformer("container", func(c googlecloud.ResourceContainer) string { return c.Identifier() })); diff != "" {
				t.Errorf("divideGroupByMaximumResourceName() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMonitorProgress(t *testing.T) {
	t.Parallel()

	progressDest := inspectionmetadata.NewTaskProgressMetadata("test-task")
	ctx := progress.WithContext(t.Context(), progressDest)
	tracker := progress.NewRatioTracker(ctx, progress.WithUnit("logs"))
	source := make(chan LogFetchProgress)
	baseLogCount := 100
	listCallIndex := 0
	totalListCalls := 2

	var wg sync.WaitGroup
	monitorProgress(ctx, &wg, source, tracker, baseLogCount, listCallIndex, totalListCalls)

	source <- LogFetchProgress{
		LogCount: 50,
		Progress: 0.5,
	}
	close(source)
	wg.Wait()

	snap := progressDest.Snapshot()
	wantRatio := float32(0+0.5) / float32(2) // 0.25
	if snap.Ratio != wantRatio {
		t.Errorf("snap.Ratio = %v, want %v", snap.Ratio, wantRatio)
	}

	wantMsg := "[1/2] 150 logs"
	if snap.Message != wantMsg {
		t.Errorf("snap.Message = %q, want %q", snap.Message, wantMsg)
	}
}
