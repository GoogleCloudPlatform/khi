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

package env

import (
	"context"
	"fmt"
	"os"

	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

const EnvironmentVariableKeyLabel = task.KHISystemPrefix + "/labels/environment-variable-key"
const EnvironmentVariableDefaultValueLabel = task.KHISystemPrefix + "/labels/environment-variable-default-value"

type EnvironmentVariable struct {
	Value  string
	Exists bool
}

// Digest implements task.CachableDependency.
func (ev *EnvironmentVariable) Digest() string {
	return fmt.Sprintf("%t-%s", ev.Exists, ev.Value)
}

var _ task.CachableDependency = (*EnvironmentVariable)(nil)

func GetEnvironmentVariableFromTaskVariables(ctx context.Context, environmentVariableName string, v *task.VariableSet) (*EnvironmentVariable, error) {
	envAny, err := v.Get(environmentVariableName)
	if err != nil {
		return nil, err
	}
	return envAny.(*EnvironmentVariable), nil
}

// EnvironmentVariableProducer creates a producer task from given arguments that is resolved with environment variable values
func EnvironmentVariableProducer(taskId string, environmentKey string, defaultValue string) task.Definition {
	return task.NewProcessorTask(taskId, []string{}, func(ctx context.Context, taskMode int, v *task.VariableSet) (any, error) {
		value, exists := os.LookupEnv(environmentKey)
		if !exists {
			value = defaultValue
		}
		return &EnvironmentVariable{
			Value:  value,
			Exists: exists,
		}, nil
	}, task.WithLabel(EnvironmentVariableKeyLabel, environmentKey), task.WithLabel(EnvironmentVariableDefaultValueLabel, defaultValue))
}
