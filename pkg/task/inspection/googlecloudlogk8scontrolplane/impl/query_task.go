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

package googlecloudlogk8scontrolplane_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	googlecloudlogk8scontrolplane_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/contract"
)

// GenerateK8sControlPlaneStructuredQuery generates a structured query for Kubernetes control plane logs.
func GenerateK8sControlPlaneStructuredQuery(cluster k8scommon.GoogleCloudClusterIdentity, controlplaneComponentFilter *gcpqueryutil.SetFilterParseResult) *logestimator.StructuredLogQuery {
	filters := []logestimator.LoggingMonitoringMatcher{
		logestimator.ResourceLabel("project_id", logestimator.Exact(cluster.ProjectID)),
		logestimator.ResourceLabel("location", logestimator.Exact(cluster.Location)),
		logestimator.ResourceLabel("cluster_name", logestimator.Exact(cluster.NameFor(k8scommon.ClusterNameUsageK8sCluster))),
	}

	if controlplaneComponentFilter != nil {
		switch {
		case controlplaneComponentFilter.ValidationError != "":
			filters = append(filters, logestimator.Comment(fmt.Sprintf(`Failed to generate component name filter due to the validation error "%s"`, controlplaneComponentFilter.ValidationError)))
		case controlplaneComponentFilter.SubtractMode:
			if len(controlplaneComponentFilter.Subtractives) > 0 {
				filters = append(filters, logestimator.ResourceLabel("component_name", logestimator.NotContainsAny(controlplaneComponentFilter.Subtractives...)))
			}
		case len(controlplaneComponentFilter.Additives) == 0:
			filters = append(filters, logestimator.Comment(`Invalid: none of the controlplane components will be selected. Ignoring component name filter.`))
		default:
			filters = append(filters, logestimator.ResourceLabel("component_name", logestimator.ContainsAny(controlplaneComponentFilter.Additives...)))
		}
	}

	filters = append(filters, logestimator.CustomFilter(`-sourceLocation.file="httplog.go" -- Ignoring the noisy log from scheduler. TODO: Support toggling this feature.`))

	return &logestimator.StructuredLogQuery{
		Incomplete:    !cluster.IsComplete(),
		ResourceTypes: []string{"k8s_control_plane_component"},
		Filters:       filters,
	}
}

// GenerateK8sControlPlaneQuery generates a query for Kubernetes control plane logs.
func GenerateK8sControlPlaneQuery(cluster k8scommon.GoogleCloudClusterIdentity, controlplaneComponentFilter *gcpqueryutil.SetFilterParseResult) string {
	return GenerateK8sControlPlaneStructuredQuery(cluster, controlplaneComponentFilter).GenerateCloudLoggingQuery()
}

type controlPlaneListLogEntriesTaskSetting struct {
}

// Dependencies implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (c *controlPlaneListLogEntriesTaskSetting) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		googlecloudlogk8scontrolplane_contract.ClusterIdentityTaskID.Ref(),
		googlecloudlogk8scontrolplane_contract.InputControlPlaneComponentNameFilterTaskID.Ref(),
	}
}

// DefaultResourceNames implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (c *controlPlaneListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudlogk8scontrolplane_contract.ClusterIdentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", cluster.ProjectID)}, nil
}

// QueryName implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (c *controlPlaneListLogEntriesTaskSetting) QueryName() string {
	return "K8s control plane logs"
}

// Queries implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (c *controlPlaneListLogEntriesTaskSetting) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudlogk8scontrolplane_contract.ClusterIdentityTaskID.Ref())
	controlplaneComponentNameFilter := coretask.GetTaskResult(ctx, googlecloudlogk8scontrolplane_contract.InputControlPlaneComponentNameFilterTaskID.Ref())
	return []*logestimator.StructuredLogQuery{GenerateK8sControlPlaneStructuredQuery(cluster, controlplaneComponentNameFilter)}, nil
}

// TaskID implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (c *controlPlaneListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogk8scontrolplane_contract.ListLogEntriesTaskID
}

// TimePartitionCount implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (c *controlPlaneListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ gcpcommon.StructuredListLogEntriesTaskSetting = (*controlPlaneListLogEntriesTaskSetting)(nil)

var ListLogEntriesTask = gcpcommon.NewStructuredListLogEntriesTask(&controlPlaneListLogEntriesTaskSetting{})
