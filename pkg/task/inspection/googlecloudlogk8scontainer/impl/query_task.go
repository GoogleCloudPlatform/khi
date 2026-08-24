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

	if nsMatcher := generateNamespacesFilter(namespacesFilter); nsMatcher != nil {
		filters = append(filters, nsMatcher)
	}

	if podMatcher := generatePodNamesFilter(podNamesFilter); podMatcher != nil {
		filters = append(filters, podMatcher)
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

func generateNamespacesFilter(namespacesFilter *gcpqueryutil.SetFilterParseResult) logestimator.LoggingMonitoringMatcher {
	if namespacesFilter == nil {
		return nil
	}
	if namespacesFilter.ValidationError != "" {
		return logestimator.Comment(fmt.Sprintf(`Failed to generate namespaces filter due to the validation error "%s"`, namespacesFilter.ValidationError))
	}
	if namespacesFilter.SubtractMode {
		if len(namespacesFilter.Subtractives) == 0 {
			return nil
		}
		return logestimator.ResourceLabel("namespace_name", logestimator.NoneOf(namespacesFilter.Subtractives...))
	}

	if len(namespacesFilter.Additives) == 0 {
		return logestimator.Comment("Invalid: none of the resources will be selected. Ignoring namespace filter.")
	}
	return logestimator.ResourceLabel("namespace_name", logestimator.OneOf(namespacesFilter.Additives...))
}

func generatePodNamesFilter(podNamesFilter *gcpqueryutil.SetFilterParseResult) logestimator.LoggingMonitoringMatcher {
	if podNamesFilter == nil {
		return nil
	}
	if podNamesFilter.ValidationError != "" {
		return logestimator.Comment(fmt.Sprintf(`Failed to generate pod name filter due to the validation error "%s"`, podNamesFilter.ValidationError))
	}
	if podNamesFilter.SubtractMode {
		if len(podNamesFilter.Subtractives) == 0 {
			return nil
		}
		return logestimator.ResourceLabel("pod_name", logestimator.NotContainsAny(podNamesFilter.Subtractives...))
	}

	if len(podNamesFilter.Additives) == 0 {
		return logestimator.Comment("Invalid: none of the resources will be selected. Ignoring pod name filter.")
	}
	return logestimator.ResourceLabel("pod_name", logestimator.ContainsAny(podNamesFilter.Additives...))
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
