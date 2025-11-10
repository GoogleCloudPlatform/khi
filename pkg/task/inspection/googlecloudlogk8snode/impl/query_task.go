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

package googlecloudlogk8snode_impl

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
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// GenerateK8sNodeLogQuery generates a query for GKE node logs.
func GenerateK8sNodeLogQuery(projectId string, clusterId string, nodeNameSubstrings []string) string {
	return fmt.Sprintf(`resource.type="k8s_node"
-logName="projects/%s/logs/events"
resource.labels.cluster_name="%s"
%s
`, projectId, clusterId, generateNodeNameSubstringLogFilter(nodeNameSubstrings))
}

func generateNodeNameSubstringLogFilter(nodeNameSubstrings []string) string {
	if len(nodeNameSubstrings) == 0 {
		return "-- No node name substring filters are specified."
	} else {
		return fmt.Sprintf("resource.labels.node_name:(%s)", strings.Join(gcpqueryutil.WrapDoubleQuoteForStringArray(nodeNameSubstrings), " OR "))
	}
}

type k8snodeListLogEntriesTaskSetting struct{}

// DefaultResourceNames implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (k *k8snodeListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", projectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (k *k8snodeListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
		googlecloudk8scommon_contract.InputClusterNameTaskID.Ref(),
		googlecloudk8scommon_contract.InputNodeNameFilterTaskID.Ref(),
	}
}

// Description implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (k *k8snodeListLogEntriesTaskSetting) Description() *googlecloudcommon_contract.ListLogEntriesTaskDescription {
	return &googlecloudcommon_contract.ListLogEntriesTaskDescription{
		DefaultLogType: enum.LogTypeNode,
		QueryName:      "Kubernetes node logs",
		ExampleQuery:   GenerateK8sNodeLogQuery("gcp-project-id", "gcp-cluster-name", []string{"gke-test-cluster-node-1", "gke-test-cluster-node-2"}),
	}
}

// LogFilters implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (k *k8snodeListLogEntriesTaskSetting) LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	clusterName := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputClusterNameTaskID.Ref())
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	nodeNameSubstrings := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputNodeNameFilterTaskID.Ref())
	return []string{GenerateK8sNodeLogQuery(projectID, clusterName, nodeNameSubstrings)}, nil
}

// TaskID implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (k *k8snodeListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogk8snode_contract.ListLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (k *k8snodeListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ googlecloudcommon_contract.ListLogEntriesTaskSetting = (*k8snodeListLogEntriesTaskSetting)(nil)

var ListLogEntriesTask = googlecloudcommon_contract.NewListLogEntriesTask(&k8snodeListLogEntriesTaskSetting{})
