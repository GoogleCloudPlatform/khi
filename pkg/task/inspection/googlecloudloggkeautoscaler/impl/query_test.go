// Copyright 2024 Google LLC
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

package googlecloudloggkeautoscaler_impl

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
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
	"github.com/google/go-cmp/cmp"
)

func TestGenerateAutoscalerStructuredQuery(t *testing.T) {
	cluster := googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ProjectID:   "my-project",
		Location:    "my-location",
		ClusterName: "my-cluster",
	}

	testCases := []struct {
		name                   string
		cluster                googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		excludeStatus          bool
		wantQuery              string
		wantMetricFilters      []string
		wantSupportMetricsFlag bool
	}{
		{
			name:          "include status (pure metric)",
			cluster:       cluster,
			excludeStatus: false,
			wantQuery: `resource.type="k8s_cluster"
resource.labels.project_id="my-project"
resource.labels.location="my-location"
resource.labels.cluster_name="my-cluster"
LOG_ID("container.googleapis.com/cluster-autoscaler-visibility")`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_cluster" AND resource.labels.project_id = "my-project" AND resource.labels.location = "my-location" AND resource.labels.cluster_name = "my-cluster" AND metric.labels.log = "container.googleapis.com/cluster-autoscaler-visibility"`,
			},
			wantSupportMetricsFlag: true,
		},
		{
			name:          "exclude status (probe fallback)",
			cluster:       cluster,
			excludeStatus: true,
			wantQuery: `resource.type="k8s_cluster"
resource.labels.project_id="my-project"
resource.labels.location="my-location"
resource.labels.cluster_name="my-cluster"
LOG_ID("container.googleapis.com/cluster-autoscaler-visibility")
-jsonPayload.status: ""`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_cluster" AND resource.labels.project_id = "my-project" AND resource.labels.location = "my-location" AND resource.labels.cluster_name = "my-cluster" AND metric.labels.log = "container.googleapis.com/cluster-autoscaler-visibility"`,
			},
			wantSupportMetricsFlag: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sq := GenerateAutoscalerStructuredQuery(tc.cluster, tc.excludeStatus)
			gotQuery := sq.GenerateCloudLoggingQuery()
			if diff := cmp.Diff(tc.wantQuery, gotQuery); diff != "" {
				t.Errorf("GenerateCloudLoggingQuery() mismatch (-want +got):\n%s", diff)
			}

			legacyQuery := generateAutoscalerQuery(tc.cluster, tc.excludeStatus)
			if diff := cmp.Diff(gotQuery, legacyQuery); diff != "" {
				t.Errorf("generateAutoscalerQuery() mismatch (-want +got):\n%s", diff)
			}

			gotMetrics := sq.GenerateMonitoringMetricFilters()
			if diff := cmp.Diff(tc.wantMetricFilters, gotMetrics); diff != "" {
				t.Errorf("GenerateMonitoringMetricFilters() mismatch (-want +got):\n%s", diff)
			}

			if sq.AllFiltersSupportMetrics() != tc.wantSupportMetricsFlag {
				t.Errorf("AllFiltersSupportMetrics() = %v, want %v", sq.AllFiltersSupportMetrics(), tc.wantSupportMetricsFlag)
			}
		})
	}
}

func TestGeneratedAutoscalerQueryIsValid(t *testing.T) {
	testCases := []struct {
		name          string
		cluster       googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		excludeStatus bool
	}{
		{
			name: "Valid Query",
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ProjectID:   "gcp-project-id",
				Location:    "gcp-location",
				ClusterName: "gcp-cluster-name",
			},
			excludeStatus: false,
		},
		{
			name: "Valid Query with Exclude Status",
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ProjectID:   "gcp-project-id",
				Location:    "gcp-location",
				ClusterName: "gcp-cluster-name",
			},
			excludeStatus: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query := generateAutoscalerQuery(tc.cluster, tc.excludeStatus)
			err := gcp_test.IsValidLogQuery(t, query)
			if err != nil {
				t.Errorf("IsValidLogQuery error: %s", err.Error())
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
		tasktest.NewTaskDependencyValuePair(googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(), cluster),
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

	wantQuery := `resource.type="k8s_cluster"
resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
LOG_ID("container.googleapis.com/cluster-autoscaler-visibility")
-jsonPayload.status: ""
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`

	if diff := cmp.Diff(wantQuery, serialized[0].Query); diff != "" {
		t.Errorf("query mismatch (-want +got):\n%s", diff)
	}
	if serialized[0].Name != "Cluster autoscaler logs" {
		t.Errorf("query name mismatch: got %q, want %q", serialized[0].Name, "Cluster autoscaler logs")
	}
}
