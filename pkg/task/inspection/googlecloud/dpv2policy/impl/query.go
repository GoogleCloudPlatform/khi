// Copyright 2026 Google LLC
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

package dpv2policy_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/dpv2policy"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// generatePolicyActionStructuredQuery generates a structured query for GKE Dataplane V2 network policy logs.
func generatePolicyActionStructuredQuery(cluster k8scommon.GoogleCloudClusterIdentity, nodeNameSubstrings []string) *logestimator.StructuredLogQuery {
	filters := []logestimator.LoggingMonitoringMatcher{
		logestimator.ResourceLabel("project_id", logestimator.Exact(cluster.ProjectID)),
		logestimator.ResourceLabel("location", logestimator.Exact(cluster.Location)),
		logestimator.ResourceLabel("cluster_name", logestimator.Exact(cluster.NameFor(k8scommon.ClusterNameUsageK8sCluster))),
		logestimator.LogID(logestimator.Exact("policy-action")),
	}

	// Network policy logs are node logs, so the node name filter shared with the other node logs applies to them as well.
	if len(nodeNameSubstrings) > 0 {
		filters = append(filters, logestimator.ResourceLabel("node_name", logestimator.ContainsAny(nodeNameSubstrings...)))
	}

	return &logestimator.StructuredLogQuery{
		Incomplete:    !cluster.IsComplete(),
		ResourceTypes: []string{"k8s_node"},
		Filters:       filters,
	}
}

// policyActionQuerySource builds the Cloud Logging query for the network policy logs of the cluster.
type policyActionQuerySource struct {
	clusterIdentity    coretask.Input[k8scommon.GoogleCloudClusterIdentity]
	nodeNameSubstrings coretask.Input[[]string]
}

// DefaultResourceNames implements gcpcommon.StructuredLogQuerySource.
func (s *policyActionQuerySource) DefaultResourceNames(ctx context.Context) ([]string, error) {
	cluster := s.clusterIdentity.Get(ctx)
	return []string{fmt.Sprintf("projects/%s", cluster.ProjectID)}, nil
}

// Queries implements gcpcommon.StructuredLogQuerySource.
func (s *policyActionQuerySource) Queries(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]*logestimator.StructuredLogQuery, error) {
	cluster := s.clusterIdentity.Get(ctx)
	nodeNameSubstrings := s.nodeNameSubstrings.Get(ctx)
	return []*logestimator.StructuredLogQuery{generatePolicyActionStructuredQuery(cluster, nodeNameSubstrings)}, nil
}

// TimePartitionCount implements gcpcommon.StructuredLogQuerySource.
func (s *policyActionQuerySource) TimePartitionCount() int {
	return 10
}

var _ gcpcommon.StructuredLogQuerySource = (*policyActionQuerySource)(nil)

// listLogEntriesTask queries the GKE Dataplane V2 network policy logs of the cluster from Cloud Logging.
var listLogEntriesTask = gcpcommon.DefineStructuredListLogEntriesTask(
	dpv2policy.ListLogEntriesTaskID,
	"Dataplane V2 network policy logs",
	func(b *coretask.Binder) gcpcommon.StructuredLogQuerySource {
		return &policyActionQuerySource{
			clusterIdentity:    coretask.Use(b, dpv2policy.ClusterIdentityTaskID.Ref()),
			nodeNameSubstrings: coretask.Use(b, k8scommon.InputNodeNameFilterTaskID.Ref()),
		}
	},
)
