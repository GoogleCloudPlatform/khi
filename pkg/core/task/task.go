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

package coretask

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
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

// LabelKeyRequiredTask is the task label to tell task resolver to always include the task in the task graph when the task is available.
var LabelKeyRequiredTask = NewTaskLabelKey[bool](KHISystemPrefix + "required-task")

// LabelKeySubsequentTaskRefs is the list of task references. These tasks are included in the task graph later and the included task reference this task.
var LabelKeySubsequentTaskRefs = NewTaskLabelKey[[]taskid.UntypedTaskReference](KHISystemPrefix + "subsquent-task-refs")

// LabelKeyTaskResultRetention indicates whether the task result should be retained in the runner after all dependent tasks finish.
var LabelKeyTaskResultRetention = NewTaskLabelKey[bool](KHISystemPrefix + "task-result-retention")

// LabelKeyAllowMultiStageExecution indicates that the task is pure and eligible for multi-stage execution during cycle resolution.
var LabelKeyAllowMultiStageExecution = NewTaskLabelKey[bool](KHISystemPrefix + "allow-multi-stage-execution")

// UntypedTask represents a task in the DAG without compile-time result type information.
type UntypedTask interface {
	UntypedID() taskid.UntypedTaskImplementationID
	// Labels returns KHITaskLabelSet assigned to this task unit.
	// The implementation of this function must return a constant value.
	Labels() *typedmap.ReadonlyTypedMap

	// Dependencies returns the list of task dependencies. Task runner will wait for these dependencies before running this task.
	Dependencies() []Dependency

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

// TaskImpl provides the default implementation of Task.
type TaskImpl[TaskResult any] struct {
	id           taskid.TaskImplementationID[TaskResult]
	labels       *typedmap.ReadonlyTypedMap
	dependencies []Dependency
	runFunc      func(ctx context.Context) (TaskResult, error)
}

// Run implements Task.
func (c *TaskImpl[TaskResult]) Run(ctx context.Context) (TaskResult, error) {
	return c.runFunc(ctx)
}

// Dependencies implements Task.
func (c *TaskImpl[TaskResult]) Dependencies() []Dependency {
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

// UntypedID implements UntypedTask.
func (c *TaskImpl[TaskResult]) UntypedID() taskid.UntypedTaskImplementationID {
	return c.ID()
}

// UntypedRun implements UntypedTask.
func (c *TaskImpl[TaskResult]) UntypedRun(ctx context.Context) (any, error) {
	return c.Run(ctx)
}

var _ Task[any] = (*TaskImpl[any])(nil)

type allowMultiStageExecutionLabelOpt struct{}

func (a *allowMultiStageExecutionLabelOpt) Write(labels *typedmap.TypedMap) {
	typedmap.Set(labels, LabelKeyAllowMultiStageExecution, true)
}

var _ LabelOpt = (*allowMultiStageExecutionLabelOpt)(nil)

// AllowMultiStageExecution returns a LabelOpt declaring that the task is eligible for multi-stage execution.
// Such tasks can be cloned and executed across multiple stages during graph resolution to break FanIn dependency cycles.
func AllowMultiStageExecution() LabelOpt {
	return &allowMultiStageExecutionLabelOpt{}
}

// NewTask constructs a new Task with the given implementation ID, dependencies, execution function, and label options.
func NewTask[TaskResult any](taskID taskid.TaskImplementationID[TaskResult], dependencies []Dependency, runFunc func(ctx context.Context) (TaskResult, error), labelOpts ...LabelOpt) *TaskImpl[TaskResult] {
	verifyTaskID(taskID)
	verifyNonNilDependencies(taskID, dependencies)
	labels := NewLabelSet(labelOpts...)
	verifyLabelKeys(taskID, labels)
	return &TaskImpl[TaskResult]{
		id:           taskID,
		labels:       labels,
		dependencies: dedupeDependencies(dependencies),
		runFunc:      runFunc,
	}
}

// mergeDependencies combines two duplicate dependencies targeting the same task or tag,
// selecting the most restrictive attributes: Data over OrderOnly, Required over Optional,
// and the broader scope (ScopeAll > ScopeActiveFeatures > ScopeActiveGraph).
func mergeDependencies(a, b Dependency) Dependency {
	kind := taskid.EdgeKindOrderOnly
	if a.DescriptorKind() == taskid.EdgeKindData || b.DescriptorKind() == taskid.EdgeKindData {
		kind = taskid.EdgeKindData
	}

	condition := taskid.ConditionOptional
	if a.DescriptorCondition() == taskid.ConditionRequired || b.DescriptorCondition() == taskid.ConditionRequired {
		condition = taskid.ConditionRequired
	}

	scope := mergeScopes(a.DescriptorScope(), b.DescriptorScope())

	switch d := a.(type) {
	case taskid.PointToPointDescriptor:
		var opts []taskid.ReferenceOption
		if kind == taskid.EdgeKindOrderOnly {
			opts = append(opts, taskid.OrderOnly)
		}
		if condition == taskid.ConditionOptional {
			opts = append(opts, taskid.Optional)
		}
		if scope != taskid.ScopeUnspecified {
			opts = append(opts, scope)
		}
		return taskid.NewTaskReference[any](d.ReferenceID(), opts...)
	case taskid.FanInDescriptor:
		var opts []taskid.FanInOption
		if kind == taskid.EdgeKindOrderOnly {
			opts = append(opts, taskid.OrderOnly)
		}
		if scope != taskid.ScopeUnspecified {
			opts = append(opts, scope)
		}
		return NewTagReference[any](d.Tag(), opts...)
	default:
		return a
	}
}

// mergeScopes returns the broader dependency scope between a and b.
// Scope breadth order: ScopeAll > ScopeActiveFeatures > ScopeActiveGraph > ScopeUnspecified.
func mergeScopes(a, b taskid.DependencyScope) taskid.DependencyScope {
	if a == taskid.ScopeAll || b == taskid.ScopeAll {
		return taskid.ScopeAll
	}
	if a == taskid.ScopeActiveFeatures || b == taskid.ScopeActiveFeatures {
		return taskid.ScopeActiveFeatures
	}
	if a == taskid.ScopeActiveGraph || b == taskid.ScopeActiveGraph {
		return taskid.ScopeActiveGraph
	}
	return taskid.ScopeUnspecified
}

func dedupeDependencies(dependencies []Dependency) []Dependency {
	result := make([]Dependency, 0, len(dependencies))
	seen := make(map[string]int, len(dependencies))
	for _, dep := range dependencies {
		key := dependencyKey(dep)
		if key == "" {
			result = append(result, dep)
			continue
		}
		if idx, ok := seen[key]; ok {
			result[idx] = mergeDependencies(result[idx], dep)
			continue
		}
		seen[key] = len(result)
		result = append(result, dep)
	}
	return result
}

func verifyTaskID[TaskResult any](taskID taskid.TaskImplementationID[TaskResult]) {
	if taskID == nil || taskID.String() == "" {
		panic(`Invalid taskID. This may be caused because of initialization order issue of global variables.
Please define task IDs and types used in its type parameter in a different package.`)
	}
}

func verifyNonNilDependencies(taskID taskid.UntypedTaskImplementationID, dependencies []Dependency) {
	for i, dependency := range dependencies {
		if dependency == nil {
			panic(fmt.Sprintf(`Invalid task definition: %s. Given task dependency list contains a nil reference at #%d. This may be caused because of initialization order issue of global variables.
Please define task IDs and types used in its type parameter in a different package.`, taskID.String(), i))
		}
	}
}

func verifyLabelKeys(taskID taskid.UntypedTaskImplementationID, labels *typedmap.ReadonlyTypedMap) {
	keys := labels.Keys()
	for i, key := range keys {
		if key == "" {
			panic(fmt.Sprintf(`Invalid task definition: %s. Given task label contains an empty key at #%d. This may be caused because of initialization order issue of global variables.
Please define label IDs and types used in its type parameter in a different package.`, taskID.String(), i))
		}
	}
}
