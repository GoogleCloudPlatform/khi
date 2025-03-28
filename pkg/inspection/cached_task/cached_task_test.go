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

package cached_task

import (
	"context"
	"testing"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	inspection_task_test "github.com/GoogleCloudPlatform/khi/pkg/inspection/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
	"github.com/google/go-cmp/cmp"
)

func TestCachedTask(t *testing.T) {
	prevValues := []PreviousTaskResult[string]{}
	testTaskID := taskid.NewDefaultImplementationID[string]("foo")
	task := NewCachedTask(testTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context, prevValue PreviousTaskResult[string]) (PreviousTaskResult[string], error) {
		prevValues = append(prevValues, prevValue)
		return PreviousTaskResult[string]{
			Value:            "foo",
			DependencyDigest: "foo",
		}, nil
	})

	ctx := inspection_task_test.WithDefaultTestInspectionTaskContext(context.Background())
	_, _, err := inspection_task_test.RunInspectionTask(ctx, task, inspection_task_interface.TaskModeRun, map[string]any{})
	if err != nil {
		t.Errorf("unexpected task error result %v", err)
	}
	_, _, err = inspection_task_test.RunInspectionTask(ctx, task, inspection_task_interface.TaskModeRun, map[string]any{})
	if err != nil {
		t.Errorf("unexpected task error result %v", err)
	}

	if diff := cmp.Diff(prevValues, []PreviousTaskResult[string]{
		{
			Value:            "",
			DependencyDigest: "",
		},
		{
			Value:            "foo",
			DependencyDigest: "foo",
		},
	}); diff != "" {
		t.Errorf("unexpected prevValues (-want +got):\n%s", diff)
	}
}
