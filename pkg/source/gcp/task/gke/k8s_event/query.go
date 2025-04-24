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

package k8s_event

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
	k8s_event_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_event/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

func GenerateK8sEventQuery(clusterName string, projectId string, namespaceFilter *queryutil.SetFilterParseResult) string {
	return fmt.Sprintf(`logName="projects/%s/logs/events"
resource.labels.cluster_name="%s"
%s`, projectId, clusterName, generateK8sEventNamespaceFilter(namespaceFilter))
}

func generateK8sEventNamespaceFilter(filter *queryutil.SetFilterParseResult) string {
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
			return `jsonPayload.involvedObject.namespace:"" -- ignore events in k8s object with namespace`
		}
		if hasClusterScope && !hasNamespacedScope {
			if len(filter.Additives) == 1 {
				return `-jsonPayload.involvedObject.namespace:"" -- ignore events in k8s object with namespace`
			}
			namespaceContains := []string{}
			for _, additive := range filter.Additives {
				if strings.HasPrefix(additive, "#") {
					continue
				}
				namespaceContains = append(namespaceContains, additive)
			}
			return fmt.Sprintf(`(jsonPayload.involvedObject.namespace=(%s) OR NOT (jsonPayload.involvedObject.namespace:""))`, strings.Join(namespaceContains, " OR "))
		}
		if len(filter.Additives) == 0 {
			return `-- Invalid: none of the resources will be selected. Ignoreing namespace filter.`
		}
		return fmt.Sprintf(`jsonPayload.involvedObject.namespace=(%s)`, strings.Join(filter.Additives, " OR "))
	}
}

var GKEK8sEventLogQueryTask = query.NewQueryGeneratorTask(k8s_event_taskid.GKEK8sEventLogQueryTaskID, "K8s event logs", enum.LogTypeEvent, []taskid.UntypedTaskReference{
	gcp_task.InputProjectIdTaskID,
	gcp_task.InputClusterNameTaskID,
	gcp_task.InputNamespaceFilterTaskID,
}, &query.ProjectIDDefaultResourceNamesGenerator{}, func(ctx context.Context, i inspection_task_interface.InspectionTaskMode) ([]string, error) {
	clusterName := task.GetTaskResult(ctx, gcp_task.InputClusterNameTaskID.GetTaskReference())
	projectID := task.GetTaskResult(ctx, gcp_task.InputProjectIdTaskID.GetTaskReference())
	namespaceFilter := task.GetTaskResult(ctx, gcp_task.InputNamespaceFilterTaskID.GetTaskReference())

	return []string{GenerateK8sEventQuery(clusterName, projectID, namespaceFilter)}, nil
}, GenerateK8sEventQuery(
	"gcp-cluster-name",
	"gcp-project-id",
	&queryutil.SetFilterParseResult{
		Additives: []string{"#cluster-scoped", "#namespaced"},
	},
))
