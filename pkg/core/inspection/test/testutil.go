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
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progress"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// TestInspectionCreationTime is a fixed time used across tests to ensure deterministic behavior.
var TestInspectionCreationTime = time.Date(2025, time.January, 1, 1, 1, 1, 1, time.UTC)

// WithDefaultTestInspectionTaskContext returns a new context used for running inspection task.
func WithDefaultTestInspectionTaskContext(baseContext context.Context) context.Context {
	taskCtx := khictx.WithValue(baseContext, inspectioncore.InspectionCreationTime, TestInspectionCreationTime)
	taskCtx = khictx.WithValue(taskCtx, inspectioncore.InspectionTaskInspectionID, "fake-inspection-id")
	taskCtx = khictx.WithValue(taskCtx, inspectioncore.InspectionTaskRunID, "fake-run-id")
	taskCtx = khictx.WithValue(taskCtx, inspectioncore.InspectionContext, baseContext)

	taskCtx = khictx.WithValue(taskCtx, inspectioncore.GlobalSharedMap, typedmap.NewTypedMap())
	taskCtx = khictx.WithValue(taskCtx, inspectioncore.InspectionSharedMap, typedmap.NewTypedMap())
	taskCtx = khictx.WithValue[inspectioncore.InspectionNameRegistry](taskCtx, inspectioncore.InspectionNameRegistryKey, inspectioncore.NewInMemoryInspectionNameRegistry())

	// If this context is used with the task runner, it should have the task result map. But if not, then this must complement the value with the default value.
	_, err := khictx.GetValue(taskCtx, core_contract.TaskResultMapContextKey)
	if err != nil {
		taskCtx = khictx.WithValue(taskCtx, core_contract.TaskResultMapContextKey, typedmap.NewTypedMap())
	}
	_, err = khictx.GetValue(taskCtx, core_contract.TaskImplementationIDContextKey)
	if err != nil {
		fakeTaskID := taskid.NewDefaultImplementationID[struct{}]("khi.google.com/fake-test-id")
		taskCtx = khictx.WithValue(taskCtx, core_contract.TaskImplementationIDContextKey, fakeTaskID.(taskid.UntypedTaskImplementationID))
	}

	ioConfig, err := inspectioncore.NewIOConfigForTest()
	if err != nil {
		panic("Failed to create test IOConfig: " + err.Error())
	}
	idGen := id.NewGenerator()
	taskCtx = khictx.WithValue(taskCtx, inspectioncore.IDGenerator, idGen)
	taskCtx = khictx.WithValue(taskCtx, inspectioncore.CurrentIOConfig, ioConfig)
	taskCtx = khictx.WithValue(taskCtx, inspectioncore.Builder, khifilev6.NewTestBuilder(idGen))
	taskCtx = khictx.WithValue(taskCtx, inspectionmetadata.MapContextKey, generateTestMetadata())
	return taskCtx
}

// NextRunTaskContext generates a new context used for running inspection task from the task context used for previous task run.
func NextRunTaskContext(originalCtx context.Context, prevRunCtx context.Context) context.Context {
	originalCtx = WithDefaultTestInspectionTaskContext(originalCtx)

	globalSharedMap := khictx.MustGetValue(prevRunCtx, inspectioncore.GlobalSharedMap)
	inspectionSharedMap := khictx.MustGetValue(prevRunCtx, inspectioncore.InspectionSharedMap)

	originalCtx = khictx.WithValue(originalCtx, inspectioncore.GlobalSharedMap, globalSharedMap)
	return khictx.WithValue(originalCtx, inspectioncore.InspectionSharedMap, inspectionSharedMap)
}

// RunInspectionTask execute a single task with given context. Use WithDefaultTestInspectionTaskContext to get the context.
func RunInspectionTask[T any](baseContext context.Context, task coretask.Task[T], mode inspectioncore.InspectionTaskModeType, input map[string]any, taskDependencyValues ...tasktest.TaskDependencyValues) (T, *typedmap.ReadonlyTypedMap, error) {
	taskCtx := khictx.WithValue(baseContext, inspectioncore.InspectionTaskInput, input)
	taskCtx = khictx.WithValue(taskCtx, inspectioncore.InspectionTaskMode, mode)
	metadata := khictx.MustGetValue(taskCtx, inspectionmetadata.MapContextKey)

	var result T
	_, err := progress.TaskInterceptor(taskCtx, task, func(ctx context.Context) (any, error) {
		var runErr error
		result, runErr = tasktest.RunTask(ctx, task, taskDependencyValues...)
		return result, runErr
	})
	return result, metadata, err
}

// RunInspectionTaskWithDependency execute a task as a graph. Supply dependencies needed to be used with the mainTask.
func RunInspectionTaskWithDependency[T any](baseContext context.Context, mainTask coretask.Task[T], dependencies []coretask.UntypedTask, mode inspectioncore.InspectionTaskModeType, input map[string]any) (T, *typedmap.ReadonlyTypedMap, error) {
	taskCtx := khictx.WithValue(baseContext, inspectioncore.InspectionTaskInput, input)
	taskCtx = khictx.WithValue(taskCtx, inspectioncore.InspectionTaskMode, mode)
	metadata := khictx.MustGetValue(taskCtx, inspectionmetadata.MapContextKey)
	result, err := tasktest.RunTaskWithDependency(taskCtx, mainTask, dependencies, progress.TaskInterceptor)
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
