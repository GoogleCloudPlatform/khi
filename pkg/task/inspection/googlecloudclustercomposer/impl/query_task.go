package googlecloudclustercomposer_impl

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

import (
	"context"
	"fmt"

	queryutil "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// ComposerSchedulerLogQueryTask defines a task that gather Cloud Composer scheduler logs from Cloud Logging.
var ComposerSchedulerLogQueryTask = queryutil.NewCloudLoggingListLogTask(
	googlecloudclustercomposer_contract.ComposerSchedulerLogQueryTaskID,
	"Composer Environment/Airflow Scheduler",
	enum.LogTypeComposerEnvironment,
	[]taskid.UntypedTaskReference{
		googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
		googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref(),
	},
	&queryutil.ProjectIDDefaultResourceNamesGenerator{},
	createGenerator("airflow-scheduler"),
	generateQueryForComponent("sample-composer-environment", "test-project", "airflow-scheduler"),
)

// ComposerDagProcessorManagerLogQueryTask defines a task that gather Cloud Composer scheduler logs from Cloud Logging.
var ComposerDagProcessorManagerLogQueryTask = queryutil.NewCloudLoggingListLogTask(
	googlecloudclustercomposer_contract.ComposerDagProcessorManagerLogQueryTaskID,
	"Composer Environment/DAG Processor Manager",
	enum.LogTypeComposerEnvironment,
	[]taskid.UntypedTaskReference{
		googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
		googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref(),
	},
	&queryutil.ProjectIDDefaultResourceNamesGenerator{},
	createGenerator("dag-processor-manager"),
	generateQueryForComponent("sample-composer-environment", "test-project", "dag-processor-manager"),
)

// ComposerMonitoringLogQueryTask defines a task that gather Cloud Composer scheduler logs from Cloud Logging.
var ComposerMonitoringLogQueryTask = queryutil.NewCloudLoggingListLogTask(
	googlecloudclustercomposer_contract.ComposerMonitoringLogQueryTaskID,
	"Composer Environment/Airflow Monitoring",
	enum.LogTypeComposerEnvironment,
	[]taskid.UntypedTaskReference{
		googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
		googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref(),
	},
	&queryutil.ProjectIDDefaultResourceNamesGenerator{},
	createGenerator("airflow-monitoring"),
	generateQueryForComponent("sample-composer-environment", "test-project", "airflow-monitoring"),
)

// ComposerWorkerLogQueryTask defines a task that gather Cloud Composer scheduler logs from Cloud Logging.
var ComposerWorkerLogQueryTask = queryutil.NewCloudLoggingListLogTask(
	googlecloudclustercomposer_contract.ComposerWorkerLogQueryTaskID,
	"Composer Environment/Airflow Worker",
	enum.LogTypeComposerEnvironment,
	[]taskid.UntypedTaskReference{
		googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
		googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref(),
	},
	&queryutil.ProjectIDDefaultResourceNamesGenerator{},
	createGenerator("airflow-worker"),
	generateQueryForComponent("sample-composer-environment", "test-project", "airflow-worker"),
)

func createGenerator(componentName string) func(ctx context.Context, i inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	// This function will generate a Cloud Logging query like;
	// resource.type="cloud_composer_environment"
	// resource.labels.environment_name="ENVIRONMENT_NAME"
	// log_name=projects/PROJECT_ID/logs/COMPONENT_NAME
	return func(ctx context.Context, i inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
		projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
		environmentName := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref())
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
