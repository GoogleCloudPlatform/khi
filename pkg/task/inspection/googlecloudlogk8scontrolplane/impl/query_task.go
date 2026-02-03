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

package googlecloudlogk8scontrolplane_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8scontrolplane_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func GenerateK8sControlPlaneQuery(cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity, controlplaneComponentFilter *gcpqueryutil.SetFilterParseResult) string {
	return fmt.Sprintf(`resource.type="k8s_control_plane_component"
resource.labels.project_id="%s"
resource.labels.location="%s"
resource.labels.cluster_name="%s"
-sourceLocation.file="httplog.go" -- Ignoring the noisy log from scheduler. TODO: Support toggling this feature.
%s`, cluster.ProjectID, cluster.Location, cluster.NameWithClusterTypePrefix(), generateK8sControlPlaneComponentFilter(controlplaneComponentFilter))
}

func generateK8sControlPlaneComponentFilter(filter *gcpqueryutil.SetFilterParseResult) string {
	if filter.ValidationError != "" {
		return fmt.Sprintf(`-- Failed to generate component name filter due to the validation error "%s"`, filter.ValidationError)
	}
	if filter.SubtractMode {
		if len(filter.Subtractives) == 0 {
			return "-- No component name filter"
		}
		return fmt.Sprintf(`-resource.labels.component_name:(%s)`, strings.Join(gcpqueryutil.WrapDoubleQuoteForStringArray(filter.Subtractives), " OR "))
	} else {
		if len(filter.Additives) == 0 {
			return `-- Invalid: none of the controlplane component will be selected. Ignoreing component name filter.`
		}
		return fmt.Sprintf(`resource.labels.component_name:(%s)`, strings.Join(gcpqueryutil.WrapDoubleQuoteForStringArray(filter.Additives), " OR "))
	}
}

type controlPlaneListLogEntriesTaskSetting struct {
}

// Dependencies implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *controlPlaneListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref(),
		googlecloudlogk8scontrolplane_contract.InputControlPlaneComponentNameFilterTaskID.Ref(),
	}
}

// DefaultResourceNames implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *controlPlaneListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", cluster.ProjectID)}, nil
}

// Description implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *controlPlaneListLogEntriesTaskSetting) Description() *googlecloudcommon_contract.ListLogEntriesTaskDescription {
	return &googlecloudcommon_contract.ListLogEntriesTaskDescription{
		DefaultLogType: enum.LogTypeControlPlaneComponent,
		QueryName:      "K8s control plane logs",
		ExampleQuery: GenerateK8sControlPlaneQuery(googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
			ProjectID:   "test-project",
			ClusterName: "test-cluster",
			Location:    "asia-northeast1",
		}, &gcpqueryutil.SetFilterParseResult{
			SubtractMode: true,
		}),
	}
}

// LogFilters implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *controlPlaneListLogEntriesTaskSetting) LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref())
	controlplaneComponentNameFilter := coretask.GetTaskResult(ctx, googlecloudlogk8scontrolplane_contract.InputControlPlaneComponentNameFilterTaskID.Ref())

	return []string{GenerateK8sControlPlaneQuery(cluster, controlplaneComponentNameFilter)}, nil
}

// TaskID implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *controlPlaneListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogk8scontrolplane_contract.ListLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *controlPlaneListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ googlecloudcommon_contract.ListLogEntriesTaskSetting = (*controlPlaneListLogEntriesTaskSetting)(nil)

var ListLogEntriesTask = googlecloudcommon_contract.NewListLogEntriesTask(&controlPlaneListLogEntriesTaskSetting{})
