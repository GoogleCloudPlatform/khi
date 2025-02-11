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

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
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

const GKENodeLogQueryTaskID = query.GKEQueryPrefix + "k8s-node"

var GKENodeQueryTask = query.NewQueryGeneratorTask(GKENodeLogQueryTaskID, "Kubernetes node log", enum.LogTypeNode, []string{
	gcp_task.InputProjectIdTaskID,
	gcp_task.InputClusterNameTaskID,
	gcp_task.InputNodeNameFilterTaskID,
}, func(ctx context.Context, i int, vs *task.VariableSet) ([]string, error) {
	clusterName, err := gcp_task.GetInputClusterNameFromTaskVariable(vs)
	if err != nil {
		return nil, err
	}
	projectId, err := gcp_task.GetInputProjectIdFromTaskVariable(vs)
	if err != nil {
		return nil, err
	}
	nodeNameSubstrings, err := gcp_task.GetNodeNameFilterFromTaskVaraible(vs)
	if err != nil {
		return nil, err
	}
	return []string{GenerateK8sNodeLogQuery(projectId, clusterName, nodeNameSubstrings)}, nil
}, GenerateK8sNodeLogQuery("gcp-project-id", "gcp-cluster-name", []string{"gke-test-cluster-node-1", "gke-test-cluster-node-2"}))
