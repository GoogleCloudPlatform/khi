// Copyright 2025 Google LLC
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

package serialport_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/serialport"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// maxNodesPerQuery is the maximum number of node names that one serial port log query filters by.
const maxNodesPerQuery = 30

// generateSerialPortStructuredQuery generates structured log queries for serial port logs.
func generateSerialPortStructuredQuery(taskMode inspectioncore.InspectionTaskModeType, foundNodeNames []string, nodeNameSubstrings []string) []*logestimator.StructuredLogQuery {
	logIDFilter := logestimator.LogID(logestimator.OneOf(
		"serialconsole.googleapis.com/serial_port_1_output",
		"serialconsole.googleapis.com/serial_port_2_output",
		"serialconsole.googleapis.com/serial_port_3_output",
		"serialconsole.googleapis.com/serial_port_debug_output",
	))

	var subFilter logestimator.LoggingMonitoringMatcher
	if len(nodeNameSubstrings) > 0 {
		subFilter = logestimator.CustomFilter(fmt.Sprintf(`labels."compute.googleapis.com/resource_name":(%s)`, strings.Join(gcpqueryutil.WrapDoubleQuoteForStringArray(nodeNameSubstrings), " OR ")))
	}

	if taskMode == inspectioncore.TaskModeDryRun {
		filters := []logestimator.LoggingMonitoringMatcher{
			logIDFilter,
			logestimator.Comment("instance name filters to be determined after node name discovery"),
		}
		if subFilter != nil {
			filters = append(filters, subFilter)
		}
		return []*logestimator.StructuredLogQuery{
			{
				Filters: filters,
				Preset:  logestimator.EstimatedCountPresetFew,
			},
		}
	}

	result := []*logestimator.StructuredLogQuery{}
	instanceNameGroups := gcpqueryutil.SplitToChildGroups(foundNodeNames, maxNodesPerQuery)
	for _, group := range instanceNameGroups {
		instanceNameFilter := logestimator.CustomFilter(fmt.Sprintf(`labels."compute.googleapis.com/resource_name"=(%s)`, strings.Join(gcpqueryutil.WrapDoubleQuoteForStringArray(group), " OR ")))
		filters := []logestimator.LoggingMonitoringMatcher{
			logIDFilter,
			instanceNameFilter,
		}
		if subFilter != nil {
			filters = append(filters, subFilter)
		}
		result = append(result, &logestimator.StructuredLogQuery{
			Filters: filters,
		})
	}
	return result
}

// serialPortQuerySource builds the serial port log queries for the cluster nodes.
type serialPortQuerySource struct {
	clusterIdentity    coretask.Input[k8scommon.GoogleCloudClusterIdentity]
	nodeNameSubstrings coretask.Input[[]string]
	nodeNames          coretask.Input[[]string]
}

// DefaultResourceNames implements gcpcommon.StructuredLogQuerySource.
func (s *serialPortQuerySource) DefaultResourceNames(ctx context.Context) ([]string, error) {
	return []string{fmt.Sprintf("projects/%s", s.clusterIdentity.Get(ctx).ProjectID)}, nil
}

// Queries implements gcpcommon.StructuredLogQuerySource.
func (s *serialPortQuerySource) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	nodeNames := s.nodeNames.Get(ctx)
	nodeNameSubstrings := s.nodeNameSubstrings.Get(ctx)
	clusterIdentity := s.clusterIdentity.Get(ctx)
	taskMode := inspectioncore.TaskModeRun
	if val, err := khictx.GetValue(ctx, inspectioncore.InspectionTaskMode); err == nil {
		taskMode = val
	}
	queries := generateSerialPortStructuredQuery(taskMode, nodeNames, nodeNameSubstrings)
	if clusterIdentity.ProjectID == "" {
		for _, q := range queries {
			q.Incomplete = true
		}
	}
	return queries, nil
}

// TimePartitionCount implements gcpcommon.StructuredLogQuerySource.
func (s *serialPortQuerySource) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ gcpcommon.StructuredLogQuerySource = (*serialPortQuerySource)(nil)

// logQueryTask queries the serial port logs of the cluster nodes from Cloud Logging.
var logQueryTask = gcpcommon.DefineStructuredListLogEntriesTask(
	serialport.LogQueryTaskID,
	"Serial port log",
	func(b *coretask.Binder) gcpcommon.StructuredLogQuerySource {
		return &serialPortQuerySource{
			clusterIdentity:    coretask.Use(b, serialport.ClusterIdentityTaskID.Ref()),
			nodeNameSubstrings: coretask.Use(b, k8scommon.InputNodeNameFilterTaskID.Ref()),
			nodeNames:          coretask.Use(b, k8saudit.NodeNameInventoryTaskID.Ref()),
		}
	},
)
