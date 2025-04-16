// Copyright 2024 Google LLC
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

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

const (
	KHISystemPrefix = "khi.google.com/"
)

// KHI allows tasks with different ID suffixes to be specified as dependencies
// using only the ID without the suffix. For example, both `a.b.c/qux#foo` and `a.b.c/qux#bar`
// can be specified as a dependency using `a.b.c/qux`.
//
// Normally, the task ID is uniquely determined by the task filter or other
// ways. However, if multiple tasks exist, the value specified with this label
// with the highest priority is used.

var LabelKeyTaskSelectionPriority = NewTaskLabelKey[int](KHISystemPrefix + "task-selection-priority")

type UntypedTask interface {
	UntypedID() taskid.UntypedTaskImplementationID
	// Labels returns KHITaskLabelSet assigned to this task unit.
	// The implementation of this function must return a constant value.
	Labels() *typedmap.ReadonlyTypedMap

	// Dependencies returns the list of task references. Task runner will wait these dependent tasks to be done before running this task.
	Dependencies() []taskid.UntypedTaskReference

	UntypedRun(ctx context.Context) (any, error)
}

// Task is the fundamental interface that all of DAG nodes in KHI task system implements.
// The implementation of ID and Labels must be deterministic when the application started.
// The implementation of Sinks and Source must be pure function not depending anything outside of the argument.
type Task[TaskResult any] interface {
	UntypedTask
	// ID returns an unique TaskID of taskid.TaskImplementationID[TaskResult]
	// The implementation of this function must return a constant value.
	ID() taskid.TaskImplementationID[TaskResult]

	Run(ctx context.Context) (TaskResult, error)
}

type TaskImpl[TaskResult any] struct {
	id           taskid.TaskImplementationID[TaskResult]
	labels       *typedmap.ReadonlyTypedMap
	dependencies []taskid.UntypedTaskReference
	runFunc      func(ctx context.Context) (TaskResult, error)
}

// Run implements Task.
func (c *TaskImpl[TaskResult]) Run(ctx context.Context) (TaskResult, error) {
	return c.runFunc(ctx)
}

// Dependencies implements Task.
func (c *TaskImpl[TaskResult]) Dependencies() []taskid.UntypedTaskReference {
	return c.dependencies
}

// ID implements Task.
func (c *TaskImpl[TaskResult]) ID() taskid.TaskImplementationID[TaskResult] {
	return c.id
}

// Labels implements Task.
func (c *TaskImpl[TaskResult]) Labels() *typedmap.ReadonlyTypedMap {
	return c.labels
}

func (c *TaskImpl[TaskResult]) UntypedID() taskid.UntypedTaskImplementationID {
	return c.ID()
}

func (c *TaskImpl[TaskResult]) UntypedRun(ctx context.Context) (any, error) {
	return c.Run(ctx)
}

var _ Task[any] = (*TaskImpl[any])(nil)

func NewTask[TaskResult any](taskId taskid.TaskImplementationID[TaskResult], dependencies []taskid.UntypedTaskReference, runFunc func(ctx context.Context) (TaskResult, error), labelOpts ...LabelOpt) *TaskImpl[TaskResult] {
	labels := NewLabelSet(labelOpts...)
	return &TaskImpl[TaskResult]{
		id:           taskId,
		labels:       labels,
		dependencies: dedupeTaskReferences(dependencies),
		runFunc:      runFunc,
	}
}

func dedupeTaskReferences(reference []taskid.UntypedTaskReference) []taskid.UntypedTaskReference {
	result := []taskid.UntypedTaskReference{}
	seen := map[string]struct{}{}
	for _, ref := range reference {
		if _, ok := seen[ref.String()]; ok {
			continue
		}
		seen[ref.String()] = struct{}{}
		result = append(result, ref)
	}
	return result

}
