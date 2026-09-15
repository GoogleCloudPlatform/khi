// Copyright 2026 Google LLC
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
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/idgenerator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/serialport"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
	"github.com/google/go-cmp/cmp"
)

func TestGenerateSerialPortStructuredQuery(t *testing.T) {
	testCases := []struct {
		name                   string
		taskMode               inspectioncore.InspectionTaskModeType
		nodeNames              []string
		nodeNameSubstrings     []string
		wantQueries            []string
		wantSupportMetricsFlag bool
	}{
		{
			name:               "dryrun with no substrings",
			taskMode:           inspectioncore.TaskModeDryRun,
			nodeNames:          []string{"node-1", "node-2"},
			nodeNameSubstrings: []string{},
			wantQueries: []string{
				`(LOG_ID("serialconsole.googleapis.com/serial_port_1_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_2_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_3_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_debug_output"))
-- instance name filters to be determined after node name discovery`,
			},
			wantSupportMetricsFlag: true,
		},
		{
			name:               "dryrun with substrings",
			taskMode:           inspectioncore.TaskModeDryRun,
			nodeNames:          []string{"node-1"},
			nodeNameSubstrings: []string{"sub-a", "sub-b"},
			wantQueries: []string{
				`(LOG_ID("serialconsole.googleapis.com/serial_port_1_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_2_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_3_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_debug_output"))
-- instance name filters to be determined after node name discovery
labels."compute.googleapis.com/resource_name":("sub-a" OR "sub-b")`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:                   "run mode with 0 nodes",
			taskMode:               inspectioncore.TaskModeRun,
			nodeNames:              []string{},
			nodeNameSubstrings:     []string{},
			wantQueries:            []string{},
			wantSupportMetricsFlag: false,
		},
		{
			name:               "run mode with single node",
			taskMode:           inspectioncore.TaskModeRun,
			nodeNames:          []string{"node-1"},
			nodeNameSubstrings: []string{},
			wantQueries: []string{
				`(LOG_ID("serialconsole.googleapis.com/serial_port_1_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_2_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_3_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_debug_output"))
labels."compute.googleapis.com/resource_name"=("node-1")`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:               "run mode with multiple nodes",
			taskMode:           inspectioncore.TaskModeRun,
			nodeNames:          []string{"node-1", "node-2", "node-3"},
			nodeNameSubstrings: []string{},
			wantQueries: []string{
				`(LOG_ID("serialconsole.googleapis.com/serial_port_1_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_2_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_3_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_debug_output"))
labels."compute.googleapis.com/resource_name"=("node-1" OR "node-2" OR "node-3")`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:               "run mode with multiple nodes and substring",
			taskMode:           inspectioncore.TaskModeRun,
			nodeNames:          []string{"node-1", "node-2", "node-3"},
			nodeNameSubstrings: []string{"node-1"},
			wantQueries: []string{
				`(LOG_ID("serialconsole.googleapis.com/serial_port_1_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_2_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_3_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_debug_output"))
labels."compute.googleapis.com/resource_name"=("node-1" OR "node-2" OR "node-3")
labels."compute.googleapis.com/resource_name":("node-1")`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:               "run mode with multiple nodes and multiple substrings",
			taskMode:           inspectioncore.TaskModeRun,
			nodeNames:          []string{"node-1", "node-2"},
			nodeNameSubstrings: []string{"sub-1", "sub-2"},
			wantQueries: []string{
				`(LOG_ID("serialconsole.googleapis.com/serial_port_1_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_2_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_3_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_debug_output"))
labels."compute.googleapis.com/resource_name"=("node-1" OR "node-2")
labels."compute.googleapis.com/resource_name":("sub-1" OR "sub-2")`,
			},
			wantSupportMetricsFlag: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sqs := GenerateSerialPortStructuredQuery(tc.taskMode, tc.nodeNames, tc.nodeNameSubstrings)
			gotQueries := make([]string, len(sqs))
			for i, sq := range sqs {
				gotQueries[i] = sq.GenerateCloudLoggingQuery()
				if sq.AllFiltersSupportMetrics() != tc.wantSupportMetricsFlag {
					t.Errorf("AllFiltersSupportMetrics()[%d] = %v, want %v", i, sq.AllFiltersSupportMetrics(), tc.wantSupportMetricsFlag)
				}
				err := gcp_test.IsValidLogQuery(t, gotQueries[i])
				if err != nil {
					t.Errorf("the generated query is invalid: %v", err)
				}
			}

			if diff := cmp.Diff(tc.wantQueries, gotQueries); diff != "" {
				t.Errorf("GenerateCloudLoggingQuery() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMaximumNodeCountNotHittingQueryLengthLimit(t *testing.T) {
	idg46 := idgenerator.NewFixedLengthIDGenerator(46)
	idg8 := idgenerator.NewFixedLengthIDGenerator(8)
	idg4 := idgenerator.NewFixedLengthIDGenerator(4)
	nodeNames := []string{}
	for i := 0; i < MaxNodesPerQuery*2+1; i++ { // This query must be split into 3 sub groups.
		nodeNames = append(nodeNames, fmt.Sprintf(`gke-%s-%s-%s`, idg46.Generate(), idg8.Generate(), idg4.Generate()))
	}
	sqs := GenerateSerialPortStructuredQuery(inspectioncore.TaskModeRun, nodeNames, []string{})
	if len(sqs) != 3 {
		t.Errorf("len(GenerateSerialPortStructuredQuery())=%d, want %d", len(sqs), 3)
	}
	for _, sq := range sqs {
		err := gcp_test.IsValidLogQuery(t, sq.GenerateCloudLoggingQuery())
		if err != nil {
			t.Errorf("the generated query is invalid. error:%v", err)
		}
	}
}

func TestLogQueryTask_DryRun(t *testing.T) {
	t.Parallel()
	startTime := time.Date(2025, time.January, 1, 1, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, time.January, 1, 1, 1, 0, 0, time.UTC)

	cluster := k8scommon.GoogleCloudClusterIdentity{
		ClusterName: "test-cluster",
		ProjectID:   "test-project",
		Location:    "us-central1",
	}

	resourceNamesInput := gcpcommon.NewResourceNamesInput()
	clientFactory, err := googlecloud.NewClientFactory()
	if err != nil {
		t.Fatalf("failed to create ClientFactory: %v", err)
	}

	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	gotLogs, _, err := inspectiontest.RunInspectionTask(ctx, LogQueryTask, inspectioncore.TaskModeDryRun, map[string]any{},
		tasktest.NewTaskDependencyValuePair(gcpcommon.InputStartTimeTaskID.Ref(), startTime),
		tasktest.NewTaskDependencyValuePair(gcpcommon.InputEndTimeTaskID.Ref(), endTime),
		tasktest.NewTaskDependencyValuePair(gcpcommon.APIClientFactoryTaskID.Ref(), clientFactory),
		tasktest.NewTaskDependencyValuePair(gcpcommon.InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
		tasktest.NewTaskDependencyValuePair(serialport.ClusterIdentityTaskID.Ref(), cluster),
		tasktest.NewTaskDependencyValuePair(k8scommon.InputNodeNameFilterTaskID.Ref(), []string{}),
		tasktest.NewTaskDependencyValuePair(k8saudit.NodeNameInventoryTaskID.Ref(), []string{}),
	)
	if err != nil {
		t.Fatalf("dry run returned unexpected error: %v", err)
	}
	if len(gotLogs) != 0 {
		t.Errorf("dry run should return 0 logs, got %d", len(gotLogs))
	}

	metadata := khictx.MustGetValue(ctx, inspectioncore.InspectionRunMetadata)
	queryMetadata, found := typedmap.Get(metadata, inspectionmetadata.QueryMetadataKey)
	if !found {
		t.Fatalf("QueryMetadata not found in metadata")
	}

	serialized := queryMetadata.ToSerializable().([]*inspectionmetadata.QueryItem)
	if len(serialized) != 1 {
		t.Fatalf("expected 1 QueryItem, got %d", len(serialized))
	}

	wantQuery := `(LOG_ID("serialconsole.googleapis.com/serial_port_1_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_2_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_3_output") OR LOG_ID("serialconsole.googleapis.com/serial_port_debug_output"))
-- instance name filters to be determined after node name discovery
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`

	if diff := cmp.Diff(wantQuery, serialized[0].Query); diff != "" {
		t.Errorf("query mismatch (-want +got):\n%s", diff)
	}
	if serialized[0].Name != "Serial port log" {
		t.Errorf("query name mismatch: got %q, want %q", serialized[0].Name, "Serial port log")
	}
	if serialized[0].Preset != logestimator.EstimatedCountPresetFew {
		t.Errorf("query preset mismatch: got %v, want %v", serialized[0].Preset, logestimator.EstimatedCountPresetFew)
	}
}
