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

type UntypedDefinition interface {
	UntypedID() taskid.UntypedTaskImplementationID
	// Labels returns KHITaskLabelSet assigned to this task unit.
	// The implementation of this function must return a constant value.
	Labels() *typedmap.ReadonlyTypedMap

	// Dependencies returns the set of Definition ids without the suffix beginning with #. Task runner will wait these dependent tasks to be done before running this task.
	Dependencies() []taskid.UntypedTaskReference

	UntypedRun(ctx context.Context) (any, error)
}

// Definition represents a task definition that behaves as a factory of the task runner itself and contains metadata of dependency and labels.
// The implementation of ID and Labels must be deterministic when the application started.
// The implementation of Sinks and Source must be pure function not depending anything outside of the argument.
type Definition[TaskResult any] interface {
	UntypedDefinition
	// ID returns a string unique for each Definition
	// The implementation of this function must return a constant value.
	// Each definition ID must be unique but ID can have suffix beginning with #. Dependencies field will ignore the suffix.
	// This is useful when task A depends on the value B but value B can be generated from task B-1 or B-2 and the task set would be differently generated by some condition.
	// In the case, task Ids would be `B#1` and `B#2` and these could be depnd by specifying `B` in the Dependencies.
	ID() taskid.TaskImplementationID[TaskResult]

	Run(ctx context.Context) (TaskResult, error)
}

type ConstantDefinitionImpl[TaskResult any] struct {
	id           taskid.TaskImplementationID[TaskResult]
	labels       *typedmap.ReadonlyTypedMap
	dependencies []taskid.UntypedTaskReference
	runFunc      func(ctx context.Context) (TaskResult, error)
}

// Run implements Definition.
func (c *ConstantDefinitionImpl[TaskResult]) Run(ctx context.Context) (TaskResult, error) {
	return c.runFunc(ctx)
}

// Dependencies implements Definition.
func (c *ConstantDefinitionImpl[TaskResult]) Dependencies() []taskid.UntypedTaskReference {
	return c.dependencies
}

// ID implements Definition.
func (c *ConstantDefinitionImpl[TaskResult]) ID() taskid.TaskImplementationID[TaskResult] {
	return c.id
}

// Labels implements Definition.
func (c *ConstantDefinitionImpl[TaskResult]) Labels() *typedmap.ReadonlyTypedMap {
	return c.labels
}

func (c *ConstantDefinitionImpl[TaskResult]) UntypedID() taskid.UntypedTaskImplementationID {
	return c.ID()
}

func (c *ConstantDefinitionImpl[TaskResult]) UntypedRun(ctx context.Context) (any, error) {
	return c.Run(ctx)
}

var _ Definition[any] = (*ConstantDefinitionImpl[any])(nil)

func NewTask[TaskResult any](taskId taskid.TaskImplementationID[TaskResult], dependencies []taskid.UntypedTaskReference, runFunc func(ctx context.Context) (TaskResult, error), labelOpts ...LabelOpt) *ConstantDefinitionImpl[TaskResult] {
	labels := NewLabelSet(labelOpts...)
	return &ConstantDefinitionImpl[TaskResult]{
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
