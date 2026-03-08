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

package googlecloudclustercomposer_impl

import (
	"context"
	"fmt"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

type composerListLogEntriesTaskSetting struct {
	taskId        taskid.TaskImplementationID[[]*log.Log]
	queryName     string
	componentName string
}

// DefaultResourceNames implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *composerListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", projectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *composerListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
		googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref(),
	}
}

// Description implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *composerListLogEntriesTaskSetting) Description() *googlecloudcommon_contract.ListLogEntriesTaskDescription {
	return &googlecloudcommon_contract.ListLogEntriesTaskDescription{
		DefaultLogType: enum.LogTypeComposerEnvironment,
		QueryName:      c.queryName,
		ExampleQuery:   generateQueryForComponent("test-project", "sample-composer-environment", c.componentName),
	}
}

// LogFilters implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *composerListLogEntriesTaskSetting) LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	environmentName := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref())
	return []string{generateQueryForComponent(projectID, environmentName, c.componentName)}, nil
}

// TaskID implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *composerListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return c.taskId
}

// TimePartitionCount implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *composerListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ googlecloudcommon_contract.ListLogEntriesTaskSetting = (*composerListLogEntriesTaskSetting)(nil)

// ComposerSchedulerLogQueryTask defines a task that gather Cloud Composer scheduler logs from Cloud Logging.
var ComposerSchedulerLogQueryTask = googlecloudcommon_contract.NewListLogEntriesTask(&composerListLogEntriesTaskSetting{
	taskId:        googlecloudclustercomposer_contract.ComposerSchedulerLogQueryTaskID,
	queryName:     "Composer Environment/Airflow Scheduler",
	componentName: "airflow-scheduler",
})

// ComposerDagProcessorManagerLogQueryTask defines a task that gather Cloud Composer scheduler logs from Cloud Logging.
var ComposerDagProcessorManagerLogQueryTask = googlecloudcommon_contract.NewListLogEntriesTask(&composerListLogEntriesTaskSetting{
	taskId:        googlecloudclustercomposer_contract.ComposerDagProcessorManagerLogQueryTaskID,
	queryName:     "Composer Environment/DAG Processor Manager",
	componentName: "dag-processor-manager",
})

// ComposerMonitoringLogQueryTask defines a task that gather Cloud Composer scheduler logs from Cloud Logging.
var ComposerMonitoringLogQueryTask = googlecloudcommon_contract.NewListLogEntriesTask(&composerListLogEntriesTaskSetting{
	taskId:        googlecloudclustercomposer_contract.ComposerMonitoringLogQueryTaskID,
	queryName:     "Composer Environment/Airflow Monitoring",
	componentName: "airflow-monitoring",
})

// ComposerWorkerLogQueryTask defines a task that gather Cloud Composer scheduler logs from Cloud Logging.
var ComposerWorkerLogQueryTask = googlecloudcommon_contract.NewListLogEntriesTask(&composerListLogEntriesTaskSetting{
	taskId:        googlecloudclustercomposer_contract.ComposerWorkerLogQueryTaskID,
	queryName:     "Composer Environment/Airflow Worker",
	componentName: "airflow-worker",
})

func generateQueryForComponent(projectId string, environmentName string, componentName string) string {
	composerFilter := composerEnvironmentLog(projectId, environmentName)
	schedulerFilter := logPath(componentName)
	return fmt.Sprintf(`%s
%s`, schedulerFilter, composerFilter)
}

func composerEnvironmentLog(projectId string, environmentName string) string {
	return fmt.Sprintf(`resource.type="cloud_composer_environment"
resource.labels.project_id="%s"
resource.labels.environment_name="%s"`, projectId, environmentName)
}

func logPath(logName string) string {
	return fmt.Sprintf(`log_id("%s")`, logName)
}
