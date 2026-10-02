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

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

// DefinedTask is a task defined with Define. It exposes its typed input declarations to callers outside the task.
type DefinedTask[T any] interface {
	Task[T]
	// Inputs returns the inputs declared through the Binder in declaration order.
	Inputs() []InputSpec
}

type definedTaskImpl[T any] struct {
	*TaskImpl[T]
	inputs []InputSpec
}

var _ DefinedTask[any] = (*definedTaskImpl[any])(nil)

// Inputs implements DefinedTask.
func (t *definedTaskImpl[T]) Inputs() []InputSpec {
	return t.inputs
}

// Define constructs a task whose dependencies are declared through a Binder.
// bind runs once when Define is called. It must declare every input with Use, UseOptional, UseTag, or After,
// and return the run function that reads the inputs through the returned handles.
// Keep handles in local variables of bind so that the compiler reports inputs that are declared but never read.
func Define[T any](
	id taskid.TaskImplementationID[T],
	bind func(b *Binder) func(ctx context.Context) (T, error),
	labelOpts ...LabelOpt,
) DefinedTask[T] {
	verifyTaskID(id)
	b := newBinder(id)
	run := bind(b)
	b.sealed = true
	return &definedTaskImpl[T]{
		TaskImpl: NewTask(id, b.dependencies(), func(ctx context.Context) (T, error) {
			return run(withActiveDefinedTask(ctx, id))
		}, labelOpts...),
		inputs: b.specs,
	}
}

// DefineConstant constructs a task that has no inputs and always returns value.
// The same value is returned on every run, so callers must not modify reference types such as slices or maps in it.
func DefineConstant[T any](id taskid.TaskImplementationID[T], value T, labelOpts ...LabelOpt) DefinedTask[T] {
	return Define(id, func(b *Binder) func(ctx context.Context) (T, error) {
		return func(ctx context.Context) (T, error) {
			return value, nil
		}
	}, labelOpts...)
}

// DefineTailTask constructs a no-op barrier task that waits for all dependencies without reading their values.
// Each dependency must be unique because the Binder rejects inputs declared twice.
func DefineTailTask(taskID taskid.TaskImplementationID[struct{}], dependencies []Dependency, labelOpts ...LabelOpt) DefinedTask[struct{}] {
	return Define(taskID, func(b *Binder) func(ctx context.Context) (struct{}, error) {
		for _, dep := range dependencies {
			After(b, dep)
		}
		return func(ctx context.Context) (struct{}, error) {
			return struct{}{}, nil
		}
	}, labelOpts...)
}
