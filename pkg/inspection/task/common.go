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
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	common_task "github.com/GoogleCloudPlatform/khi/pkg/task"
)

type InspectionRequest struct {
	Values map[string]any
}

var InspectionTimeTaskID = InspectionTaskPrefix + "task/time"

// InspectionTimeProducer is a provider of inspection time.
// Tasks shouldn't use time.Now() directly to make test easier.
var InspectionTimeProducer common_task.Definition = common_task.NewProcessorTask(InspectionTimeTaskID, []string{}, func(ctx context.Context, taskMode int, v *common_task.VariableSet) (any, error) {
	return time.Now(), nil
})

// TestInspectionTimeTaskProducer is a function to generate a fake InspectionTimeProducer task with the given time string.
var TestInspectionTimeTaskProducer func(timeStr string) common_task.Definition = func(timeStr string) common_task.Definition {
	return common_task.NewProcessorTask(InspectionTimeTaskID, []string{}, func(ctx context.Context, taskMode int, v *common_task.VariableSet) (any, error) {
		time, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return nil, err
		}
		return time, nil
	})
}

func GetMetadataSetFromVariable(v *common_task.VariableSet) (*typedmap.ReadonlyTypedMap, error) {
	return common_task.GetTypedVariableFromTaskVariable[*typedmap.ReadonlyTypedMap](v, MetadataVariableName, nil)
}

func GetInspectionRequestFromVariable(v *common_task.VariableSet) (*InspectionRequest, error) {
	return common_task.GetTypedVariableFromTaskVariable[*InspectionRequest](v, InspectionRequestVariableName, nil)
}

func GetInspectionTimeFromTaskVariable(v *common_task.VariableSet) (time.Time, error) {
	return common_task.GetTypedVariableFromTaskVariable[time.Time](v, InspectionTimeTaskID, time.Time{})
}
