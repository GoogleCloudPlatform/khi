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

package googlecloudlogk8scontainer_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8scontainer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontainer/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// GenerateK8sContainerQuery generates a Cloud Logging query for Kubernetes container logs.
func GenerateK8sContainerQuery(clusterName string, namespacesFilter *gcpqueryutil.SetFilterParseResult, podNamesFilter *gcpqueryutil.SetFilterParseResult) string {
	return fmt.Sprintf(`resource.type="k8s_container"
resource.labels.cluster_name="%s"
%s
%s`, clusterName, generateNamespacesFilter(namespacesFilter), generatePodNamesFilter(podNamesFilter))
}

func generateNamespacesFilter(namespacesFilter *gcpqueryutil.SetFilterParseResult) string {
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

func generatePodNamesFilter(podNamesFilter *gcpqueryutil.SetFilterParseResult) string {
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

// GKEContainerQueryTask is a query generator task for GKE container logs.
var GKEContainerQueryTask = googlecloudcommon_contract.NewCloudLoggingListLogTask(googlecloudlogk8scontainer_contract.GKEContainerLogQueryTaskID, "K8s container logs", enum.LogTypeContainer, []taskid.UntypedTaskReference{
	googlecloudk8scommon_contract.InputClusterNameTaskID.Ref(),
	googlecloudlogk8scontainer_contract.InputContainerQueryNamespacesTaskID.Ref(),
	googlecloudlogk8scontainer_contract.InputContainerQueryPodNamesTaskID.Ref(),
}, &googlecloudcommon_contract.ProjectIDDefaultResourceNamesGenerator{}, func(ctx context.Context, i inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	clusterName := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputClusterNameTaskID.Ref())
	namespacesFilter := coretask.GetTaskResult(ctx, googlecloudlogk8scontainer_contract.InputContainerQueryNamespacesTaskID.Ref())
	podNamesFilter := coretask.GetTaskResult(ctx, googlecloudlogk8scontainer_contract.InputContainerQueryPodNamesTaskID.Ref())

	return []string{GenerateK8sContainerQuery(clusterName, namespacesFilter, podNamesFilter)}, nil
}, GenerateK8sContainerQuery("gcp-cluster-name", &gcpqueryutil.SetFilterParseResult{
	Additives: []string{"default"},
},
	&gcpqueryutil.SetFilterParseResult{
		Subtractives: []string{"nginx-", "redis"},
		SubtractMode: true,
	}))
