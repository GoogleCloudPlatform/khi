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

package autoscaler

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

var AutoscalerQueryTaskID = query.GKEQueryPrefix + "autoscaler"

func GenerateAutoscalerQuery(projectId string, clusterName string, excludeStatus bool) string {
	excludeStatusQueryFragment := "-- include query for status log"
	if excludeStatus {
		excludeStatusQueryFragment = `-jsonPayload.status: ""`
	}
	return fmt.Sprintf(`resource.type="k8s_cluster"
resource.labels.project_id="%s"
resource.labels.cluster_name="%s"
%s
logName="projects/%s/logs/container.googleapis.com%%2Fcluster-autoscaler-visibility"`, projectId, clusterName, excludeStatusQueryFragment, projectId)
}

var AutoscalerQueryTask = query.NewQueryGeneratorTask(AutoscalerQueryTaskID, "Autoscaler logs", enum.LogTypeAutoscaler, []string{
	gcp_task.InputProjectIdTaskID,
	gcp_task.InputClusterNameTaskID,
}, func(ctx context.Context, i int, vs *task.VariableSet) ([]string, error) {
	projectId, err := gcp_task.GetInputProjectIdFromTaskVariable(vs)
	if err != nil {
		return []string{}, err
	}
	clusterName, err := gcp_task.GetInputClusterNameFromTaskVariable(vs)
	if err != nil {
		return []string{}, err
	}
	return []string{GenerateAutoscalerQuery(projectId, clusterName, true)}, nil
}, GenerateAutoscalerQuery("gcp-project-id", "gcp-cluster-name", true))
