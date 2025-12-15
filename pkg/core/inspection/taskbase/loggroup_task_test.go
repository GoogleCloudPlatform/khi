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

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

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
				return l.ReadStringOrDefault("id", "unknown")[:1]
			},
			resultLogIDs: map[string][]string{},
		},
		{
			name:     "should group logs correctly based on the provided function on task run mode",
			taskMode: inspectioncore_contract.TaskModeRun,
			logYamls: sourceLogs,
			logGrouper: func(ctx context.Context, l *log.Log) string {
				return l.ReadStringOrDefault("id", "unknown")[:1]
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
				return l.ReadStringOrDefault("id", "unknown")[:1]
			},
			resultLogIDs: map[string][]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logs := []*log.Log{}
			for _, logYaml := range tc.logYamls {
				l, err := log.NewLogFromYAMLString(logYaml)
				if err != nil {
					t.Fatal(err.Error())
				}
				logs = append(logs, l)
			}

			testSourceTaskID := taskid.NewDefaultImplementationID[[]*log.Log]("source")
			testTaskID := taskid.NewDefaultImplementationID[LogGroupMap]("dest")
			task := NewLogGrouperTask(testTaskID, testSourceTaskID.Ref(), tc.logGrouper)

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
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
					gotLogIDs = append(gotLogIDs, l.ReadStringOrDefault("id", "unknown"))
				}
				if diff := cmp.Diff(wantLogIDs, gotLogIDs); diff != "" {
					t.Errorf("log IDs for group %q mismatch (-want +got):\n%s", key, diff)
				}
			}
		})
	}
}
