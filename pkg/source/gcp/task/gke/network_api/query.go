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

package network_api

import (
	"context"
	"fmt"
	"strings"

	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"

	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	gke_k8saudit_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/taskid"
	network_api_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/network_api/taskid"
)

func GenerateGCPNetworkAPIQuery(taskMode inspection_task_interface.InspectionTaskMode, negNames []string) []string {
	nodeNamesWithNetworkEndpointGroups := []string{}
	for _, negName := range negNames {
		nodeNamesWithNetworkEndpointGroups = append(nodeNamesWithNetworkEndpointGroups, fmt.Sprintf("networkEndpointGroups/%s", negName))
	}
	if taskMode == inspection_task_interface.TaskModeDryRun {
		return []string{queryFromNegNameFilter("-- neg name filters to be determined after audit log query")}
	} else {
		result := []string{}
		groups := queryutil.SplitToChildGroups(nodeNamesWithNetworkEndpointGroups, 10)
		for _, group := range groups {
			negNameFilter := fmt.Sprintf("protoPayload.resourceName:(%s)", strings.Join(group, " OR "))
			result = append(result, queryFromNegNameFilter(negNameFilter))
		}
		return result
	}
}

func queryFromNegNameFilter(negNameFilter string) string {
	return fmt.Sprintf(`resource.type="gce_network"
-protoPayload.methodName:("list" OR "get" OR "watch")
%s
`, negNameFilter)
}

var GCPNetworkLogQueryTask = query.NewQueryGeneratorTask(network_api_taskid.GCPNetworkLogQueryTaskID, "GCP network log", enum.LogTypeNetworkAPI, []taskid.UntypedTaskReference{
	gke_k8saudit_taskid.K8sAuditParseTaskID,
}, &query.ProjectIDDefaultResourceNamesGenerator{}, func(ctx context.Context, i inspection_task_interface.InspectionTaskMode) ([]string, error) {
	builder := task.GetTaskResult(ctx, inspection_task.BuilderGeneratorTaskID.GetTaskReference())
	return GenerateGCPNetworkAPIQuery(i, builder.ClusterResource.NEGs.GetAllIdentifiers()), nil
}, GenerateGCPNetworkAPIQuery(inspection_task_interface.TaskModeRun, []string{"neg-id-1", "neg-id-2"})[0])
