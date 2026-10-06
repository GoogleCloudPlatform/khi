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

package inspectiontaskbase

import (
	"context"
	"fmt"
	"testing"

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

func TestDefineInspectionTask(t *testing.T) {
	sourceRef := taskid.NewTaskReference[string]("define-inspection-test.source")
	task := DefineInspectionTask(taskid.NewDefaultImplementationID[string]("define-inspection-test.task"), func(b *coretask.Binder) InspectionTaskFunc[string] {
		source := coretask.Use(b, sourceRef)
		return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (string, error) {
			return fmt.Sprintf("%s:%d", source.Get(ctx), taskMode), nil
		}
	})

	testCases := []struct {
		name     string
		taskMode inspectioncore.InspectionTaskModeType
		want     string
	}{
		{
			name:     "passes the run mode and reads the declared input",
			taskMode: inspectioncore.TaskModeRun,
			want:     "source-value:2",
		},
		{
			name:     "passes the dry run mode and reads the declared input",
			taskMode: inspectioncore.TaskModeDryRun,
			want:     "source-value:1",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			got, _, err := inspectiontest.Run(t, ctx, task, tc.taskMode, map[string]any{}, tasktest.Given(sourceRef, "source-value"))
			if err != nil {
				t.Fatalf("Run() returned unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("Run() = %q, want %q", got, tc.want)
			}
		})
	}
}
