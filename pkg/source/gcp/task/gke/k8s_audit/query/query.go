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

package query

import (
	"context"
	"fmt"
	"slices"
	"strings"

	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	gke_k8saudit_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"

	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

var Task = query.NewQueryGeneratorTask(gke_k8saudit_taskid.K8sAuditQueryTaskID, "K8s audit logs", enum.LogTypeAudit, []taskid.UntypedTaskReference{
	gcp_task.InputClusterNameTaskID.Ref(),
	gcp_task.InputKindFilterTaskID.Ref(),
	gcp_task.InputNamespaceFilterTaskID.Ref(),
}, &query.ProjectIDDefaultResourceNamesGenerator{}, func(ctx context.Context, i inspection_task_interface.InspectionTaskMode) ([]string, error) {
	clusterName := task.GetTaskResult(ctx, gcp_task.InputClusterNameTaskID.Ref())
	kindFilter := task.GetTaskResult(ctx, gcp_task.InputKindFilterTaskID.Ref())
	namespaceFilter := task.GetTaskResult(ctx, gcp_task.InputNamespaceFilterTaskID.Ref())

	return []string{GenerateK8sAuditQuery(clusterName, kindFilter, namespaceFilter)}, nil
}, GenerateK8sAuditQuery(
	"gcp-cluster-name",
	&queryutil.SetFilterParseResult{
		Additives: []string{"deployments", "replicasets", "pods", "nodes"},
	},
	&queryutil.SetFilterParseResult{
		Additives: []string{"#cluster-scoped", "#namespaced"},
	},
))

func GenerateK8sAuditQuery(clusterName string, auditKindFilter *queryutil.SetFilterParseResult, namespaceFilter *queryutil.SetFilterParseResult) string {
	return fmt.Sprintf(`resource.type="k8s_cluster"
resource.labels.cluster_name="%s"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
%s
%s
`, clusterName, generateAuditKindFilter(auditKindFilter), generateK8sAuditNamespaceFilter(namespaceFilter))
}

func generateAuditKindFilter(filter *queryutil.SetFilterParseResult) string {
	if filter.ValidationError != "" {
		return fmt.Sprintf(`-- Failed to generate kind filter due to the validation error "%s"`, filter.ValidationError)
	}
	if filter.SubtractMode {
		if len(filter.Subtractives) == 0 {
			return "-- No kind filter"
		}
		return fmt.Sprintf(`-protoPayload.methodName=~"\.(%s)\."`, strings.Join(filter.Subtractives, "|"))
	} else {
		if len(filter.Additives) == 0 {
			return `-- Invalid: none of the resources will be selected. Ignoreing kind filter.`
		}
		return fmt.Sprintf(`protoPayload.methodName=~"\.(%s)\."`, strings.Join(filter.Additives, "|"))
	}
}

func generateK8sAuditNamespaceFilter(filter *queryutil.SetFilterParseResult) string {
	if filter.ValidationError != "" {
		return fmt.Sprintf(`-- Failed to generate namespace filter due to the validation error "%s"`, filter.ValidationError)
	}
	if filter.SubtractMode {
		return "-- Unsupported operation"
	} else {
		hasClusterScope := slices.Contains(filter.Additives, "#cluster-scoped")
		hasNamespacedScope := slices.Contains(filter.Additives, "#namespaced")
		if hasClusterScope && hasNamespacedScope {
			return "-- No namespace filter"
		}
		if !hasClusterScope && hasNamespacedScope {
			return `protoPayload.resourceName:"namespaces/"`
		}
		if hasClusterScope && !hasNamespacedScope {
			if len(filter.Additives) == 1 { // 1 is used for #cluster-scope
				return `-protoPayload.resourceName:"/namespaces/"`
			}
			resourceNameContains := []string{}
			for _, additive := range filter.Additives {
				if strings.HasPrefix(additive, "#") {
					continue
				}
				resourceNameContains = append(resourceNameContains, fmt.Sprintf(`"/namespaces/%s"`, additive))
			}
			return fmt.Sprintf(`(protoPayload.resourceName:(%s) OR NOT (protoPayload.resourceName:"/namespaces/"))`, strings.Join(resourceNameContains, " OR "))
		}
		if len(filter.Additives) == 0 {
			return `-- Invalid: none of the resources will be selected. Ignoreing namespace filter.`
		}
		resourceNameContains := []string{}
		for _, additive := range filter.Additives {
			resourceNameContains = append(resourceNameContains, fmt.Sprintf(`"/namespaces/%s"`, additive))
		}
		return fmt.Sprintf(`protoPayload.resourceName:(%s)`, strings.Join(resourceNameContains, " OR "))
	}
}
