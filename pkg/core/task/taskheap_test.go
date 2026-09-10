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

package coretask

import (
	"container/heap"
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/google/go-cmp/cmp"
)

func createHeapTestTask(refID string, hash string) UntypedTask {
	return NewTask(
		taskid.NewImplementationID(taskid.NewTaskReference[any](refID), hash),
		nil,
		func(ctx context.Context) (any, error) {
			return nil, nil
		},
	)
}

func TestTaskMinHeap_TableDriven(t *testing.T) {
	testCases := []struct {
		name      string
		inputs    []UntypedTask
		wantOrder []string
	}{
		{
			name:      "empty heap",
			inputs:    []UntypedTask{},
			wantOrder: nil,
		},
		{
			name: "single element",
			inputs: []UntypedTask{
				createHeapTestTask("task-a", "default"),
			},
			wantOrder: []string{"task-a#default"},
		},
		{
			name: "multiple elements inserted out of order are popped lexicographically",
			inputs: []UntypedTask{
				createHeapTestTask("task-c", "v1"),
				createHeapTestTask("task-a", "v2"),
				createHeapTestTask("task-b", "v1"),
				createHeapTestTask("task-a", "v1"),
			},
			wantOrder: []string{
				"task-a#v1",
				"task-a#v2",
				"task-b#v1",
				"task-c#v1",
			},
		},
		{
			name: "tasks with reverse alphabetical order",
			inputs: []UntypedTask{
				createHeapTestTask("z", "1"),
				createHeapTestTask("y", "1"),
				createHeapTestTask("x", "1"),
				createHeapTestTask("w", "1"),
			},
			wantOrder: []string{
				"w#1",
				"x#1",
				"y#1",
				"z#1",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := &taskMinHeap{}
			heap.Init(h)

			for _, task := range tc.inputs {
				heap.Push(h, task)
			}

			if gotLen := h.Len(); gotLen != len(tc.inputs) {
				t.Errorf("taskMinHeap.Len() mismatch: got %d, want %d", gotLen, len(tc.inputs))
			}

			var gotOrder []string
			for h.Len() > 0 {
				popped := heap.Pop(h).(UntypedTask)
				gotOrder = append(gotOrder, popped.UntypedID().String())
			}

			if diff := cmp.Diff(tc.wantOrder, gotOrder); diff != "" {
				t.Errorf("taskMinHeap popped order mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTaskMinHeap_DirectMethods(t *testing.T) {
	t1 := createHeapTestTask("task-1", "hash-a")
	t2 := createHeapTestTask("task-2", "hash-b")

	h := taskMinHeap{t1, t2}

	if !h.Less(0, 1) {
		t.Errorf("taskMinHeap.Less(0, 1) should be true for %s < %s", t1.UntypedID(), t2.UntypedID())
	}
	if h.Less(1, 0) {
		t.Errorf("taskMinHeap.Less(1, 0) should be false for %s < %s", t1.UntypedID(), t2.UntypedID())
	}

	h.Swap(0, 1)
	if h[0].UntypedID().String() != t2.UntypedID().String() {
		t.Errorf("taskMinHeap.Swap() failed to swap elements")
	}

	// Verify Pop() zeros out the popped element in the backing array to avoid memory leaks.
	rawSlice := make([]UntypedTask, 2)
	rawSlice[0] = t1
	rawSlice[1] = t2
	hPtr := (*taskMinHeap)(&rawSlice)
	popped := hPtr.Pop()
	if popped != t2 {
		t.Errorf("taskMinHeap.Pop() returned %v, want %v", popped, t2)
	}
	if rawSlice[:2][1] != nil {
		t.Errorf("taskMinHeap.Pop() did not clear backing array slot at index 1")
	}
}
