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
	"errors"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/logging/apiv2/loggingpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

type mockStructuredListLogEntriesTaskSetting struct {
	dependencies       []taskid.UntypedTaskReference
	resourceNames      []string
	queries            []*logestimator.StructuredLogQuery
	timePartitionCount int
	queryName          string
}

func (s *mockStructuredListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return s.dependencies
}

func (s *mockStructuredListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	return s.resourceNames, nil
}

func (s *mockStructuredListLogEntriesTaskSetting) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	return s.queries, nil
}

func (s *mockStructuredListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return taskid.NewDefaultImplementationID[[]*log.Log]("structured-test")
}

func (s *mockStructuredListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return s.timePartitionCount, nil
}

func (s *mockStructuredListLogEntriesTaskSetting) QueryName() string {
	return s.queryName
}

var _ StructuredListLogEntriesTaskSetting = (*mockStructuredListLogEntriesTaskSetting)(nil)

func TestStructuredListLogEntriesTask_DryRun_FallbackWhenNoClient(t *testing.T) {
	t.Parallel()
	startTime := time.Date(2025, time.January, 1, 1, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, time.January, 1, 1, 1, 0, 0, time.UTC)

	setting := &mockStructuredListLogEntriesTaskSetting{
		queryName:     "container-logs",
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

	task := NewStructuredListLogEntriesTask(setting)
	resourceNamesInput := NewResourceNamesInput()
	clientFactory, err := googlecloud.NewClientFactory()
	if err != nil {
		t.Fatalf("failed to create ClientFactory: %v", err)
	}

	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	gotLogs, _, err := inspectiontest.RunInspectionTask(ctx, task, inspectioncore_contract.TaskModeDryRun, map[string]any{},
		tasktest.NewTaskDependencyValuePair(InputStartTimeTaskID.Ref(), startTime),
		tasktest.NewTaskDependencyValuePair(InputEndTimeTaskID.Ref(), endTime),
		tasktest.NewTaskDependencyValuePair(APIClientFactoryTaskID.Ref(), clientFactory),
		tasktest.NewTaskDependencyValuePair(InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
	)
	if err != nil {
		t.Fatalf("DryRun returned unexpected error: %v", err)
	}
	if len(gotLogs) != 0 {
		t.Errorf("DryRun should return empty logs, got %d", len(gotLogs))
	}

	// Verify QueryMetadata
	metadata := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)
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
		setting        *mockStructuredListLogEntriesTaskSetting
		fetcherFactory func(t *testing.T) *mockLogFetcher
		wantLogsString []string
		wantError      error
	}{
		{
			desc: "successful log fetch with single query",
			setting: &mockStructuredListLogEntriesTaskSetting{
				queryName:     "container-logs",
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
			setting: &mockStructuredListLogEntriesTaskSetting{
				queryName:     "container-logs",
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
			task := NewStructuredListLogEntriesTask(tc.setting)
			fetcher := tc.fetcherFactory(t)
			resourceNamesInput := NewResourceNamesInput()
			clientFactory, err := googlecloud.NewClientFactory()
			if err != nil {
				t.Fatalf("failed to create ClientFactory: %v", err)
			}

			firstCtx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			_, _, err = inspectiontest.RunInspectionTask(firstCtx, task, inspectioncore_contract.TaskModeDryRun, map[string]any{},
				tasktest.NewTaskDependencyValuePair(InputStartTimeTaskID.Ref(), startTime),
				tasktest.NewTaskDependencyValuePair(InputEndTimeTaskID.Ref(), endTime),
				tasktest.NewTaskDependencyValuePair(APIClientFactoryTaskID.Ref(), clientFactory),
				tasktest.NewTaskDependencyValuePair[LogFetcher](LoggingFetcherTaskID.Ref(), fetcher),
				tasktest.NewTaskDependencyValuePair(InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
			)
			if err != nil {
				t.Fatalf("dry run failed: %v", err)
			}

			nextCtx := inspectiontest.NextRunTaskContext(t.Context(), firstCtx)
			gotLogs, _, err := inspectiontest.RunInspectionTask(nextCtx, task, inspectioncore_contract.TaskModeRun, map[string]any{},
				tasktest.NewTaskDependencyValuePair(InputStartTimeTaskID.Ref(), startTime),
				tasktest.NewTaskDependencyValuePair(InputEndTimeTaskID.Ref(), endTime),
				tasktest.NewTaskDependencyValuePair[LogFetcher](LoggingFetcherTaskID.Ref(), fetcher),
				tasktest.NewTaskDependencyValuePair(APIClientFactoryTaskID.Ref(), clientFactory),
				tasktest.NewTaskDependencyValuePair(InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
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
				yaml, err := l.Serialize("", &structured.YAMLNodeSerializer{})
				if err != nil {
					t.Fatalf("failed to serialize to yaml: %v", err)
				}
				gotLogsString = append(gotLogsString, string(yaml))
				_, err = log.GetFieldSet(l, &log.CommonFieldSet{})
				if err != nil {
					t.Errorf("CommonFieldSet not found on log: %v", err)
				}
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

			metadata := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)
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
	taskSetting := &mockStructuredListLogEntriesTaskSetting{
		queryName:     "test-cache-query",
		resourceNames: []string{"projects/test-project"},
		queries: []*logestimator.StructuredLogQuery{
			{
				ResourceTypes: []string{"k8s_cluster"},
				Filters:       []logestimator.LoggingMonitoringMatcher{logestimator.ResourceLabel("cluster_name", logestimator.Exact("test-cluster"))},
			},
		},
	}
	task := NewStructuredListLogEntriesTask(taskSetting)
	clientFactory, err := googlecloud.NewClientFactory()
	if err != nil {
		t.Fatalf("failed to create clientFactory: %v", err)
	}

	startTime := time.Date(2025, time.January, 1, 1, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, time.January, 1, 1, 1, 0, 0, time.UTC)
	resourceNamesInput := NewResourceNamesInput()

	// Run DryRun 1
	firstCtx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	_, _, err = inspectiontest.RunInspectionTask(firstCtx, task, inspectioncore_contract.TaskModeDryRun, map[string]any{},
		tasktest.NewTaskDependencyValuePair(InputStartTimeTaskID.Ref(), startTime),
		tasktest.NewTaskDependencyValuePair(InputEndTimeTaskID.Ref(), endTime),
		tasktest.NewTaskDependencyValuePair(APIClientFactoryTaskID.Ref(), clientFactory),
		tasktest.NewTaskDependencyValuePair(InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
	)
	if err != nil {
		t.Fatalf("first dryrun failed: %v", err)
	}

	sharedMap := khictx.MustGetValue(firstCtx, inspectioncore_contract.InspectionSharedMap)
	cachedEstimator, found := typedmap.Get(sharedMap, LogEstimatorCacheKey)
	if !found || cachedEstimator == nil {
		t.Fatalf("expected CachedStructuredLogEstimator to be stored in InspectionSharedMap")
	}

	// Run DryRun 2 in the same inspection session
	nextCtx := inspectiontest.NextRunTaskContext(t.Context(), firstCtx)
	_, _, err = inspectiontest.RunInspectionTask(nextCtx, task, inspectioncore_contract.TaskModeDryRun, map[string]any{},
		tasktest.NewTaskDependencyValuePair(InputStartTimeTaskID.Ref(), startTime),
		tasktest.NewTaskDependencyValuePair(InputEndTimeTaskID.Ref(), endTime),
		tasktest.NewTaskDependencyValuePair(APIClientFactoryTaskID.Ref(), clientFactory),
		tasktest.NewTaskDependencyValuePair(InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
	)
	if err != nil {
		t.Fatalf("second dryrun failed: %v", err)
	}

	// Verify that the cached estimator instance was reused
	nextSharedMap := khictx.MustGetValue(nextCtx, inspectioncore_contract.InspectionSharedMap)
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
			taskSetting := &mockStructuredListLogEntriesTaskSetting{
				queryName:     "test-query",
				resourceNames: tc.resourceNames,
				queries:       tc.queries,
			}
			task := NewStructuredListLogEntriesTask(taskSetting)
			clientFactory, err := googlecloud.NewClientFactory()
			if err != nil {
				t.Fatalf("failed to create clientFactory: %v", err)
			}

			startTime := time.Date(2025, time.January, 1, 1, 0, 0, 0, time.UTC)
			endTime := time.Date(2025, time.January, 1, 1, 1, 0, 0, time.UTC)
			resourceNamesInput := NewResourceNamesInput()
			resourceNamesInput.UpdateDefaultResourceNamesForQuery("structured-test", tc.resourceNames)

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			_, _, err = inspectiontest.RunInspectionTask(ctx, task, inspectioncore_contract.TaskModeDryRun, map[string]any{},
				tasktest.NewTaskDependencyValuePair(InputStartTimeTaskID.Ref(), startTime),
				tasktest.NewTaskDependencyValuePair(InputEndTimeTaskID.Ref(), endTime),
				tasktest.NewTaskDependencyValuePair(APIClientFactoryTaskID.Ref(), clientFactory),
				tasktest.NewTaskDependencyValuePair(InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
			)
			if err != nil {
				t.Fatalf("dryrun failed: %v", err)
			}

			metadata := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)
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
