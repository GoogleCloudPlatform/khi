// Copyright 2025 Google LLC
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

package googlecloudlogmulticloudapiaudit_impl

import (
	"context"
	"fmt"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogmulticloudapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogmulticloudapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// generateQuery generates a query for multicloud API logs.
func generateQuery(clusterNameWithPrefix string) string {
	return fmt.Sprintf(`resource.type="audited_resource"
resource.labels.service="gkemulticloud.googleapis.com"
resource.labels.method:("Update" OR "Create" OR "Delete")
protoPayload.resourceName:"%s"
`, clusterNameWithPrefix)
}

type multicloudAPIListLogEntriesTaskSetting struct {
}

// DefaultResourceNames implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (g *multicloudAPIListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", projectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (g *multicloudAPIListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
		googlecloudk8scommon_contract.InputClusterNameTaskID.Ref(),
	}
}

// Description implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (g *multicloudAPIListLogEntriesTaskSetting) Description() *googlecloudcommon_contract.ListLogEntriesTaskDescription {
	return &googlecloudcommon_contract.ListLogEntriesTaskDescription{
		DefaultLogType: enum.LogTypeMulticloudAPI,
		QueryName:      "Multicloud API Logs",
		ExampleQuery:   generateQuery("awsClusters/gcp-cluster-name"),
	}
}

// LogFilters implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (g *multicloudAPIListLogEntriesTaskSetting) LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	clusterName := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputClusterNameTaskID.Ref())

	return []string{generateQuery(clusterName)}, nil
}

// TaskID implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (g *multicloudAPIListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogmulticloudapiaudit_contract.ListLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (g *multicloudAPIListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 1, nil
}

var _ googlecloudcommon_contract.ListLogEntriesTaskSetting = (*multicloudAPIListLogEntriesTaskSetting)(nil)

var ListLogEntriesTask = googlecloudcommon_contract.NewListLogEntriesTask(&multicloudAPIListLogEntriesTaskSetting{})
