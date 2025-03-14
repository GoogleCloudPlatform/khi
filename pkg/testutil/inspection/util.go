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

package inspection_test

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/inspectiondata"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/form"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/header"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	task_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/task"
)

// DryRunInspectionTaskGraph executes the task graph just with provided dependency tasks with several variables given in default local runner
func DryRunInspectionTaskGraph(target task.Definition, requestParams map[string]any, dependencies ...task.Definition) (*task.VariableSet, error) {
	ms := typedmap.NewTypedMap()
	typedmap.Set(ms, header.HeaderMetadataKey, &header.Header{})
	typedmap.Set(ms, form.FormFieldSetMetadataKey, form.NewFormFieldSet())

	return task_test.RunTaskGraph(target, inspection_task.TaskModeDryRun, map[string]any{
		inspection_task.MetadataVariableName: ms.AsReadonly(),
		inspection_task.InspectionRequestVariableName: &inspection_task.InspectionRequest{
			Values: requestParams,
		},
	}, dependencies...)
}

// RunInspectionTaskGraph executes the task graph just with provided dependency tasks with several variables given in the default local runner
func RunInspectionTaskGraph(target task.Definition, requestParams map[string]any, dependencies ...task.Definition) (*task.VariableSet, error) {
	ms := typedmap.NewTypedMap()
	typedmap.Set(ms, header.HeaderMetadataKey, &header.Header{})
	typedmap.Set(ms, form.FormFieldSetMetadataKey, form.NewFormFieldSet())

	return task_test.RunTaskGraph(target, inspection_task.TaskModeRun, map[string]any{
		inspection_task.MetadataVariableName: ms.AsReadonly(),
		inspection_task.InspectionRequestVariableName: &inspection_task.InspectionRequest{
			Values: requestParams,
		},
		inspection_task.InspectionResultVariableName: inspectiondata.NewFileSystemInspectionResultRepository("foo"),
	}, dependencies...)
}
