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

package k8scontrolplane_impl

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scontrolplane"
)

// generateK8sControlPlaneStructuredQuery generates a structured query for Kubernetes control plane logs.
func generateK8sControlPlaneStructuredQuery(cluster k8scommon.GoogleCloudClusterIdentity, controlplaneComponentFilter *gcpqueryutil.SetFilterParseResult) *logestimator.StructuredLogQuery {
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

// controlPlaneLogQuerySource builds the Cloud Logging query for the Kubernetes control plane component logs of the cluster.
type controlPlaneLogQuerySource struct {
	clusterIdentity     coretask.Input[k8scommon.GoogleCloudClusterIdentity]
	componentNameFilter coretask.Input[*gcpqueryutil.SetFilterParseResult]
}

// DefaultResourceNames implements gcpcommon.StructuredLogQuerySource.
func (s *controlPlaneLogQuerySource) DefaultResourceNames(ctx context.Context) ([]string, error) {
	cluster := s.clusterIdentity.Get(ctx)
	return []string{fmt.Sprintf("projects/%s", cluster.ProjectID)}, nil
}

// Queries implements gcpcommon.StructuredLogQuerySource.
func (s *controlPlaneLogQuerySource) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	cluster := s.clusterIdentity.Get(ctx)
	controlplaneComponentNameFilter := s.componentNameFilter.Get(ctx)
	return []*logestimator.StructuredLogQuery{generateK8sControlPlaneStructuredQuery(cluster, controlplaneComponentNameFilter)}, nil
}

// TimePartitionCount implements gcpcommon.StructuredLogQuerySource.
func (s *controlPlaneLogQuerySource) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ gcpcommon.StructuredLogQuerySource = (*controlPlaneLogQuerySource)(nil)

// listLogEntriesTask queries the Kubernetes control plane component logs of the cluster from Cloud Logging.
var listLogEntriesTask = gcpcommon.DefineStructuredListLogEntriesTask(
	k8scontrolplane.ListLogEntriesTaskID,
	"K8s control plane logs",
	func(b *coretask.Binder) gcpcommon.StructuredLogQuerySource {
		return &controlPlaneLogQuerySource{
			clusterIdentity:     coretask.Use(b, k8scontrolplane.ClusterIdentityTaskID.Ref()),
			componentNameFilter: coretask.Use(b, k8scontrolplane.InputControlPlaneComponentNameFilterTaskID.Ref()),
		}
	},
)
