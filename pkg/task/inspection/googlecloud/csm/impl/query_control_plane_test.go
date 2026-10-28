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

package csm_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/csm"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
)

func TestGenerateCSMControlPlaneStructuredQuery(t *testing.T) {
	type queryResult struct {
		Query              string
		SupportMetricsFlag bool
		Incomplete         bool
	}
	testCases := []struct {
		name           string
		fleetProjectID string
		want           queryResult
	}{
		{
			name:           "valid fleet project id",
			fleetProjectID: "fleet-project",
			want: queryResult{
				Query: `resource.type="istio_control_plane"
resource.labels.project_id="fleet-project"`,
				SupportMetricsFlag: true,
				Incomplete:         false,
			},
		},
		{
			name:           "empty fleet project id",
			fleetProjectID: "",
			want: queryResult{
				Query: `resource.type="istio_control_plane"
resource.labels.project_id=""`,
				SupportMetricsFlag: true,
				Incomplete:         true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sq := generateCSMControlPlaneStructuredQuery(tc.fleetProjectID)
			got := queryResult{
				Query:              sq.GenerateCloudLoggingQuery(),
				SupportMetricsFlag: sq.AllFiltersSupportMetrics(),
				Incomplete:         sq.Incomplete,
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("generateCSMControlPlaneStructuredQuery() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestListCSMControlPlaneLogEntriesTask_DryRun(t *testing.T) {
	t.Parallel()
	startTime := time.Date(2025, time.January, 1, 1, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, time.January, 1, 1, 1, 0, 0, time.UTC)

	clientFactory, err := googlecloud.NewClientFactory()
	if err != nil {
		t.Fatalf("failed to create ClientFactory: %v", err)
	}

	testCases := []struct {
		name           string
		fleetProjectID string
		want           []*inspectionmetadata.QueryItem
	}{
		{
			name:           "valid fleet project id",
			fleetProjectID: "fleet-project",
			want: []*inspectionmetadata.QueryItem{
				{
					Id:   csm.ListCSMControlPlaneLogEntriesTaskID.String(),
					Name: "CSM Control Plane logs",
					Query: `resource.type="istio_control_plane"
resource.labels.project_id="fleet-project"
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`,
					Incomplete: false,
					Pending:    true,
				},
			},
		},
		{
			name:           "missing fleet project id is incomplete",
			fleetProjectID: "",
			want: []*inspectionmetadata.QueryItem{
				{
					Id:   csm.ListCSMControlPlaneLogEntriesTaskID.String(),
					Name: "CSM Control Plane logs",
					Query: `resource.type="istio_control_plane"
resource.labels.project_id=""
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`,
					Incomplete: true,
					Pending:    false,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resourceNamesInput := gcpcommon.NewResourceNamesInput()
			if tc.fleetProjectID != "" {
				resourceNamesInput.UpdateDefaultResourceNamesForQuery(csm.ListCSMControlPlaneLogEntriesTaskID.ReferenceIDString(), []string{"projects/" + tc.fleetProjectID})
			}

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			gotLogs, _, err := inspectiontest.Run(t, ctx, listCSMControlPlaneLogEntriesTask, inspectioncore.TaskModeDryRun, map[string]any{},
				tasktest.Given(gcpcommon.InputStartTimeTaskID.Ref(), startTime),
				tasktest.Given(gcpcommon.InputEndTimeTaskID.Ref(), endTime),
				tasktest.Given(gcpcommon.InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
				tasktest.Given[gcpcommon.LogFetcher](gcpcommon.LoggingFetcherTaskID.Ref(), nil),
				tasktest.Given(gcpcommon.APIClientFactoryTaskID.Ref(), clientFactory),
				tasktest.Given(gcpcommon.APIClientCallOptionsInjectorTaskID.Ref(), googlecloud.NewCallOptionInjector()),
				tasktest.Given(csm.InputFleetProjectIDTaskID.Ref(), tc.fleetProjectID),
			)
			if err != nil {
				t.Fatalf("dry run returned unexpected error: %v", err)
			}
			if len(gotLogs) != 0 {
				t.Errorf("dry run should return 0 logs, got %d", len(gotLogs))
			}

			metadata := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
			queryMetadata, found := typedmap.Get(metadata, inspectionmetadata.QueryMetadataKey)
			if !found {
				t.Fatalf("QueryMetadata not found in metadata")
			}

			serialized := queryMetadata.ToSerializable().([]*inspectionmetadata.QueryItem)
			if diff := cmp.Diff(tc.want, serialized); diff != "" {
				t.Errorf("QueryMetadata mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
