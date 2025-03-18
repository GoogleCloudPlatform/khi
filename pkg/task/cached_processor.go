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
	"crypto/md5"
	"fmt"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

type CachableDependency interface {
	Digest() string
}

const TaskCacheTaskID = KHISystemPrefix + "cache"

func GetCacheStoreFromTaskVariable(tv *VariableSet) (TaskVariableCache, error) {
	return GetTypedVariableFromTaskVariable[TaskVariableCache](tv, TaskCacheTaskID, nil)
}

func NewCachedProcessor[TaskResult any](taskId taskid.TaskImplementationID[TaskResult], dependencies []taskid.UntypedTaskReference, processorFunc ProcessorFunc[TaskResult], labelOpt ...LabelOpt) Definition[TaskResult] {
	return NewProcessorTask(taskId, dependencies, func(ctx context.Context, taskMode int, v *VariableSet) (TaskResult, error) {
		cacheKey := fmt.Sprintf("%s-%d-", taskId, taskMode)
		for _, source := range dependencies {
			rawValue, err := GetTypedVariableFromTaskVariable[any](v, source.ReferenceIDString(), "")
			if err != nil {
				return *new(TaskResult), err
			}
			if rawValue == nil {
				cacheKey += fmt.Sprintf("%s=nil,", source)
				continue
			}
			if rawValueStr, stringConvertable := rawValue.(string); stringConvertable {
				cacheKey += fmt.Sprintf("%s=%s,", source, md5.Sum([]byte(rawValueStr)))
				continue
			}
			if cachable, cachableConvertible := rawValue.(CachableDependency); cachableConvertible {
				cacheKey += fmt.Sprintf("%s=%s,", source, md5.Sum([]byte(cachable.Digest())))
				continue
			}
			return *new(TaskResult), fmt.Errorf("failed to generate cache key from the source `%s`.The source must be a string or implementing CachableDependency. %v can't be converted to the desired value.", source, rawValue)
		}

		tvc, err := GetCacheStoreFromTaskVariable(v)
		if err != nil {
			return *new(TaskResult), err
		}

		// processor-cache-lock is used to wait the runnable cache to be available if there were the other task graph shareing the same task graph run it already.
		lockKey := cacheKey + "processor-cache-lock"
		lockAny, _ := tvc.LoadOrStore(lockKey, &sync.Mutex{})
		lock := (lockAny).(*sync.Mutex)
		lock.Lock()
		defer lock.Unlock()
		cachedValue, exists := tvc.Load(cacheKey)
		if exists {
			return cachedValue.(TaskResult), nil
		}
		value, err := processorFunc(ctx, taskMode, v)
		if err != nil {
			return *new(TaskResult), err
		}
		tvc.Store(cacheKey, value)
		return value, nil
	}, labelOpt...)
}
