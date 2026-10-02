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

// Package computeapiaudit_impl defines the implementation of compute API audit inspection tasks.
package computeapiaudit_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/computeapiaudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// generateComputeAPIStructuredQuery generates a structured query slice for compute API logs.
func generateComputeAPIStructuredQuery(taskMode inspectioncore.InspectionTaskModeType, nodeNames []string) []*logestimator.StructuredLogQuery {
	if taskMode == inspectioncore.TaskModeDryRun {
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

// computeAuditQuerySource builds the Compute API audit log queries for the cluster.
type computeAuditQuerySource struct {
	clusterIdentity coretask.Input[k8scommon.GoogleCloudClusterIdentity]
	nodeNames       coretask.Input[[]string]
}

// DefaultResourceNames implements gcpcommon.StructuredLogQuerySource.
func (s *computeAuditQuerySource) DefaultResourceNames(ctx context.Context) ([]string, error) {
	return []string{fmt.Sprintf("projects/%s", s.clusterIdentity.Get(ctx).ProjectID)}, nil
}

// Queries implements gcpcommon.StructuredLogQuerySource.
func (s *computeAuditQuerySource) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	taskMode, err := khictx.GetValue(ctx, inspectioncore.InspectionTaskMode)
	if err != nil {
		taskMode = inspectioncore.TaskModeRun
	}
	var nodeNames []string
	if taskMode == inspectioncore.TaskModeRun {
		nodeNames = s.nodeNames.Get(ctx)
	}
	clusterIdentity := s.clusterIdentity.Get(ctx)
	queries := generateComputeAPIStructuredQuery(taskMode, nodeNames)
	if !clusterIdentity.IsComplete() {
		for _, q := range queries {
			q.Incomplete = true
		}
	}
	return queries, nil
}

// TimePartitionCount implements gcpcommon.StructuredLogQuerySource.
func (s *computeAuditQuerySource) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ gcpcommon.StructuredLogQuerySource = (*computeAuditQuerySource)(nil)

// listLogEntriesTask queries compute API audit logs from Cloud Logging.
var listLogEntriesTask = gcpcommon.DefineStructuredListLogEntriesTask(
	computeapiaudit.ListLogEntriesTaskID,
	"Compute API Audit log",
	func(b *coretask.Binder) gcpcommon.StructuredLogQuerySource {
		return &computeAuditQuerySource{
			clusterIdentity: coretask.Use(b, computeapiaudit.ClusterIdentityTaskID.Ref()),
			nodeNames:       coretask.Use(b, k8saudit.NodeNameInventoryTaskID.Ref()),
		}
	},
)
