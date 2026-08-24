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

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8scontainer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontainer/contract"
)

// GenerateK8sContainerStructuredQuery constructs a StructuredLogQuery for Kubernetes container logs.
func GenerateK8sContainerStructuredQuery(
	cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity,
	namespacesFilter *gcpqueryutil.SetFilterParseResult,
	podNamesFilter *gcpqueryutil.SetFilterParseResult,
) *logestimator.StructuredLogQuery {
	filters := []logestimator.LoggingMonitoringMatcher{
		logestimator.ResourceLabel("project_id", logestimator.Exact(cluster.ProjectID)),
		logestimator.ResourceLabel("location", logestimator.Exact(cluster.Location)),
		logestimator.ResourceLabel("cluster_name", logestimator.Exact(cluster.NameFor(googlecloudk8scommon_contract.ClusterNameUsageK8sCluster))),
		logestimator.LogID(logestimator.NoneOf("server-accesslog-stackdriver", "client-accesslog-stackdriver")),
	}

	if namespacesFilter != nil {
		switch {
		case namespacesFilter.ValidationError != "":
			filters = append(filters, logestimator.CustomFilter(generateNamespacesFilter(namespacesFilter)))
		case namespacesFilter.SubtractMode:
			if len(namespacesFilter.Subtractives) > 0 {
				filters = append(filters, logestimator.ResourceLabel("namespace_name", logestimator.NoneOf(namespacesFilter.Subtractives...)))
			}
		case len(namespacesFilter.Additives) == 0:
			filters = append(filters, logestimator.CustomFilter(generateNamespacesFilter(namespacesFilter)))
		default:
			filters = append(filters, logestimator.ResourceLabel("namespace_name", logestimator.OneOf(namespacesFilter.Additives...)))
		}
	}

	if podNamesFilter != nil {
		switch {
		case podNamesFilter.ValidationError != "":
			filters = append(filters, logestimator.CustomFilter(generatePodNamesFilter(podNamesFilter)))
		case podNamesFilter.SubtractMode:
			if len(podNamesFilter.Subtractives) > 0 {
				filters = append(filters, logestimator.CustomFilter(generatePodNamesFilter(podNamesFilter)))
			}
		default:
			filters = append(filters, logestimator.CustomFilter(generatePodNamesFilter(podNamesFilter)))
		}
	}

	return &logestimator.StructuredLogQuery{
		Incomplete:    !cluster.IsComplete(),
		ResourceTypes: []string{"k8s_container"},
		Filters:       filters,
	}
}

// GenerateK8sContainerQuery generates a Cloud Logging query for Kubernetes container logs.
func GenerateK8sContainerQuery(cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity, namespacesFilter *gcpqueryutil.SetFilterParseResult, podNamesFilter *gcpqueryutil.SetFilterParseResult) string {
	return GenerateK8sContainerStructuredQuery(cluster, namespacesFilter, podNamesFilter).GenerateCloudLoggingQuery()
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

// DefaultResourceNames implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *containerListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudlogk8scontainer_contract.ClusterIdentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", cluster.ProjectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *containerListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudlogk8scontainer_contract.ClusterIdentityTaskID.Ref(),
		googlecloudlogk8scontainer_contract.InputContainerQueryNamespacesTaskID.Ref(),
		googlecloudlogk8scontainer_contract.InputContainerQueryPodNamesTaskID.Ref(),
	}
}

// QueryName implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *containerListLogEntriesTaskSetting) QueryName() string {
	return "K8s container logs"
}

// Queries implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *containerListLogEntriesTaskSetting) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudlogk8scontainer_contract.ClusterIdentityTaskID.Ref())
	namespacesFilter := coretask.GetTaskResult(ctx, googlecloudlogk8scontainer_contract.InputContainerQueryNamespacesTaskID.Ref())
	podNamesFilter := coretask.GetTaskResult(ctx, googlecloudlogk8scontainer_contract.InputContainerQueryPodNamesTaskID.Ref())

	return []*logestimator.StructuredLogQuery{
		GenerateK8sContainerStructuredQuery(cluster, namespacesFilter, podNamesFilter),
	}, nil
}

// TaskID implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *containerListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogk8scontainer_contract.ListLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *containerListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ googlecloudcommon_contract.StructuredListLogEntriesTaskSetting = (*containerListLogEntriesTaskSetting)(nil)

var ListLogEntriesTask = googlecloudcommon_contract.NewStructuredListLogEntriesTask(&containerListLogEntriesTaskSetting{})
