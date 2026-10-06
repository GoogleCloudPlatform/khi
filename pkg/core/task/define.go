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
	"reflect"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

type taskImpl[T any] struct {
	id     taskid.TaskImplementationID[T]
	labels *typedmap.ReadonlyTypedMap
	inputs []InputSpec
	run    func(ctx context.Context) (T, error)
}

var _ Task[any] = (*taskImpl[any])(nil)

// ID implements Task.
func (t *taskImpl[T]) ID() taskid.TaskImplementationID[T] {
	return t.id
}

// UntypedID implements UntypedTask.
func (t *taskImpl[T]) UntypedID() taskid.UntypedTaskImplementationID {
	return t.id
}

// Labels implements UntypedTask.
func (t *taskImpl[T]) Labels() *typedmap.ReadonlyTypedMap {
	return t.labels
}

// Inputs implements UntypedTask.
func (t *taskImpl[T]) Inputs() []InputSpec {
	return t.inputs
}

// ResultType implements UntypedTask.
func (t *taskImpl[T]) ResultType() reflect.Type {
	return reflect.TypeFor[T]()
}

// Run implements Task. It marks ctx with the running task so that input handles can verify their owner.
func (t *taskImpl[T]) Run(ctx context.Context) (T, error) {
	return t.run(withActiveTask(ctx, t.id))
}

// UntypedRun implements UntypedTask.
func (t *taskImpl[T]) UntypedRun(ctx context.Context) (any, error) {
	return t.Run(ctx)
}

// Define constructs a task whose dependencies are declared through a Binder.
// bind runs once when Define is called. It must declare every input with Use, UseOptional, UseTag, or After,
// and return the run function that reads the inputs through the returned handles.
// Keep handles in local variables of bind so that the compiler reports inputs that are declared but never read.
func Define[T any](
	id taskid.TaskImplementationID[T],
	bind func(b *Binder) func(ctx context.Context) (T, error),
	labelOpts ...LabelOpt,
) Task[T] {
	verifyTaskID(id)
	b := newBinder(id)
	run := callBind(id, b, bind)
	b.sealed = true
	labelOpts = append([]LabelOpt{WithLabelValue(LabelKeyTaskResultType, reflect.TypeFor[T]().String())}, labelOpts...)
	labels := NewLabelSet(labelOpts...)
	verifyLabelKeys(id, labels)
	return &taskImpl[T]{
		id:     id,
		labels: labels,
		inputs: b.specs,
		run:    run,
	}
}

// callBind calls bind and re-panics any panic raised in it with the task ID prepended.
// Bind functions run at package initialization, where the stack trace alone rarely tells which task failed.
func callBind[T any](id taskid.TaskImplementationID[T], b *Binder, bind func(b *Binder) func(ctx context.Context) (T, error)) func(ctx context.Context) (T, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Sprintf("task %s: %v", id, r))
		}
	}()
	return bind(b)
}

// DefineConstant constructs a task that has no inputs and always returns value.
// The same value is returned on every run, so callers must not modify reference types such as slices or maps in it.
func DefineConstant[T any](id taskid.TaskImplementationID[T], value T, labelOpts ...LabelOpt) Task[T] {
	return Define(id, func(b *Binder) func(ctx context.Context) (T, error) {
		return func(ctx context.Context) (T, error) {
			return value, nil
		}
	}, labelOpts...)
}

// DefineTailTask constructs a no-op barrier task that waits for all dependencies without reading their values.
// Duplicate dependencies are merged into one input by the Binder.
func DefineTailTask(taskID taskid.TaskImplementationID[struct{}], dependencies []Dependency, labelOpts ...LabelOpt) Task[struct{}] {
	return Define(taskID, func(b *Binder) func(ctx context.Context) (struct{}, error) {
		for _, dep := range dependencies {
			After(b, dep)
		}
		return func(ctx context.Context) (struct{}, error) {
			return struct{}{}, nil
		}
	}, labelOpts...)
}
