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

package task_test

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

// Deprecated: Use testtask package instead.
func MockProcessorTaskFromTaskID(taskId string, value any) task.Definition {
	return task.NewProcessorTask(taskId, []string{}, func(ctx context.Context, taskMode int, v *task.VariableSet) (any, error) {
		return value, nil
	})
}

// Deprecated: Use testtask package instead.
// RunTaskGraph executes the task graph just with provided dependency tasks
func RunTaskGraph(target task.Definition, mode int, initialParameters map[string]any, dependencies ...task.Definition) (*task.VariableSet, error) {
	sourceDs, err := task.NewSet([]task.Definition{target})
	if err != nil {
		return nil, err
	}
	availableDs, err := task.NewSet(dependencies)
	if err != nil {
		return nil, err
	}

	resolved, err := sourceDs.ResolveTask(availableDs)
	if err != nil {
		return nil, err
	}

	localRunner, err := task.NewLocalRunner(resolved)
	if err != nil {
		return nil, err
	}
	localRunner = localRunner.WithCacheProvider(&task.LocalTaskVariableCache{})

	err = localRunner.Run(context.Background(), mode, initialParameters)
	if err != nil {
		return nil, err
	}

	<-localRunner.Wait()
	return localRunner.Result()
}
