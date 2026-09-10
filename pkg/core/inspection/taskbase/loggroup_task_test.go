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

package inspectiontaskbase

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

var pathGroupTestID = structured.CompileFieldPath("id")

func TestNwewLogGrouperTask(t *testing.T) {
	sourceLogs := []string{
		`id: foo`,
		`id: bar`,
		`id: qux`,
		`id: quux`,
	}
	testCases := []struct {
		name         string
		taskMode     inspectioncore_contract.InspectionTaskModeType
		logYamls     []string
		logGrouper   LogGrouperFunc
		resultLogIDs map[string][]string
	}{
		{
			name:     "should return an empty map for empty log input on task run mode",
			taskMode: inspectioncore_contract.TaskModeRun,
			logYamls: []string{},
			logGrouper: func(ctx context.Context, l *log.Log) string {
				return l.ReadStringOrDefault(pathGroupTestID, "unknown")[:1]
			},
			resultLogIDs: map[string][]string{},
		},
		{
			name:     "should group logs correctly based on the provided function on task run mode",
			taskMode: inspectioncore_contract.TaskModeRun,
			logYamls: sourceLogs,
			logGrouper: func(ctx context.Context, l *log.Log) string {
				return l.ReadStringOrDefault(pathGroupTestID, "unknown")[:1]
			},
			resultLogIDs: map[string][]string{
				"f": {"foo"},
				"b": {"bar"},
				"q": {"qux", "quux"},
			},
		},
		{
			name:     "should return an empty map on task dry run mode",
			taskMode: inspectioncore_contract.TaskModeDryRun,
			logYamls: sourceLogs,
			logGrouper: func(ctx context.Context, l *log.Log) string {
				return l.ReadStringOrDefault(pathGroupTestID, "unknown")[:1]
			},
			resultLogIDs: map[string][]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			logs := []*log.Log{}
			for _, logYaml := range tc.logYamls {
				logs = append(logs, mustNewLogFromYAML(t, ctx, logYaml))
			}

			testSourceTaskID := taskid.NewDefaultImplementationID[[]*log.Log]("source")
			testTaskID := taskid.NewDefaultImplementationID[LogGroupMap]("dest")
			task := NewLogGrouperTask(testTaskID, testSourceTaskID.Ref(), tc.logGrouper)
			result, _, err := inspectiontest.RunInspectionTask(ctx, task, tc.taskMode, map[string]any{}, tasktest.NewTaskDependencyValuePair(testSourceTaskID.Ref(), logs))
			if err != nil {
				t.Fatalf("RunInspectionTask returned an unexpected error: %v", err)
			}

			if len(result) != len(tc.resultLogIDs) {
				t.Fatalf("unexpected number of groups: got %d, want %d", len(result), len(tc.resultLogIDs))
			}

			for key, gotLogGroup := range result {
				wantLogIDs, groupFound := tc.resultLogIDs[key]
				if !groupFound {
					t.Fatalf("unexpected group key found: %q", key)
				}
				gotLogIDs := []string{}
				for _, l := range gotLogGroup.Logs {
					gotLogIDs = append(gotLogIDs, l.ReadStringOrDefault(pathGroupTestID, "unknown"))
				}
				if diff := cmp.Diff(wantLogIDs, gotLogIDs); diff != "" {
					t.Errorf("log IDs for group %q mismatch (-want +got):\n%s", key, diff)
				}
			}
		})
	}
}

func TestNewLogGrouperTaskWithDependencies(t *testing.T) {
	testCases := []struct {
		name       string
		extraValue string
		logYamls   []string
		wantGroups map[string][]string
	}{
		{
			name:       "accesses extra dependency within grouper",
			extraValue: "prefix-",
			logYamls: []string{
				`id: foo`,
				`id: bar`,
			},
			wantGroups: map[string][]string{
				"prefix-f": {"foo"},
				"prefix-b": {"bar"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			logs := []*log.Log{}
			for _, logYaml := range tc.logYamls {
				logs = append(logs, mustNewLogFromYAML(t, ctx, logYaml))
			}

			testSourceTaskID := taskid.NewDefaultImplementationID[[]*log.Log]("source")
			testExtraTaskID := taskid.NewDefaultImplementationID[string]("extra")
			testTaskID := taskid.NewDefaultImplementationID[LogGroupMap]("dest")

			task := NewLogGrouperTaskWithDependencies(
				testTaskID,
				testSourceTaskID.Ref(),
				[]coretask.Dependency{testExtraTaskID.Ref()},
				func(ctx context.Context, l *log.Log) string {
					extra := coretask.GetTaskResult(ctx, testExtraTaskID.Ref())
					return extra + l.ReadStringOrDefault(pathGroupTestID, "unknown")[:1]
				},
			)
			result, _, err := inspectiontest.RunInspectionTask(
				ctx,
				task,
				inspectioncore_contract.TaskModeRun,
				map[string]any{},
				tasktest.NewTaskDependencyValuePair(testSourceTaskID.Ref(), logs),
				tasktest.NewTaskDependencyValuePair(testExtraTaskID.Ref(), tc.extraValue),
			)
			if err != nil {
				t.Fatalf("RunInspectionTask returned an unexpected error: %v", err)
			}

			gotGroups := map[string][]string{}
			for key, group := range result {
				for _, l := range group.Logs {
					gotGroups[key] = append(gotGroups[key], l.ReadStringOrDefault(pathGroupTestID, "unknown"))
				}
			}
			if diff := cmp.Diff(tc.wantGroups, gotGroups); diff != "" {
				t.Errorf("grouped log IDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
