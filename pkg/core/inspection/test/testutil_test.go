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

package inspectiontest

import (
	"context"
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

func TestRun(t *testing.T) {
	sourceRef := taskid.NewTaskReference[string]("testutil-test.source")
	task := inspectiontaskbase.DefineInspectionTask(taskid.NewDefaultImplementationID[string]("testutil-test.task"), func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[string] {
		source := coretask.Use(b, sourceRef)
		return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (string, error) {
			input := khictx.MustGetValue(ctx, inspectioncore.InspectionTaskInput)
			return fmt.Sprintf("source=%s mode=%d form=%v", source.Get(ctx), taskMode, input["form"]), nil
		}
	})

	testCases := []struct {
		name string
		mode inspectioncore.InspectionTaskModeType
		want string
	}{
		{
			name: "run mode",
			mode: inspectioncore.TaskModeRun,
			want: fmt.Sprintf("source=foo mode=%d form=bar", inspectioncore.TaskModeRun),
		},
		{
			name: "dry run mode",
			mode: inspectioncore.TaskModeDryRun,
			want: fmt.Sprintf("source=foo mode=%d form=bar", inspectioncore.TaskModeDryRun),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := WithDefaultTestInspectionTaskContext(t.Context())
			wantMetadata := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)

			got, gotMetadata, err := Run(t, ctx, task, tc.mode, map[string]any{"form": "bar"}, tasktest.Given(sourceRef, "foo"))
			if err != nil {
				t.Fatalf("Run() returned unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("Run() = %q, want %q", got, tc.want)
			}
			if gotMetadata != wantMetadata {
				t.Errorf("Run() returned metadata %p, want the metadata in the context %p", gotMetadata, wantMetadata)
			}
		})
	}
}
