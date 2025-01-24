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

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/k8saudittask"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

var Task = query.NewQueryGeneratorTask(k8saudittask.K8sAuditQueryTaskID, "K8s audit logs", enum.LogTypeAudit, []string{
	gcp_task.InputClusterNameTaskID,
	gcp_task.InputKindFilterTaskID,
	gcp_task.InputNamespaceFilterTaskID,
}, func(ctx context.Context, i int, vs *task.VariableSet) ([]string, error) {
	clusterName, err := gcp_task.GetInputClusterNameFromTaskVariable(vs)
	if err != nil {
		return []string{}, err
	}
	kindFilter, err := gcp_task.GetInputKindNameFromTaskVariable(vs)
	if err != nil {
		return []string{}, err
	}
	namespaceFilter, err := gcp_task.GetInputNamespaceFilterFromTaskVariable(vs)
	if err != nil {
		return []string{}, err
	}
	return []string{GenerateK8sAuditQuery(clusterName, kindFilter, namespaceFilter)}, nil
})

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
