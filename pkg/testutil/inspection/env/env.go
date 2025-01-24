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

package env_test

import (
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/env"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	task_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/task"
)

// MockedEnvironmentVariableProducer creates a mocked producer task from the EnvironmentVariableProducer.
// The environment variable is resolved with the default value when the value is empty string.
// This is only for testing purpose.
func MockedEnvironmentVariableProducer(parentVariableDefinition task.Definition, value string) task.Definition {
	dv, found := parentVariableDefinition.Labels().Get(env.EnvironmentVariableDefaultValueLabel)
	if !found {
		panic(fmt.Errorf("the given parent variable definition is not declared with EnvironmentVariableProducer"))
	}
	if value == "" {
		return task_test.MockProcessorTaskFromTaskID(parentVariableDefinition.ID().String(), &env.EnvironmentVariable{
			Value:  dv.(string),
			Exists: false,
		})
	} else {
		return task_test.MockProcessorTaskFromTaskID(parentVariableDefinition.ID().String(), &env.EnvironmentVariable{
			Value:  value,
			Exists: true,
		})
	}
}
