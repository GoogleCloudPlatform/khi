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
	"log/slog"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/errorreport"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/task/contextkey"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
	"golang.org/x/sync/errgroup"
)

// LocalRunner executes a task graph defined by a TaskSet on the local machine.
// It manages task dependencies, concurrent execution, and result aggregation.
type LocalRunner struct {
	resolvedTaskSet *TaskSet
	resultVariable  *typedmap.TypedMap
	resultError     error
	started         bool
	stopped         bool
	taskWaiters     *typedmap.ReadonlyTypedMap
	waiter          chan interface{}
	taskStatuses    []*LocalRunnerTaskStat
}

// LocalRunner implements task_interface.TaskRunner
var _ TaskRunner = (*LocalRunner)(nil)

// LocalRunnerTaskStat holds the status and metrics for a single task
// executed by the LocalRunner.
type LocalRunnerTaskStat struct {
	Phase     string
	Error     error
	StartTime time.Time
	EndTime   time.Time
}

const (
	// LocalRunnerTaskStatPhaseWaiting indicates that the task is waiting for its dependencies to complete.
	LocalRunnerTaskStatPhaseWaiting = "WAITING"
	// LocalRunnerTaskStatPhaseRunning indicates that the task is currently running.
	LocalRunnerTaskStatPhaseRunning = "RUNNING"
	// LocalRunnerTaskStatPhaseStopped indicates that the task has finished its execution (either completed or failed).
	LocalRunnerTaskStatPhaseStopped = "STOPPED"
)

// NewLocalRunner creates and initializes a new LocalRunner for a given TaskSet.
// The TaskSet must be runnable (i.e., topologically sorted with all dependencies met).
// It returns an error if the provided TaskSet is not runnable.
func NewLocalRunner(taskSet *TaskSet) (*LocalRunner, error) {
	if !taskSet.runnable {
		return nil, fmt.Errorf("given taskset must be runnable")
	}
	taskStatuses := []*LocalRunnerTaskStat{}
	taskWaiters := typedmap.NewTypedMap()
	for i := 0; i < len(taskSet.tasks); i++ {
		taskStatuses = append(taskStatuses, &LocalRunnerTaskStat{
			Phase: LocalRunnerTaskStatPhaseWaiting,
		})

		// lock the task waiter until its task finished.
		waiter := sync.RWMutex{}
		waiter.Lock()
		typedmap.Set(taskWaiters, waiterKeyForTask(taskSet.tasks[i].UntypedID().GetUntypedReference()), &waiter)
	}
	return &LocalRunner{
		resolvedTaskSet: taskSet,
		started:         false,
		resultVariable:  nil,
		resultError:     nil,
		stopped:         false,
		taskWaiters:     taskWaiters.AsReadonly(),
		waiter:          make(chan interface{}),
		taskStatuses:    taskStatuses,
	}, nil
}

// GetTaskResultFromLocalRunner is a helper function to safely extract a specific
// task's result from a LocalRunner's result map. It provides a type-safe way
// to access results using a TaskReference.
func GetTaskResultFromLocalRunner[TaskResult any](runner *LocalRunner, taskRef taskid.TaskReference[TaskResult]) (TaskResult, bool) {
	return typedmap.Get(runner.resultVariable, typedmap.NewTypedKey[TaskResult](taskRef.String()))
}

// Run starts the execution of the task graph in a non-blocking manner.
// It launches a goroutine to manage the entire execution process.
// It returns an error if the runner has already been started.
func (r *LocalRunner) Run(ctx context.Context) error {
	if r.started {
		return fmt.Errorf("this task is already started before")
	}
	go func() {
		defer r.finalizeExecution()

		// Setting up graph context
		r.resultVariable = typedmap.NewTypedMap()
		ctx = khictx.WithValue(ctx, task_contextkey.TaskResultMapContextKey, r.resultVariable)

		tasks := r.resolvedTaskSet.GetAll()
		cancelableCtx, cancel := context.WithCancel(ctx)
		currentErrGrp, currentErrCtx := errgroup.WithContext(cancelableCtx)
		for i := range tasks {
			taskDefIndex := i
			currentErrGrp.Go(func() error {
				defer errorreport.CheckAndReportPanic()
				err := r.runTask(currentErrCtx, taskDefIndex)
				if err != nil {
					cancel()
					return err
				}
				return nil
			})
		}
		err := currentErrGrp.Wait()
		if err != nil {
			r.resultError = err
		}
		cancel()
	}()
	return nil
}

// Wait returns a channel that is closed when the runner finishes executing all tasks
// in the graph. This is the primary mechanism for waiting for the completion of the
// entire task set.
func (r *LocalRunner) Wait() <-chan interface{} {
	return r.waiter
}

