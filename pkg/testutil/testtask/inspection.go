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

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/form"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/header"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

func RunSingleTask[T any](target task.Definition[T], mode int, opts ...TestRunTaskParameterOpt) (T, error) {
	return RunMultipleTask[T](target, []task.UntypedDefinition{}, mode, opts...)
}

func RunMultipleTask[T any](target task.Definition[T], availableTasks []task.UntypedDefinition, mode int, opts ...TestRunTaskParameterOpt) (T, error) {
	params := generateVariableSetFromOpts(opts...)
	sourceTaskSet, err := task.NewSet([]task.UntypedDefinition{target})
	if err != nil {
		return *new(T), err
	}

	mockedParameterTasks := []task.UntypedDefinition{}
	for key, value := range params {
		nextTaskValue := value
		mockedParameterTasks = append(mockedParameterTasks, task.NewProcessorTask(taskid.NewDefaultImplementationID[any](key), []taskid.UntypedTaskReference{}, func(ctx context.Context, taskMode int, v *task.VariableSet) (any, error) {
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

	metadata := typedmap.NewTypedMap()
	typedmap.Set(metadata, header.HeaderMetadataKey, &header.Header{})
	typedmap.Set(metadata, form.FormFieldSetMetadataKey, form.NewFormFieldSet())
	typedmap.Set(metadata, progress.ProgressMetadataKey, progress.NewProgress())

	err = localRunner.Run(context.Background(), mode, map[string]any{
		inspection_task.MetadataVariableName: metadata.AsReadonly(),
	})
	if err != nil {
		return *new(T), err
	}

	<-localRunner.Wait()
	result, err := localRunner.Result()
	if err != nil {
		return *new(T), err
	}
	return task.GetTypedVariableFromTaskVariable(result, target.UntypedID().ReferenceIDString(), *new(T))
}

func generateVariableSetFromOpts(opts ...TestRunTaskParameterOpt) map[string]any {
	parameters := map[string]any{}
	for _, opt := range opts {
		opt.AddParam(parameters)
	}
	return parameters
}
