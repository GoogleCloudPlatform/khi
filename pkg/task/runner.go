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
	"fmt"
	"log/slog"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
	"golang.org/x/sync/errgroup"
)

// Runner receives the runnable DefinitionSet and run tasks with topological sorted order.
type Runner interface {
	Run(ctx context.Context, runMode int, initialVariables map[string]any) error
	Wait() <-chan interface{}
	Result() (*VariableSet, error)
}

var _ Runner = (*LocalRunner)(nil)

type LocalRunner struct {
	resolvedDefinitionSet *DefinitionSet
	resultVariable        *VariableSet
	resultError           error
	started               bool
	stopped               bool
	waiter                chan interface{}
	taskStatuses          []*LocalRunnerTaskStat
	cache                 TaskVariableCache
}

type LocalRunnerTaskStat struct {
	Phase     string
	Error     error
	StartTime time.Time
	EndTime   time.Time
}

const (
	LocalRunnerTaskStatPhaseWaiting = "WAITING"
	LocalRunnerTaskStatPhaseRunning = "RUNNING"
	LocalRunnerTaskStatPhaseStopped = "STOPPED"
)

func (r *LocalRunner) Wait() <-chan interface{} {
	return r.waiter
}

// Result implements Runner.
func (r *LocalRunner) Result() (*VariableSet, error) {
	if !r.stopped {
		return nil, fmt.Errorf("this task runner hasn't finished yet")
	}
	if r.resultError != nil {
		return nil, r.resultError
	}
	return r.resultVariable, nil
}

// Run implements Runner.
func (r *LocalRunner) Run(ctx context.Context, taskMode int, initialVariables map[string]any) error {
	if r.started {
		return fmt.Errorf("this task is already started before")
	}
	go func() {
		defer r.markDone()
		r.resultVariable = NewVariableSet(initialVariables)
		r.resultVariable.Set(TaskCacheTaskID, r.cache)
		definitions := r.resolvedDefinitionSet.GetAll()
		cancelableCtx, cancel := context.WithCancel(ctx)
		currentErrGrp, currentErrCtx := errgroup.WithContext(cancelableCtx)
		currentTasks := []taskid.TaskImplementationId{}
		for i, definition := range definitions {
			taskDefIndex := i
			// if this task was thread-unsafe, wait all tasks started before.
			if isThreadUnsafe := definition.Labels().GetOrDefault(LabelKeyThreadUnsafe, false); isThreadUnsafe.(bool) {
				err := currentErrGrp.Wait()
				if err != nil {
					slog.WarnContext(currentErrCtx, fmt.Sprintf("tasks return errors during waiting for starting task %s", definitions[taskDefIndex].ID()))
					cancel()
					r.resultError = err
					return
				}
				nextTaskId := definitions[taskDefIndex].ID()
				currentTasks = []taskid.TaskImplementationId{nextTaskId}
				err = r.runTask(cancelableCtx, taskDefIndex, taskMode)
				if err != nil {
					cancel()
					r.resultError = err
					return
				}
				currentErrGrp, currentErrCtx = errgroup.WithContext(cancelableCtx)
			} else {
				nextTaskId := definitions[taskDefIndex].ID()
				currentTasks = append(currentTasks, nextTaskId)
				currentErrGrp.Go(func() error {
					err := r.runTask(currentErrCtx, taskDefIndex, taskMode)
					if err != nil {
						cancel()
						return err
					}
					return nil
				})
			}
		}
		err := currentErrGrp.Wait()
		if err != nil {
			r.resultError = err
		}
		cancel()
	}()
	return nil
}

