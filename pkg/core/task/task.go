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
	"reflect"

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

// LabelKeyTaskResultRetention indicates whether the task result should be retained in the runner after all dependent tasks finish.
var LabelKeyTaskResultRetention = NewTaskLabelKey[bool](KHISystemPrefix + "task-result-retention")

// LabelKeyTaskDescription is the task label to record a human-readable description of the task.
var LabelKeyTaskDescription = NewTaskLabelKey[string](KHISystemPrefix + "task-description")

// LabelKeyTaskResultType is the task label to record the string representation of the task output type.
var LabelKeyTaskResultType = NewTaskLabelKey[string](KHISystemPrefix + "task-result-type")

// LabelKeyFeatureGateTaskRef is a task label key specifying the task reference
// whose presence in the active graph gates inclusion of this task when resolved
// via ScopeActiveFeatures.
var LabelKeyFeatureGateTaskRef = NewTaskLabelKey[taskid.UntypedTaskReference](KHISystemPrefix + "feature-gate-task-ref")

type UntypedTask interface {
	UntypedID() taskid.UntypedTaskImplementationID
	// Labels returns KHITaskLabelSet assigned to this task unit.
	// The implementation of this function must return a constant value.
	Labels() *typedmap.ReadonlyTypedMap

	// Dependencies returns the list of task dependencies. Task runner will wait for these dependencies before running this task.
	Dependencies() []Dependency

	// ResultType returns the reflection Type of the task output.
	ResultType() reflect.Type

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

func verifyTaskID[TaskResult any](taskID taskid.TaskImplementationID[TaskResult]) {
	if taskID == nil || taskID.String() == "" {
		panic(`Invalid taskID. This may be caused because of initialization order issue of global variables.
Please define task IDs and types used in its type parameter in a different package.`)
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
