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

package googlecloudloggkeautoscaler_impl

import (
	"context"
	"fmt"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudloggkeautoscaler_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudloggkeautoscaler/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func generateAutoscalerQuery(cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity, excludeStatus bool) string {
	excludeStatusQueryFragment := "-- include query for status log"
	if excludeStatus {
		excludeStatusQueryFragment = `-jsonPayload.status: ""`
	}
	return fmt.Sprintf(`resource.type="k8s_cluster"
resource.labels.project_id="%s"
resource.labels.location="%s"
resource.labels.cluster_name="%s"
%s
logName="projects/%s/logs/container.googleapis.com%%2Fcluster-autoscaler-visibility"`, cluster.ProjectID, cluster.Location, cluster.NameWithClusterTypePrefix(), excludeStatusQueryFragment, cluster.ProjectID)
}

type autoscalerListLogEntriesTaskSetting struct{}

// DefaultResourceNames implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (a *autoscalerListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", cluster.ProjectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (a *autoscalerListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref(),
	}
}

// Description implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (a *autoscalerListLogEntriesTaskSetting) Description() *googlecloudcommon_contract.ListLogEntriesTaskDescription {
	return &googlecloudcommon_contract.ListLogEntriesTaskDescription{
		DefaultLogType: enum.LogTypeAutoscaler,
		QueryName:      "Cluster autoscaler logs",
		ExampleQuery: generateAutoscalerQuery(googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
			ProjectID:   "gcp-project-id",
			Location:    "gcp-location",
			ClusterName: "gcp-cluster-name",
		}, true),
	}
}

// LogFilters implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (a *autoscalerListLogEntriesTaskSetting) LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref())
	return []string{generateAutoscalerQuery(cluster, true)}, nil
}

// TaskID implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (a *autoscalerListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudloggkeautoscaler_contract.ListLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (a *autoscalerListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 1, nil
}

var _ googlecloudcommon_contract.ListLogEntriesTaskSetting = (*autoscalerListLogEntriesTaskSetting)(nil)

var ListLogEntriesTask = googlecloudcommon_contract.NewListLogEntriesTask(&autoscalerListLogEntriesTaskSetting{})
