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
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/networkapiaudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// GenerateGCPNetworkAPIStructuredQuery generates a structured query slice for network API logs.
func GenerateGCPNetworkAPIStructuredQuery(taskMode inspectioncore.InspectionTaskModeType, negNames []string) []*logestimator.StructuredLogQuery {
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

type networkAPIListLogEntriesTaskSetting struct{}

// DefaultResourceNames implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (n *networkAPIListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, networkapiaudit.ClusterIdentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", clusterIdentity.ProjectID)}, nil
}

// Dependencies implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (n *networkAPIListLogEntriesTaskSetting) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		networkapiaudit.ClusterIdentityTaskID.Ref(),
		k8scommon.NEGNamesInventoryTaskID.Ref(),
	}
}

// QueryName implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (n *networkAPIListLogEntriesTaskSetting) QueryName() string {
	return "GCP network log"
}

// Queries implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (n *networkAPIListLogEntriesTaskSetting) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	taskMode, err := khictx.GetValue(ctx, inspectioncore.InspectionTaskMode)
	if err != nil {
		taskMode = inspectioncore.TaskModeRun
	}
	var negNames []string
	if taskMode == inspectioncore.TaskModeRun {
		negs := coretask.GetTaskResult(ctx, k8scommon.NEGNamesInventoryTaskID.Ref())
		for negName := range negs {
			negNames = append(negNames, negName)
		}
	}
	clusterIdentity := coretask.GetTaskResult(ctx, networkapiaudit.ClusterIdentityTaskID.Ref())
	queries := GenerateGCPNetworkAPIStructuredQuery(taskMode, negNames)
	if clusterIdentity.ProjectID == "" {
		for _, q := range queries {
			q.Incomplete = true
		}
	}
	return queries, nil
}

// TaskID implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (n *networkAPIListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return networkapiaudit.ListLogEntriesTaskID
}

// TimePartitionCount implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (n *networkAPIListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 1, nil
}

var _ gcpcommon.StructuredListLogEntriesTaskSetting = (*networkAPIListLogEntriesTaskSetting)(nil)

var ListLogEntriesTask = gcpcommon.NewStructuredListLogEntriesTask(&networkAPIListLogEntriesTaskSetting{})
