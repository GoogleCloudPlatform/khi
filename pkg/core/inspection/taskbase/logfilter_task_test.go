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

func TestNewLogFilterTask(t *testing.T) {
	sourceLogs := []string{
		`id: foo`,
		`id: bar`,
		`id: qux`,
	}
	testCases := []struct {
		name         string
		taskMode     inspectioncore_contract.InspectionTaskModeType
		logYAMLs     []string
		logFilter    LogFilterFunc
		resultLogIDs []string
	}{
		{
			name:         "should return an empty slice for an empty log input on run mode",
			taskMode:     inspectioncore_contract.TaskModeRun,
			logYAMLs:     []string{},
			resultLogIDs: []string{},
		},
		{
			name:     "should filter logs based on the provided function on run mode",
			taskMode: inspectioncore_contract.TaskModeRun,
			logYAMLs: sourceLogs,
			logFilter: func(ctx context.Context, l *log.Log) bool {
				id := l.ReadStringOrDefault("id", "unknown")
				return id == "foo" || id == "qux"
			},
			resultLogIDs: []string{"foo", "qux"},
		},
		{
			name:     "should return an empty slice and perform no filtering for dryrun mode",
			taskMode: inspectioncore_contract.TaskModeDryRun,
			logFilter: func(ctx context.Context, l *log.Log) bool {
				return true
			},
			resultLogIDs: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logs := []*log.Log{}
			for _, logYaml := range tc.logYAMLs {
				l, err := log.NewLogFromYAMLString(logYaml)
				if err != nil {
					t.Fatal(err.Error())
				}
				logs = append(logs, l)
			}

			testSourceTaskID := taskid.NewDefaultImplementationID[[]*log.Log]("source")
			testTaskID := taskid.NewDefaultImplementationID[[]*log.Log]("dest")
			task := NewLogFilterTask(testTaskID, testSourceTaskID.Ref(), tc.logFilter)

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			result, _, err := inspectiontest.RunInspectionTask(ctx, task, tc.taskMode, map[string]any{}, tasktest.NewTaskDependencyValuePair(testSourceTaskID.Ref(), logs))
			if err != nil {
				t.Fatalf("RunInspectionTask returned an unexpected error: %v", err)
			}

			logIDs := []string{}
			for _, resultLog := range result {
				logIDs = append(logIDs, resultLog.ReadStringOrDefault("id", "unknown"))
			}

			if diff := cmp.Diff(tc.resultLogIDs, logIDs); diff != "" {
				t.Errorf("Log IDs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
