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
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
	"github.com/google/go-cmp/cmp"
)

func TestDefine_DeclaresInputsAsDependencies(t *testing.T) {
	requiredRef := taskid.NewTaskReference[string]("define-test.required")
	optionalRef := taskid.NewTaskReference[int]("define-test.optional", taskid.ScopeActiveGraph)
	orderingRef := taskid.NewTaskReference[struct{}]("define-test.ordering")
	tag := NewTag[string]("define-test-tag")

	task := Define(taskid.NewDefaultImplementationID[string]("define-test.task"), func(b *Binder) func(ctx context.Context) (string, error) {
		required := Use(b, requiredRef)
		optional := UseOptional(b, optionalRef)
		tagged := UseTag(b, tag.Ref())
		After(b, orderingRef)
		return func(ctx context.Context) (string, error) {
			value, _ := optional.Get(ctx)
			return fmt.Sprint(required.Get(ctx), value, tagged.Get(ctx)), nil
		}
	})

	wantKeys := []string{"ref:define-test.required", "ref:define-test.optional", "tag:define-test-tag", "ref:define-test.ordering"}
	gotKeys := make([]string, 0, len(task.Dependencies()))
	for _, dep := range task.Dependencies() {
		gotKeys = append(gotKeys, dependencyKey(dep))
	}
	if diff := cmp.Diff(wantKeys, gotKeys); diff != "" {
		t.Errorf("Dependencies() mismatch (-want +got):\n%s", diff)
	}

	wantInputs := []string{
		"required ref:define-test.required",
		"optional ref:define-test.optional",
		"tag tag:define-test-tag",
		"ordering ref:define-test.ordering",
	}
	if diff := cmp.Diff(wantInputs, describeInputSpecs(task.Inputs())); diff != "" {
		t.Errorf("Inputs() mismatch (-want +got):\n%s", diff)
	}
}

func TestDefine_RejectsMisuseInRunFunction(t *testing.T) {
	taskID := taskid.NewDefaultImplementationID[string]("define-test.misuse")
	lateRef := taskid.NewTaskReference[string]("define-test.late")

	testCases := []struct {
		name      string
		run       func(ctx context.Context, b *Binder)
		wantPanic string
	}{
		{
			name:      "declaring an input after bind returned panics",
			run:       func(ctx context.Context, b *Binder) { Use(b, lateRef) },
			wantPanic: "declares input ref:define-test.late after its bind function returned",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			task := Define(taskID, func(b *Binder) func(ctx context.Context) (string, error) {
				return func(ctx context.Context) (string, error) {
					tc.run(ctx, b)
					return "", nil
				}
			})
			ctx := khictx.WithValue[taskid.UntypedTaskImplementationID](context.Background(), core_contract.TaskImplementationIDContextKey, taskID)

			gotPanic := panicMessage(func() { _, _ = task.Run(ctx) })
			if !strings.Contains(gotPanic, tc.wantPanic) {
				t.Errorf("Run() panic = %q, want substring %q", gotPanic, tc.wantPanic)
			}
		})
	}
}

