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
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	googlecloudlogcsm_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcsm/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func TestCSMTrafficDirectorListLogEntryTaskSetting_LogFilters(t *testing.T) {
	tests := []struct {
		name               string
		fleetProjectID     string
		clusterIdentifiers []string
		taskMode           inspectioncore_contract.InspectionTaskModeType
		want               []string
	}{
		{
			name:               "single identifier",
			fleetProjectID:     "fleet-project",
			clusterIdentifiers: []string{"cluster1"},
			taskMode:           inspectioncore_contract.TaskModeRun,
			want: []string{
				`(log_id("cloudaudit.googleapis.com/activity") OR log_id("cloudaudit.googleapis.com/data_access"))
protoPayload.resourceName:"gsmrsvd-cluster1"
resource.labels.project_id="fleet-project"`,
			},
		},
		{
			name:               "multiple identifiers",
			fleetProjectID:     "fleet-project",
			clusterIdentifiers: []string{"cluster1", "cluster2"},
			taskMode:           inspectioncore_contract.TaskModeRun,
			want: []string{
				`(log_id("cloudaudit.googleapis.com/activity") OR log_id("cloudaudit.googleapis.com/data_access"))
protoPayload.resourceName:("gsmrsvd-cluster1" OR "gsmrsvd-cluster2")
resource.labels.project_id="fleet-project"`,
			},
		},
		{
			name:               "no identifiers",
			fleetProjectID:     "fleet-project",
			clusterIdentifiers: []string{},
			taskMode:           inspectioncore_contract.TaskModeRun,
			want:               nil,
		},
		{
			name:               "dry run",
			fleetProjectID:     "fleet-project",
			clusterIdentifiers: []string{"cluster1"}, // Should be ignored in dry run
			taskMode:           inspectioncore_contract.TaskModeDryRun,
			want: []string{
				`(log_id("cloudaudit.googleapis.com/activity") OR log_id("cloudaudit.googleapis.com/data_access"))
protoPayload.resourceName:"gsmrsvd-dummy" -- The actual resource name selector will be generated from other logs in the middle of the pipeline.
resource.labels.project_id="fleet-project"`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			setting := &CSMTrafficDirectorListLogEntryTaskSetting{}

			// Mocking dependencies by providing them in the context
			ctx = tasktest.WithTaskResult(ctx, googlecloudlogcsm_contract.InputFleetProjectIDTaskID.Ref(), tc.fleetProjectID)
			ctx = tasktest.WithTaskResult(ctx, googlecloudlogcsm_contract.CSMClusterIdentifierTaskID.Ref(), tc.clusterIdentifiers)

			got, err := setting.LogFilters(ctx, tc.taskMode)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("LogFilters() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
