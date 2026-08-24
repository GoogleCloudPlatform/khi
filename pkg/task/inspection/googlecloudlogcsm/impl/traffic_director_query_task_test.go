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

package googlecloudlogcsm_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogcsm_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcsm/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

func TestGenerateCSMTrafficDirectorStructuredQuery(t *testing.T) {
	tests := []struct {
		name                   string
		fleetProjectID         string
		clusterIdentifiers     []string
		isDryRun               bool
		wantQuery              string
		wantSupportMetricsFlag bool
	}{
		{
			name:               "single identifier",
			fleetProjectID:     "fleet-project",
			clusterIdentifiers: []string{"cluster1"},
			isDryRun:           false,
			wantQuery: `resource.labels.project_id="fleet-project"
(LOG_ID("cloudaudit.googleapis.com/activity") OR LOG_ID("cloudaudit.googleapis.com/data_access"))
protoPayload.resourceName:"gsmrsvd-cluster1"`,
			wantSupportMetricsFlag: false,
		},
		{
			name:               "multiple identifiers",
			fleetProjectID:     "fleet-project",
			clusterIdentifiers: []string{"cluster1", "cluster2"},
			isDryRun:           false,
			wantQuery: `resource.labels.project_id="fleet-project"
(LOG_ID("cloudaudit.googleapis.com/activity") OR LOG_ID("cloudaudit.googleapis.com/data_access"))
protoPayload.resourceName:("gsmrsvd-cluster1" OR "gsmrsvd-cluster2")`,
			wantSupportMetricsFlag: false,
		},
		{
			name:                   "no identifiers",
			fleetProjectID:         "fleet-project",
			clusterIdentifiers:     []string{},
			isDryRun:               false,
			wantQuery:              "",
			wantSupportMetricsFlag: false,
		},
		{
			name:               "dry run",
			fleetProjectID:     "fleet-project",
			clusterIdentifiers: []string{"cluster1"}, // Ignored in dry run
			isDryRun:           true,
			wantQuery: `resource.labels.project_id="fleet-project"
(LOG_ID("cloudaudit.googleapis.com/activity") OR LOG_ID("cloudaudit.googleapis.com/data_access"))
protoPayload.resourceName:"gsmrsvd-dummy" -- The actual resource name selector will be generated from other logs in the middle of the pipeline.`,
			wantSupportMetricsFlag: false,
		},
		{
			name:               "dry run with empty identifiers",
			fleetProjectID:     "fleet-project",
			clusterIdentifiers: []string{},
			isDryRun:           true,
			wantQuery: `resource.labels.project_id="fleet-project"
(LOG_ID("cloudaudit.googleapis.com/activity") OR LOG_ID("cloudaudit.googleapis.com/data_access"))
protoPayload.resourceName:"gsmrsvd-dummy" -- The actual resource name selector will be generated from other logs in the middle of the pipeline.`,
			wantSupportMetricsFlag: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sq := GenerateCSMTrafficDirectorStructuredQuery(tc.fleetProjectID, tc.clusterIdentifiers, tc.isDryRun)
			if sq == nil {
				if tc.wantQuery != "" {
					t.Errorf("GenerateCSMTrafficDirectorStructuredQuery() = nil, want %q", tc.wantQuery)
				}
				legacyQuery := GenerateCSMTrafficDirectorQuery(tc.fleetProjectID, tc.clusterIdentifiers, tc.isDryRun)
				if diff := cmp.Diff(tc.wantQuery, legacyQuery); diff != "" {
					t.Errorf("GenerateCSMTrafficDirectorQuery() mismatch (-want +got):\n%s", diff)
				}
				return
			}
			gotQuery := sq.GenerateCloudLoggingQuery()
			if diff := cmp.Diff(tc.wantQuery, gotQuery); diff != "" {
				t.Errorf("GenerateCloudLoggingQuery() mismatch (-want +got):\n%s", diff)
			}

			legacyQuery := GenerateCSMTrafficDirectorQuery(tc.fleetProjectID, tc.clusterIdentifiers, tc.isDryRun)
			if diff := cmp.Diff(gotQuery, legacyQuery); diff != "" {
				t.Errorf("GenerateCSMTrafficDirectorQuery() mismatch (-want +got):\n%s", diff)
			}

			if sq.AllFiltersSupportMetrics() != tc.wantSupportMetricsFlag {
				t.Errorf("AllFiltersSupportMetrics() = %v, want %v", sq.AllFiltersSupportMetrics(), tc.wantSupportMetricsFlag)
			}
		})
	}
}

func TestListCSMTrafficDirectorLogEntriesTask_DryRun(t *testing.T) {
	t.Parallel()
	startTime := time.Date(2025, time.January, 1, 1, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, time.January, 1, 1, 1, 0, 0, time.UTC)

	resourceNamesInput := googlecloudcommon_contract.NewResourceNamesInput()
	clientFactory, err := googlecloud.NewClientFactory()
	if err != nil {
		t.Fatalf("failed to create ClientFactory: %v", err)
	}

	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	gotLogs, _, err := inspectiontest.RunInspectionTask(ctx, ListCSMTrafficDirectorLogEntriesTask, inspectioncore_contract.TaskModeDryRun, map[string]any{},
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputStartTimeTaskID.Ref(), startTime),
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputEndTimeTaskID.Ref(), endTime),
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.APIClientFactoryTaskID.Ref(), clientFactory),
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
		tasktest.NewTaskDependencyValuePair(googlecloudlogcsm_contract.InputFleetProjectIDTaskID.Ref(), "fleet-project"),
		tasktest.NewTaskDependencyValuePair(googlecloudlogcsm_contract.CSMClusterIdentifierTaskID.Ref(), []string{}),
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

	wantQuery := `resource.labels.project_id="fleet-project"
(LOG_ID("cloudaudit.googleapis.com/activity") OR LOG_ID("cloudaudit.googleapis.com/data_access"))
protoPayload.resourceName:"gsmrsvd-dummy" -- The actual resource name selector will be generated from other logs in the middle of the pipeline.
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`

	if diff := cmp.Diff(wantQuery, serialized[0].Query); diff != "" {
		t.Errorf("query mismatch (-want +got):\n%s", diff)
	}
	if serialized[0].Name != "CSM Traffic Director logs" {
		t.Errorf("query name mismatch: got %q, want %q", serialized[0].Name, "CSM Traffic Director logs")
	}
}
