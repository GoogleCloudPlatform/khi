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
	"errors"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/google/go-cmp/cmp"
)

func TestNewTask(t *testing.T) {
	taskID := taskid.NewDefaultImplementationID[string]("task.test")
	depA := taskid.NewTaskReference[string]("task.a")
	depB := taskid.NewTaskReference[string]("task.b")
	tag := NewTag[int]("tag.test")
	testLabelKey := NewTaskLabelKey[string]("test-label")
	expectedErr := errors.New("execution failure")

	testCases := []struct {
		name            string
		taskID          taskid.TaskImplementationID[string]
		deps            []Dependency
		labelOpts       []LabelOpt
		runFunc         func(ctx context.Context) (string, error)
		wantDepCount    int
		wantLabelVal    string
		wantProvidedTag string
		wantErr         error
		shouldPanic     bool
		panicMatch      string
		verifyDeps      func(t *testing.T, deps []Dependency)
	}{
		{
			name:         "creates task with deduplicated p2p and tag dependencies",
			taskID:       taskID,
			deps:         []Dependency{depA, depB, depA, tag.Ref(), tag.Ref()},
			wantDepCount: 3,
			verifyDeps: func(t *testing.T, gotDeps []Dependency) {
				if len(gotDeps) != 3 {
					t.Fatalf("expected 3 dependencies, got %d", len(gotDeps))
				}
				p2p0, ok0 := gotDeps[0].(taskid.PointToPointDescriptor)
				p2p1, ok1 := gotDeps[1].(taskid.PointToPointDescriptor)
				fanIn2, ok2 := gotDeps[2].(taskid.FanInDescriptor)
				if !ok0 || p2p0.ReferenceID() != "task.a" {
					t.Errorf("dep[0] mismatch, want task.a, got %v", gotDeps[0])
				}
				if !ok1 || p2p1.ReferenceID() != "task.b" {
					t.Errorf("dep[1] mismatch, want task.b, got %v", gotDeps[1])
				}
				if !ok2 || fanIn2.Tag() != "tag.test" {
					t.Errorf("dep[2] mismatch, want tag.test, got %v", gotDeps[2])
				}
			},
		},
		{
			name:         "upgrades order-only dependency to data dependency when duplicated",
			taskID:       taskID,
			deps:         []Dependency{ToOrderOnly(depA), depA},
			wantDepCount: 1,
			verifyDeps: func(t *testing.T, gotDeps []Dependency) {
				if len(gotDeps) != 1 {
					t.Fatalf("expected 1 dependency, got %d", len(gotDeps))
				}
				if diff := cmp.Diff(taskid.EdgeKindData, gotDeps[0].DescriptorKind()); diff != "" {
					t.Errorf("dep[0].DescriptorKind() mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:         "preserves data dependency when followed by order-only duplicate",
			taskID:       taskID,
			deps:         []Dependency{depA, ToOrderOnly(depA)},
			wantDepCount: 1,
			verifyDeps: func(t *testing.T, gotDeps []Dependency) {
				if len(gotDeps) != 1 {
					t.Fatalf("expected 1 dependency, got %d", len(gotDeps))
				}
				if diff := cmp.Diff(taskid.EdgeKindData, gotDeps[0].DescriptorKind()); diff != "" {
					t.Errorf("dep[0].DescriptorKind() mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:         "upgrades optional dependency to required when duplicate is required",
			taskID:       taskID,
			deps:         []Dependency{taskid.NewTaskReference[string]("task.b", taskid.Optional), depB},
			wantDepCount: 1,
			verifyDeps: func(t *testing.T, gotDeps []Dependency) {
				if len(gotDeps) != 1 {
					t.Fatalf("expected 1 dependency, got %d", len(gotDeps))
				}
				if diff := cmp.Diff(taskid.ConditionRequired, gotDeps[0].DescriptorCondition()); diff != "" {
					t.Errorf("dep[0].DescriptorCondition() mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:         "preserves required dependency when followed by optional duplicate",
			taskID:       taskID,
			deps:         []Dependency{depB, taskid.NewTaskReference[string]("task.b", taskid.Optional)},
			wantDepCount: 1,
			verifyDeps: func(t *testing.T, gotDeps []Dependency) {
				if len(gotDeps) != 1 {
					t.Fatalf("expected 1 dependency, got %d", len(gotDeps))
				}
				if diff := cmp.Diff(taskid.ConditionRequired, gotDeps[0].DescriptorCondition()); diff != "" {
					t.Errorf("dep[0].DescriptorCondition() mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:         "upgrades order-only dependency to data dependency even when candidate is optional",
			taskID:       taskID,
			deps:         []Dependency{ToOrderOnly(depA), taskid.NewTaskReference[string]("task.a", taskid.Optional)},
			wantDepCount: 1,
			verifyDeps: func(t *testing.T, gotDeps []Dependency) {
				if len(gotDeps) != 1 {
					t.Fatalf("expected 1 dependency, got %d", len(gotDeps))
				}
				if diff := cmp.Diff(taskid.EdgeKindData, gotDeps[0].DescriptorKind()); diff != "" {
					t.Errorf("dep[0].DescriptorKind() mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:         "preserves data dependency when optional data is followed by required order-only",
			taskID:       taskID,
			deps:         []Dependency{taskid.NewTaskReference[string]("task.a", taskid.Optional), ToOrderOnly(depA)},
			wantDepCount: 1,
			verifyDeps: func(t *testing.T, gotDeps []Dependency) {
				if len(gotDeps) != 1 {
					t.Fatalf("expected 1 dependency, got %d", len(gotDeps))
				}
				if diff := cmp.Diff(taskid.EdgeKindData, gotDeps[0].DescriptorKind()); diff != "" {
					t.Errorf("dep[0].DescriptorKind() mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:         "upgrades order-only tag reference to data tag reference when duplicated",
			taskID:       taskID,
			deps:         []Dependency{tag.Ref(taskid.OrderOnly), tag.Ref()},
			wantDepCount: 1,
			verifyDeps: func(t *testing.T, gotDeps []Dependency) {
				if len(gotDeps) != 1 {
					t.Fatalf("expected 1 dependency, got %d", len(gotDeps))
				}
				if diff := cmp.Diff(taskid.EdgeKindData, gotDeps[0].DescriptorKind()); diff != "" {
					t.Errorf("dep[0].DescriptorKind() mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:         "preserves data tag reference when followed by order-only duplicate",
			taskID:       taskID,
			deps:         []Dependency{tag.Ref(), tag.Ref(taskid.OrderOnly)},
			wantDepCount: 1,
			verifyDeps: func(t *testing.T, gotDeps []Dependency) {
				if len(gotDeps) != 1 {
					t.Fatalf("expected 1 dependency, got %d", len(gotDeps))
				}
				if diff := cmp.Diff(taskid.EdgeKindData, gotDeps[0].DescriptorKind()); diff != "" {
					t.Errorf("dep[0].DescriptorKind() mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:   "creates task with valid label options",
			taskID: taskID,
			deps:   []Dependency{},
			labelOpts: []LabelOpt{
				labelOptFunc(func(labels *typedmap.TypedMap) {
					typedmap.Set(labels, testLabelKey, "foo")
				}),
			},
			wantLabelVal: "foo",
		},
		{
			name:            "creates task with ProvidesTag label option",
			taskID:          taskID,
			deps:            []Dependency{},
			labelOpts:       []LabelOpt{ProvidesTag(tag)},
			wantProvidedTag: "tag.test",
		},
		{
			name:   "propagates error from run function",
			taskID: taskID,
			deps:   []Dependency{},
			runFunc: func(ctx context.Context) (string, error) {
				return "", expectedErr
			},
			wantErr: expectedErr,
		},
		{
			name:        "panics when taskID is nil",
			taskID:      nil,
			deps:        []Dependency{},
			shouldPanic: true,
			panicMatch:  "Invalid taskID",
		},
		{
			name:        "panics when dependencies contains nil",
			taskID:      taskID,
			deps:        []Dependency{depA, nil},
			shouldPanic: true,
			panicMatch:  "contains a nil reference",
		},
		{
			name:   "panics when label contains empty key",
			taskID: taskID,
			deps:   []Dependency{},
			labelOpts: []LabelOpt{
				labelOptFunc(func(labels *typedmap.TypedMap) {
					typedmap.Set(labels, typedmap.NewTypedKey[string](""), "empty")
				}),
			},
			shouldPanic: true,
			panicMatch:  "contains an empty key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.shouldPanic {
				defer func() {
					r := recover()
					if r == nil {
						t.Errorf("expected panic containing %q, but none occurred", tc.panicMatch)
						return
					}
					msg := ""
					if err, ok := r.(error); ok {
						msg = err.Error()
					} else if s, ok := r.(string); ok {
						msg = s
					}
					if !strings.Contains(msg, tc.panicMatch) {
						t.Errorf("expected panic message to contain %q, got %q", tc.panicMatch, msg)
					}
				}()
			}

			runFunc := tc.runFunc
			if runFunc == nil {
				runFunc = func(ctx context.Context) (string, error) {
					return "result", nil
				}
			}

			task := NewTask(tc.taskID, tc.deps, runFunc, tc.labelOpts...)

			if !tc.shouldPanic {
				if diff := cmp.Diff(tc.taskID.String(), task.ID().String()); diff != "" {
					t.Errorf("task.ID() mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(tc.taskID.String(), task.UntypedID().String()); diff != "" {
					t.Errorf("task.UntypedID() mismatch (-want +got):\n%s", diff)
				}
				if tc.wantDepCount > 0 {
					if diff := cmp.Diff(tc.wantDepCount, len(task.Dependencies())); diff != "" {
						t.Errorf("len(task.Dependencies()) mismatch (-want +got):\n%s", diff)
					}
				}

				if tc.verifyDeps != nil {
					tc.verifyDeps(t, task.Dependencies())
				}

				if tc.wantLabelVal != "" {
					val, ok := typedmap.Get(task.Labels(), testLabelKey)
					if !ok || val != tc.wantLabelVal {
						t.Errorf("expected label %v, got %v (found: %v)", tc.wantLabelVal, val, ok)
					}
				}

				if tc.wantProvidedTag != "" {
					val, ok := typedmap.Get(task.Labels(), LabelKeyProvidedTag(tc.wantProvidedTag))
					if !ok || !val {
						t.Errorf("expected provided tag label %v, got %v (found: %v)", tc.wantProvidedTag, val, ok)
					}
				}

				res, err := task.Run(t.Context())
				if tc.wantErr != nil {
					if !errors.Is(err, tc.wantErr) {
						t.Errorf("task.Run() error mismatch: want %v, got %v", tc.wantErr, err)
					}
				} else {
					if err != nil {
						t.Fatalf("task.Run() unexpected error: %v", err)
					}
					if diff := cmp.Diff("result", res); diff != "" {
						t.Errorf("task.Run() mismatch (-want +got):\n%s", diff)
					}
				}

				untypedRes, err := task.UntypedRun(t.Context())
				if tc.wantErr != nil {
					if !errors.Is(err, tc.wantErr) {
						t.Errorf("task.UntypedRun() error mismatch: want %v, got %v", tc.wantErr, err)
					}
				} else {
					if err != nil {
						t.Fatalf("task.UntypedRun() unexpected error: %v", err)
					}
					if diff := cmp.Diff("result", untypedRes); diff != "" {
						t.Errorf("task.UntypedRun() mismatch (-want +got):\n%s", diff)
					}
				}
			}
		})
	}
}

type labelOptFunc func(labels *typedmap.TypedMap)

func (f labelOptFunc) Write(labels *typedmap.TypedMap) {
	f(labels)
}

func TestNewTailTask(t *testing.T) {
	tailID := taskid.NewDefaultImplementationID[struct{}]("tail.test")
	depA := taskid.NewTaskReference[string]("task.a")
	depB := taskid.NewTaskReference[string]("task.b", taskid.Optional)
	tag := NewTag[int]("tag.test")
	tailLabelKey := NewTaskLabelKey[string]("tail-label")

	testCases := []struct {
		name         string
		taskID       taskid.TaskImplementationID[struct{}]
		deps         []Dependency
		wantDepCount int
		shouldPanic  bool
		panicMatch   string
		verifyDeps   func(t *testing.T, deps []Dependency)
	}{
		{
			name:         "empty dependencies",
			taskID:       tailID,
			deps:         nil,
			wantDepCount: 0,
		},
		{
			name:         "converts all dependencies to order-only and preserves attributes",
			taskID:       tailID,
			deps:         []Dependency{depA, depB, tag.Ref()},
			wantDepCount: 3,
			verifyDeps: func(t *testing.T, deps []Dependency) {
				if len(deps) != 3 {
					t.Fatalf("expected 3 dependencies, got %d", len(deps))
				}
				for i, dep := range deps {
					if diff := cmp.Diff(taskid.EdgeKindOrderOnly, dep.DescriptorKind()); diff != "" {
						t.Errorf("dep[%d].DescriptorKind() mismatch (-want +got):\n%s", i, diff)
					}
				}
				// Verify depB preserves ConditionOptional
				if diff := cmp.Diff(taskid.ConditionOptional, deps[1].DescriptorCondition()); diff != "" {
					t.Errorf("dep[1].DescriptorCondition() mismatch (-want +got):\n%s", diff)
				}
				// Verify tag dependency preserves tag ID
				if fanIn, ok := deps[2].(taskid.FanInDescriptor); !ok || fanIn.Tag() != "tag.test" {
					t.Errorf("dep[2] tag mismatch, want tag.test, got %v", deps[2])
				}
			},
		},
		{
			name:         "deduplicates duplicate dependencies",
			taskID:       tailID,
			deps:         []Dependency{depA, depA},
			wantDepCount: 1,
			verifyDeps: func(t *testing.T, deps []Dependency) {
				if len(deps) != 1 {
					t.Fatalf("expected 1 dependency, got %d", len(deps))
				}
				if diff := cmp.Diff(taskid.EdgeKindOrderOnly, deps[0].DescriptorKind()); diff != "" {
					t.Errorf("dep[0].DescriptorKind() mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:        "panics when taskID is nil",
			taskID:      nil,
			deps:        []Dependency{},
			shouldPanic: true,
			panicMatch:  "Invalid taskID",
		},
		{
			name:        "panics when dependencies contains nil",
			taskID:      tailID,
			deps:        []Dependency{depA, nil},
			shouldPanic: true,
			panicMatch:  "contains a nil reference",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.shouldPanic {
				defer func() {
					r := recover()
					if r == nil {
						t.Errorf("expected panic containing %q, but none occurred", tc.panicMatch)
						return
					}
					msg := ""
					if err, ok := r.(error); ok {
						msg = err.Error()
					} else if s, ok := r.(string); ok {
						msg = s
					}
					if !strings.Contains(msg, tc.panicMatch) {
						t.Errorf("expected panic message to contain %q, got %q", tc.panicMatch, msg)
					}
				}()
			}

			tailTask := NewTailTask(tc.taskID, tc.deps, labelOptFunc(func(labels *typedmap.TypedMap) {
				typedmap.Set(labels, tailLabelKey, "tail-label-val")
			}))

			if !tc.shouldPanic {
				if diff := cmp.Diff(tailID.String(), tailTask.ID().String()); diff != "" {
					t.Errorf("tailTask.ID() mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(tailID.String(), tailTask.UntypedID().String()); diff != "" {
					t.Errorf("tailTask.UntypedID() mismatch (-want +got):\n%s", diff)
				}

				val, ok := typedmap.Get(tailTask.Labels(), tailLabelKey)
				if !ok || val != "tail-label-val" {
					t.Errorf("expected tail task label tail-label-val, got %v (found: %v)", val, ok)
				}

				if diff := cmp.Diff(tc.wantDepCount, len(tailTask.Dependencies())); diff != "" {
					t.Errorf("len(tailTask.Dependencies()) mismatch (-want +got):\n%s", diff)
				}

				if tc.verifyDeps != nil {
					tc.verifyDeps(t, tailTask.Dependencies())
				}

				res, err := tailTask.Run(t.Context())
				if err != nil {
					t.Fatalf("tailTask.Run() unexpected error: %v", err)
				}
				if diff := cmp.Diff(struct{}{}, res); diff != "" {
					t.Errorf("tailTask.Run() mismatch (-want +got):\n%s", diff)
				}

				untypedRes, err := tailTask.UntypedRun(t.Context())
				if err != nil {
					t.Fatalf("tailTask.UntypedRun() unexpected error: %v", err)
				}
				if diff := cmp.Diff(struct{}{}, untypedRes); diff != "" {
					t.Errorf("tailTask.UntypedRun() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
