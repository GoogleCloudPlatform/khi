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
	"context"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
	"github.com/google/go-cmp/cmp"
)

var (
	inputTestOwnerID = taskid.NewDefaultImplementationID[string]("input-test.owner")
	inputTestOtherID = taskid.NewDefaultImplementationID[string]("input-test.other")
)

// newInputTestContext builds the context a runner passes to a task.
// When active is nil, the context does not mark any task defined with Define as running.
func newInputTestContext(active taskid.UntypedTaskImplementationID, results map[string]string, meta core_contract.TaskGraphMetadata) context.Context {
	taskResults := typedmap.NewTypedMap()
	for id, value := range results {
		typedmap.Set(taskResults, typedmap.NewTypedKey[string](id), value)
	}
	ctx := context.Background()
	ctx = khictx.WithValue[taskid.UntypedTaskImplementationID](ctx, core_contract.TaskImplementationIDContextKey, inputTestOwnerID)
	ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
	ctx = khictx.WithValue(ctx, core_contract.TaskGraphMetadataContextKey, meta)
	if active != nil {
		ctx = withActiveTask(ctx, active)
	}
	return ctx
}

func TestInput_Get(t *testing.T) {
	ref := taskid.NewTaskReference[string]("input-test.producer")

	testCases := []struct {
		name      string
		active    taskid.UntypedTaskImplementationID
		want      string
		wantPanic string
	}{
		{
			name:   "returns the producer result when read by the declaring task",
			active: inputTestOwnerID,
			want:   "value",
		},
		{
			name:      "panics when read outside of a task defined with Define",
			active:    nil,
			wantPanic: "is read outside of a task defined with Define",
		},
		{
			name:      "panics when read by another task",
			active:    inputTestOtherID,
			wantPanic: "is read from task input-test.other#default",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := Use(newBinder(inputTestOwnerID), ref)
			ctx := newInputTestContext(tc.active, map[string]string{"input-test.producer": "value"}, &mockGraphMetadata{})

			var got string
			gotPanic := panicMessage(func() { got = input.Get(ctx) })
			if tc.wantPanic != "" {
				if !strings.Contains(gotPanic, tc.wantPanic) {
					t.Fatalf("Input.Get() panic = %q, want substring %q", gotPanic, tc.wantPanic)
				}
				return
			}
			if gotPanic != "" {
				t.Fatalf("Input.Get() panicked unexpectedly: %s", gotPanic)
			}
			if got != tc.want {
				t.Errorf("Input.Get() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestOptionalInput_Get(t *testing.T) {
	ref := taskid.NewTaskReference[string]("input-test.producer", taskid.ScopeActiveGraph)

	testCases := []struct {
		name      string
		active    taskid.UntypedTaskImplementationID
		bound     bool
		want      string
		wantFound bool
		wantPanic string
	}{
		{
			name:      "returns the producer result when the producer is bound",
			active:    inputTestOwnerID,
			bound:     true,
			want:      "value",
			wantFound: true,
		},
		{
			name:      "returns the zero value and false when the producer is not bound",
			active:    inputTestOwnerID,
			bound:     false,
			want:      "",
			wantFound: false,
		},
		{
			name:      "panics when read outside of a task defined with Define",
			active:    nil,
			bound:     true,
			wantPanic: "is read outside of a task defined with Define",
		},
		{
			name:      "panics when read by another task",
			active:    inputTestOtherID,
			bound:     true,
			wantPanic: "is read from task input-test.other#default",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := UseOptional(newBinder(inputTestOwnerID), ref)
			results := map[string]string{}
			if tc.bound {
				results["input-test.producer"] = "value"
			}
			meta := &mockGraphMetadata{boundTasks: map[string]bool{"input-test.producer": tc.bound}}
			ctx := newInputTestContext(tc.active, results, meta)

			var got string
			var gotFound bool
			gotPanic := panicMessage(func() { got, gotFound = input.Get(ctx) })
			if tc.wantPanic != "" {
				if !strings.Contains(gotPanic, tc.wantPanic) {
					t.Fatalf("OptionalInput.Get() panic = %q, want substring %q", gotPanic, tc.wantPanic)
				}
				return
			}
			if gotPanic != "" {
				t.Fatalf("OptionalInput.Get() panicked unexpectedly: %s", gotPanic)
			}
			if got != tc.want || gotFound != tc.wantFound {
				t.Errorf("OptionalInput.Get() = (%q, %v), want (%q, %v)", got, gotFound, tc.want, tc.wantFound)
			}
		})
	}
}

func TestTagInput_Get(t *testing.T) {
	tag := NewTag[string]("input-test-tag")
	meta := &mockGraphMetadata{
		boundFanInRefIDsByTaskImplID: map[string]map[string][]string{
			inputTestOwnerID.String(): {tag.ID(): {"p1", "p2"}},
		},
	}

	testCases := []struct {
		name      string
		active    taskid.UntypedTaskImplementationID
		want      []string
		wantPanic string
	}{
		{
			name:   "returns the results of producers bound to the declaring task",
			active: inputTestOwnerID,
			want:   []string{"apple", "banana"},
		},
		{
			name:      "panics when read outside of a task defined with Define",
			active:    nil,
			wantPanic: "is read outside of a task defined with Define",
		},
		{
			name:      "panics when read by another task",
			active:    inputTestOtherID,
			wantPanic: "is read from task input-test.other#default",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := UseTag(newBinder(inputTestOwnerID), tag.Ref())
			ctx := newInputTestContext(tc.active, map[string]string{"p1": "apple", "p2": "banana"}, meta)

			var got []string
			gotPanic := panicMessage(func() { got = input.Get(ctx) })
			if tc.wantPanic != "" {
				if !strings.Contains(gotPanic, tc.wantPanic) {
					t.Fatalf("TagInput.Get() panic = %q, want substring %q", gotPanic, tc.wantPanic)
				}
				return
			}
			if gotPanic != "" {
				t.Fatalf("TagInput.Get() panicked unexpectedly: %s", gotPanic)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TagInput.Get() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
