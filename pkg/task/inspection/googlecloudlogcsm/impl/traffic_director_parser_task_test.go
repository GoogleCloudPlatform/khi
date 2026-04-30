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
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestCSMTrafficDirectorLogToTimelineMapperSetting_ProcessLogByGroup(t *testing.T) {
	now := time.Now().Truncate(time.Second) // Truncate to avoid sub-second diffs if needed, though cmp.Diff handles it.

	createReader := func(t *testing.T, data map[string]any) *structured.NodeReader {
		if data == nil {
			return nil
		}
		node, err := structured.FromGoValue(data, &structured.AlphabeticalGoMapKeyOrderProvider{})
		if err != nil {
			t.Fatalf("failed to create node: %v", err)
		}
		return structured.NewNodeReader(node)
	}

	tests := []struct {
		name           string
		auditFieldSets []*googlecloudcommon_contract.GCPAuditLogFieldSet
		want           [][]testchangeset.ChangeSetAsserter
	}{
		{
			name: "Mesh creation log",
			auditFieldSets: []*googlecloudcommon_contract.GCPAuditLogFieldSet{
				{
					MethodName:     "google.cloud.networkservices.v1.NetworkServices.CreateMesh",
					ResourceName:   "projects/test-project/locations/global/meshes/test-mesh",
					PrincipalEmail: "user@example.com",
					OperationFirst: true,
					OperationLast:  true,
					Response:       createReader(t, map[string]any{"name": "test-mesh", "description": "test"}),
				},
			},
			want: [][]testchangeset.ChangeSetAsserter{
				{
					&testchangeset.HasEvent{ResourcePath: "@GCP#CSM#meshes#test-mesh"},
					&testchangeset.HasLogSummary{WantLogSummary: "CreateMesh"},
				},
			},
		},
		{
			name: "Long running operation sequence",
			auditFieldSets: []*googlecloudcommon_contract.GCPAuditLogFieldSet{
				{
					OperationID:    "op1",
					MethodName:     "google.cloud.networkservices.v1.NetworkServices.CreateMesh",
					ResourceName:   "projects/p/locations/global/meshes/m1",
					PrincipalEmail: "u@e.c",
					OperationFirst: true,
					OperationLast:  false,
					Request:        createReader(t, map[string]any{"description": "init"}),
				},
				{
					OperationID:    "op1",
					MethodName:     "google.cloud.networkservices.v1.NetworkServices.CreateMesh",
					ResourceName:   "projects/p/locations/global/meshes/m1",
					PrincipalEmail: "u@e.c",
					OperationFirst: false,
					OperationLast:  true,
					// No response
				},
			},
			want: [][]testchangeset.ChangeSetAsserter{
				{
					// Creation log shouldn't generate its resource timeline before the operation completes.
					&testchangeset.HasNoRevision{ResourcePath: "@GCP#meshes#m1"},
					&testchangeset.HasRevision{
						ResourcePath: "@GCP#CSM#meshes#m1#CreateMesh-op1",
						WantRevision: history.StagingResourceRevision{
							Verb:       enum.RevisionVerbOperationStart,
							State:      enum.RevisionStateOperationStarted,
							Requestor:  "u@e.c",
							Body:       "description: init\n",
							ChangeTime: now,
						},
					},
				},
				{
					&testchangeset.HasRevision{
						ResourcePath: "@GCP#CSM#meshes#m1",
						WantRevision: history.StagingResourceRevision{
							Verb:       enum.RevisionVerbCreate,
							State:      enum.RevisionStateExisting,
							Requestor:  "u@e.c",
							Body:       "description: init\n",
							ChangeTime: now.Add(time.Second),
						},
					},
					&testchangeset.HasRevision{
						ResourcePath: "@GCP#CSM#meshes#m1#CreateMesh-op1",
						WantRevision: history.StagingResourceRevision{
							Verb:       enum.RevisionVerbOperationFinish,
							State:      enum.RevisionStateOperationFinished,
							Requestor:  "u@e.c",
							Body:       "",
							ChangeTime: now.Add(time.Second),
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mapper := &csmTrafficDirectorLogToTimelineMapperSetting{}
			tracker := googlecloudcommon_contract.NewGCPOperationStateTracker()

			for i, auditFieldSet := range tc.auditFieldSets {
				logTime := now.Add(time.Duration(i) * time.Second)
				l := log.NewLogWithFieldSetsForTest(
					&log.CommonFieldSet{Timestamp: logTime},
					auditFieldSet,
				)
				cs := history.NewChangeSet(l)

				_, err := mapper.ProcessLogByGroup(context.Background(), l, cs, nil, tracker)
				if err != nil {
					t.Fatalf("log %d: unexpected error: %v", i, err)
				}

				if i < len(tc.want) {
					for _, asserter := range tc.want[i] {
						asserter.Assert(t, cs)
					}
				}
			}
		})
	}
}
