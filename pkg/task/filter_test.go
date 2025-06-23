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

package task

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
	"github.com/google/go-cmp/cmp"
)

func TestSubset(t *testing.T) {
	colorKey := NewTaskLabelKey[string]("color")
	enabledKey := NewTaskLabelKey[bool]("enabled")

	task1 := NewTask(taskid.NewDefaultImplementationID[any]("task1"), nil, func(ctx context.Context) (any, error) { return nil, nil })
	labels1 := typedmap.NewTypedMap()
	typedmap.Set(labels1, colorKey, "red")
	typedmap.Set(labels1, enabledKey, true)
	task1.labels = labels1.AsReadonly()

	task2 := NewTask(taskid.NewDefaultImplementationID[any]("task2"), nil, func(ctx context.Context) (any, error) { return nil, nil })
	labels2 := typedmap.NewTypedMap()
	typedmap.Set(labels2, colorKey, "blue")
	typedmap.Set(labels2, enabledKey, false)
	task2.labels = labels2.AsReadonly()

	task3 := NewTask(taskid.NewDefaultImplementationID[any]("task3"), nil, func(ctx context.Context) (any, error) { return nil, nil })
	labels3 := typedmap.NewTypedMap()
	typedmap.Set(labels3, colorKey, "red")
	typedmap.Set(labels3, enabledKey, true)
	task3.labels = labels3.AsReadonly()

	task4 := NewTask(taskid.NewDefaultImplementationID[any]("task4"), nil, func(ctx context.Context) (any, error) { return nil, nil })

	taskSet, _ := NewTaskSet([]UntypedTask{task1, task2, task3, task4})

	t.Run("Filter by color", func(t *testing.T) {
		t.Run("includeIfMissing=false", func(t *testing.T) {
			predicate := WhereLabelEquals(colorKey, "red", false)
			result := Subset(taskSet, predicate)
			expected, _ := NewTaskSet([]UntypedTask{task1, task3})
			if diff := cmp.Diff(getTaskIDs(expected.GetAll()), getTaskIDs(result.GetAll())); diff != "" {
				t.Errorf("Subset() mismatch (-want +got):\n%s", diff)
			}
		})
		t.Run("includeIfMissing=true", func(t *testing.T) {
			predicate := WhereLabelEquals(colorKey, "red", true)
			result := Subset(taskSet, predicate)
			expected, _ := NewTaskSet([]UntypedTask{task1, task3, task4})
			if diff := cmp.Diff(getTaskIDs(expected.GetAll()), getTaskIDs(result.GetAll())); diff != "" {
				t.Errorf("Subset() mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("Filter by enabled", func(t *testing.T) {
		t.Run("includeIfMissing=false", func(t *testing.T) {
			predicate := WhereLabelIsEnabled(enabledKey, false)
			result := Subset(taskSet, predicate)
			expected, _ := NewTaskSet([]UntypedTask{task1, task3})
			if diff := cmp.Diff(getTaskIDs(expected.GetAll()), getTaskIDs(result.GetAll())); diff != "" {
				t.Errorf("Subset() mismatch (-want +got):\n%s", diff)
			}
		})
		t.Run("includeIfMissing=true", func(t *testing.T) {
			predicate := WhereLabelIsEnabled(enabledKey, true)
			result := Subset(taskSet, predicate)
			expected, _ := NewTaskSet([]UntypedTask{task1, task3, task4})
			if diff := cmp.Diff(getTaskIDs(expected.GetAll()), getTaskIDs(result.GetAll())); diff != "" {
				t.Errorf("Subset() mismatch (-want +got):\n%s", diff)
			}
		})
	})
}

func getTaskIDs(tasks []UntypedTask) []string {
	ids := make([]string, len(tasks))
	for i, task := range tasks {
		ids[i] = task.UntypedID().String()
	}
	return ids
}