func TestDefine_RunsAlongsideLegacyTasks(t *testing.T) {
	tag := NewTag[string]("define-test-runner-tag")
	legacyProducer := NewTask(taskid.NewDefaultImplementationID[string]("define-test.legacy-producer"), nil, func(ctx context.Context) (string, error) {
		return "legacy", nil
	})
	tagProducer := NewTask(taskid.NewDefaultImplementationID[string]("define-test.tag-producer"), nil, func(ctx context.Context) (string, error) {
		return "tagged", nil
	}, ProvidesTag(tag))
	orderingProducer := NewTask(taskid.NewDefaultImplementationID[struct{}]("define-test.ordering-producer"), nil, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, nil
	})
	bound := Define(taskid.NewDefaultImplementationID[string]("define-test.bound"), func(b *Binder) func(ctx context.Context) (string, error) {
		legacy := Use(b, taskid.NewTaskReference[string]("define-test.legacy-producer"))
		absent := UseOptional(b, taskid.NewTaskReference[string]("define-test.absent", taskid.ScopeActiveGraph))
		tagged := UseTag(b, tag.Ref())
		After(b, taskid.NewTaskReference[struct{}]("define-test.ordering-producer"))
		return func(ctx context.Context) (string, error) {
			absentValue, found := absent.Get(ctx)
			return fmt.Sprintf("%s|%q,%v|%v", legacy.Get(ctx), absentValue, found, tagged.Get(ctx)), nil
		}
	}, NewTaskResultRetentionLabel(true))
	definedConsumer := Define(taskid.NewDefaultImplementationID[string]("define-test.defined-consumer"), func(b *Binder) func(ctx context.Context) (string, error) {
		bound := Use(b, taskid.NewTaskReference[string]("define-test.bound"))
		return func(ctx context.Context) (string, error) {
			return "consumed:" + bound.Get(ctx), nil
		}
	}, NewTaskResultRetentionLabel(true))

	tasks := []UntypedTask{legacyProducer, tagProducer, orderingProducer, bound, definedConsumer}
	runnableSet, err := ResolveGraph(tasks, tasks, nil)
	if err != nil {
		t.Fatalf("ResolveGraph() returned unexpected error: %v", err)
	}
	runner, err := NewLocalRunner(runnableSet)
	if err != nil {
		t.Fatalf("NewLocalRunner() returned unexpected error: %v", err)
	}
	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	<-runner.Wait()
	if _, err := runner.Result(); err != nil {
		t.Fatalf("Result() returned unexpected error: %v", err)
	}

	testCases := []struct {
		name string
		ref  taskid.TaskReference[string]
		want string
	}{
		{
			name: "task defined with Define reads legacy, optional and tag inputs",
			ref:  taskid.NewTaskReference[string]("define-test.bound"),
			want: `legacy|"",false|[tagged]`,
		},
		{
			name: "task defined with Define reads the result of another task defined with Define",
			ref:  taskid.NewTaskReference[string]("define-test.defined-consumer"),
			want: `consumed:legacy|"",false|[tagged]`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, found := GetTaskResultFromLocalRunner(runner, tc.ref)
			if !found {
				t.Fatalf("GetTaskResultFromLocalRunner(%s) found no result", tc.ref.ReferenceIDString())
			}
			if got != tc.want {
				t.Errorf("GetTaskResultFromLocalRunner(%s) = %q, want %q", tc.ref.ReferenceIDString(), got, tc.want)
			}
		})
	}
}

func TestDefineTailTask(t *testing.T) {
	t.Run("declares dependencies and preserves descriptors", func(t *testing.T) {
		taskID := taskid.NewDefaultImplementationID[struct{}]("tail-test.task")
		deps := []Dependency{
			taskid.NewTaskReference[string]("tail-test.a"),
			taskid.NewTaskReference[int]("tail-test.b", taskid.ScopeActiveGraph),
			NewTag[string]("tail-test-tag").Ref(),
		}
		testLabelKey := typedmap.NewTypedKey[string]("tail-test-label")
		task := DefineTailTask(taskID, deps, labelOptFunc(func(labels *typedmap.TypedMap) {
			typedmap.Set(labels, testLabelKey, "tail-label-val")
		}))

		wantInputs := []string{
			"ordering ref:tail-test.a",
			"ordering ref:tail-test.b",
			"ordering tag:tail-test-tag",
		}
		if diff := cmp.Diff(wantInputs, describeInputSpecs(task.Inputs())); diff != "" {
			t.Errorf("Inputs() mismatch (-want +got):\n%s", diff)
		}

		if got := task.Dependencies()[1].DescriptorScope(); got != taskid.ScopeActiveGraph {
			t.Errorf("task.Dependencies()[1].DescriptorScope() = %v, want %v", got, taskid.ScopeActiveGraph)
		}

		val, ok := typedmap.Get(task.Labels(), testLabelKey)
		if !ok {
			t.Errorf("expected label to be present")
		} else if val != "tail-label-val" {
			t.Errorf("label value = %q, want %q", val, "tail-label-val")
		}

		ctx := khictx.WithValue[taskid.UntypedTaskImplementationID](context.Background(), core_contract.TaskImplementationIDContextKey, taskID)
		got, err := task.Run(ctx)
		if err != nil {
			t.Fatalf("task.Run() returned unexpected error: %v", err)
		}
		if got != (struct{}{}) {
			t.Errorf("task.Run() = %v, want %v", got, struct{}{})
		}
	})

	t.Run("panics on duplicate dependencies", func(t *testing.T) {
		taskID := taskid.NewDefaultImplementationID[struct{}]("tail-test.dup-task")
		dupRef := taskid.NewTaskReference[string]("tail-test.dup")
		gotPanic := panicMessage(func() {
			DefineTailTask(taskID, []Dependency{dupRef, dupRef})
		})
		wantSubstring := "declares input ref:tail-test.dup twice"
		if !strings.Contains(gotPanic, wantSubstring) {
			t.Errorf("DefineTailTask() panic = %q, want substring %q", gotPanic, wantSubstring)
		}
	})
}
