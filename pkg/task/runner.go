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
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/task/contextkey"
	task_interface "github.com/GoogleCloudPlatform/khi/pkg/task/inteface"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
	"golang.org/x/sync/errgroup"
)

var _ task_interface.TaskRunner = (*LocalRunner)(nil)

type LocalRunner struct {
	resolvedDefinitionSet *DefinitionSet
	resultVariable        *typedmap.TypedMap
	resultError           error
	started               bool
	stopped               bool
	taskWaiters           *sync.Map // sync.Map[string(taskRefID), sync.RWMutex], runner acquire the write lock at the beginning. All dependents will acquire read lock, it will be released when the task run finished.
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
func (r *LocalRunner) Result() (*typedmap.ReadonlyTypedMap, error) {
	if !r.stopped {
		return nil, fmt.Errorf("this task runner hasn't finished yet")
	}
	if r.resultError != nil {
		return nil, r.resultError
	}
	return r.resultVariable.AsReadonly(), nil
}

// Run implements Runner.
func (r *LocalRunner) Run(ctx context.Context) error {
	if r.started {
		return fmt.Errorf("this task is already started before")
	}
	go func() {
		defer r.markDone()

		// Setting up graph context
		r.resultVariable = typedmap.NewTypedMap()
		ctx = khictx.WithValue(ctx, task_contextkey.TaskResultMapContextKey, r.resultVariable)

		definitions := r.resolvedDefinitionSet.GetAll()
		cancelableCtx, cancel := context.WithCancel(ctx)
		currentErrGrp, currentErrCtx := errgroup.WithContext(cancelableCtx)
		for i := range definitions {
			taskDefIndex := i
			currentErrGrp.Go(func() error {
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

func (r *LocalRunner) runTask(graphCtx context.Context, taskDefIndex int) error {
	definition := r.resolvedDefinitionSet.GetAll()[taskDefIndex]
	sources := definition.Dependencies()
	taskStatus := r.taskStatuses[taskDefIndex]
	taskCtx := khictx.WithValue(graphCtx, task_contextkey.TaskImplementationIDContextKey, definition.UntypedID())
	slog.DebugContext(taskCtx, fmt.Sprintf("task %s started", definition.UntypedID().String()))
	r.waitDependencies(taskCtx, sources)
	if taskCtx.Err() == context.Canceled {
		return context.Canceled
	}

	taskStatus.StartTime = time.Now()
	taskStatus.Phase = LocalRunnerTaskStatPhaseRunning
	slog.DebugContext(taskCtx, fmt.Sprintf("task %s started", definition.UntypedID()))

	result, err := definition.UntypedRun(taskCtx)

	taskStatus.Phase = LocalRunnerTaskStatPhaseStopped
	taskStatus.EndTime = time.Now()
	slog.DebugContext(taskCtx, fmt.Sprintf("task %s stopped after %f sec", definition.UntypedID(), taskStatus.EndTime.Sub(taskStatus.StartTime).Seconds()))
	taskStatus.Error = err
	if taskCtx.Err() == context.Canceled {
		return context.Canceled
	}
	if err != nil {
		detailedErr := r.wrapWithTaskError(err, definition)
		r.resultError = detailedErr
		slog.ErrorContext(taskCtx, err.Error())
		return detailedErr
	}
	typedmap.Set(r.resultVariable, typedmap.NewTypedKey[any](definition.UntypedID().GetUntypedReference().ReferenceIDString()), result)
	taskWaiter, _ := r.taskWaiters.Load(definition.UntypedID().GetUntypedReference().String())
	taskWaiter.(*sync.RWMutex).Unlock()
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
	taskWaiters := sync.Map{}
	for i := 0; i < len(taskSet.definitions); i++ {
		taskStatuses = append(taskStatuses, newLocalRunnerTaskStatus())

		// lock the task waiter until its task finished.
		waiter := sync.RWMutex{}
		waiter.Lock()
		taskWaiters.Store(taskSet.definitions[i].UntypedID().ReferenceIDString(), &waiter)
	}
	return &LocalRunner{
		resolvedDefinitionSet: taskSet,
		started:               false,
		resultVariable:        nil,
		resultError:           nil,
		stopped:               false,
		taskWaiters:           &taskWaiters,
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
	r.taskWaiters.Range(func(key, value any) bool {
		mutex, _ := value.(*sync.RWMutex)
		if !mutex.TryRLock() {
			mutex.Unlock()
		}
		return true
	})
}

func (r *LocalRunner) waitDependencies(ctx context.Context, dependencies []taskid.UntypedTaskReference) error {
	for _, dependency := range dependencies {
		select { // wait for getting the RLock for the task result, or context cancel
		case <-ctx.Done():
			return ctx.Err()
		case <-func() chan struct{} {
			ch := make(chan struct{})
			go func() {
				waiter, _ := r.taskWaiters.Load(dependency.ReferenceIDString())
				taskWaiter := waiter.(*sync.RWMutex)
				taskWaiter.RLock()
				close(ch)
			}()
			return ch
		}():
			continue
		}
	}
	return nil
}

func (r *LocalRunner) wrapWithTaskError(err error, definition UntypedDefinition) error {
	errMsg := fmt.Sprintf("failed to run a task graph.\n definition ID=%s got an error. \n ERROR:\n%v", definition.UntypedID(), err)
	return fmt.Errorf("%s", errMsg)
}

// GetTaskResultFromLocalRunner returns task results from the local runner task results.
func GetTaskResultFromLocalRunner[TaskResult any](runner *LocalRunner, taskRef taskid.TaskReference[TaskResult]) (TaskResult, bool) {
	return typedmap.Get(runner.resultVariable, typedmap.NewTypedKey[TaskResult](taskRef.String()))
}
