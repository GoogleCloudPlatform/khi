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

package googlecloudloggkeapiaudit_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudloggkeapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudloggkeapiaudit/contract"
)

// GenerateGKEAuditStructuredQuery generates a structured query for GKE API audit logs.
func GenerateGKEAuditStructuredQuery(cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity) *logestimator.StructuredLogQuery {
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

// GenerateGKEAuditQuery generates a query for GKE API audit logs.
func GenerateGKEAuditQuery(cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity) string {
	return GenerateGKEAuditStructuredQuery(cluster).GenerateCloudLoggingQuery()
}

type gkeAPIListLogEntriesTaskSetting struct {
}

// DefaultResourceNames implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (g *gkeAPIListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudloggkeapiaudit_contract.ClusterIdentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", clusterIdentity.ProjectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (g *gkeAPIListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudloggkeapiaudit_contract.ClusterIdentityTaskID.Ref(),
	}
}

// QueryName implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (g *gkeAPIListLogEntriesTaskSetting) QueryName() string {
	return "GKE Audit logs"
}

// Queries implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (g *gkeAPIListLogEntriesTaskSetting) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudloggkeapiaudit_contract.ClusterIdentityTaskID.Ref())
	return []*logestimator.StructuredLogQuery{GenerateGKEAuditStructuredQuery(cluster)}, nil
}

// TaskID implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (g *gkeAPIListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudloggkeapiaudit_contract.ListLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (g *gkeAPIListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 1, nil
}

var _ googlecloudcommon_contract.StructuredListLogEntriesTaskSetting = (*gkeAPIListLogEntriesTaskSetting)(nil)

var ListLogEntriesTask = googlecloudcommon_contract.NewStructuredListLogEntriesTask(&gkeAPIListLogEntriesTaskSetting{})
