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

func TestDefine_RunsInGraph(t *testing.T) {
	tag := NewTag[string]("define-test-runner-tag")
	producer := DefineConstant(taskid.NewDefaultImplementationID[string]("define-test.producer"), "produced")
	tagProducer := DefineConstant(taskid.NewDefaultImplementationID[string]("define-test.tag-producer"), "tagged", ProvidesTag(tag))
	orderingProducer := DefineConstant(taskid.NewDefaultImplementationID[struct{}]("define-test.ordering-producer"), struct{}{})
	multiInputConsumer := Define(taskid.NewDefaultImplementationID[string]("define-test.multi-input-consumer"), func(b *Binder) func(ctx context.Context) (string, error) {
		producerInput := Use(b, taskid.NewTaskReference[string]("define-test.producer"))
		absent := UseOptional(b, taskid.NewTaskReference[string]("define-test.absent", taskid.ScopeActiveGraph))
		tagged := UseTag(b, tag.Ref())
		After(b, taskid.NewTaskReference[struct{}]("define-test.ordering-producer"))
		return func(ctx context.Context) (string, error) {
			absentValue, found := absent.Get(ctx)
			return fmt.Sprintf("%s|%q,%v|%v", producerInput.Get(ctx), absentValue, found, tagged.Get(ctx)), nil
		}
	}, NewTaskResultRetentionLabel(true))
	chainedConsumer := Define(taskid.NewDefaultImplementationID[string]("define-test.chained-consumer"), func(b *Binder) func(ctx context.Context) (string, error) {
		multiInputConsumerInput := Use(b, taskid.NewTaskReference[string]("define-test.multi-input-consumer"))
		return func(ctx context.Context) (string, error) {
			return "consumed:" + multiInputConsumerInput.Get(ctx), nil
		}
	}, NewTaskResultRetentionLabel(true))

	tasks := []UntypedTask{producer, tagProducer, orderingProducer, multiInputConsumer, chainedConsumer}
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
			name: "reads required, optional and tag inputs",
			ref:  taskid.NewTaskReference[string]("define-test.multi-input-consumer"),
			want: `produced|"",false|[tagged]`,
		},
		{
			name: "reads the result of another task defined with inputs",
			ref:  taskid.NewTaskReference[string]("define-test.chained-consumer"),
			want: `consumed:produced|"",false|[tagged]`,
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

func TestDefine_RunsTasksWithMergedInputs(t *testing.T) {
	sharedRef := taskid.NewTaskReference[string]("define-test.shared")
	shared := DefineConstant(taskid.NewDefaultImplementationID[string]("define-test.shared"), "shared")
	wrappedBind := func(b *Binder) func(ctx context.Context) (string, error) {
		sharedInput := Use(b, sharedRef)
		return func(ctx context.Context) (string, error) {
			return "wrapped:" + sharedInput.Get(ctx), nil
		}
	}
	wrapper := Define(taskid.NewDefaultImplementationID[string]("define-test.wrapper"), func(b *Binder) func(ctx context.Context) (string, error) {
		sharedInput := Use(b, sharedRef)
		wrappedRun := wrappedBind(b)
		return func(ctx context.Context) (string, error) {
			wrapped, err := wrappedRun(ctx)
			if err != nil {
				return "", err
			}
			return sharedInput.Get(ctx) + "|" + wrapped, nil
		}
	}, NewTaskResultRetentionLabel(true))
	scopeAllOrderingConsumer := Define(taskid.NewDefaultImplementationID[string]("define-test.scope-all-ordering-consumer"), func(b *Binder) func(ctx context.Context) (string, error) {
		sharedInput := UseOptional(b, sharedRef.Ref(taskid.ScopeActiveGraph))
		After(b, sharedRef)
		return func(ctx context.Context) (string, error) {
			value, found := sharedInput.Get(ctx)
			return fmt.Sprintf("%s,%t", value, found), nil
		}
	}, NewTaskResultRetentionLabel(true))

	tasks := []UntypedTask{shared, wrapper, scopeAllOrderingConsumer}
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
		name       string
		task       DefinedTask[string]
		wantInputs []string
		want       string
	}{
		{
			name:       "wrapper and wrapped bind both read an input they declared",
			task:       wrapper,
			wantInputs: []string{"required ref:define-test.shared"},
			want:       "shared|wrapped:shared",
		},
		{
			name:       "optional input merged with a ScopeAll ordering dependency reads the produced value",
			task:       scopeAllOrderingConsumer,
			wantInputs: []string{"required ref:define-test.shared"},
			want:       "shared,true",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if diff := cmp.Diff(tc.wantInputs, describeInputSpecs(tc.task.Inputs())); diff != "" {
				t.Errorf("Inputs() mismatch (-want +got):\n%s", diff)
			}
			got, found := GetTaskResultFromLocalRunner(runner, tc.task.ID().Ref())
			if !found {
				t.Fatalf("GetTaskResultFromLocalRunner(%s) found no result", tc.task.ID())
			}
			if got != tc.want {
				t.Errorf("GetTaskResultFromLocalRunner(%s) = %q, want %q", tc.task.ID(), got, tc.want)
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
			t.Fatalf("Inputs() mismatch (-want +got):\n%s", diff)
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

	t.Run("merges duplicate dependencies", func(t *testing.T) {
		dupRef := taskid.NewTaskReference[string]("tail-test.dup")
		task := DefineTailTask(taskid.NewDefaultImplementationID[struct{}]("tail-test.dup-task"), []Dependency{dupRef, dupRef})
		want := []string{"ordering ref:tail-test.dup"}
		if diff := cmp.Diff(want, describeInputSpecs(task.Inputs())); diff != "" {
			t.Errorf("Inputs() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("panics with the task ID on dependencies with conflicting result types", func(t *testing.T) {
		taskID := taskid.NewDefaultImplementationID[struct{}]("tail-test.conflict-task")
		gotPanic := panicMessage(func() {
			DefineTailTask(taskID, []Dependency{
				taskid.NewTaskReference[string]("tail-test.conflict"),
				taskid.NewTaskReference[int]("tail-test.conflict"),
			})
		})
		want := "task tail-test.conflict-task#default: declares input ref:tail-test.conflict with conflicting result types string and int"
		if gotPanic != want {
			t.Errorf("DefineTailTask() panic = %q, want %q", gotPanic, want)
		}
	})

	t.Run("accepts empty dependencies", func(t *testing.T) {
		task := DefineTailTask(taskid.NewDefaultImplementationID[struct{}]("tail-test.empty"), nil)
		if got := len(task.Dependencies()); got != 0 {
			t.Errorf("len(task.Dependencies()) = %d, want 0", got)
		}
	})
}

func TestDefine_BuildsTask(t *testing.T) {
	taskID := taskid.NewDefaultImplementationID[string]("task.test")
	depA := taskid.NewTaskReference[string]("task.a")
	tag := NewTag[int]("tag.test")
	testLabelKey := NewTaskLabelKey[string]("test-label")
	expectedErr := errors.New("execution failure")

	testCases := []struct {
		name            string
		taskID          taskid.TaskImplementationID[string]
		deps            []Dependency
		labelOpts       []LabelOpt
		runErr          error
		bindPanic       error
		wantLabelVal    string
		wantProvidedTag string
		wantPanic       string
	}{
		{
			name:   "applies label options",
			taskID: taskID,
			labelOpts: []LabelOpt{
				labelOptFunc(func(labels *typedmap.TypedMap) {
					typedmap.Set(labels, testLabelKey, "foo")
				}),
			},
			wantLabelVal: "foo",
		},
		{
			name:            "applies ProvidesTag label option",
			taskID:          taskID,
			labelOpts:       []LabelOpt{ProvidesTag(tag)},
			wantProvidedTag: "tag.test",
		},
		{
			name:   "propagates error from run function",
			taskID: taskID,
			runErr: expectedErr,
		},
		{
			name:      "panics when taskID is nil",
			taskID:    nil,
			wantPanic: "Invalid taskID",
		},
		{
			name:      "panics with the task ID when a declared input is nil",
			taskID:    taskID,
			deps:      []Dependency{depA, nil},
			wantPanic: "task task.test#default: unsupported dependency <nil>",
		},
		{
			name:      "panics with the task ID when user code in bind panics",
			taskID:    taskID,
			bindPanic: errors.New("boom"),
			wantPanic: "task task.test#default: boom",
		},
		{
			name:   "panics when label contains empty key",
			taskID: taskID,
			labelOpts: []LabelOpt{
				labelOptFunc(func(labels *typedmap.TypedMap) {
					typedmap.Set(labels, typedmap.NewTypedKey[string](""), "empty")
				}),
			},
			wantPanic: "contains an empty key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			define := func() DefinedTask[string] {
				return Define(tc.taskID, func(b *Binder) func(ctx context.Context) (string, error) {
					for _, dep := range tc.deps {
						After(b, dep)
					}
					if tc.bindPanic != nil {
						panic(tc.bindPanic)
					}
					return func(ctx context.Context) (string, error) {
						if tc.runErr != nil {
							return "", tc.runErr
						}
						return "result", nil
					}
				}, tc.labelOpts...)
			}
			if tc.wantPanic != "" {
				gotPanic := panicMessage(func() { define() })
				if !strings.Contains(gotPanic, tc.wantPanic) {
					t.Errorf("Define() panic = %q, want substring %q", gotPanic, tc.wantPanic)
				}
				return
			}

			task := define()
			if got := task.ID().String(); got != tc.taskID.String() {
				t.Errorf("task.ID() = %q, want %q", got, tc.taskID.String())
			}
			if got := task.UntypedID().String(); got != tc.taskID.String() {
				t.Errorf("task.UntypedID() = %q, want %q", got, tc.taskID.String())
			}
			if tc.wantLabelVal != "" {
				val, ok := typedmap.Get(task.Labels(), testLabelKey)
				if !ok || val != tc.wantLabelVal {
					t.Errorf("test label = %q (found: %v), want %q", val, ok, tc.wantLabelVal)
				}
			}
			if tc.wantProvidedTag != "" {
				val, ok := typedmap.Get(task.Labels(), LabelKeyProvidedTag(tc.wantProvidedTag))
				if !ok || !val {
					t.Errorf("provided tag label %s = %v (found: %v), want true", tc.wantProvidedTag, val, ok)
				}
			}

			ctx := khictx.WithValue[taskid.UntypedTaskImplementationID](t.Context(), core_contract.TaskImplementationIDContextKey, taskID)
			res, err := task.Run(ctx)
			untypedRes, untypedErr := task.UntypedRun(ctx)
			if tc.runErr != nil {
				if !errors.Is(err, tc.runErr) {
					t.Errorf("task.Run() error = %v, want %v", err, tc.runErr)
				}
				if !errors.Is(untypedErr, tc.runErr) {
					t.Errorf("task.UntypedRun() error = %v, want %v", untypedErr, tc.runErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("task.Run() returned unexpected error: %v", err)
			}
			if res != "result" {
				t.Errorf("task.Run() = %q, want %q", res, "result")
			}
			if untypedErr != nil {
				t.Fatalf("task.UntypedRun() returned unexpected error: %v", untypedErr)
			}
			if untypedRes != "result" {
				t.Errorf("task.UntypedRun() = %v, want %q", untypedRes, "result")
			}
		})
	}
}

type labelOptFunc func(labels *typedmap.TypedMap)

func (f labelOptFunc) Write(labels *typedmap.TypedMap) {
	f(labels)
}
