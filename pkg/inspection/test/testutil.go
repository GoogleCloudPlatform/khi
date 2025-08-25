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

package inspection_task_test

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	error_metadata "github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/error"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/form"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/header"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/query"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	inspection_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/contract"
	task_test "github.com/GoogleCloudPlatform/khi/pkg/task/test"
)

// WithDefaultTestInspectionTaskContext returns a new context used for running inspection task.
func WithDefaultTestInspectionTaskContext(baseContext context.Context) context.Context {
	taskCtx := khictx.WithValue(baseContext, inspection_contract.InspectionTaskInspectionID, "fake-inspection-id")
	taskCtx = khictx.WithValue(taskCtx, inspection_contract.InspectionTaskRunID, "fake-run-id")

	taskCtx = khictx.WithValue(taskCtx, inspection_contract.GlobalSharedMap, typedmap.NewTypedMap())
	taskCtx = khictx.WithValue(taskCtx, inspection_contract.InspectionSharedMap, typedmap.NewTypedMap())

	ioConfig, err := inspection_contract.NewIOConfigForTest()
	if err != nil {
		panic("Failed to create test IOConfig: " + err.Error())
	}
	taskCtx = khictx.WithValue(taskCtx, inspection_contract.CurrentIOConfig, ioConfig)
	taskCtx = khictx.WithValue(taskCtx, inspection_contract.CurrentHistoryBuilder, history.NewBuilder(ioConfig.TemporaryFolder))
	taskCtx = khictx.WithValue(taskCtx, inspection_contract.InspectionRunMetadata, generateTestMetadata())
	return taskCtx
}

// RunInspectionTask execute a single task with given context. Use WithDefaultTestInspectionTaskContext to get the context.
func RunInspectionTask[T any](baseContext context.Context, task coretask.Task[T], mode inspection_contract.InspectionTaskModeType, input map[string]any, taskDependencyValues ...task_test.TaskDependencyValues) (T, *typedmap.ReadonlyTypedMap, error) {
	taskCtx := khictx.WithValue(baseContext, inspection_contract.InspectionTaskInput, input)
	taskCtx = khictx.WithValue(taskCtx, inspection_contract.InspectionTaskMode, mode)

	result, err := task_test.RunTask(taskCtx, task, taskDependencyValues...)
	metadata := khictx.MustGetValue(taskCtx, inspection_contract.InspectionRunMetadata)
	return result, metadata, err
}

// RunInspectionTaskWithDependency execute a task as a graph. Supply dependencies needed to be used with the mainTask.
func RunInspectionTaskWithDependency[T any](baseContext context.Context, mainTask coretask.Task[T], dependencies []coretask.UntypedTask, mode inspection_contract.InspectionTaskModeType, input map[string]any) (T, *typedmap.ReadonlyTypedMap, error) {
	taskCtx := khictx.WithValue(baseContext, inspection_contract.InspectionTaskInput, input)
	taskCtx = khictx.WithValue(taskCtx, inspection_contract.InspectionTaskMode, mode)
	result, err := task_test.RunTaskWithDependency(taskCtx, mainTask, dependencies)
	metadata := khictx.MustGetValue(taskCtx, inspection_contract.InspectionRunMetadata)
	return result, metadata, err
}

func generateTestMetadata() *typedmap.ReadonlyTypedMap {
	writableMetadata := typedmap.NewTypedMap()
	typedmap.Set(writableMetadata, header.HeaderMetadataKey, &header.Header{})
	typedmap.Set(writableMetadata, error_metadata.ErrorMessageSetMetadataKey, error_metadata.NewErrorMessageSet())
	typedmap.Set(writableMetadata, form.FormFieldSetMetadataKey, form.NewFormFieldSet())
	typedmap.Set(writableMetadata, query.QueryMetadataKey, query.NewQueryMetadata())
	typedmap.Set(writableMetadata, progress.ProgressMetadataKey, progress.NewProgress())
	return writableMetadata.AsReadonly()
}
