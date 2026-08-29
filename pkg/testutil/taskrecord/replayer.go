// Copyright 2025 Google LLC
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

package taskrecord

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/pprof"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var (
	taskTypeRegistryMu sync.RWMutex
	taskTypeRegistry   = make(map[string]reflect.Type)
)

// RegisterTaskType associates a task reference with its expected Go return type.
func RegisterTaskType[T any](taskRef taskid.UntypedTaskReference) {
	taskTypeRegistryMu.Lock()
	defer taskTypeRegistryMu.Unlock()
	taskTypeRegistry[taskRef.ReferenceIDString()] = reflect.TypeOf((*T)(nil)).Elem()
}

// GetTaskType returns the registered Go return type for a task reference, defaulting to []*log.Log.
func GetTaskType(taskRef taskid.UntypedTaskReference) reflect.Type {
	taskTypeRegistryMu.RLock()
	defer taskTypeRegistryMu.RUnlock()
	if t, ok := taskTypeRegistry[taskRef.ReferenceIDString()]; ok {
		return t
	}
	return reflect.TypeOf([]*log.Log{})
}

// LoadRecordedTaskResult loads the recorded fixture for a specific task reference and type T.
func LoadRecordedTaskResult[T any](fixtureDir string, taskRef taskid.UntypedTaskReference) (T, error) {
	targetType := reflect.TypeOf((*T)(nil)).Elem()
	val, err := loadRecordedTaskResultForType(fixtureDir, taskRef, targetType)
	if err != nil {
		var zero T
		return zero, err
	}
	typedVal, ok := val.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("loaded task result has type %T, want %T", val, zero)
	}
	return typedVal, nil
}

// loadRecordedTaskResultForType loads and deserializes fixture data for a given reflect.Type.
func loadRecordedTaskResultForType(fixtureDir string, taskRef taskid.UntypedTaskReference, t reflect.Type) (any, error) {
	fileName := sanitizeTaskReferenceForFileName(taskRef.ReferenceIDString()) + ".json"
	filePath := filepath.Join(fixtureDir, fileName)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read fixture file %s: %w", filePath, err)
	}

	return DefaultCodecRegistry.DeserializeForType(data, t)
}

// newReplayStubTask creates a stub task with highest selection priority and no upstream dependencies.
func newReplayStubTask(taskRef taskid.UntypedTaskReference, val any) coretask.UntypedTask {
	typedRef := taskid.NewTaskReference[any](taskRef.ReferenceIDString())
	implID := taskid.NewImplementationID(typedRef, "replay")
	return coretask.NewTask[any](
		implID,
		[]taskid.UntypedTaskReference{},
		func(ctx context.Context) (any, error) {
			return val, nil
		},
		coretask.WithSelectionPriority(1000000),
	)
}

// newReplayInspectionInterceptor creates an InspectionInterceptor managing isolated execution and profiling.
func newReplayInspectionInterceptor(
	targetTask taskid.UntypedTaskReference,
	cpuProfilePath string,
	memProfilePath string,
	cancelFunc func(),
	onTargetComplete func(any),
) coreinspection.InspectionInterceptor {
	var targetRefID string
	if targetTask != nil {
		targetRefID = targetTask.ReferenceIDString()
	}

	var runningMu sync.Mutex
	var runningCond = sync.NewCond(&runningMu)
	runningCount := 0
	targetExecuting := false

	return func(ctx context.Context, req *inspectioncore_contract.InspectionRequest, next func(context.Context) error) error {
		runner := khictx.MustGetValue(ctx, inspectioncore_contract.TaskRunner)

		runner.AddInterceptor(func(taskCtx context.Context, task coretask.UntypedTask, taskNext func(context.Context) (any, error)) (any, error) {
			currentRefID := task.UntypedID().GetUntypedReference().ReferenceIDString()
			isTarget := targetRefID != "" && currentRefID == targetRefID

			if isTarget {
				// Wait for all other running tasks to finish
				runningMu.Lock()
				for runningCount > 0 {
					runningCond.Wait()
				}
				targetExecuting = true
				runningMu.Unlock()

				var cpuFile *os.File
				if cpuProfilePath != "" {
					if err := os.MkdirAll(filepath.Dir(cpuProfilePath), 0755); err == nil {
						f, err := os.Create(cpuProfilePath)
						if err == nil {
							cpuFile = f
							_ = pprof.StartCPUProfile(cpuFile)
						}
					}
				}

				res, err := taskNext(taskCtx)

				if cpuFile != nil {
					pprof.StopCPUProfile()
					_ = cpuFile.Close()
				}

				if memProfilePath != "" {
					runtime.GC()
					if err := os.MkdirAll(filepath.Dir(memProfilePath), 0755); err == nil {
						if f, err := os.Create(memProfilePath); err == nil {
							_ = pprof.WriteHeapProfile(f)
							_ = f.Close()
						}
					}
				}

				if onTargetComplete != nil && err == nil {
					onTargetComplete(res)
				}

				if cancelFunc != nil {
					cancelFunc()
				}

				runningMu.Lock()
				targetExecuting = false
				runningMu.Unlock()

				return res, err
			}

			// Preceding or other tasks
			runningMu.Lock()
			for targetExecuting {
				runningCond.Wait()
			}
			runningCount++
			runningMu.Unlock()

			res, err := taskNext(taskCtx)

			runningMu.Lock()
			runningCount--
			runningCond.Broadcast()
			runningMu.Unlock()

			return res, err
		})

		return next(ctx)
	}
}
