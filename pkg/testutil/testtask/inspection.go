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

package testtask

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

func RunSingleTask[T any](target task.Definition, mode int, opts ...TestRunTaskParameterOpt) (T, error) {
	return RunMultipleTask[T](target, []task.Definition{}, mode, opts...)
}

func RunMultipleTask[T any](target task.Definition, availableTasks []task.Definition, mode int, opts ...TestRunTaskParameterOpt) (T, error) {
	params := generateVariableSetFromOpts(opts...)
	sourceTaskSet, err := task.NewSet([]task.Definition{target})
	if err != nil {
		return *new(T), err
	}

	mockedParameterTasks := []task.Definition{}
	for key, value := range params {
		nextTaskValue := value
		mockedParameterTasks = append(mockedParameterTasks, task.NewProcessorTask(key, []string{}, func(ctx context.Context, taskMode int, v *task.VariableSet) (any, error) {
			return nextTaskValue, nil
		}))
	}

	availableTaskSet, err := task.NewSet(append(availableTasks, mockedParameterTasks...))
	if err != nil {
		return *new(T), err
	}

	resolved, err := sourceTaskSet.ResolveTask(availableTaskSet)
	if err != nil {
		return *new(T), err
	}

	localRunner, err := task.NewLocalRunner(resolved)
	if err != nil {
		return *new(T), err
	}

	localRunner = localRunner.WithCacheProvider(&task.LocalTaskVariableCache{})

	err = localRunner.Run(context.Background(), mode, map[string]any{
		inspection_task.MetadataVariableName: metadata.NewSet(),
	})
	if err != nil {
		return *new(T), err
	}

	<-localRunner.Wait()
	result, err := localRunner.Result()
	if err != nil {
		return *new(T), err
	}
	return task.GetTypedVariableFromTaskVariable(result, target.ID().String(), *new(T))
}

func generateVariableSetFromOpts(opts ...TestRunTaskParameterOpt) map[string]any {
	parameters := map[string]any{}
	parameters[inspection_task.MetadataVariableName] = metadata.NewSet()
	for _, opt := range opts {
		opt.AddParam(parameters)
	}
	return parameters
}
