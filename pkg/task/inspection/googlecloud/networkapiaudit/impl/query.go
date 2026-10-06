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

package networkapiaudit_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/networkapiaudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// generateGCPNetworkAPIStructuredQuery generates a structured query slice for network API logs.
func generateGCPNetworkAPIStructuredQuery(taskMode inspectioncore.InspectionTaskModeType, negNames []string) []*logestimator.StructuredLogQuery {
	if taskMode == inspectioncore.TaskModeDryRun {
		return []*logestimator.StructuredLogQuery{
			{
				ResourceTypes: []string{"gce_network"},
				Filters: []logestimator.LoggingMonitoringMatcher{
					logestimator.CustomFilter(`-protoPayload.methodName:("list" OR "get" OR "watch")`),
					logestimator.Comment("neg name filters to be determined after audit log query"),
				},
			},
		}
	}

	nodeNamesWithNetworkEndpointGroups := []string{}
	for _, negName := range negNames {
		nodeNamesWithNetworkEndpointGroups = append(nodeNamesWithNetworkEndpointGroups, fmt.Sprintf("networkEndpointGroups/%s", negName))
	}
	result := []*logestimator.StructuredLogQuery{}
	groups := gcpqueryutil.SplitToChildGroups(nodeNamesWithNetworkEndpointGroups, 10)
	for _, group := range groups {
		negNameFilter := fmt.Sprintf("protoPayload.resourceName:(%s)", strings.Join(group, " OR "))
		result = append(result, &logestimator.StructuredLogQuery{
			ResourceTypes: []string{"gce_network"},
			Filters: []logestimator.LoggingMonitoringMatcher{
				logestimator.CustomFilter(`-protoPayload.methodName:("list" OR "get" OR "watch")`),
				logestimator.CustomFilter(negNameFilter),
			},
		})
	}
	return result
}

// networkAuditQuerySource generates log queries for GCE Network API audit logs.
type networkAuditQuerySource struct {
	clusterIdentity coretask.Input[k8scommon.GoogleCloudClusterIdentity]
	negs            coretask.Input[k8scommon.NEGNameToResourceIdentityMap]
}

// DefaultResourceNames implements gcpcommon.StructuredLogQuerySource.
func (s *networkAuditQuerySource) DefaultResourceNames(ctx context.Context) ([]string, error) {
	return []string{fmt.Sprintf("projects/%s", s.clusterIdentity.Get(ctx).ProjectID)}, nil
}

// Queries implements gcpcommon.StructuredLogQuerySource.
func (s *networkAuditQuerySource) Queries(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]*logestimator.StructuredLogQuery, error) {
	var negNames []string
	if taskMode == inspectioncore.TaskModeRun {
		negs := s.negs.Get(ctx)
		for negName := range negs {
			negNames = append(negNames, negName)
		}
	}
	clusterIdentity := s.clusterIdentity.Get(ctx)
	queries := generateGCPNetworkAPIStructuredQuery(taskMode, negNames)
	if clusterIdentity.ProjectID == "" {
		for _, q := range queries {
			q.Incomplete = true
		}
	}
	return queries, nil
}

// TimePartitionCount implements gcpcommon.StructuredLogQuerySource.
func (s *networkAuditQuerySource) TimePartitionCount() int {
	return 1
}

var _ gcpcommon.StructuredLogQuerySource = (*networkAuditQuerySource)(nil)

// listLogEntriesTask queries GCP network logs from Cloud Logging.
var listLogEntriesTask = gcpcommon.DefineStructuredListLogEntriesTask(
	networkapiaudit.ListLogEntriesTaskID,
	"GCP network log",
	func(b *coretask.Binder) gcpcommon.StructuredLogQuerySource {
		return &networkAuditQuerySource{
			clusterIdentity: coretask.Use(b, networkapiaudit.ClusterIdentityTaskID.Ref()),
			negs:            coretask.Use(b, k8scommon.NEGNamesInventoryTaskID.Ref()),
		}
	},
)
