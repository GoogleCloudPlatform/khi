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

	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"

	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/k8saudittask"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

const GCPNetworkLogQueryTaskID = query.GKEQueryPrefix + "network-api"

func GenerateGCPNetworkAPIQuery(taskMode int, negNames []string) []string {
	nodeNamesWithNetworkEndpointGroups := []string{}
	for _, negName := range negNames {
		nodeNamesWithNetworkEndpointGroups = append(nodeNamesWithNetworkEndpointGroups, fmt.Sprintf("networkEndpointGroups/%s", negName))
	}
	if taskMode == inspection_task.TaskModeDryRun {
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

var GCPNetworkLogQueryTask = query.NewQueryGeneratorTask(GCPNetworkLogQueryTaskID, "GCP network log", enum.LogTypeNetworkAPI, []string{
	k8saudittask.K8sAuditParseTaskID,
}, func(ctx context.Context, i int, vs *task.VariableSet) ([]string, error) {
	builder, err := inspection_task.GetHistoryBuilderFromTaskVariable(vs)
	if err != nil {
		return []string{}, err
	}
	return GenerateGCPNetworkAPIQuery(i, builder.ClusterResource.NEGs.GetAllIdentifiers()), nil
}, GenerateGCPNetworkAPIQuery(inspection_task.TaskModeRun, []string{"neg-id-1", "neg-id-2"})[0])
