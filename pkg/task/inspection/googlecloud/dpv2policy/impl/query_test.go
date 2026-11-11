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

package dpv2policy_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/dpv2policy"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
	"github.com/google/go-cmp/cmp"
)

func TestGeneratePolicyActionStructuredQuery(t *testing.T) {
	cluster := k8scommon.GoogleCloudClusterIdentity{
		ClusterName: "test-cluster",
		Location:    "us-central1-a",
		ProjectID:   "test-project",
	}

	testCases := []struct {
		name                   string
		cluster                k8scommon.GoogleCloudClusterIdentity
		nodeNameSubstrings     []string
		wantQuery              string
		wantMetricFilters      []string
		wantSupportMetricsFlag bool
	}{
		{
			name:               "empty node name substrings",
			cluster:            cluster,
			nodeNameSubstrings: []string{},
			wantQuery: `resource.type="k8s_node"
resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
LOG_ID("policy-action")`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_node" AND resource.labels.project_id = "test-project" AND resource.labels.location = "us-central1-a" AND resource.labels.cluster_name = "test-cluster" AND metric.labels.log = "policy-action"`,
			},
			wantSupportMetricsFlag: true,
		},
		{
			name:               "single node name substring",
			cluster:            cluster,
			nodeNameSubstrings: []string{"node-1"},
			wantQuery: `resource.type="k8s_node"
resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
LOG_ID("policy-action")
resource.labels.node_name:"node-1"`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_node" AND resource.labels.project_id = "test-project" AND resource.labels.location = "us-central1-a" AND resource.labels.cluster_name = "test-cluster" AND metric.labels.log = "policy-action" AND resource.labels.node_name = has_substring("node-1")`,
			},
			wantSupportMetricsFlag: true,
		},
		{
			name:               "multiple node name substrings",
			cluster:            cluster,
			nodeNameSubstrings: []string{"node-1", "node-2"},
			wantQuery: `resource.type="k8s_node"
resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
LOG_ID("policy-action")
resource.labels.node_name:("node-1" OR "node-2")`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_node" AND resource.labels.project_id = "test-project" AND resource.labels.location = "us-central1-a" AND resource.labels.cluster_name = "test-cluster" AND metric.labels.log = "policy-action" AND (resource.labels.node_name = has_substring("node-1") OR resource.labels.node_name = has_substring("node-2"))`,
			},
			wantSupportMetricsFlag: true,
		},
		{
			name: "cluster name prefix for GKE on AWS",
			cluster: k8scommon.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
				Location:    "us-central1-a",
				ProjectID:   "test-project",
				PrefixPolicy: k8scommon.ClusterPrefixPolicy{
					Prefix:         "awsClusters/",
					RequiredUsages: []k8scommon.ClusterNameUsage{k8scommon.ClusterNameUsageK8sCluster},
				},
			},
			nodeNameSubstrings: []string{},
			wantQuery: `resource.type="k8s_node"
resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="awsClusters/test-cluster"
LOG_ID("policy-action")`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_node" AND resource.labels.project_id = "test-project" AND resource.labels.location = "us-central1-a" AND resource.labels.cluster_name = "awsClusters/test-cluster" AND metric.labels.log = "policy-action"`,
			},
			wantSupportMetricsFlag: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sq := generatePolicyActionStructuredQuery(tc.cluster, tc.nodeNameSubstrings)
			gotQuery := sq.GenerateCloudLoggingQuery()
			if diff := cmp.Diff(tc.wantQuery, gotQuery); diff != "" {
				t.Errorf("GenerateCloudLoggingQuery() mismatch (-want +got):\n%s", diff)
			}

			gotMetrics := sq.GenerateMonitoringMetricFilters()
			if diff := cmp.Diff(tc.wantMetricFilters, gotMetrics); diff != "" {
				t.Errorf("GenerateMonitoringMetricFilters() mismatch (-want +got):\n%s", diff)
			}

			if got := sq.AllFiltersSupportMetrics(); got != tc.wantSupportMetricsFlag {
				t.Errorf("AllFiltersSupportMetrics() = %v, want %v", got, tc.wantSupportMetricsFlag)
			}
		})
	}
}

func TestGeneratePolicyActionStructuredQueryIsValid(t *testing.T) {
	testCases := []struct {
		name               string
		nodeNameSubstrings []string
	}{
		{
			name:               "valid query without node name substrings",
			nodeNameSubstrings: []string{},
		},
		{
			name:               "valid query with multiple node name substrings",
			nodeNameSubstrings: []string{"node-1", "node-2"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cluster := k8scommon.GoogleCloudClusterIdentity{
				ProjectID:   "test-project",
				Location:    "test-location",
				ClusterName: "test-cluster",
			}
			query := generatePolicyActionStructuredQuery(cluster, tc.nodeNameSubstrings).GenerateCloudLoggingQuery()
			if err := gcp_test.IsValidLogQuery(t, query); err != nil {
				t.Errorf("IsValidLogQuery() returned an error: %v", err)
			}
		})
	}
}

func TestListLogEntriesTask_DryRun(t *testing.T) {
	t.Parallel()
	startTime := time.Date(2026, time.January, 1, 1, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, time.January, 1, 1, 1, 0, 0, time.UTC)

	cluster := k8scommon.GoogleCloudClusterIdentity{
		ClusterName: "test-cluster",
		ProjectID:   "test-project",
		Location:    "us-central1-a",
	}

	resourceNamesInput := gcpcommon.NewResourceNamesInput()
	clientFactory, err := googlecloud.NewClientFactory()
	if err != nil {
		t.Fatalf("failed to create ClientFactory: %v", err)
	}

	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	gotLogs, _, err := inspectiontest.Run(t, ctx, listLogEntriesTask, inspectioncore.TaskModeDryRun, map[string]any{},
		tasktest.Given(gcpcommon.InputStartTimeTaskID.Ref(), startTime),
		tasktest.Given(gcpcommon.InputEndTimeTaskID.Ref(), endTime),
		tasktest.Given(gcpcommon.InputLoggingFilterResourceNameTaskID.Ref(), resourceNamesInput),
		// Dry run records the query without fetching logs.
		tasktest.Given[gcpcommon.LogFetcher](gcpcommon.LoggingFetcherTaskID.Ref(), nil),
		tasktest.Given(gcpcommon.APIClientFactoryTaskID.Ref(), clientFactory),
		tasktest.Given(gcpcommon.APIClientCallOptionsInjectorTaskID.Ref(), googlecloud.NewCallOptionInjector()),
		tasktest.Given(dpv2policy.ClusterIdentityTaskID.Ref(), cluster),
		tasktest.Given(k8scommon.InputNodeNameFilterTaskID.Ref(), []string{"node-1"}),
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
	if len(serialized) != 1 {
		t.Fatalf("expected 1 QueryItem, got %d", len(serialized))
	}

	wantQuery := `resource.type="k8s_node"
resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
LOG_ID("policy-action")
resource.labels.node_name:"node-1"
timestamp >= "2026-01-01T01:00:00+0000"
timestamp <= "2026-01-01T01:01:00+0000"`

	if diff := cmp.Diff(wantQuery, serialized[0].Query); diff != "" {
		t.Errorf("query mismatch (-want +got):\n%s", diff)
	}
	if serialized[0].Name != "Dataplane V2 network policy logs" {
		t.Errorf("query name mismatch: got %q, want %q", serialized[0].Name, "Dataplane V2 network policy logs")
	}
}
