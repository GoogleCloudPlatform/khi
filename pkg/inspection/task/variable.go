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
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

const (
	MetadataVariableName          = InspectionTaskPrefix + "metadata"
	InspectionResultVariableName  = InspectionTaskPrefix + "inspection-result"
	InspectionRequestVariableName = InspectionTaskPrefix + "request"
	InspectionIdVariableName      = InspectionTaskPrefix + "inspection-id"
)

func GetHistoryBuilderFromTaskVariable(v *task.VariableSet) (*history.Builder, error) {
	return task.GetTypedVariableFromTaskVariable[*history.Builder](v, BuilderGeneratorTaskID, nil)
}

func GetInspectionIdFromTaskVariable(v *task.VariableSet) (string, error) {
	return task.GetTypedVariableFromTaskVariable[string](v, InspectionIdVariableName, "<INVALID>")
}
