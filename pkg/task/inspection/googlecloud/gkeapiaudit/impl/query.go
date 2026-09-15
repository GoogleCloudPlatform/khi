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
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gkeapiaudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
)

// GenerateGKEAuditStructuredQuery generates a structured query for GKE API audit logs.
func GenerateGKEAuditStructuredQuery(cluster k8scommon.GoogleCloudClusterIdentity) *logestimator.StructuredLogQuery {
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

type gkeAPIListLogEntriesTaskSetting struct {
}

// DefaultResourceNames implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (g *gkeAPIListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, gkeapiaudit.ClusterIdentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", clusterIdentity.ProjectID)}, nil
}

// Dependencies implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (g *gkeAPIListLogEntriesTaskSetting) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		gkeapiaudit.ClusterIdentityTaskID.Ref(),
	}
}

// QueryName implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (g *gkeAPIListLogEntriesTaskSetting) QueryName() string {
	return "GKE Audit logs"
}

// Queries implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (g *gkeAPIListLogEntriesTaskSetting) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	cluster := coretask.GetTaskResult(ctx, gkeapiaudit.ClusterIdentityTaskID.Ref())
	return []*logestimator.StructuredLogQuery{GenerateGKEAuditStructuredQuery(cluster)}, nil
}

// TaskID implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (g *gkeAPIListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return gkeapiaudit.ListLogEntriesTaskID
}

// TimePartitionCount implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (g *gkeAPIListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 1, nil
}

var _ gcpcommon.StructuredListLogEntriesTaskSetting = (*gkeAPIListLogEntriesTaskSetting)(nil)

var ListLogEntriesTask = gcpcommon.NewStructuredListLogEntriesTask(&gkeAPIListLogEntriesTaskSetting{})
