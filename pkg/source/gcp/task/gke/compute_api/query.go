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

	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/k8saudittask"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

var ComputeAPIQueryTaskID = query.GKEQueryPrefix + "compute-api"

func GenerateComputeAPIQuery(taskMode int, nodeNames []string) []string {
	if taskMode == inspection_task.TaskModeDryRun {
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

var ComputeAPIQueryTask = query.NewQueryGeneratorTask(ComputeAPIQueryTaskID, "Compute API Logs", enum.LogTypeComputeApi, []string{
	k8saudittask.K8sAuditParseTaskID,
}, func(ctx context.Context, i int, vs *task.VariableSet) ([]string, error) {
	builder, err := inspection_task.GetHistoryBuilderFromTaskVariable(vs)
	if err != nil {
		return []string{}, err
	}
	return GenerateComputeAPIQuery(i, builder.ClusterResource.GetNodes()), nil
})
