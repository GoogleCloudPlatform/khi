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

package k8scontainer_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scontainer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// generateK8sContainerStructuredQuery constructs a StructuredLogQuery for Kubernetes container logs.
func generateK8sContainerStructuredQuery(
	cluster k8scommon.GoogleCloudClusterIdentity,
	namespacesFilter *gcpqueryutil.SetFilterParseResult,
	podNamesFilter *gcpqueryutil.SetFilterParseResult,
) *logestimator.StructuredLogQuery {
	filters := []logestimator.LoggingMonitoringMatcher{
		logestimator.ResourceLabel("project_id", logestimator.Exact(cluster.ProjectID)),
		logestimator.ResourceLabel("location", logestimator.Exact(cluster.Location)),
		logestimator.ResourceLabel("cluster_name", logestimator.Exact(cluster.NameFor(k8scommon.ClusterNameUsageK8sCluster))),
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

// containerLogQuerySource builds the Cloud Logging query for the Kubernetes container logs of the cluster.
type containerLogQuerySource struct {
	clusterIdentity  coretask.Input[k8scommon.GoogleCloudClusterIdentity]
	namespacesFilter coretask.Input[*gcpqueryutil.SetFilterParseResult]
	podNamesFilter   coretask.Input[*gcpqueryutil.SetFilterParseResult]
}

// DefaultResourceNames implements gcpcommon.StructuredLogQuerySource.
func (s *containerLogQuerySource) DefaultResourceNames(ctx context.Context) ([]string, error) {
	cluster := s.clusterIdentity.Get(ctx)
	return []string{fmt.Sprintf("projects/%s", cluster.ProjectID)}, nil
}

// Queries implements gcpcommon.StructuredLogQuerySource.
func (s *containerLogQuerySource) Queries(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]*logestimator.StructuredLogQuery, error) {
	cluster := s.clusterIdentity.Get(ctx)
	namespacesFilter := s.namespacesFilter.Get(ctx)
	podNamesFilter := s.podNamesFilter.Get(ctx)

	return []*logestimator.StructuredLogQuery{
		generateK8sContainerStructuredQuery(cluster, namespacesFilter, podNamesFilter),
	}, nil
}

// TimePartitionCount implements gcpcommon.StructuredLogQuerySource.
func (s *containerLogQuerySource) TimePartitionCount() int {
	return 10
}

var _ gcpcommon.StructuredLogQuerySource = (*containerLogQuerySource)(nil)

// listLogEntriesTask queries the Kubernetes container logs of the cluster from Cloud Logging.
var listLogEntriesTask = gcpcommon.DefineStructuredListLogEntriesTask(
	k8scontainer.ListLogEntriesTaskID,
	"K8s container logs",
	func(b *coretask.Binder) gcpcommon.StructuredLogQuerySource {
		return &containerLogQuerySource{
			clusterIdentity:  coretask.Use(b, k8scontainer.ClusterIdentityTaskID.Ref()),
			namespacesFilter: coretask.Use(b, k8scontainer.InputContainerQueryNamespacesTaskID.Ref()),
			podNamesFilter:   coretask.Use(b, k8scontainer.InputContainerQueryPodNamesTaskID.Ref()),
		}
	},
)
