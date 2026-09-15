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

package googlecloudloggkeautoscaler_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	googlecloudloggkeautoscaler_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudloggkeautoscaler/contract"
)

// GenerateAutoscalerStructuredQuery generates a structured query for GKE cluster autoscaler logs.
func GenerateAutoscalerStructuredQuery(cluster k8scommon.GoogleCloudClusterIdentity, excludeStatus bool) *logestimator.StructuredLogQuery {
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

type autoscalerListLogEntriesTaskSetting struct{}

// DefaultResourceNames implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (a *autoscalerListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, k8scommon.ClusterIdentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", cluster.ProjectID)}, nil
}

// Dependencies implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (a *autoscalerListLogEntriesTaskSetting) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		k8scommon.ClusterIdentityTaskID.Ref(),
	}
}

// QueryName implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (a *autoscalerListLogEntriesTaskSetting) QueryName() string {
	return "Cluster autoscaler logs"
}

// Queries implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (a *autoscalerListLogEntriesTaskSetting) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	cluster := coretask.GetTaskResult(ctx, k8scommon.ClusterIdentityTaskID.Ref())
	return []*logestimator.StructuredLogQuery{GenerateAutoscalerStructuredQuery(cluster, true)}, nil
}

// TaskID implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (a *autoscalerListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudloggkeautoscaler_contract.ListLogEntriesTaskID
}

// TimePartitionCount implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (a *autoscalerListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 1, nil
}

var _ gcpcommon.StructuredListLogEntriesTaskSetting = (*autoscalerListLogEntriesTaskSetting)(nil)

var ListLogEntriesTask = gcpcommon.NewStructuredListLogEntriesTask(&autoscalerListLogEntriesTaskSetting{})
