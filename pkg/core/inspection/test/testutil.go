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

package inspectiontest

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// WithDefaultTestInspectionTaskContext returns a new context used for running inspection task.
func WithDefaultTestInspectionTaskContext(baseContext context.Context) context.Context {
	taskCtx := khictx.WithValue(baseContext, inspectioncore_contract.InspectionTaskInspectionID, "fake-inspection-id")
	taskCtx = khictx.WithValue(taskCtx, inspectioncore_contract.InspectionTaskRunID, "fake-run-id")

	taskCtx = khictx.WithValue(taskCtx, inspectioncore_contract.GlobalSharedMap, typedmap.NewTypedMap())
	taskCtx = khictx.WithValue(taskCtx, inspectioncore_contract.InspectionSharedMap, typedmap.NewTypedMap())

	ioConfig, err := inspectioncore_contract.NewIOConfigForTest()
	if err != nil {
		panic("Failed to create test IOConfig: " + err.Error())
	}
	taskCtx = khictx.WithValue(taskCtx, inspectioncore_contract.CurrentIOConfig, ioConfig)
	taskCtx = khictx.WithValue(taskCtx, inspectioncore_contract.CurrentHistoryBuilder, history.NewBuilder(ioConfig.TemporaryFolder))
	taskCtx = khictx.WithValue(taskCtx, inspectioncore_contract.InspectionRunMetadata, generateTestMetadata())
	return taskCtx
}

// NextRunTaskContext generates a new context used for running inspection task from the task context used for previous task run.
func NextRunTaskContext(originalCtx context.Context, prevRunCtx context.Context) context.Context {
	originalCtx = WithDefaultTestInspectionTaskContext(originalCtx)

	globalSharedMap := khictx.MustGetValue(prevRunCtx, inspectioncore_contract.GlobalSharedMap)
	inspectionSharedMap := khictx.MustGetValue(prevRunCtx, inspectioncore_contract.InspectionSharedMap)

	originalCtx = khictx.WithValue(originalCtx, inspectioncore_contract.GlobalSharedMap, globalSharedMap)
	return khictx.WithValue(originalCtx, inspectioncore_contract.InspectionSharedMap, inspectionSharedMap)
}

// RunInspectionTask execute a single task with given context. Use WithDefaultTestInspectionTaskContext to get the context.
func RunInspectionTask[T any](baseContext context.Context, task coretask.Task[T], mode inspectioncore_contract.InspectionTaskModeType, input map[string]any, taskDependencyValues ...tasktest.TaskDependencyValues) (T, *typedmap.ReadonlyTypedMap, error) {
	taskCtx := khictx.WithValue(baseContext, inspectioncore_contract.InspectionTaskInput, input)
	taskCtx = khictx.WithValue(taskCtx, inspectioncore_contract.InspectionTaskMode, mode)

	result, err := tasktest.RunTask(taskCtx, task, taskDependencyValues...)
	metadata := khictx.MustGetValue(taskCtx, inspectioncore_contract.InspectionRunMetadata)
	return result, metadata, err
}

// RunInspectionTaskWithDependency execute a task as a graph. Supply dependencies needed to be used with the mainTask.
func RunInspectionTaskWithDependency[T any](baseContext context.Context, mainTask coretask.Task[T], dependencies []coretask.UntypedTask, mode inspectioncore_contract.InspectionTaskModeType, input map[string]any) (T, *typedmap.ReadonlyTypedMap, error) {
	taskCtx := khictx.WithValue(baseContext, inspectioncore_contract.InspectionTaskInput, input)
	taskCtx = khictx.WithValue(taskCtx, inspectioncore_contract.InspectionTaskMode, mode)
	result, err := tasktest.RunTaskWithDependency(taskCtx, mainTask, dependencies)
	metadata := khictx.MustGetValue(taskCtx, inspectioncore_contract.InspectionRunMetadata)
	return result, metadata, err
}

func generateTestMetadata() *typedmap.ReadonlyTypedMap {
	writableMetadata := typedmap.NewTypedMap()
	typedmap.Set(writableMetadata, inspectionmetadata.HeaderMetadataKey, &inspectionmetadata.HeaderMetadata{})
	typedmap.Set(writableMetadata, inspectionmetadata.ErrorMessageSetMetadataKey, inspectionmetadata.NewErrorMessageSetMetadata())
	typedmap.Set(writableMetadata, inspectionmetadata.FormFieldSetMetadataKey, inspectionmetadata.NewFormFieldSetMetadata())
	typedmap.Set(writableMetadata, inspectionmetadata.QueryMetadataKey, inspectionmetadata.NewQueryMetadata())
	typedmap.Set(writableMetadata, inspectionmetadata.ProgressMetadataKey, inspectionmetadata.NewProgress())
	return writableMetadata.AsReadonly()
}
