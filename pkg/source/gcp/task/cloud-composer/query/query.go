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

package composer_query

import (
	"context"
	"fmt"

	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	composer_form "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer/form"
	composer_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var ComposerSchedulerLogQueryTask = query.NewQueryGeneratorTask(
	composer_taskid.ComposerSchedulerLogQueryTaskID,
	"Composer Environment/Airflow Scheduler",
	enum.LogTypeComposerEnvironment,
	[]taskid.UntypedTaskReference{
		gcp_task.InputProjectIdTaskID,
		composer_taskid.InputComposerEnvironmentTaskID,
	},
	&query.ProjectIDDefaultResourceNamesGenerator{},
	createGenerator("airflow-scheduler"),
	generateQueryForComponent("sample-composer-environment", "test-project", "airflow-scheduler"),
)

var ComposerDagProcessorManagerLogQueryTask = query.NewQueryGeneratorTask(
	composer_taskid.ComposerDagProcessorManagerLogQueryTaskID,
	"Composer Environment/DAG Processor Manager",
	enum.LogTypeComposerEnvironment,
	[]taskid.UntypedTaskReference{
		gcp_task.InputProjectIdTaskID,
		composer_taskid.InputComposerEnvironmentTaskID,
	},
	&query.ProjectIDDefaultResourceNamesGenerator{},
	createGenerator("dag-processor-manager"),
	generateQueryForComponent("sample-composer-environment", "test-project", "dag-processor-manager"),
)

var ComposerMonitoringLogQueryTask = query.NewQueryGeneratorTask(
	composer_taskid.ComposerMonitoringLogQueryTaskID,
	"Composer Environment/Airflow Monitoring",
	enum.LogTypeComposerEnvironment,
	[]taskid.UntypedTaskReference{
		gcp_task.InputProjectIdTaskID,
		composer_taskid.InputComposerEnvironmentTaskID,
	},
	&query.ProjectIDDefaultResourceNamesGenerator{},
	createGenerator("airflow-monitoring"),
	generateQueryForComponent("sample-composer-environment", "test-project", "airflow-monitoring"),
)

var ComposerWorkerLogQueryTask = query.NewQueryGeneratorTask(
	composer_taskid.ComposerWorkerLogQueryTaskID,
	"Composer Environment/Airflow Worker",
	enum.LogTypeComposerEnvironment,
	[]taskid.UntypedTaskReference{
		gcp_task.InputProjectIdTaskID,
		composer_taskid.InputComposerEnvironmentTaskID,
	},
	&query.ProjectIDDefaultResourceNamesGenerator{},
	createGenerator("airflow-worker"),
	generateQueryForComponent("sample-composer-environment", "test-project", "airflow-worker"),
)

func createGenerator(componentName string) func(ctx context.Context, i inspection_task_interface.InspectionTaskMode) ([]string, error) {
	// This function will generate a Cloud Logging query like;
	// resource.type="cloud_composer_environment"
	// resource.labels.environment_name="ENVIRONMENT_NAME"
	// log_name=projects/PROJECT_ID/logs/COMPONENT_NAME
	return func(ctx context.Context, i inspection_task_interface.InspectionTaskMode) ([]string, error) {
		projectID := task.GetTaskResult(ctx, gcp_task.InputProjectIdTaskID.GetTaskReference())
		environmentName := task.GetTaskResult(ctx, composer_form.InputComposerEnvironmentNameTask.ID().GetTaskReference())
		return []string{generateQueryForComponent(environmentName, projectID, componentName)}, nil
	}
}

func generateQueryForComponent(environmentName string, projectId string, componentName string) string {
	composerFilter := composerEnvironmentLog(environmentName)
	schedulerFilter := logPath(projectId, componentName)
	return fmt.Sprintf(`%s
%s`, composerFilter, schedulerFilter)
}

func composerEnvironmentLog(environmentName string) string {
	return fmt.Sprintf(`resource.type="cloud_composer_environment"
resource.labels.environment_name="%s"`, environmentName)
}

func logPath(projectId string, logName string) string {
	// log_name=projects/PROJECT_ID/logs/dag-processor-manager
	return fmt.Sprintf(`log_name=projects/%s/logs/%s`, projectId, logName)
}
