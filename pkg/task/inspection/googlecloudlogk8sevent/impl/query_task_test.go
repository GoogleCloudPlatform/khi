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

package googlecloudlogk8sevent_impl

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
	googlecloudlogk8sevent_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8sevent/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
	"github.com/google/go-cmp/cmp"
)

func TestGenerateK8sEventStructuredQuery(t *testing.T) {
	cluster := googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ClusterName: "test-cluster",
		Location:    "us-central1-a",
		ProjectID:   "test-project",
	}

	testCases := []struct {
		name                   string
		cluster                googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		namespaceFilter        *gcpqueryutil.SetFilterParseResult
		wantQuery              string
		wantMetricFilters      []string
		wantSupportMetricsFlag bool
	}{
		{
			name:    "both cluster scoped and namespaced scopes (pure metric)",
			cluster: cluster,
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"#cluster-scoped", "#namespaced"},
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
LOG_ID("events")`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.labels.project_id = "test-project" AND resource.labels.location = "us-central1-a" AND resource.labels.cluster_name = "test-cluster" AND metric.labels.log = "events"`,
			},
			wantSupportMetricsFlag: true,
		},
		{
			name:    "namespaced scope only",
			cluster: cluster,
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"#namespaced"},
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
LOG_ID("events")
jsonPayload.involvedObject.namespace:"" -- ignore events in k8s object with namespace`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.labels.project_id = "test-project" AND resource.labels.location = "us-central1-a" AND resource.labels.cluster_name = "test-cluster" AND metric.labels.log = "events"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:    "cluster-scoped only",
			cluster: cluster,
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"#cluster-scoped"},
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
LOG_ID("events")
-jsonPayload.involvedObject.namespace:"" -- ignore events in k8s object with namespace`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.labels.project_id = "test-project" AND resource.labels.location = "us-central1-a" AND resource.labels.cluster_name = "test-cluster" AND metric.labels.log = "events"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:    "cluster-scoped with specific namespaces",
			cluster: cluster,
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"#cluster-scoped", "default", "kube-system"},
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
LOG_ID("events")
(jsonPayload.involvedObject.namespace=(default OR kube-system) OR NOT (jsonPayload.involvedObject.namespace:""))`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.labels.project_id = "test-project" AND resource.labels.location = "us-central1-a" AND resource.labels.cluster_name = "test-cluster" AND metric.labels.log = "events"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:    "single specific namespace",
			cluster: cluster,
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"default"},
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
LOG_ID("events")
jsonPayload.involvedObject.namespace=(default)`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.labels.project_id = "test-project" AND resource.labels.location = "us-central1-a" AND resource.labels.cluster_name = "test-cluster" AND metric.labels.log = "events"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:    "multiple specific namespaces",
			cluster: cluster,
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"default", "kube-system"},
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
LOG_ID("events")
jsonPayload.involvedObject.namespace=(default OR kube-system)`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.labels.project_id = "test-project" AND resource.labels.location = "us-central1-a" AND resource.labels.cluster_name = "test-cluster" AND metric.labels.log = "events"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:    "subtractive mode",
			cluster: cluster,
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
LOG_ID("events")
-- Unsupported operation`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.labels.project_id = "test-project" AND resource.labels.location = "us-central1-a" AND resource.labels.cluster_name = "test-cluster" AND metric.labels.log = "events"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:    "validation error",
			cluster: cluster,
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				ValidationError: "invalid syntax",
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
LOG_ID("events")
-- Failed to generate namespace filter due to the validation error "invalid syntax"`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.labels.project_id = "test-project" AND resource.labels.location = "us-central1-a" AND resource.labels.cluster_name = "test-cluster" AND metric.labels.log = "events"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:    "empty additives",
			cluster: cluster,
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{},
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
LOG_ID("events")
-- Invalid: none of the resources will be selected. Ignoring namespace filter.`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.labels.project_id = "test-project" AND resource.labels.location = "us-central1-a" AND resource.labels.cluster_name = "test-cluster" AND metric.labels.log = "events"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:            "nil namespace filter",
			cluster:         cluster,
			namespaceFilter: nil,
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
LOG_ID("events")`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.labels.project_id = "test-project" AND resource.labels.location = "us-central1-a" AND resource.labels.cluster_name = "test-cluster" AND metric.labels.log = "events"`,
			},
			wantSupportMetricsFlag: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sq := GenerateK8sEventStructuredQuery(tc.cluster, tc.namespaceFilter)
			gotQuery := sq.GenerateCloudLoggingQuery()
			if diff := cmp.Diff(tc.wantQuery, gotQuery); diff != "" {
				t.Errorf("GenerateCloudLoggingQuery() mismatch (-want +got):\n%s", diff)
			}

			legacyQuery := GenerateK8sEventQuery(tc.cluster, tc.namespaceFilter)
			if diff := cmp.Diff(gotQuery, legacyQuery); diff != "" {
				t.Errorf("GenerateK8sEventQuery() mismatch (-want +got):\n%s", diff)
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

func TestGenerateK8sEventQueryIsValid(t *testing.T) {
	testCluster := googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ClusterName: "test-cluster",
		Location:    "us-central1-a",
		ProjectID:   "test-project",
	}
	testCases := []struct {
		name            string
		cluster         googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		namespaceFilter *gcpqueryutil.SetFilterParseResult
	}{
		{
			name:            "ClusterScoped",
			cluster:         testCluster,
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"#cluster-scoped"}},
		},
		{
			name:            "Namespaced",
			cluster:         testCluster,
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"#namespaced"}},
		},
		{
			name:            "Namespaced with specific namespace",
			cluster:         testCluster,
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"default"}},
		},
		{
			name:            "Namespaced with multiple namespaces",
			cluster:         testCluster,
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"default", "kube-system"}},
		},
		{
			name:            "ClusterScoped with specific namespace",
			cluster:         testCluster,
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"#cluster-scoped", "default"}},
		},
		{
			name:            "ClusterScoped with multiple namespaces",
			cluster:         testCluster,
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"#cluster-scoped", "default", "kube-system"}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query := GenerateK8sEventQuery(tc.cluster, tc.namespaceFilter)
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
		tasktest.NewTaskDependencyValuePair(googlecloudlogk8sevent_contract.ClusterIdentityTaskID.Ref(), cluster),
		tasktest.NewTaskDependencyValuePair(googlecloudk8scommon_contract.InputNamespaceFilterTaskID.Ref(), &gcpqueryutil.SetFilterParseResult{Additives: []string{"#cluster-scoped", "#namespaced"}}),
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

	wantQuery := `resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
LOG_ID("events")
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`

	if diff := cmp.Diff(wantQuery, serialized[0].Query); diff != "" {
		t.Errorf("query mismatch (-want +got):\n%s", diff)
	}
	if serialized[0].Name != "Kubernetes Event Logs" {
		t.Errorf("query name mismatch: got %q, want %q", serialized[0].Name, "Kubernetes Event Logs")
	}
}
