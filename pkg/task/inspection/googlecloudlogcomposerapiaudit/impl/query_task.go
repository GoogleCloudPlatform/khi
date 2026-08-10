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

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogcomposerapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcomposerapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// GenerateComposerAuditQuery generates a Cloud Logging filter string for Cloud Composer audit logs.
func GenerateComposerAuditQuery(projectID, location, environmentName string) string {
	locationFilter := ""
	if location != "" && location != "all" {
		locationFilter = fmt.Sprintf("resource.labels.location=\"%s\"\n", location)
	}
	environmentFilter := ""
	if environmentName != "" {
		environmentFilter = fmt.Sprintf("resource.labels.environment_name=\"%s\"\n", environmentName)
	}

	return fmt.Sprintf(`(log_id("cloudaudit.googleapis.com/activity") OR log_id("cloudaudit.googleapis.com/data_access"))
resource.type="cloud_composer_environment"
resource.labels.project_id="%s"
%s%sprotoPayload.serviceName="composer.googleapis.com"
`, projectID, locationFilter, environmentFilter)
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

// Description returns user-facing task metadata and example query.
func (s *composerAPIListLogEntriesTaskSetting) Description() *googlecloudcommon_contract.ListLogEntriesTaskDescription {
	return &googlecloudcommon_contract.ListLogEntriesTaskDescription{
		QueryName:    "Composer API Audit logs",
		ExampleQuery: GenerateComposerAuditQuery("test-project", "us-central1", "test-environment"),
	}
}

// LogFilters returns the Cloud Logging query string.
func (s *composerAPIListLogEntriesTaskSetting) LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudlogcomposerapiaudit_contract.ClusterIdentityTaskID.Ref())
	environmentName := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref())
	return []string{GenerateComposerAuditQuery(clusterIdentity.ProjectID, clusterIdentity.Location, environmentName)}, nil
}

// TaskID returns the implementation ID of this query task.
func (s *composerAPIListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogcomposerapiaudit_contract.ListLogEntriesTaskID
}

// TimePartitionCount returns the number of time partitions to use when querying logs.
func (s *composerAPIListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 1, nil
}

var _ googlecloudcommon_contract.ListLogEntriesTaskSetting = (*composerAPIListLogEntriesTaskSetting)(nil)

// ListLogEntriesTask is the task that queries Cloud Composer audit logs from Cloud Logging.
var ListLogEntriesTask = googlecloudcommon_contract.NewListLogEntriesTask(&composerAPIListLogEntriesTaskSetting{})
