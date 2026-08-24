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

package googlecloudlogk8snode_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
)

// GenerateK8sNodeStructuredQuery generates a structured query for GKE node logs.
func GenerateK8sNodeStructuredQuery(cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity, nodeNameSubstrings []string) *logestimator.StructuredLogQuery {
	filters := []logestimator.LoggingMonitoringMatcher{
		logestimator.ResourceLabel("project_id", logestimator.Exact(cluster.ProjectID)),
		logestimator.ResourceLabel("location", logestimator.Exact(cluster.Location)),
		logestimator.ResourceLabel("cluster_name", logestimator.Exact(cluster.NameFor(googlecloudk8scommon_contract.ClusterNameUsageK8sCluster))),
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

// GenerateK8sNodeLogQuery generates a query for GKE node logs.
func GenerateK8sNodeLogQuery(cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity, nodeNameSubstrings []string) string {
	return GenerateK8sNodeStructuredQuery(cluster, nodeNameSubstrings).GenerateCloudLoggingQuery()
}

type k8snodeListLogEntriesTaskSetting struct{}

// DefaultResourceNames implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *k8snodeListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudlogk8snode_contract.ClusterIdentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", cluster.ProjectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *k8snodeListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudlogk8snode_contract.ClusterIdentityTaskID.Ref(),
		googlecloudk8scommon_contract.InputNodeNameFilterTaskID.Ref(),
	}
}

// QueryName implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *k8snodeListLogEntriesTaskSetting) QueryName() string {
	return "Kubernetes node logs"
}

// Queries implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *k8snodeListLogEntriesTaskSetting) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudlogk8snode_contract.ClusterIdentityTaskID.Ref())
	nodeNameSubstrings := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputNodeNameFilterTaskID.Ref())
	return []*logestimator.StructuredLogQuery{GenerateK8sNodeStructuredQuery(cluster, nodeNameSubstrings)}, nil
}

// TaskID implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *k8snodeListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogk8snode_contract.ListLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (k *k8snodeListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ googlecloudcommon_contract.StructuredListLogEntriesTaskSetting = (*k8snodeListLogEntriesTaskSetting)(nil)

var ListLogEntriesTask = googlecloudcommon_contract.NewStructuredListLogEntriesTask(&k8snodeListLogEntriesTaskSetting{})
