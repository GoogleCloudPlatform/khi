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

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// GrouperTaskTestCase is a test case for log grouper task.
type GrouperTaskTestCase struct {
	Description string
	LogFields   []log.FieldSet
	WantGroup   string
}

// AssertGrouperTask asserts that the given grouper task behaves as expected for the given test cases.
func AssertGrouperTask(t *testing.T, task coretask.Task[inspectiontaskbase.LogGroupMap], sourceRef taskid.TaskReference[[]*log.Log], testCases []GrouperTaskTestCase) {
	t.Helper()
	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(tc.LogFields...)
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())

			result, _, err := inspectiontest.RunInspectionTask(ctx, task, inspectioncore_contract.TaskModeRun, map[string]any{}, tasktest.NewTaskDependencyValuePair(sourceRef, []*log.Log{l}))
			if err != nil {
				t.Fatalf("RunInspectionTask failed: %v", err)
			}
			if len(result) != 1 {
				t.Fatalf("unexpected element count found in result: want=1, got=%d", len(result))
			}
			for _, group := range result {
				if group.Group != tc.WantGroup {
					t.Fatalf("unexpected group found in result: want=%s, got=%s", tc.WantGroup, group.Group)
				}
			}
		})
	}
}
