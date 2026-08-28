// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package googlecloudlogcomputeapiaudit_impl defines the implementation of the googlecloudlogcomputeapiaudit task.
package googlecloudlogcomputeapiaudit_impl

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
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogcomputeapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcomputeapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// GenerateComputeAPIStructuredQuery generates a structured query slice for compute API logs.
func GenerateComputeAPIStructuredQuery(taskMode inspectioncore_contract.InspectionTaskModeType, nodeNames []string) []*logestimator.StructuredLogQuery {
	if taskMode == inspectioncore_contract.TaskModeDryRun {
		return []*logestimator.StructuredLogQuery{
			{
				ResourceTypes: []string{"gce_instance"},
				Filters: []logestimator.LoggingMonitoringMatcher{
					logestimator.CustomFilter(`-protoPayload.methodName:("list" OR "get" OR "watch")`),
					logestimator.Comment("instance name filters to be determined after node name discovery"),
				},
				Preset: logestimator.EstimatedCountPresetFew,
			},
		}
	}

	result := []*logestimator.StructuredLogQuery{}
	instanceNameGroups := gcpqueryutil.SplitToChildGroups(nodeNames, 30)
	for _, group := range instanceNameGroups {
		nodeNamesWithInstance := []string{}
		for _, nodeName := range group {
			nodeNamesWithInstance = append(nodeNamesWithInstance, fmt.Sprintf("instances/%s", nodeName))
		}
		instanceNameFilter := fmt.Sprintf("protoPayload.resourceName:(%s)", strings.Join(nodeNamesWithInstance, " OR "))
		result = append(result, &logestimator.StructuredLogQuery{
			ResourceTypes: []string{"gce_instance"},
			Filters: []logestimator.LoggingMonitoringMatcher{
				logestimator.CustomFilter(`-protoPayload.methodName:("list" OR "get" OR "watch")`),
				logestimator.CustomFilter(instanceNameFilter),
			},
		})
	}
	return result
}

// GenerateComputeAPIQuery generates a query for compute API logs.
func GenerateComputeAPIQuery(taskMode inspectioncore_contract.InspectionTaskModeType, nodeNames []string) []string {
	structuredQueries := GenerateComputeAPIStructuredQuery(taskMode, nodeNames)
	result := make([]string, 0, len(structuredQueries))
	for _, sq := range structuredQueries {
		result = append(result, sq.GenerateCloudLoggingQuery())
	}
	return result
}

type computeAPIListLogEntriesTaskSetting struct {
}

// DefaultResourceNames implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *computeAPIListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudlogcomputeapiaudit_contract.ClusterIdentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", clusterIdentity.ProjectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *computeAPIListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		commonlogk8saudit_contract.NodeNameInventoryTaskID.Ref(),
		googlecloudlogcomputeapiaudit_contract.ClusterIdentityTaskID.Ref(),
	}
}

// QueryName implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *computeAPIListLogEntriesTaskSetting) QueryName() string {
	return "Compute API Audit log"
}

// Queries implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *computeAPIListLogEntriesTaskSetting) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	taskMode, err := khictx.GetValue(ctx, inspectioncore_contract.InspectionTaskMode)
	if err != nil {
		taskMode = inspectioncore_contract.TaskModeRun
	}
	var nodeNames []string
	if taskMode == inspectioncore_contract.TaskModeRun {
		nodeNames = coretask.GetTaskResult(ctx, commonlogk8saudit_contract.NodeNameInventoryTaskID.Ref())
	}
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudlogcomputeapiaudit_contract.ClusterIdentityTaskID.Ref())
	queries := GenerateComputeAPIStructuredQuery(taskMode, nodeNames)
	if !clusterIdentity.IsComplete() {
		for _, q := range queries {
			q.Incomplete = true
		}
	}
	return queries, nil
}

// TaskID implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *computeAPIListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogcomputeapiaudit_contract.ListLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (c *computeAPIListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ googlecloudcommon_contract.StructuredListLogEntriesTaskSetting = (*computeAPIListLogEntriesTaskSetting)(nil)

var ListLogEntriesTask = googlecloudcommon_contract.NewStructuredListLogEntriesTask(&computeAPIListLogEntriesTaskSetting{})
