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
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progress"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/summary"
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

var writableMetadataContextKey = typedmap.NewTypedKey[*typedmap.TypedMap]("test-writable-metadata")

func newTestSummaryCollector(tasks ...coretask.UntypedTask) *summary.Collector {
	fakeID := taskid.NewDefaultImplementationID[struct{}]("khi.google.com/fake-test-id")
	fakeTask := coretask.DefineConstant(fakeID, struct{}{})
	allTasks := make([]coretask.UntypedTask, 0, len(tasks)+1)
	allTasks = append(allTasks, fakeTask)
	for _, t := range tasks {
		if t.UntypedID().String() == fakeTask.UntypedID().String() {
			continue
		}
		allTasks = append(allTasks, t)
	}
	taskGraph := coretask.NewResolvedTaskSet(allTasks, nil, nil)
	return summary.NewCollector(taskGraph)
}

func setTestSummaryCollector(ctx context.Context, tasks ...coretask.UntypedTask) {
	writable := khictx.MustGetValue(ctx, writableMetadataContextKey)
	typedmap.Set(writable, summary.MetadataKey, newTestSummaryCollector(tasks...))
}

// WithDefaultTestInspectionTaskContext returns a new context used for running inspection task.
func WithDefaultTestInspectionTaskContext(baseContext context.Context) context.Context {
	taskCtx := khictx.WithValue(baseContext, inspectioncore.InspectionCreationTime, TestInspectionCreationTime)
	taskCtx = khictx.WithValue(taskCtx, inspectioncore.InspectionTaskInspectionID, "fake-inspection-id")
	taskCtx = khictx.WithValue(taskCtx, inspectioncore.InspectionTaskRunID, "fake-run-id")
	taskCtx = khictx.WithValue(taskCtx, inspectioncore.InspectionContext, baseContext)

	taskCtx = khictx.WithValue(taskCtx, inspectioncore.GlobalSharedMap, typedmap.NewTypedMap())
	taskCtx = khictx.WithValue(taskCtx, inspectioncore.InspectionSharedMap, typedmap.NewTypedMap())
	taskCtx = khictx.WithValue[inspectioncore.InspectionNameRegistry](taskCtx, inspectioncore.InspectionNameRegistryKey, inspectioncore.NewInMemoryInspectionNameRegistry())

	_, err := khictx.GetValue(taskCtx, core_contract.TaskImplementationIDContextKey)
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
	writableMetadata, readonlyMetadata := generateTestMetadata()
	taskCtx = khictx.WithValue(taskCtx, writableMetadataContextKey, writableMetadata)
	taskCtx = khictx.WithValue(taskCtx, inspectionmetadata.MapContextKey, readonlyMetadata)
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

// Run validates inputs against the inputs declared by task and runs the task in the given inspection mode.
// It fails t on invalid inputs in the same way as tasktest.Run. Use WithDefaultTestInspectionTaskContext to get the base context.
// It returns the task result, the inspection metadata and the error returned by the task.
func Run[T any](t testing.TB, baseContext context.Context, task coretask.Task[T], mode inspectioncore.InspectionTaskModeType, inspectionInput map[string]any, inputs ...tasktest.InputValue) (T, *typedmap.ReadonlyTypedMap, error) {
	t.Helper()
	metadata := khictx.MustGetValue(baseContext, inspectionmetadata.MapContextKey)
	setTestSummaryCollector(baseContext, task)
	taskCtx := khictx.WithValue(baseContext, inspectioncore.InspectionTaskInput, inspectionInput)
	taskCtx = khictx.WithValue(taskCtx, inspectioncore.InspectionTaskMode, mode)

	var result T
	_, err := progress.TaskInterceptor(taskCtx, task, func(ctx context.Context) (any, error) {
		var runErr error
		result, runErr = tasktest.Run(t, ctx, task, inputs...)
		return result, runErr
	})
	return result, metadata, err
}

// RunInspectionTaskWithDependency execute a task as a graph. Supply dependencies needed to be used with the mainTask.
func RunInspectionTaskWithDependency[T any](baseContext context.Context, mainTask coretask.Task[T], dependencies []coretask.UntypedTask, mode inspectioncore.InspectionTaskModeType, input map[string]any) (T, *typedmap.ReadonlyTypedMap, error) {
	allTasks := append([]coretask.UntypedTask{mainTask}, dependencies...)
	metadata := khictx.MustGetValue(baseContext, inspectionmetadata.MapContextKey)
	setTestSummaryCollector(baseContext, allTasks...)
	taskCtx := khictx.WithValue(baseContext, inspectioncore.InspectionTaskInput, input)
	taskCtx = khictx.WithValue(taskCtx, inspectioncore.InspectionTaskMode, mode)
	result, err := tasktest.RunTaskWithDependency(taskCtx, mainTask, dependencies, progress.TaskInterceptor)
	return result, metadata, err
}

func generateTestMetadata() (*typedmap.TypedMap, *typedmap.ReadonlyTypedMap) {
	writableMetadata := typedmap.NewTypedMap()
	typedmap.Set(writableMetadata, inspectionmetadata.HeaderMetadataKey, &inspectionmetadata.HeaderMetadata{})
	typedmap.Set(writableMetadata, inspectionmetadata.ErrorMessageSetMetadataKey, inspectionmetadata.NewErrorMessageSetMetadata())
	typedmap.Set(writableMetadata, inspectionmetadata.FormFieldSetMetadataKey, inspectionmetadata.NewFormFieldSetMetadata())
	typedmap.Set(writableMetadata, inspectionmetadata.QueryMetadataKey, inspectionmetadata.NewQueryMetadata())
	typedmap.Set(writableMetadata, inspectionmetadata.ProgressMetadataKey, inspectionmetadata.NewProgress())
	typedmap.Set(writableMetadata, summary.MetadataKey, newTestSummaryCollector())
	return writableMetadata, writableMetadata.AsReadonly()
}
