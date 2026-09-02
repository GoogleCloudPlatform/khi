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
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/logging/apiv2/loggingpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockListLogEntriesTaskSetting struct {
	dependencies       []taskid.UntypedTaskReference
	resourceNames      []string
	logFilters         []string
	timePartitionCount int
	description        *ListLogEntriesTaskDescription
}

// Dependencies implements ListLogEntriesTaskSetting.
func (s *mockListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return s.dependencies
}

// Description implements ListLogEntriesTaskSetting.
func (s *mockListLogEntriesTaskSetting) Description() *ListLogEntriesTaskDescription {
	return s.description
}

// LogFilters implements ListLogEntriesTaskSetting.
func (s *mockListLogEntriesTaskSetting) LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	return s.logFilters, nil
}

// DefaultResourceNames implements ListLogEntriesTaskSetting.
func (s *mockListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	return s.resourceNames, nil
}

// TaskID implements ListLogEntriesTaskSetting.
func (s *mockListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return taskid.NewDefaultImplementationID[[]*log.Log]("test")
}

// TimePartitionCount implements ListLogEntriesTaskSetting.
func (s *mockListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return s.timePartitionCount, nil
}

var _ ListLogEntriesTaskSetting = (*mockListLogEntriesTaskSetting)(nil)

func TestNewListLogEntriesTask(t *testing.T) {
	t.Parallel()
	startTime := time.Date(2025, time.January, 1, 1, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, time.January, 1, 1, 1, 0, 0, time.UTC)
	testErr := fmt.Errorf("test error")
	description := &ListLogEntriesTaskDescription{
		QueryName:    "query-foo",
		ExampleQuery: "resource.type=gce_instance AND severity=ERROR",
	}
	testCase := []struct {
		desc               string
		setting            *mockListLogEntriesTaskSetting
		fetcherFactory     func(t *testing.T) *mockLogFetcher
		mode               inspectioncore_contract.InspectionTaskModeType
		inputResourceNames string
		wantLogsString     []string
		wantError          error
	}{
		{
			desc: "dryrun doesn't call fetcher",
			setting: &mockListLogEntriesTaskSetting{
				logFilters:         []string{"foo"},
				resourceNames:      []string{"projects/bar"},
				timePartitionCount: 1,
				description:        description,
			},
			fetcherFactory: func(t *testing.T) *mockLogFetcher {
				return getMockFetcherFromFakeLogUpstreamPairs(t, []fakeLogUpstreamPair{})
			},
			mode:           inspectioncore_contract.TaskModeDryRun, // DryRun mode should not call the fetcher
			wantLogsString: []string{},
		},
		{
			desc: "run with empty filter listt doesn't call fetcher",
			setting: &mockListLogEntriesTaskSetting{
				logFilters:         []string{},
				resourceNames:      []string{"projects/bar"},
				timePartitionCount: 1,
				description:        description,
			},
			fetcherFactory: func(t *testing.T) *mockLogFetcher {
				return getMockFetcherFromFakeLogUpstreamPairs(t, []fakeLogUpstreamPair{})
			},
			mode:           inspectioncore_contract.TaskModeRun,
			wantLogsString: []string{},
		},
		{
			desc: "with a single log filter not producing any log",
			setting: &mockListLogEntriesTaskSetting{
				logFilters:         []string{"foo"},
				resourceNames:      []string{"projects/bar"},
				timePartitionCount: 1,
				description:        description,
			},
			fetcherFactory: func(t *testing.T) *mockLogFetcher {
				return getMockFetcherFromFakeLogUpstreamPairs(t, []fakeLogUpstreamPair{
					newFakeLogUpstreamPair(`foo
timestamp >= "2025-01-01T01:00:00+0000"
timestamp < "2025-01-01T01:01:00+0000"`, func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {

					}),
				})
			},
			mode:           inspectioncore_contract.TaskModeRun,
			wantLogsString: []string{},
		},
		{
			desc: "with a single log filter not producing logs",
			setting: &mockListLogEntriesTaskSetting{
				logFilters:         []string{"foo"},
				resourceNames:      []string{"projects/bar"},
				timePartitionCount: 1,
				description:        description,
			},
			fetcherFactory: func(t *testing.T) *mockLogFetcher {
				return getMockFetcherFromFakeLogUpstreamPairs(t, []fakeLogUpstreamPair{
					newFakeLogUpstreamPair(`foo
timestamp >= "2025-01-01T01:00:00+0000"
timestamp < "2025-01-01T01:01:00+0000"`, func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
						logSource <- &loggingpb.LogEntry{InsertId: "foo", LogName: "foo"}
						<-time.After(time.Second)
						logSource <- &loggingpb.LogEntry{InsertId: "bar", LogName: "bar"}
						logSource <- &loggingpb.LogEntry{InsertId: "qux", LogName: "qux"}
					}),
				})
			},
			mode: inspectioncore_contract.TaskModeRun,
			wantLogsString: []string{
				"insertId: foo\nlogName: foo\n",
				"insertId: bar\nlogName: bar\n",
				"insertId: qux\nlogName: qux\n",
			},
		},
		{
			desc: "with multiple log filter",
			setting: &mockListLogEntriesTaskSetting{
				logFilters:         []string{"foo", "bar"},
				resourceNames:      []string{"projects/bar"},
				timePartitionCount: 1,
				description:        description,
			},
			fetcherFactory: func(t *testing.T) *mockLogFetcher {
				return getMockFetcherFromFakeLogUpstreamPairs(t, []fakeLogUpstreamPair{
					newFakeLogUpstreamPair(`foo
timestamp >= "2025-01-01T01:00:00+0000"
timestamp < "2025-01-01T01:01:00+0000"`, func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
						logSource <- &loggingpb.LogEntry{InsertId: "foo", LogName: "foo"}
						<-time.After(time.Second)
						logSource <- &loggingpb.LogEntry{InsertId: "bar", LogName: "bar"}
						logSource <- &loggingpb.LogEntry{InsertId: "qux", LogName: "qux"}
					}),
					newFakeLogUpstreamPair(`bar
timestamp >= "2025-01-01T01:00:00+0000"
timestamp < "2025-01-01T01:01:00+0000"`, func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
						logSource <- &loggingpb.LogEntry{InsertId: "quux", LogName: "quux"}
					}),
				})
			},
			mode: inspectioncore_contract.TaskModeRun,
			wantLogsString: []string{
				"insertId: foo\nlogName: foo\n",
				"insertId: bar\nlogName: bar\n",
				"insertId: qux\nlogName: qux\n",
				"insertId: quux\nlogName: quux\n",
			},
		},
		{
			desc: "with error",
			setting: &mockListLogEntriesTaskSetting{
				logFilters:         []string{"foo"},
				resourceNames:      []string{"projects/bar"},
				timePartitionCount: 1,
				description:        description,
			},
			fetcherFactory: func(t *testing.T) *mockLogFetcher {
				return getMockFetcherFromFakeLogUpstreamPairs(t, []fakeLogUpstreamPair{
					newFakeLogUpstreamPair(`foo
timestamp >= "2025-01-01T01:00:00+0000"
timestamp < "2025-01-01T01:01:00+0000"`, func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
						logSource <- &loggingpb.LogEntry{InsertId: "foo", LogName: "foo"}
						<-time.After(time.Second)
						errSource <- testErr
					}),
				})
			},
			mode:      inspectioncore_contract.TaskModeRun,
			wantError: testErr,
		},
	}

	for _, tt := range testCase {
		t.Run(tt.desc, func(t *testing.T) {
			task := NewListLogEntriesTask(tt.setting)
			fetcher := tt.fetcherFactory(t)

			if task.ID().String() != "test#default" {
				t.Errorf("Task ID mismatch: got %s, want %s", task.ID().String(), "test#default")
			}
			gotIsQueryTask, found := typedmap.Get(task.Labels(), inspectioncore_contract.TaskLabelKeyIsQueryTask)
			if !found {
				t.Errorf("isQueryTask label not found")
			}
			if !gotIsQueryTask {
				t.Errorf("isQueryTask label is not true")
			}
			gotSampleQuery, found := typedmap.Get(task.Labels(), inspectioncore_contract.TaskLabelKeyQueryTaskSampleQuery)
			if !found {
				t.Errorf("sampleQuery label not found")
			}
			if diff := cmp.Diff(description.ExampleQuery, gotSampleQuery); diff != "" {
				t.Errorf("sampleQuery label mismatch (-want +got):\n%s", diff)
			}

			resourceNamesInput := NewResourceNamesInput()
			firstCtx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			_, _, err := inspectiontest.RunInspectionTask(firstCtx, task, inspectioncore_contract.TaskModeDryRun, map[string]any{},
				tasktest.NewTaskDependencyValuePair(InputStartTimeTaskID.Ref(), startTime),
				tasktest.NewTaskDependencyValuePair(InputEndTimeTaskID.Ref(), endTime),
				tasktest.NewTaskDependencyValuePair[LogFetcher](LoggingFetcherTaskID.Ref(), fetcher),
				tasktest.NewTaskDependencyValuePair(InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput))
			if err != nil {
				t.Errorf("first NewCloudLoggingFilterTask dry run failed:%v", err)
			}

			nextCtx := inspectiontest.NextRunTaskContext(t.Context(), firstCtx)
			inputIDForResourceName := (&QueryResourceNames{
				QueryID: "test",
			}).GetInputID()
			gotLogs, _, err := inspectiontest.RunInspectionTask(nextCtx, task, tt.mode, map[string]any{
				inputIDForResourceName: tt.inputResourceNames,
			},
				tasktest.NewTaskDependencyValuePair(InputStartTimeTaskID.Ref(), startTime),
				tasktest.NewTaskDependencyValuePair(InputEndTimeTaskID.Ref(), endTime),
				tasktest.NewTaskDependencyValuePair[LogFetcher](LoggingFetcherTaskID.Ref(), fetcher),
				tasktest.NewTaskDependencyValuePair(InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
			)
			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Errorf("NewCloudLoggingFilterTask() error = %v, wantErr %v", err, tt.wantError)
				}
				return
			}

			gotLogsString := []string{}
			for _, l := range gotLogs {
				yaml, err := l.Serialize(structured.EmptyFieldPath, &structured.YAMLNodeSerializer{})
				if err != nil {
					t.Fatalf("failed to serialize to yaml error=%v", err)
				}
				gotLogsString = append(gotLogsString, string(yaml))
			}

			if diff := cmp.Diff(tt.wantLogsString, gotLogsString); diff != "" {
				t.Errorf("NewCloudLoggingFilterTask() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSetQueryInfo(t *testing.T) {
	t.Parallel()
	taskID := "task-foo"
	startTime := time.Date(2025, time.January, 1, 1, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, time.January, 1, 1, 1, 0, 0, time.UTC)
	description := &ListLogEntriesTaskDescription{
		QueryName: "query-foo",
	}
	baseLogFilter := "resource.type=gce_instance AND severity=ERROR"

	tests := []struct {
		desc                string
		logFilterIndex      int
		totalLogFilterCount int
		wantQuery           *inspectionmetadata.QueryItem
	}{{
		desc:                "single filter, no special name",
		logFilterIndex:      0,
		totalLogFilterCount: 1,
		wantQuery: &inspectionmetadata.QueryItem{
			Id:    taskID,
			Name:  "query-foo",
			Query: "resource.type=gce_instance AND severity=ERROR\ntimestamp >= \"2025-01-01T01:00:00+0000\"\ntimestamp <= \"2025-01-01T01:01:00+0000\"",
		},
	},
		{
			desc:                "multiple filters, first one",
			logFilterIndex:      0,
			totalLogFilterCount: 2,
			wantQuery: &inspectionmetadata.QueryItem{
				Id:    taskID,
				Name:  "query-foo-0",
				Query: "resource.type=gce_instance AND severity=ERROR\ntimestamp >= \"2025-01-01T01:00:00+0000\"\ntimestamp <= \"2025-01-01T01:01:00+0000\"",
			},
		},
		{
			desc:                "multiple filters, second one",
			logFilterIndex:      1,
			totalLogFilterCount: 2,
			wantQuery: &inspectionmetadata.QueryItem{
				Id:    taskID,
				Name:  "query-foo-1",
				Query: "resource.type=gce_instance AND severity=ERROR\ntimestamp >= \"2025-01-01T01:00:00+0000\"\ntimestamp <= \"2025-01-01T01:01:00+0000\"",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			setQueryInfo(ctx, taskID, baseLogFilter, tt.logFilterIndex, tt.totalLogFilterCount, startTime, endTime, description)

			metadata := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)
			errorMessageSet, found := typedmap.Get(metadata, inspectionmetadata.QueryMetadataKey)
			if !found {
				t.Fatalf("query metadata not found")
			}

			if diff := cmp.Diff(tt.wantQuery, errorMessageSet.Queries[0]); diff != "" {
				t.Errorf("setQueryInfo() mismatch (-want +got):\n%s", diff)
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

			metadata := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)
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

func TestFormatETA(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input time.Duration
		want  string
	}{
		{
			name:  "negative duration",
			input: -5 * time.Second,
			want:  "0s",
		},
		{
			name:  "zero duration",
			input: 0,
			want:  "0s",
		},
		{
			name:  "seconds under one minute",
			input: 45 * time.Second,
			want:  "45s",
		},
		{
			name:  "rounding seconds",
			input: 14*time.Second + 600*time.Millisecond,
			want:  "15s",
		},
		{
			name:  "exact one minute",
			input: 60 * time.Second,
			want:  "1m00s",
		},
		{
			name:  "minutes and single-digit seconds",
			input: 65 * time.Second,
			want:  "1m05s",
		},
		{
			name:  "minutes and double-digit seconds",
			input: 95 * time.Second,
			want:  "1m35s",
		},
		{
			name:  "exact one hour",
			input: 3600 * time.Second,
			want:  "1h00m",
		},
		{
			name:  "hours and single-digit minutes",
			input: 3600*time.Second + 5*time.Minute,
			want:  "1h05m",
		},
		{
			name:  "hours and double-digit minutes",
			input: 2*time.Hour + 30*time.Minute,
			want:  "2h30m",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := formatETA(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("formatETA() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCalculateETA(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		elapsed       time.Duration
		completeRatio float32
		want          string
	}{
		{
			name:          "warmup elapsed under 2 seconds",
			elapsed:       1500 * time.Millisecond,
			completeRatio: 0.5,
			want:          "--",
		},
		{
			name:          "warmup complete ratio too small",
			elapsed:       10 * time.Second,
			completeRatio: 0.003,
			want:          "--",
		},
		{
			name:          "zero complete ratio",
			elapsed:       10 * time.Second,
			completeRatio: 0,
			want:          "--",
		},
		{
			name:          "complete ratio 100%",
			elapsed:       10 * time.Second,
			completeRatio: 1.0,
			want:          "0s",
		},
		{
			name:          "complete ratio over 100%",
			elapsed:       10 * time.Second,
			completeRatio: 1.2,
			want:          "0s",
		},
		{
			name:          "normal remaining seconds",
			elapsed:       10 * time.Second,
			completeRatio: 0.2, // 10s * (0.8 / 0.2) = 40s
			want:          "40s",
		},
		{
			name:          "normal remaining minutes",
			elapsed:       30 * time.Second,
			completeRatio: 0.25, // 30s * (0.75 / 0.25) = 90s = 1m30s
			want:          "1m30s",
		},
		{
			name:          "normal remaining hours",
			elapsed:       10 * time.Minute,
			completeRatio: 0.1, // 10m * (0.9 / 0.1) = 90m = 1h30m
			want:          "1h30m",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := calculateETA(tc.elapsed, tc.completeRatio)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("calculateETA() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMonitorProgress(t *testing.T) {
	t.Parallel()

	progressDest := inspectionmetadata.NewTaskProgressMetadata("test-task")
	source := make(chan LogFetchProgress)
	taskStartTime := time.Now().Add(-10 * time.Second)
	baseLogCount := 100
	listCallIndex := 0
	allListCalls := 2

	var wg sync.WaitGroup
	monitorProgress(t.Context(), &wg, source, progressDest, taskStartTime, baseLogCount, listCallIndex, allListCalls)

	source <- LogFetchProgress{
		LogCount: 50,
		Progress: 0.5,
	}
	close(source)
	wg.Wait()

	wantRatio := float32(0+0.5) / float32(2) // 0.25
	if diff := cmp.Diff(wantRatio, progressDest.Percentage); diff != "" {
		t.Errorf("progressDest.Percentage mismatch (-want +got):\n%s", diff)
	}

	// 100 base + 50 current = 150 logs. Elapsed ~10s -> ~15 lps. Ratio 0.25 -> ETA ~30s.
	if !strings.HasPrefix(progressDest.Message, "150 logs fetched(") {
		t.Errorf("expected message to start with '150 logs fetched(', got %q", progressDest.Message)
	}
	if !strings.Contains(progressDest.Message, ", ETA ") {
		t.Errorf("expected message to contain ', ETA ', got %q", progressDest.Message)
	}
	if !strings.HasSuffix(progressDest.Message, ")[1/2]") {
		t.Errorf("expected message to end with ')[1/2]', got %q", progressDest.Message)
	}
}
