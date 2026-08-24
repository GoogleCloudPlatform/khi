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

package googlecloudlogk8saudit_impl

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
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
	"github.com/google/go-cmp/cmp"
)

func TestGenerateK8sAuditStructuredQuery(t *testing.T) {
	cluster := googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ClusterName: "foo-cluster",
		ProjectID:   "foo-project",
		Location:    "foo-location",
	}

	testCases := []struct {
		name                   string
		cluster                googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		kindFilter             *gcpqueryutil.SetFilterParseResult
		namespaceFilter        *gcpqueryutil.SetFilterParseResult
		wantQuery              string
		wantMetricFilters      []string
		wantSupportMetricsFlag bool
	}{
		{
			name:    "kind additive and namespaced scope",
			cluster: cluster,
			kindFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"pods", "deployments", "jobs"},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"#namespaced"},
			},
			wantQuery: `resource.type="k8s_cluster"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
protoPayload.methodName=~"\.(pods|deployments|jobs)\."
protoPayload.resourceName:"namespaces/"`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_cluster" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:    "kind subtractive and both scopes",
			cluster: cluster,
			kindFilter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{"pods", "deployments"},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"#cluster-scoped", "#namespaced"},
			},
			wantQuery: `resource.type="k8s_cluster"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
-protoPayload.methodName=~"\.(pods|deployments)\."`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_cluster" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:    "empty kind filter and cluster-scoped with specific namespace",
			cluster: cluster,
			kindFilter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"#cluster-scoped", "kube-system", "default"},
			},
			wantQuery: `resource.type="k8s_cluster"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
(protoPayload.resourceName:("/namespaces/kube-system" OR "/namespaces/default") OR NOT (protoPayload.resourceName:"/namespaces/"))`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_cluster" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:    "cluster-scoped only",
			cluster: cluster,
			kindFilter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"#cluster-scoped"},
			},
			wantQuery: `resource.type="k8s_cluster"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
-protoPayload.resourceName:"/namespaces/"`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_cluster" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:    "specific namespaces only",
			cluster: cluster,
			kindFilter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"kube-system", "default"},
			},
			wantQuery: `resource.type="k8s_cluster"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
protoPayload.resourceName:("/namespaces/kube-system" OR "/namespaces/default")`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_cluster" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:    "subtractive namespace mode (unsupported)",
			cluster: cluster,
			kindFilter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
			},
			wantQuery: `resource.type="k8s_cluster"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
-- Unsupported operation`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_cluster" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:    "validation error on kind and namespace",
			cluster: cluster,
			kindFilter: &gcpqueryutil.SetFilterParseResult{
				ValidationError: "kind error",
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				ValidationError: "namespace error",
			},
			wantQuery: `resource.type="k8s_cluster"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
-- Failed to generate kind filter due to the validation error "kind error"
-- Failed to generate namespace filter due to the validation error "namespace error"`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_cluster" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:    "empty additives for kind and namespace",
			cluster: cluster,
			kindFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{},
			},
			wantQuery: `resource.type="k8s_cluster"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
-- Invalid: none of the resources will be selected. Ignoring kind filter.
-- Invalid: none of the resources will be selected. Ignoring namespace filter.`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_cluster" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster"`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:            "nil kind and namespace filters",
			cluster:         cluster,
			kindFilter:      nil,
			namespaceFilter: nil,
			wantQuery: `resource.type="k8s_cluster"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_cluster" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster"`,
			},
			wantSupportMetricsFlag: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sq := GenerateK8sAuditStructuredQuery(tc.cluster, tc.kindFilter, tc.namespaceFilter)
			gotQuery := sq.GenerateCloudLoggingQuery()
			if diff := cmp.Diff(tc.wantQuery, gotQuery); diff != "" {
				t.Errorf("GenerateCloudLoggingQuery() mismatch (-want +got):\n%s", diff)
			}

			legacyQuery := GenerateK8sAuditQuery(tc.cluster, tc.kindFilter, tc.namespaceFilter)
			if diff := cmp.Diff(gotQuery, legacyQuery); diff != "" {
				t.Errorf("GenerateK8sAuditQuery() mismatch (-want +got):\n%s", diff)
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

func TestGenerateK8sAuditQueryIsValid(t *testing.T) {
	testCases := []struct {
		Name            string
		Cluster         googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		KindFilter      *gcpqueryutil.SetFilterParseResult
		NamespaceFilter *gcpqueryutil.SetFilterParseResult
	}{
		{
			Name: "ClusterScoped",
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			KindFilter:      &gcpqueryutil.SetFilterParseResult{Additives: []string{"pods"}},
			NamespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"#cluster-scoped"}},
		},
		{
			Name: "Namespaced",
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			KindFilter:      &gcpqueryutil.SetFilterParseResult{Additives: []string{"pods"}},
			NamespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"#namespaced"}},
		},
		{
			Name: "Namespaced with specific namespace",
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			KindFilter:      &gcpqueryutil.SetFilterParseResult{Additives: []string{"pods"}},
			NamespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"default"}},
		},
		{
			Name: "Namespaced with multiple namespaces",
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			KindFilter:      &gcpqueryutil.SetFilterParseResult{Additives: []string{"pods"}},
			NamespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"default", "kube-system"}},
		},
		{
			Name: "ClusterScoped with specific namespace",
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			KindFilter:      &gcpqueryutil.SetFilterParseResult{Additives: []string{"pods"}},
			NamespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"#cluster-scoped", "default"}},
		},
		{
			Name: "ClusterScoped with multiple namespaces",
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			KindFilter:      &gcpqueryutil.SetFilterParseResult{Additives: []string{"pods"}},
			NamespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"#cluster-scoped", "default", "kube-system"}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			query := GenerateK8sAuditQuery(tc.Cluster, tc.KindFilter, tc.NamespaceFilter)
			err := gcp_test.IsValidLogQuery(t, query)
			if err != nil {
				t.Errorf("IsValidLogQuery error: %s", err.Error())
			}
		})
	}
}

func TestGCPK8sAuditLogListLogEntriesTask_DryRun(t *testing.T) {
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
	gotLogs, _, err := inspectiontest.RunInspectionTask(ctx, GCPK8sAuditLogListLogEntriesTask, inspectioncore_contract.TaskModeDryRun, map[string]any{},
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputStartTimeTaskID.Ref(), startTime),
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputEndTimeTaskID.Ref(), endTime),
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.APIClientFactoryTaskID.Ref(), clientFactory),
		tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
		tasktest.NewTaskDependencyValuePair(googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(), cluster),
		tasktest.NewTaskDependencyValuePair(googlecloudk8scommon_contract.InputKindFilterTaskID.Ref(), &gcpqueryutil.SetFilterParseResult{Additives: []string{"pods"}}),
		tasktest.NewTaskDependencyValuePair(googlecloudk8scommon_contract.InputNamespaceFilterTaskID.Ref(), &gcpqueryutil.SetFilterParseResult{Additives: []string{"#namespaced"}}),
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

	wantQuery := `resource.type="k8s_cluster"
resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
protoPayload.methodName=~"\.(pods)\."
protoPayload.resourceName:"namespaces/"
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`

	if diff := cmp.Diff(wantQuery, serialized[0].Query); diff != "" {
		t.Errorf("Query mismatch (-want +got):\n%s", diff)
	}
	if serialized[0].Name != "K8s audit logs" {
		t.Errorf("Query Name mismatch: got %q, want %q", serialized[0].Name, "K8s audit logs")
	}
}
