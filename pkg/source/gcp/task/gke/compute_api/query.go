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

package compute_api

import (
	"context"
	"fmt"
	"strings"

	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	gke_compute_api_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/compute_api/taskid"
	gke_k8saudit_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

func GenerateComputeAPIQuery(taskMode inspection_task_interface.InspectionTaskMode, nodeNames []string) []string {
	if taskMode == inspection_task_interface.TaskModeDryRun {
		return []string{
			generateComputeAPIQueryWithInstanceNameFilter("-- instance name filters to be determined after audit log query"),
		}
	} else {
		result := []string{}
		instanceNameGroups := queryutil.SplitToChildGroups(nodeNames, 30)
		for _, group := range instanceNameGroups {
			nodeNamesWithInstance := []string{}
			for _, nodeName := range group {
				nodeNamesWithInstance = append(nodeNamesWithInstance, fmt.Sprintf("instances/%s", nodeName))
			}
			instanceNameFilter := fmt.Sprintf("protoPayload.resourceName:(%s)", strings.Join(nodeNamesWithInstance, " OR "))
			result = append(result, generateComputeAPIQueryWithInstanceNameFilter(instanceNameFilter))
		}
		return result
	}
}

func generateComputeAPIQueryWithInstanceNameFilter(instanceNameFilter string) string {
	return fmt.Sprintf(`resource.type="gce_instance"
-protoPayload.methodName:("list" OR "get" OR "watch")
%s
`, instanceNameFilter)
}

var ComputeAPIQueryTask = query.NewQueryGeneratorTask(gke_compute_api_taskid.ComputeAPIQueryTaskID, "Compute API Logs", enum.LogTypeComputeApi, []taskid.UntypedTaskReference{
	gke_k8saudit_taskid.K8sAuditParseTaskID,
}, &query.ProjectIDDefaultResourceNamesGenerator{}, func(ctx context.Context, i inspection_task_interface.InspectionTaskMode) ([]string, error) {
	builder := task.GetTaskResult(ctx, inspection_task.BuilderGeneratorTaskID.GetTaskReference())

	return GenerateComputeAPIQuery(i, builder.ClusterResource.GetNodes()), nil
}, GenerateComputeAPIQuery(inspection_task_interface.TaskModeRun, []string{
	"gke-test-cluster-node-1",
	"gke-test-cluster-node-2",
})[0])
