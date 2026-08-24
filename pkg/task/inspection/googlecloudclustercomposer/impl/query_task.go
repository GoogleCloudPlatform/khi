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

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
)

// GenerateComposerLogsStructuredQuery generates a structured query for Composer environment logs.
func GenerateComposerLogsStructuredQuery(projectID, location, environmentName string, selectedComponents []string) *logestimator.StructuredLogQuery {
	filters := []logestimator.LoggingMonitoringMatcher{
		logestimator.ResourceLabel("project_id", logestimator.Exact(projectID)),
		logestimator.ResourceLabel("location", logestimator.Exact(location)),
		logestimator.ResourceLabel("environment_name", logestimator.Exact(environmentName)),
	}

	if !slices.Contains(selectedComponents, "@any") && len(selectedComponents) > 0 {
		filters = append(filters, logestimator.LogID(logestimator.OneOf(selectedComponents...)))
	}

	filters = append(filters, logestimator.LogID(logestimator.NoneOf("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")))

	return &logestimator.StructuredLogQuery{
		Incomplete:    projectID == "" || location == "" || environmentName == "",
		ResourceTypes: []string{"cloud_composer_environment"},
		Filters:       filters,
	}
}

// GenerateComposerLogsQuery generates a query for Composer environment logs.
func GenerateComposerLogsQuery(projectID, location, environmentName string, selectedComponents []string) string {
	return GenerateComposerLogsStructuredQuery(projectID, location, environmentName, selectedComponents).GenerateCloudLoggingQuery()
}

type composerListLogEntriesTaskSetting struct {
	taskId    taskid.TaskImplementationID[[]*log.Log]
	queryName string
}

// DefaultResourceNames implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *composerListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.ClusterIdentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", clusterIdentity.ProjectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *composerListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudclustercomposer_contract.ClusterIdentityTaskID.Ref(),
		googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref(),
		googlecloudclustercomposer_contract.InputComposerComponentsTaskID.Ref(),
	}
}

// QueryName implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *composerListLogEntriesTaskSetting) QueryName() string {
	return c.queryName
}

// Queries implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *composerListLogEntriesTaskSetting) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.ClusterIdentityTaskID.Ref())
	environmentName := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref())
	selectedComponents := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.InputComposerComponentsTaskID.Ref())

	return []*logestimator.StructuredLogQuery{GenerateComposerLogsStructuredQuery(clusterIdentity.ProjectID, clusterIdentity.Location, environmentName, selectedComponents)}, nil
}

// TaskID implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *composerListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return c.taskId
}

// TimePartitionCount implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *composerListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ googlecloudcommon_contract.StructuredListLogEntriesTaskSetting = (*composerListLogEntriesTaskSetting)(nil)

// ComposerLogsQueryTask defines a task that gathers logs from Cloud Logging for multiple Composer components.
var ComposerLogsQueryTask = googlecloudcommon_contract.NewStructuredListLogEntriesTask(&composerListLogEntriesTaskSetting{
	taskId:    googlecloudclustercomposer_contract.ComposerLogsQueryTaskID,
	queryName: "Composer Environment Logs",
})
