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
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8scontainer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontainer/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// GenerateK8sContainerQuery generates a Cloud Logging query for Kubernetes container logs.
func GenerateK8sContainerQuery(cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity, namespacesFilter *gcpqueryutil.SetFilterParseResult, podNamesFilter *gcpqueryutil.SetFilterParseResult) string {
	return fmt.Sprintf(`resource.type="k8s_container"
resource.labels.project_id="%s"
resource.labels.location="%s"
resource.labels.cluster_name="%s"
%s
%s`, cluster.ProjectID, cluster.Location, cluster.NameWithClusterTypePrefix(), generateNamespacesFilter(namespacesFilter), generatePodNamesFilter(podNamesFilter))
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
		return `-- Invalid: none of the resources will be selected. Ignoring namespace filter.`
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
		return `-- Invalid: none of the resources will be selected. Ignoring pod name filter.`
	}
	podNamesWithQuotes := []string{}
	for _, podName := range podNamesFilter.Additives {
		podNamesWithQuotes = append(podNamesWithQuotes, fmt.Sprintf(`"%s"`, podName))
	}
	return fmt.Sprintf(`resource.labels.pod_name:(%s)`, strings.Join(podNamesWithQuotes, " OR "))
}

type containerListLogEntriesTaskSetting struct {
}

// DefaultResourceNames implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *containerListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", cluster.ProjectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *containerListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref(),
		googlecloudlogk8scontainer_contract.InputContainerQueryNamespacesTaskID.Ref(),
		googlecloudlogk8scontainer_contract.InputContainerQueryPodNamesTaskID.Ref(),
	}
}

// Description implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *containerListLogEntriesTaskSetting) Description() *googlecloudcommon_contract.ListLogEntriesTaskDescription {
	return &googlecloudcommon_contract.ListLogEntriesTaskDescription{
		DefaultLogType: enum.LogTypeContainer,
		QueryName:      "K8s container logs",
		ExampleQuery: GenerateK8sContainerQuery(googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
			ProjectID:   "test-project",
			Location:    "test-location",
			ClusterName: "test-cluster",
		},
			&gcpqueryutil.SetFilterParseResult{
				Additives: []string{"default"},
			},
			&gcpqueryutil.SetFilterParseResult{
				Subtractives: []string{"nginx-", "redis"},
				SubtractMode: true,
			},
		),
	}
}

// LogFilters implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *containerListLogEntriesTaskSetting) LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref())
	namespacesFilter := coretask.GetTaskResult(ctx, googlecloudlogk8scontainer_contract.InputContainerQueryNamespacesTaskID.Ref())
	podNamesFilter := coretask.GetTaskResult(ctx, googlecloudlogk8scontainer_contract.InputContainerQueryPodNamesTaskID.Ref())

	return []string{GenerateK8sContainerQuery(cluster, namespacesFilter, podNamesFilter)}, nil
}

// TaskID implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *containerListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogk8scontainer_contract.ListLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (c *containerListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ googlecloudcommon_contract.ListLogEntriesTaskSetting = (*containerListLogEntriesTaskSetting)(nil)

var ListLogEntriesTask = googlecloudcommon_contract.NewListLogEntriesTask(&containerListLogEntriesTaskSetting{})
