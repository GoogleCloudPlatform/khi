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

package composer_task

import (
	"context"
	"fmt"
	"testing"

	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

func TestCreateGeneratorCreatesComposerQuery(t *testing.T) {
	ctx := context.Background()
	projectId := "test-project"
	environmentName := "test-environment"
	vs := &task.VariableSet{}
	vs.Set(gcp_task.InputProjectIdTaskID, projectId)
	vs.Set(InputComposerEnvironmentTaskID, environmentName)

	// resource.type="cloud_composer_environment"
	// resource.labels.environment_name="test-environment"
	// log_name=projects/test-project/logs/airflow-scheduler
	expected := fmt.Sprintf(`resource.type="cloud_composer_environment"
resource.labels.environment_name="test-environment"
log_name=projects/%s/logs/airflow-scheduler`, projectId)

	taskMode := 0                                     // any int is fine
	generator := createGenerator("airflow-scheduler") // sample: airflow-scheduler
	actual, err := generator(ctx, taskMode, vs)
	if err != nil {
		t.Fatalf("GenerateQuery: %v", err)
	}
	if len(actual) != 1 {
		t.Errorf("Unexpected query count %d", len(actual))
	}
	if actual[0] != expected {
		t.Errorf("GenerateQuery: expected %q, got %q", expected, actual)
	}
}
