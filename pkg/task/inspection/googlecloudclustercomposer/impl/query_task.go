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
	"slices"
	"strings"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

type composerListLogEntriesTaskSetting struct {
	taskId    taskid.TaskImplementationID[[]*log.Log]
	queryName string
}

// DefaultResourceNames implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *composerListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.ClusterIdentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", clusterIdentity.ProjectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *composerListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudclustercomposer_contract.ClusterIdentityTaskID.Ref(),
		googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref(),
		googlecloudclustercomposer_contract.InputComposerComponentsTaskID.Ref(),
	}
}

// Description implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *composerListLogEntriesTaskSetting) Description() *googlecloudcommon_contract.ListLogEntriesTaskDescription {
	return &googlecloudcommon_contract.ListLogEntriesTaskDescription{
		DefaultLogType: enum.LogTypeComposerEnvironment,
		QueryName:      c.queryName,
		ExampleQuery:   generateExampleQuery("test-project", "sample-composer-environment"),
	}
}

// LogFilters implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *composerListLogEntriesTaskSetting) LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.ClusterIdentityTaskID.Ref())
	environmentName := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref())
	selectedComponents := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.InputComposerComponentsTaskID.Ref())

	logIds := []string{}
	logIDSelector := "-- no component filter specified. fetching all logs."
	if !slices.Contains(selectedComponents, "@any") {
		logIds = append(logIds, selectedComponents...)
		for i := range logIds {
			logIds[i] = fmt.Sprintf(`log_id("%s")`, logIds[i])
		}
		logIDSelector = fmt.Sprintf("(%s)", strings.Join(logIds, " OR "))
	}

	return []string{fmt.Sprintf(`%s
resource.type="cloud_composer_environment"
resource.labels.project_id="%s"
resource.labels.location="%s"
resource.labels.environment_name="%s"`, logIDSelector, clusterIdentity.ProjectID, clusterIdentity.Location, environmentName)}, nil
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

// ComposerLogsQueryTask defines a task that gathers logs from Cloud Logging for multiple Composer components.
var ComposerLogsQueryTask = googlecloudcommon_contract.NewListLogEntriesTask(&composerListLogEntriesTaskSetting{
	taskId:    googlecloudclustercomposer_contract.ComposerLogsQueryTaskID,
	queryName: "Composer Environment Logs",
})

func generateExampleQuery(projectId string, environmentName string) string {
	composerFilter := composerEnvironmentLog(projectId, environmentName)
	return fmt.Sprintf(`(log_id("airflow-worker") OR log_id("worker") OR log_id("airflow-scheduler") OR log_id("scheduler"))
%s`, composerFilter)
}

func composerEnvironmentLog(projectId string, environmentName string) string {
	return fmt.Sprintf(`resource.type="cloud_composer_environment"
resource.labels.project_id="%s"
resource.labels.environment_name="%s"`, projectId, environmentName)
}
