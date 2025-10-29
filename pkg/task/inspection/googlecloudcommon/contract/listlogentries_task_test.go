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
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
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
		QueryName:      "query-foo",
		ExampleQuery:   "resource.type=gce_instance AND severity=ERROR",
		DefaultLogType: enum.LogTypeContainer,
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
			gotTargetLogType, found := typedmap.Get(task.Labels(), inspectioncore_contract.TaskLabelKeyQueryTaskTargetLogType)
			if !found {
				t.Errorf("targetLogType label not found")
			}
			if gotTargetLogType != description.DefaultLogType {
				t.Errorf("targetLogType label is not %v", description.DefaultLogType)
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
				yaml, err := l.Serialize("", &structured.YAMLNodeSerializer{})
				if err != nil {
					t.Fatalf("failed to serialize to yaml error=%v", err)
				}
				gotLogsString = append(gotLogsString, string(yaml))
				if l.LogType != description.DefaultLogType {
					t.Errorf("log type mismatch: got %v, want %v", l.LogType, description.DefaultLogType)
				}
				_, err = log.GetFieldSet(l, &log.CommonFieldSet{})
				if err != nil {
					t.Errorf("CommonFieldSet is not set for a log entry: %v", err)
				}
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
