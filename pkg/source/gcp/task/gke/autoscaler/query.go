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

	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	gke_autoscaler_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/autoscaler/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

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

var AutoscalerQueryTask = query.NewQueryGeneratorTask(gke_autoscaler_taskid.AutoscalerQueryTaskID, "Autoscaler logs", enum.LogTypeAutoscaler, []taskid.UntypedTaskReference{
	gcp_task.InputProjectIdTaskID,
	gcp_task.InputClusterNameTaskID,
}, func(ctx context.Context, i inspection_task_interface.InspectionTaskMode) ([]string, error) {
	projectID := task.GetTaskResult(ctx, gcp_task.InputProjectIdTaskID.GetTaskReference())
	clusterName := task.GetTaskResult(ctx, gcp_task.InputClusterNameTaskID.GetTaskReference())

	return []string{GenerateAutoscalerQuery(projectID, clusterName, true)}, nil
}, GenerateAutoscalerQuery("gcp-project-id", "gcp-cluster-name", true))
