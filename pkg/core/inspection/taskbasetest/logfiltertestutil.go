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

package inspectiontaskbasetest

import (
	"testing"

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// FilterTaskTestCase is a test case for testing a filter task.
// It contains the log fields to be filtered and the expected result.
type FilterTaskTestCase struct {
	Description  string
	LogFields    []log.FieldSet
	WantIncluded bool
}

// AssertFilterTask asserts that the given filter task behaves as expected for the given test cases.
// It runs the task with the given log fields and checks if the log is included or excluded from the result.
func AssertFilterTask(t *testing.T, task coretask.Task[[]*log.Log], sourceRef taskid.TaskReference[[]*log.Log], testCases []FilterTaskTestCase) {
	t.Helper()
	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(tc.LogFields...)
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())

			result, _, err := inspectiontest.RunInspectionTask(ctx, task, inspectioncore_contract.TaskModeRun, map[string]any{}, tasktest.NewTaskDependencyValuePair(sourceRef, []*log.Log{l}))
			if err != nil {
				t.Fatalf("RunInspectionTask failed: %v", err)
			}
			if tc.WantIncluded {
				if len(result) == 0 {
					t.Errorf("given log was unexpectedly filtered out. want=1, got=0")
				}
			} else {
				if len(result) != 0 {
					t.Errorf("given log was unexpectedly included. want=0, got=1")
				}
			}
		})
	}
}
