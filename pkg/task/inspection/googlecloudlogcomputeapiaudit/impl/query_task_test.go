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

package googlecloudlogcomputeapiaudit_impl

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogcomputeapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcomputeapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
	"github.com/google/go-cmp/cmp"
)

func TestGenerateComputeAPIStructuredQuery(t *testing.T) {
	generateNodes := func(count int) []string {
		nodes := make([]string, count)
		for i := 0; i < count; i++ {
			nodes[i] = fmt.Sprintf("node-%d", i+1)
		}
		return nodes
	}

	testCases := []struct {
		name                   string
		taskMode               inspectioncore_contract.InspectionTaskModeType
		nodeNames              []string
		wantQueries            []string
		wantSupportMetricsFlag bool
	}{
		{
			name:      "DryRun mode",
			taskMode:  inspectioncore_contract.TaskModeDryRun,
			nodeNames: []string{},
			wantQueries: []string{
				`resource.type="gce_instance"
-protoPayload.methodName:("list" OR "get" OR "watch")
-- instance name filters to be determined after node name discovery`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:                   "Run mode with empty nodes",
			taskMode:               inspectioncore_contract.TaskModeRun,
			nodeNames:              []string{},
			wantQueries:            []string{},
			wantSupportMetricsFlag: false,
		},
		{
			name:      "Run mode with a few nodes",
			taskMode:  inspectioncore_contract.TaskModeRun,
			nodeNames: []string{"node1", "node2"},
			wantQueries: []string{
				`resource.type="gce_instance"
-protoPayload.methodName:("list" OR "get" OR "watch")
protoPayload.resourceName:(instances/node1 OR instances/node2)`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:      "Run mode with >30 nodes chunked into multiple queries",
			taskMode:  inspectioncore_contract.TaskModeRun,
			nodeNames: generateNodes(32),
			wantQueries: []string{
				func() string {
					parts := []string{}
					for i := 1; i <= 30; i++ {
						parts = append(parts, fmt.Sprintf("instances/node-%d", i))
					}
					return fmt.Sprintf("resource.type=\"gce_instance\"\n-protoPayload.methodName:(\"list\" OR \"get\" OR \"watch\")\nprotoPayload.resourceName:(%s)", strings.Join(parts, " OR "))
				}(),
				`resource.type="gce_instance"
-protoPayload.methodName:("list" OR "get" OR "watch")
protoPayload.resourceName:(instances/node-31 OR instances/node-32)`,
			},
			wantSupportMetricsFlag: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sqs := GenerateComputeAPIStructuredQuery(tc.taskMode, tc.nodeNames)
			gotQueries := make([]string, len(sqs))
			for i, sq := range sqs {
				gotQueries[i] = sq.GenerateCloudLoggingQuery()
				if sq.AllFiltersSupportMetrics() != tc.wantSupportMetricsFlag {
					t.Errorf("AllFiltersSupportMetrics() = %v, want %v", sq.AllFiltersSupportMetrics(), tc.wantSupportMetricsFlag)
				}
			}

			if diff := cmp.Diff(tc.wantQueries, gotQueries); diff != "" {
				t.Errorf("GenerateCloudLoggingQuery() mismatch (-want +got):\n%s", diff)
			}

			legacyQueries := GenerateComputeAPIQuery(tc.taskMode, tc.nodeNames)
			if diff := cmp.Diff(gotQueries, legacyQueries); diff != "" {
				t.Errorf("GenerateComputeAPIQuery() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateComputeAPIQueryIsValid(t *testing.T) {
	testCases := []struct {
		name      string
		taskMode  inspectioncore_contract.InspectionTaskModeType
		nodeNames []string
	}{
		{
			name:      "Valid Query in DryRun mode",
			taskMode:  inspectioncore_contract.TaskModeDryRun,
			nodeNames: []string{},
		},
		{
			name:      "Valid Query in Run mode",
			taskMode:  inspectioncore_contract.TaskModeRun,
			nodeNames: []string{"gke-test-cluster-node-1"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			queries := GenerateComputeAPIQuery(tc.taskMode, tc.nodeNames)
			for _, query := range queries {
				err := gcp_test.IsValidLogQuery(t, query)
				if err != nil {
					t.Errorf("IsValidLogQuery error: %s", err.Error())
				}
			}
		})
	}
}

func TestListLogEntriesTask_DryRun(t *testing.T) {
	t.Parallel()
	startTime := time.Date(2025, time.January, 1, 1, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, time.January, 1, 1, 1, 0, 0, time.UTC)

	cluster := googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ClusterName: "test-cluster",
		ProjectID:   "test-project",
		Location:    "us-central1-a",
	}

	resourceNamesInput := googlecloudcommon_contract.NewResourceNamesInput()
	clientFactory, err := googlecloud.NewClientFactory()
	if err != nil {
		t.Fatalf("failed to create ClientFactory: %v", err)
	}

	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	gotLogs, _, err := inspectiontest.RunInspectionTask(ctx, ListLogEntriesTask, inspectioncore_contract.TaskModeDryRun, map[string]any{},
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputStartTimeTaskID.Ref(), startTime),
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputEndTimeTaskID.Ref(), endTime),
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.APIClientFactoryTaskID.Ref(), clientFactory),
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
		tasktest.NewTaskDependencyValuePair(googlecloudlogcomputeapiaudit_contract.ClusterIdentityTaskID.Ref(), cluster),
		tasktest.NewTaskDependencyValuePair(commonlogk8saudit_contract.NodeNameInventoryTaskID.Ref(), []string{}),
	)
	if err != nil {
		t.Fatalf("dry run returned unexpected error: %v", err)
	}
	if len(gotLogs) != 0 {
		t.Errorf("dry run should return 0 logs, got %d", len(gotLogs))
	}

	metadata := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)
	queryMetadata, found := typedmap.Get(metadata, inspectionmetadata.QueryMetadataKey)
	if !found {
		t.Fatalf("QueryMetadata not found in metadata")
	}

	serialized := queryMetadata.ToSerializable().([]*inspectionmetadata.QueryItem)
	if len(serialized) != 1 {
		t.Fatalf("expected 1 QueryItem, got %d", len(serialized))
	}

	wantQuery := `resource.type="gce_instance"
-protoPayload.methodName:("list" OR "get" OR "watch")
-- instance name filters to be determined after node name discovery
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`

	if diff := cmp.Diff(wantQuery, serialized[0].Query); diff != "" {
		t.Errorf("query mismatch (-want +got):\n%s", diff)
	}
	if serialized[0].Name != "Compute API Audit log" {
		t.Errorf("query name mismatch: got %q, want %q", serialized[0].Name, "Compute API Audit log")
	}
	if serialized[0].Preset != logestimator.EstimatedCountPresetFew {
		t.Errorf("query preset mismatch: got %v, want %v", serialized[0].Preset, logestimator.EstimatedCountPresetFew)
	}
}
