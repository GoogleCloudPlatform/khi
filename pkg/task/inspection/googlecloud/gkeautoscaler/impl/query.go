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

package gkeautoscaler_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gkeautoscaler"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// generateAutoscalerStructuredQuery generates a structured query for GKE cluster autoscaler logs.
func generateAutoscalerStructuredQuery(cluster k8scommon.GoogleCloudClusterIdentity, excludeStatus bool) *logestimator.StructuredLogQuery {
	filters := []logestimator.LoggingMonitoringMatcher{
		logestimator.ResourceLabel("project_id", logestimator.Exact(cluster.ProjectID)),
		logestimator.ResourceLabel("location", logestimator.Exact(cluster.Location)),
		logestimator.ResourceLabel("cluster_name", logestimator.Exact(cluster.NameFor(k8scommon.ClusterNameUsageK8sCluster))),
		logestimator.LogID(logestimator.Exact("container.googleapis.com/cluster-autoscaler-visibility")),
	}
	if excludeStatus {
		filters = append(filters, logestimator.CustomFilter(`-jsonPayload.status: ""`))
	}

	return &logestimator.StructuredLogQuery{
		Incomplete:    !cluster.IsComplete(),
		ResourceTypes: []string{"k8s_cluster"},
		Filters:       filters,
	}
}

// autoscalerQuerySource builds the GKE cluster autoscaler log queries for the cluster.
type autoscalerQuerySource struct {
	clusterIdentity coretask.Input[k8scommon.GoogleCloudClusterIdentity]
}

// DefaultResourceNames implements gcpcommon.StructuredLogQuerySource.
func (s *autoscalerQuerySource) DefaultResourceNames(ctx context.Context) ([]string, error) {
	cluster := s.clusterIdentity.Get(ctx)
	return []string{fmt.Sprintf("projects/%s", cluster.ProjectID)}, nil
}

// Queries implements gcpcommon.StructuredLogQuerySource.
func (s *autoscalerQuerySource) Queries(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]*logestimator.StructuredLogQuery, error) {
	cluster := s.clusterIdentity.Get(ctx)
	return []*logestimator.StructuredLogQuery{generateAutoscalerStructuredQuery(cluster, true)}, nil
}

// TimePartitionCount implements gcpcommon.StructuredLogQuerySource.
func (s *autoscalerQuerySource) TimePartitionCount() int {
	return 1
}

var _ gcpcommon.StructuredLogQuerySource = (*autoscalerQuerySource)(nil)

// listLogEntriesTask queries the GKE cluster autoscaler visibility logs of the cluster from Cloud Logging.
var listLogEntriesTask = gcpcommon.DefineStructuredListLogEntriesTask(
	gkeautoscaler.ListLogEntriesTaskID,
	"Cluster autoscaler logs",
	func(b *coretask.Binder) gcpcommon.StructuredLogQuerySource {
		return &autoscalerQuerySource{
			clusterIdentity: coretask.Use(b, k8scommon.ClusterIdentityTaskID.Ref()),
		}
	},
)