func (r *LocalRunner) runTask(ctx context.Context, taskDefIndex int, taskMode int) error {
	definition := r.resolvedDefinitionSet.GetAll()[taskDefIndex]
	sources := definition.Dependencies()
	runnable := definition.Runnable(taskMode)
	taskStatus := r.taskStatuses[taskDefIndex]
	ctx = context.WithValue(ctx, "tid", definition.ID())
	slog.DebugContext(ctx, fmt.Sprintf("task %s started", definition.ID().String()))
	r.waitDependencies(ctx, sources)
	if ctx.Err() == context.Canceled {
		return context.Canceled
	}

	taskStatus.StartTime = time.Now()
	taskStatus.Phase = LocalRunnerTaskStatPhaseRunning
	slog.DebugContext(ctx, fmt.Sprintf("task %s started", definition.ID()))

	err := runnable.Run(ctx, r.resultVariable)

	taskStatus.Phase = LocalRunnerTaskStatPhaseStopped
	taskStatus.EndTime = time.Now()
	slog.DebugContext(ctx, fmt.Sprintf("task %s stopped after %f sec", definition.ID(), taskStatus.EndTime.Sub(taskStatus.StartTime).Seconds()))
	taskStatus.Error = err
	if ctx.Err() == context.Canceled {
		return context.Canceled
	}
	if err != nil {
		detailedErr := r.wrapWithTaskError(err, definition)
		r.resultError = detailedErr
		slog.ErrorContext(ctx, err.Error())
		return detailedErr
	}
	err = r.resolveTask(ctx, definition.ID())
	if err != nil {
		detailedErr := r.wrapWithTaskError(err, definition)
		taskStatus.Error = err
		r.resultError = detailedErr
		slog.ErrorContext(ctx, err.Error())
		return detailedErr
	}
	return nil
}

func (r *LocalRunner) TaskStatuses() []*LocalRunnerTaskStat {
	return r.taskStatuses
}

func newLocalRunnerTaskStatus() *LocalRunnerTaskStat {
	return &LocalRunnerTaskStat{
		Phase: LocalRunnerTaskStatPhaseWaiting,
	}
}

func NewLocalRunner(taskSet *DefinitionSet) (*LocalRunner, error) {
	if !taskSet.runnable {
		return nil, fmt.Errorf("given taskset must be runnable")
	}
	taskStatuses := []*LocalRunnerTaskStat{}
	for i := 0; i < len(taskSet.definitions); i++ {
		taskStatuses = append(taskStatuses, newLocalRunnerTaskStatus())
	}
	return &LocalRunner{
		resolvedDefinitionSet: taskSet,
		started:               false,
		resultVariable:        nil,
		resultError:           nil,
		stopped:               false,
		waiter:                make(chan interface{}),
		taskStatuses:          taskStatuses,
		cache:                 &GlobalTaskVariableCache{},
	}, nil
}

func (r *LocalRunner) WithCacheProvider(cache TaskVariableCache) *LocalRunner {
	r.cache = cache
	return r
}

func (r *LocalRunner) markDone() {
	r.stopped = true
	close(r.waiter)
}

func (r *LocalRunner) resolveTask(ctx context.Context, taskId taskid.TaskImplementationId) error {
	err := r.resultVariable.Set(fmt.Sprintf("%s/resolved", taskId.ReferenceId().String()), struct{}{})
	if err != nil {
		return err
	}
	return nil
}

func (r *LocalRunner) waitDependencies(ctx context.Context, sources []taskid.TaskReferenceId) error {
	errGrp := errgroup.Group{}
	for _, source := range sources {
		captureSource := source
		if !r.resultVariable.IsResolved(captureSource.String()) {
			errGrp.Go(func() error {
				_, err := r.resultVariable.Wait(ctx, fmt.Sprintf("%s/resolved", captureSource))
				return err
			})
		}
	}
	return errGrp.Wait()
}

func (r *LocalRunner) wrapWithTaskError(err error, definition Definition) error {
	errMsg := fmt.Sprintf("failed to run a task graph.\n definition ID=%s got an error. \n ERROR:\n%v", definition.ID(), err)
	return fmt.Errorf(errMsg)
}