// Result returns the final results of the task graph execution.
// It returns a map of task results if the execution was successful, or an error
// if any task failed or the runner has not yet completed.
// This method should only be called after the channel from Wait() has been closed.
func (r *LocalRunner) Result() (*typedmap.ReadonlyTypedMap, error) {
	if !r.stopped {
		return nil, fmt.Errorf("this task runner hasn't finished yet")
	}
	if r.resultError != nil {
		return nil, r.resultError
	}
	return r.resultVariable.AsReadonly(), nil
}

// TaskStatuses returns a slice of LocalRunnerTaskStat, providing the status
// and execution details for each task in the runner's task set.
// The order of statuses corresponds to the order of tasks in the resolved TaskSet.
func (r *LocalRunner) TaskStatuses() []*LocalRunnerTaskStat {
	return r.taskStatuses
}

// runTask manages the entire lifecycle of a single task within the graph.
// It waits for all task dependencies to complete, executes the task,
// records its status, and stores its result or handles any errors.
func (r *LocalRunner) runTask(graphCtx context.Context, taskDefIndex int) error {
	task := r.resolvedTaskSet.GetAll()[taskDefIndex]
	taskStatus := r.taskStatuses[taskDefIndex]
	taskCtx := khictx.WithValue(graphCtx, task_contextkey.TaskImplementationIDContextKey, task.UntypedID())

	// Wait for completions of all dependencies.
	for _, dependency := range task.Dependencies() {
		err := r.waitForDependency(taskCtx, dependency)
		if err != nil {
			return err
		}
	}

	taskStatus.StartTime = time.Now()
	taskStatus.Phase = LocalRunnerTaskStatPhaseRunning
	slog.DebugContext(taskCtx, fmt.Sprintf("task %s started", task.UntypedID()))

	// Run the task
	result, err := task.UntypedRun(taskCtx)

	taskStatus.Phase = LocalRunnerTaskStatPhaseStopped
	taskStatus.EndTime = time.Now()
	slog.DebugContext(taskCtx, fmt.Sprintf("task %s stopped after %f sec", task.UntypedID(), taskStatus.EndTime.Sub(taskStatus.StartTime).Seconds()))
	taskStatus.Error = err
	if taskCtx.Err() == context.Canceled {
		return context.Canceled
	}
	if err != nil {
		detailedErr := r.wrapWithTaskError(err, task)
		r.resultError = detailedErr
		slog.ErrorContext(taskCtx, err.Error())
		return detailedErr
	}

	// store the task result to result map
	typedmap.Set(r.resultVariable, typedmap.NewTypedKey[any](task.UntypedID().GetUntypedReference().ReferenceIDString()), result)

	r.releaseTaskWaiter(task.UntypedID())

	return nil
}

// wrapWithTaskError creates a detailed error message, wrapping the original error
// with the ID of the task that produced it for better debugging context.
func (r *LocalRunner) wrapWithTaskError(err error, task UntypedTask) error {
	errMsg := fmt.Sprintf("failed to run a task graph.\n task ID=%s got an error. \n ERROR:\n%v", task.UntypedID(), err)
	return fmt.Errorf("%s", errMsg)
}

// waitForDependency blocks until a specified dependency task has completed.
// It handles context cancellation, allowing the wait to be interrupted.
func (r *LocalRunner) waitForDependency(ctx context.Context, task taskid.UntypedTaskReference) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-func() chan struct{} {
		ch := make(chan struct{})
		go func() {
			waiter, found := typedmap.Get(r.taskWaiters, waiterKeyForTask(task))
			if !found {
				slog.ErrorContext(ctx, fmt.Sprintf("unreachable error. Task waiter lock not found for the key `%s`", task.String()))
				close(ch)
				return
			}
			waiter.RLock()
			close(ch)
		}()
		return ch
	}():
		return nil
	}
}

// releaseTaskWaiter releases a waiter for a single task as complete by unlocking its corresponding
// RWMutex. This allows any tasks that depend on it to proceed.
func (r *LocalRunner) releaseTaskWaiter(task taskid.UntypedTaskImplementationID) error {
	lock, found := typedmap.Get(r.taskWaiters, waiterKeyForTask(task.GetUntypedReference()))
	if !found {
		return fmt.Errorf("unreachable error. Task waiter lock not found for the key `%s`", task.GetUntypedReference().String())
	}
	if !lock.TryRLock() {
		lock.Unlock()
	}
	return nil
}

// finalizeExecution performs cleanup after the entire task graph has finished.
// It marks the runner as stopped, closes the main waiter channel to signal completion,
// and ensures all task waiter locks are released.
func (r *LocalRunner) finalizeExecution() {
	r.stopped = true
	close(r.waiter)
	for _, task := range r.resolvedTaskSet.tasks {
		r.releaseTaskWaiter(task.UntypedID())
	}
}

// waiterKeyForTask is a helper function that creates a type-safe
// key for accessing the waiter RWMutex in the taskWaiters map.
func waiterKeyForTask(taskID taskid.UntypedTaskReference) typedmap.TypedKey[*sync.RWMutex] {
	return typedmap.NewTypedKey[*sync.RWMutex](taskID.ReferenceIDString())
}
