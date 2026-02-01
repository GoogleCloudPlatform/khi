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

package googlecloudloggkeapiaudit_impl

import (
	"context"
	"fmt"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudloggkeapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudloggkeapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func GenerateGKEAuditQuery(cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity) string {
	return fmt.Sprintf(`log_id("cloudaudit.googleapis.com/activity") OR log_id("cloudaudit.googleapis.com/data_access")
resource.type=("gke_cluster" OR "gke_nodepool")
resource.labels.project_id="%s"
resource.labels.location="%s"
resource.labels.cluster_name="%s"
protoPayload.serviceName="container.googleapis.com"
`, cluster.ProjectID, cluster.Location, cluster.ClusterName)
}

type gkeAPIListLogEntriesTaskSetting struct {
}

// DefaultResourceNames implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (g *gkeAPIListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", projectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (g *gkeAPIListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
		googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref(),
	}
}

// Description implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (g *gkeAPIListLogEntriesTaskSetting) Description() *googlecloudcommon_contract.ListLogEntriesTaskDescription {
	return &googlecloudcommon_contract.ListLogEntriesTaskDescription{
		DefaultLogType: enum.LogTypeGkeAudit,
		QueryName:      "GKE Audit logs",
		ExampleQuery:   GenerateGKEAuditQuery(googlecloudk8scommon_contract.GoogleCloudClusterIdentity{ProjectID: "gcp-project-id", Location: "gcp-location", ClusterName: "gcp-cluster-name"}),
	}
}

// LogFilters implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (g *gkeAPIListLogEntriesTaskSetting) LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref())
	return []string{GenerateGKEAuditQuery(cluster)}, nil
}

// TaskID implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (g *gkeAPIListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudloggkeapiaudit_contract.ListLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (g *gkeAPIListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 1, nil
}

var _ googlecloudcommon_contract.ListLogEntriesTaskSetting = (*gkeAPIListLogEntriesTaskSetting)(nil)

var ListLogEntriesTask = googlecloudcommon_contract.NewListLogEntriesTask(&gkeAPIListLogEntriesTaskSetting{})
