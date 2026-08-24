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

package googlecloudlogk8scontrolplane_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8scontrolplane_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
	"github.com/google/go-cmp/cmp"
)

func TestGenerateK8sControlPlaneStructuredQuery(t *testing.T) {
	cluster := googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ClusterName: "foo-cluster",
		ProjectID:   "foo-project",
		Location:    "foo-location",
	}

	testCases := []struct {
		name                   string
		cluster                googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		componentFilter        *gcpqueryutil.SetFilterParseResult
		wantQuery              string
		wantMetricFilters      []string
		wantSupportMetricsFlag bool
	}{
		{
			name:            "no component name filter (subtract mode empty)",
			cluster:         cluster,
			componentFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: true},
			wantQuery: `resource.type="k8s_control_plane_component"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-sourceLocation.file="httplog.go" -- Ignoring the noisy log from scheduler. TODO: Support toggling this feature.`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_control_plane_component" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:            "subtract mode with multiple subtractives",
			cluster:         cluster,
			componentFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: true, Subtractives: []string{"apiserver", "autoscaler"}},
			wantQuery: `resource.type="k8s_control_plane_component"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-resource.labels.component_name:("apiserver" OR "autoscaler")
-sourceLocation.file="httplog.go" -- Ignoring the noisy log from scheduler. TODO: Support toggling this feature.`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_control_plane_component" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster" AND (resource.labels.component_name != has_substring("apiserver") AND resource.labels.component_name != has_substring("autoscaler"))`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:            "additive mode with single component",
			cluster:         cluster,
			componentFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: false, Additives: []string{"apiserver"}},
			wantQuery: `resource.type="k8s_control_plane_component"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
resource.labels.component_name:"apiserver"
-sourceLocation.file="httplog.go" -- Ignoring the noisy log from scheduler. TODO: Support toggling this feature.`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_control_plane_component" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster" AND resource.labels.component_name = has_substring("apiserver")`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:            "additive mode with multiple components",
			cluster:         cluster,
			componentFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: false, Additives: []string{"apiserver", "autoscaler"}},
			wantQuery: `resource.type="k8s_control_plane_component"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
resource.labels.component_name:("apiserver" OR "autoscaler")
-sourceLocation.file="httplog.go" -- Ignoring the noisy log from scheduler. TODO: Support toggling this feature.`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_control_plane_component" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster" AND (resource.labels.component_name = has_substring("apiserver") OR resource.labels.component_name = has_substring("autoscaler"))`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:            "additive mode with empty additives",
			cluster:         cluster,
			componentFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: false, Additives: []string{}},
			wantQuery: `resource.type="k8s_control_plane_component"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-- Invalid: none of the controlplane components will be selected. Ignoring component name filter.
-sourceLocation.file="httplog.go" -- Ignoring the noisy log from scheduler. TODO: Support toggling this feature.`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_control_plane_component" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:            "subtract mode with single subtractive",
			cluster:         cluster,
			componentFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: true, Subtractives: []string{"apiserver"}},
			wantQuery: `resource.type="k8s_control_plane_component"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-resource.labels.component_name:"apiserver"
-sourceLocation.file="httplog.go" -- Ignoring the noisy log from scheduler. TODO: Support toggling this feature.`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_control_plane_component" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster" AND resource.labels.component_name != has_substring("apiserver")`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:            "validation error",
			cluster:         cluster,
			componentFilter: &gcpqueryutil.SetFilterParseResult{ValidationError: "test error"},
			wantQuery: `resource.type="k8s_control_plane_component"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-- Failed to generate component name filter due to the validation error "test error"
-sourceLocation.file="httplog.go" -- Ignoring the noisy log from scheduler. TODO: Support toggling this feature.`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_control_plane_component" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:            "nil component filter",
			cluster:         cluster,
			componentFilter: nil,
			wantQuery: `resource.type="k8s_control_plane_component"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-sourceLocation.file="httplog.go" -- Ignoring the noisy log from scheduler. TODO: Support toggling this feature.`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_control_plane_component" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster"`,
			},
			wantSupportMetricsFlag: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sq := GenerateK8sControlPlaneStructuredQuery(tc.cluster, tc.componentFilter)
			gotQuery := sq.GenerateCloudLoggingQuery()
			if diff := cmp.Diff(tc.wantQuery, gotQuery); diff != "" {
				t.Errorf("GenerateCloudLoggingQuery() mismatch (-want +got):\n%s", diff)
			}

			legacyQuery := GenerateK8sControlPlaneQuery(tc.cluster, tc.componentFilter)
			if diff := cmp.Diff(gotQuery, legacyQuery); diff != "" {
				t.Errorf("GenerateK8sControlPlaneQuery() mismatch (-want +got):\n%s", diff)
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

func TestGenerateK8sControlPlaneQueryIsValid(t *testing.T) {
	testCases := []struct {
		name                                 string
		cluster                              googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		inputControlplaneComponentNameFilter *gcpqueryutil.SetFilterParseResult
	}{
		{
			name: "All components",
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			inputControlplaneComponentNameFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: true},
		},
		{
			name: "Specific components (inclusive)",
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			inputControlplaneComponentNameFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: false, Additives: []string{"apiserver", "autoscaler"}},
		},
		{
			name: "Specific components (exclusive)",
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			inputControlplaneComponentNameFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: true, Subtractives: []string{"apiserver", "autoscaler"}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query := GenerateK8sControlPlaneQuery(tc.cluster, tc.inputControlplaneComponentNameFilter)
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
		tasktest.NewTaskDependencyValuePair(googlecloudlogk8scontrolplane_contract.ClusterIdentityTaskID.Ref(), cluster),
		tasktest.NewTaskDependencyValuePair(googlecloudlogk8scontrolplane_contract.InputControlPlaneComponentNameFilterTaskID.Ref(), &gcpqueryutil.SetFilterParseResult{SubtractMode: true}),
	)
	if err != nil {
		t.Fatalf("DryRun returned unexpected error: %v", err)
	}
	if len(gotLogs) != 0 {
		t.Errorf("DryRun should return 0 logs, got %d", len(gotLogs))
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

	wantQuery := `resource.type="k8s_control_plane_component"
resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
-sourceLocation.file="httplog.go" -- Ignoring the noisy log from scheduler. TODO: Support toggling this feature.
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`

	if diff := cmp.Diff(wantQuery, serialized[0].Query); diff != "" {
		t.Errorf("Query mismatch (-want +got):\n%s", diff)
	}
	if serialized[0].Name != "K8s control plane logs" {
		t.Errorf("Query Name mismatch: got %q, want %q", serialized[0].Name, "K8s control plane logs")
	}
}
