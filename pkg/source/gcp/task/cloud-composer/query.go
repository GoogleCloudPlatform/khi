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

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

const ComposerQueryPrefix = gcp_task.GCPPrefix + "query/composer/"

const ComposerSchedulerLogQueryTaskName = ComposerQueryPrefix + "scheduler"
const ComposerDagProcessorManagerLogQueryTaskName = ComposerQueryPrefix + "dag-processor-manager"
const ComposerMonitoringLogQueryTaskName = ComposerQueryPrefix + "monitoring"
const ComposerWorkerLogQueryTaskName = ComposerQueryPrefix + "worker"

var ComposerSchedulerLogQueryTask = query.NewQueryGeneratorTask(
	ComposerSchedulerLogQueryTaskName,
	"Composer Environment/Airflow Scheduler",
	enum.LogTypeComposerEnvironment,
	[]string{
		gcp_task.InputProjectIdTaskID,
		InputComposerEnvironmentTaskID,
	},
	createGenerator("airflow-scheduler"),
)

var ComposerDagProcessorManagerLogQueryTask = query.NewQueryGeneratorTask(
	ComposerDagProcessorManagerLogQueryTaskName,
	"Composer Environment/DAG Processor Manager",
	enum.LogTypeComposerEnvironment,
	[]string{
		gcp_task.InputProjectIdTaskID,
		InputComposerEnvironmentTaskID,
	},
	createGenerator("dag-processor-manager"),
)

var ComposerMonitoringLogQueryTask = query.NewQueryGeneratorTask(
	ComposerMonitoringLogQueryTaskName,
	"Composer Environment/Airflow Monitoring",
	enum.LogTypeComposerEnvironment,
	[]string{
		gcp_task.InputProjectIdTaskID,
		InputComposerEnvironmentTaskID,
	},
	createGenerator("airflow-monitoring"),
)

var ComposerWorkerLogQueryTask = query.NewQueryGeneratorTask(
	ComposerWorkerLogQueryTaskName,
	"Composer Environment/Airflow Worker",
	enum.LogTypeComposerEnvironment,
	[]string{
		gcp_task.InputProjectIdTaskID,
		InputComposerEnvironmentTaskID,
	},
	createGenerator("airflow-worker"),
)

func createGenerator(componentName string) func(ctx context.Context, i int, vs *task.VariableSet) ([]string, error) {
	// This function will generate a Cloud Logging query like;
	// resource.type="cloud_composer_environment"
	// resource.labels.environment_name="ENVIRONMENT_NAME"
	// log_name=projects/PROJECT_ID/logs/COMPONENT_NAME
	return func(ctx context.Context, i int, vs *task.VariableSet) ([]string, error) {
		projectId, err := gcp_task.GetInputProjectIdFromTaskVariable(vs)
		if err != nil {
			return []string{}, err
		}
		environmentName, err := GetInputComposerEnvironmentVariable(vs)
		if err != nil {
			return []string{}, err
		}

		composerFilter := composerEnvironmentLog(environmentName)
		schedulerFilter := logPath(projectId, componentName)

		return []string{fmt.Sprintf(`%s
%s`, composerFilter, schedulerFilter)}, nil
	}
}

func composerEnvironmentLog(environmentName string) string {
	return fmt.Sprintf(`resource.type="cloud_composer_environment"
resource.labels.environment_name="%s"`, environmentName)
}

func logPath(projectId string, logName string) string {
	// log_name=projects/PROJECT_ID/logs/dag-processor-manager
	return fmt.Sprintf(`log_name=projects/%s/logs/%s`, projectId, logName)
}
