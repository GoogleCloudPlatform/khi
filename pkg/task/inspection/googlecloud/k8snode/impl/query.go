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

package k8snode_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8snode"
)

// generateK8sNodeStructuredQuery generates a structured query for GKE node logs.
func generateK8sNodeStructuredQuery(cluster k8scommon.GoogleCloudClusterIdentity, nodeNameSubstrings []string) *logestimator.StructuredLogQuery {
	filters := []logestimator.LoggingMonitoringMatcher{
		logestimator.ResourceLabel("project_id", logestimator.Exact(cluster.ProjectID)),
		logestimator.ResourceLabel("location", logestimator.Exact(cluster.Location)),
		logestimator.ResourceLabel("cluster_name", logestimator.Exact(cluster.NameFor(k8scommon.ClusterNameUsageK8sCluster))),
		logestimator.LogID(logestimator.NoneOf("events")),
	}

	if len(nodeNameSubstrings) > 0 {
		filters = append(filters, logestimator.ResourceLabel("node_name", logestimator.ContainsAny(nodeNameSubstrings...)))
	}

	return &logestimator.StructuredLogQuery{
		Incomplete:    !cluster.IsComplete(),
		ResourceTypes: []string{"k8s_node"},
		Filters:       filters,
	}
}

// nodeLogQuerySource builds the Cloud Logging query for the Kubernetes node logs of the cluster.
type nodeLogQuerySource struct {
	clusterIdentity    coretask.Input[k8scommon.GoogleCloudClusterIdentity]
	nodeNameSubstrings coretask.Input[[]string]
}

// DefaultResourceNames implements gcpcommon.StructuredLogQuerySource.
func (s *nodeLogQuerySource) DefaultResourceNames(ctx context.Context) ([]string, error) {
	cluster := s.clusterIdentity.Get(ctx)
	return []string{fmt.Sprintf("projects/%s", cluster.ProjectID)}, nil
}

// Queries implements gcpcommon.StructuredLogQuerySource.
func (s *nodeLogQuerySource) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	cluster := s.clusterIdentity.Get(ctx)
	nodeNameSubstrings := s.nodeNameSubstrings.Get(ctx)
	return []*logestimator.StructuredLogQuery{generateK8sNodeStructuredQuery(cluster, nodeNameSubstrings)}, nil
}

// TimePartitionCount implements gcpcommon.StructuredLogQuerySource.
func (s *nodeLogQuerySource) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ gcpcommon.StructuredLogQuerySource = (*nodeLogQuerySource)(nil)

// listLogEntriesTask queries the Kubernetes node logs of the cluster from Cloud Logging.
var listLogEntriesTask = gcpcommon.DefineStructuredListLogEntriesTask(
	k8snode.ListLogEntriesTaskID,
	"Kubernetes node logs",
	func(b *coretask.Binder) gcpcommon.StructuredLogQuerySource {
		return &nodeLogQuerySource{
			clusterIdentity:    coretask.Use(b, k8snode.ClusterIdentityTaskID.Ref()),
			nodeNameSubstrings: coretask.Use(b, k8scommon.InputNodeNameFilterTaskID.Ref()),
		}
	},
)
