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

package gkeapiaudit_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gkeapiaudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// generateGKEAuditStructuredQuery generates a structured query for GKE API audit logs.
func generateGKEAuditStructuredQuery(cluster k8scommon.GoogleCloudClusterIdentity) *logestimator.StructuredLogQuery {
	return &logestimator.StructuredLogQuery{
		Incomplete:                !cluster.IsComplete(),
		ResourceTypes:             []string{"gke_cluster", "gke_nodepool"},
		IgnoreMetricsResourceType: []string{"gke_cluster", "gke_nodepool"},
		Filters: []logestimator.LoggingMonitoringMatcher{
			logestimator.ResourceLabel("project_id", logestimator.Exact(cluster.ProjectID)),
			logestimator.ResourceLabel("location", logestimator.Exact(cluster.Location)),
			logestimator.ResourceLabel("cluster_name", logestimator.Exact(cluster.ClusterName)),
			logestimator.LogID(logestimator.OneOf("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")),
			logestimator.CustomFilter(`protoPayload.serviceName="container.googleapis.com"`),
		},
	}
}

// gkeAuditQuerySource builds the GKE audit log query for the cluster read through the cluster identity input.
type gkeAuditQuerySource struct {
	clusterIdentity coretask.Input[k8scommon.GoogleCloudClusterIdentity]
}

// DefaultResourceNames implements gcpcommon.StructuredLogQuerySource.
func (s *gkeAuditQuerySource) DefaultResourceNames(ctx context.Context) ([]string, error) {
	return []string{fmt.Sprintf("projects/%s", s.clusterIdentity.Get(ctx).ProjectID)}, nil
}

// Queries implements gcpcommon.StructuredLogQuerySource.
func (s *gkeAuditQuerySource) Queries(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]*logestimator.StructuredLogQuery, error) {
	return []*logestimator.StructuredLogQuery{generateGKEAuditStructuredQuery(s.clusterIdentity.Get(ctx))}, nil
}

// TimePartitionCount implements gcpcommon.StructuredLogQuerySource.
func (s *gkeAuditQuerySource) TimePartitionCount() int {
	return 1
}

var _ gcpcommon.StructuredLogQuerySource = (*gkeAuditQuerySource)(nil)

// listLogEntriesTask queries GKE audit logs of the cluster from Cloud Logging.
var listLogEntriesTask = gcpcommon.DefineStructuredListLogEntriesTask(gkeapiaudit.ListLogEntriesTaskID, "GKE Audit logs", func(b *coretask.Binder) gcpcommon.StructuredLogQuerySource {
	return &gkeAuditQuerySource{clusterIdentity: coretask.Use(b, gkeapiaudit.ClusterIdentityTaskID.Ref())}
})
