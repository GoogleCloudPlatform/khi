// Copyright 2026 Google LLC
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

package googlecloudlogcomposerapiaudit_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogcomposerapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcomposerapiaudit/contract"
)

// GenerateComposerAuditStructuredQuery generates a structured query for Cloud Composer audit logs.
func GenerateComposerAuditStructuredQuery(projectID, location, environmentName string) *logestimator.StructuredLogQuery {
	filters := []logestimator.LoggingMonitoringMatcher{
		logestimator.ResourceLabel("project_id", logestimator.Exact(projectID)),
	}
	if location != "" && location != "all" {
		filters = append(filters, logestimator.ResourceLabel("location", logestimator.Exact(location)))
	}
	if environmentName != "" {
		filters = append(filters, logestimator.ResourceLabel("environment_name", logestimator.Exact(environmentName)))
	}
	filters = append(filters,
		logestimator.LogID(logestimator.OneOf("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")),
		logestimator.CustomFilter(`protoPayload.serviceName="composer.googleapis.com"`),
	)

	return &logestimator.StructuredLogQuery{
		Incomplete:    projectID == "" || environmentName == "",
		ResourceTypes: []string{"cloud_composer_environment"},
		Filters:       filters,
	}
}

// GenerateComposerAuditQuery generates a Cloud Logging filter string for Cloud Composer audit logs.
func GenerateComposerAuditQuery(projectID, location, environmentName string) string {
	return GenerateComposerAuditStructuredQuery(projectID, location, environmentName).GenerateCloudLoggingQuery()
}

type composerAPIListLogEntriesTaskSetting struct{}

// DefaultResourceNames returns the resource name for Cloud Logging queries.
func (s *composerAPIListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudlogcomposerapiaudit_contract.ClusterIdentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", clusterIdentity.ProjectID)}, nil
}

// Dependencies returns task dependencies for query construction.
func (s *composerAPIListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudlogcomposerapiaudit_contract.ClusterIdentityTaskID.Ref(),
		googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref(),
	}
}

// QueryName returns human-readable name of the query.
func (s *composerAPIListLogEntriesTaskSetting) QueryName() string {
	return "Composer API Audit logs"
}

// Queries returns the list of structured log queries for estimation and execution.
func (s *composerAPIListLogEntriesTaskSetting) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudlogcomposerapiaudit_contract.ClusterIdentityTaskID.Ref())
	environmentName := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref())
	return []*logestimator.StructuredLogQuery{GenerateComposerAuditStructuredQuery(clusterIdentity.ProjectID, clusterIdentity.Location, environmentName)}, nil
}

// TaskID returns the implementation ID of this query task.
func (s *composerAPIListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogcomposerapiaudit_contract.ListLogEntriesTaskID
}

// TimePartitionCount returns the number of time partitions to use when querying logs.
func (s *composerAPIListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 1, nil
}

var _ googlecloudcommon_contract.StructuredListLogEntriesTaskSetting = (*composerAPIListLogEntriesTaskSetting)(nil)

// ListLogEntriesTask is the task that queries Cloud Composer audit logs from Cloud Logging.
var ListLogEntriesTask = googlecloudcommon_contract.NewStructuredListLogEntriesTask(&composerAPIListLogEntriesTaskSetting{})
