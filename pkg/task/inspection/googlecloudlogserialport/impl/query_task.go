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

package googlecloudlogserialport_impl

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
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	googlecloudlogserialport_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogserialport/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

const MaxNodesPerQuery = 30

// GenerateSerialPortStructuredQuery generates structured log queries for serial port logs.
func GenerateSerialPortStructuredQuery(taskMode inspectioncore.InspectionTaskModeType, foundNodeNames []string, nodeNameSubstrings []string) []*logestimator.StructuredLogQuery {
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
	instanceNameGroups := gcpqueryutil.SplitToChildGroups(foundNodeNames, MaxNodesPerQuery)
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

type serialPortLoggingFilterTaskSetting struct {
}

// Dependencies implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (s *serialPortLoggingFilterTaskSetting) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		googlecloudlogserialport_contract.ClusterIdentityTaskID.Ref(),
		k8scommon.InputNodeNameFilterTaskID.Ref(),
		k8saudit.NodeNameInventoryTaskID.Ref(),
	}
}

// QueryName implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (s *serialPortLoggingFilterTaskSetting) QueryName() string {
	return "Serial port log"
}

// Queries implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (s *serialPortLoggingFilterTaskSetting) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	nodeNames := coretask.GetTaskResult(ctx, k8saudit.NodeNameInventoryTaskID.Ref())
	nodeNameSubstrings := coretask.GetTaskResult(ctx, k8scommon.InputNodeNameFilterTaskID.Ref())
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudlogserialport_contract.ClusterIdentityTaskID.Ref())
	taskMode := inspectioncore.TaskModeRun
	if val, err := khictx.GetValue(ctx, inspectioncore.InspectionTaskMode); err == nil {
		taskMode = val
	}
	queries := GenerateSerialPortStructuredQuery(taskMode, nodeNames, nodeNameSubstrings)
	if clusterIdentity.ProjectID == "" {
		for _, q := range queries {
			q.Incomplete = true
		}
	}
	return queries, nil
}

// DefaultResourceNames implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (s *serialPortLoggingFilterTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudlogserialport_contract.ClusterIdentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", clusterIdentity.ProjectID)}, nil
}

// TaskID implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (s *serialPortLoggingFilterTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogserialport_contract.LogQueryTaskID
}

// TimePartitionCount implements gcpcommon.StructuredListLogEntriesTaskSetting.
func (s *serialPortLoggingFilterTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ gcpcommon.StructuredListLogEntriesTaskSetting = (*serialPortLoggingFilterTaskSetting)(nil)

var LogQueryTask = gcpcommon.NewStructuredListLogEntriesTask(&serialPortLoggingFilterTaskSetting{})
