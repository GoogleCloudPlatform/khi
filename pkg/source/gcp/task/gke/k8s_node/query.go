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

package k8s_node

import (
	"context"
	"fmt"
	"strings"

	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	k8s_node_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_node/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

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
		return fmt.Sprintf("resource.labels.node_name:(%s)", strings.Join(queryutil.WrapDoubleQuoteForStringArray(nodeNameSubstrings), " OR "))
	}
}

var GKENodeQueryTask = query.NewQueryGeneratorTask(k8s_node_taskid.GKENodeLogQueryTaskID, "Kubernetes node log", enum.LogTypeNode, []taskid.UntypedTaskReference{
	gcp_task.InputProjectIdTaskID.Ref(),
	gcp_task.InputClusterNameTaskID.Ref(),
	gcp_task.InputNodeNameFilterTaskID.Ref(),
}, &query.ProjectIDDefaultResourceNamesGenerator{}, func(ctx context.Context, i inspection_task_interface.InspectionTaskMode) ([]string, error) {
	clusterName := task.GetTaskResult(ctx, gcp_task.InputClusterNameTaskID.Ref())
	projectID := task.GetTaskResult(ctx, gcp_task.InputProjectIdTaskID.Ref())
	nodeNameSubstrings := task.GetTaskResult(ctx, gcp_task.InputNodeNameFilterTaskID.Ref())

	return []string{GenerateK8sNodeLogQuery(projectID, clusterName, nodeNameSubstrings)}, nil
}, GenerateK8sNodeLogQuery("gcp-project-id", "gcp-cluster-name", []string{"gke-test-cluster-node-1", "gke-test-cluster-node-2"}))
