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

package k8s_container

import (
	"context"
	"fmt"
	"strings"

	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	gke_k8s_container_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_container/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

func GenerateK8sContainerQuery(clusterName string, namespacesFilter *queryutil.SetFilterParseResult, podNamesFilter *queryutil.SetFilterParseResult) string {
	return fmt.Sprintf(`resource.type="k8s_container"
resource.labels.cluster_name="%s"
%s
%s`, clusterName, generateNamespacesFilter(namespacesFilter), generatePodNamesFilter(podNamesFilter))
}

func generateNamespacesFilter(namespacesFilter *queryutil.SetFilterParseResult) string {
	if namespacesFilter.ValidationError != "" {
		return fmt.Sprintf("-- Failed to generate namespaces filter due to the validation error \"%s\"", namespacesFilter.ValidationError)
	}
	if namespacesFilter.SubtractMode {
		if len(namespacesFilter.Subtractives) == 0 {
			return "-- No namespace filter"
		}
		namespacesWithQuotes := []string{}
		for _, namespace := range namespacesFilter.Subtractives {
			namespacesWithQuotes = append(namespacesWithQuotes, fmt.Sprintf(`"%s"`, namespace))
		}
		return fmt.Sprintf(`-resource.labels.namespace_name=(%s)`, strings.Join(namespacesWithQuotes, " OR "))
	}

	if len(namespacesFilter.Additives) == 0 {
		return `-- Invalid: none of the resources will be selected. Ignoreing kind filter.`
	}
	namespacesWithQuotes := []string{}
	for _, namespace := range namespacesFilter.Additives {
		namespacesWithQuotes = append(namespacesWithQuotes, fmt.Sprintf(`"%s"`, namespace))
	}
	return fmt.Sprintf(`resource.labels.namespace_name=(%s)`, strings.Join(namespacesWithQuotes, " OR "))

}

func generatePodNamesFilter(podNamesFilter *queryutil.SetFilterParseResult) string {
	if podNamesFilter.ValidationError != "" {
		return fmt.Sprintf("-- Failed to generate pod name filter due to the validation error \"%s\"", podNamesFilter.ValidationError)
	}
	if podNamesFilter.SubtractMode {
		if len(podNamesFilter.Subtractives) == 0 {
			return "-- No pod name filter"
		}
		podNamesWithQuotes := []string{}
		for _, podName := range podNamesFilter.Subtractives {
			podNamesWithQuotes = append(podNamesWithQuotes, fmt.Sprintf(`"%s"`, podName))
		}
		return fmt.Sprintf(`-resource.labels.pod_name:(%s)`, strings.Join(podNamesWithQuotes, " OR "))
	}

	if len(podNamesFilter.Additives) == 0 {
		return `-- Invalid: none of the resources will be selected. Ignoreing kind filter.`
	}
	podNamesWithQuotes := []string{}
	for _, podName := range podNamesFilter.Additives {
		podNamesWithQuotes = append(podNamesWithQuotes, fmt.Sprintf(`"%s"`, podName))
	}
	return fmt.Sprintf(`resource.labels.pod_name:(%s)`, strings.Join(podNamesWithQuotes, " OR "))
}

var GKEContainerQueryTask = query.NewQueryGeneratorTask(gke_k8s_container_taskid.GKEContainerLogQueryTaskID, "K8s container logs", enum.LogTypeContainer, []taskid.UntypedTaskReference{
	gcp_task.InputClusterNameTaskID.Ref(),
	gke_k8s_container_taskid.InputContainerQueryNamespacesTaskID.Ref(),
	gke_k8s_container_taskid.InputContainerQueryPodNamesTaskID.Ref(),
}, &query.ProjectIDDefaultResourceNamesGenerator{}, func(ctx context.Context, i inspection_task_interface.InspectionTaskMode) ([]string, error) {
	clusterName := task.GetTaskResult(ctx, gcp_task.InputClusterNameTaskID.Ref())
	namespacesFilter := task.GetTaskResult(ctx, gke_k8s_container_taskid.InputContainerQueryNamespacesTaskID.Ref())
	podNamesFilter := task.GetTaskResult(ctx, gke_k8s_container_taskid.InputContainerQueryPodNamesTaskID.Ref())

	return []string{GenerateK8sContainerQuery(clusterName, namespacesFilter, podNamesFilter)}, nil
}, GenerateK8sContainerQuery("gcp-cluster-name", &queryutil.SetFilterParseResult{
	Additives: []string{"default"},
}, &queryutil.SetFilterParseResult{
	Subtractives: []string{"nginx-", "redis"},
	SubtractMode: true,
}))
