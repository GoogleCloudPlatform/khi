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

package googlecloudlogmulticloudapiaudit_impl

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
	googlecloudlogmulticloudapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogmulticloudapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
	"github.com/google/go-cmp/cmp"
)

func TestGenerateMultiCloudAPIStructuredQuery(t *testing.T) {
	testCases := []struct {
		name                   string
		cluster                googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		wantQuery              string
		wantMetricFilters      []string
		wantSupportMetricsFlag bool
	}{
		{
			name: "AWS cluster",
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ProjectID:   "test-project",
				ClusterName: "test-cluster",
				PrefixPolicy: googlecloudk8scommon_contract.ClusterPrefixPolicy{
					Prefix: "awsClusters/",
					RequiredUsages: []googlecloudk8scommon_contract.ClusterNameUsage{
						googlecloudk8scommon_contract.ClusterNameUsageK8sPlatformAudit,
					},
				},
				Location: "asia-northeast1",
			},
			wantQuery: `resource.type="audited_resource"
resource.labels.service="gkemulticloud.googleapis.com"
resource.labels.method:("Update" OR "Create" OR "Delete")
(LOG_ID("cloudaudit.googleapis.com/activity") OR LOG_ID("cloudaudit.googleapis.com/data_access"))
protoPayload.resourceName:"projects/test-project/locations/asia-northeast1/"
protoPayload.resourceName:"awsClusters/test-cluster"`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "audited_resource" AND resource.labels.service = "gkemulticloud.googleapis.com" AND (resource.labels.method = has_substring("Update") OR resource.labels.method = has_substring("Create") OR resource.labels.method = has_substring("Delete")) AND metric.labels.log = one_of("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name: "Azure cluster",
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ProjectID:   "test-project",
				ClusterName: "test-cluster",
				PrefixPolicy: googlecloudk8scommon_contract.ClusterPrefixPolicy{
					Prefix: "azureClusters/",
					RequiredUsages: []googlecloudk8scommon_contract.ClusterNameUsage{
						googlecloudk8scommon_contract.ClusterNameUsageK8sPlatformAudit,
					},
				},
				Location: "us-east4",
			},
			wantQuery: `resource.type="audited_resource"
resource.labels.service="gkemulticloud.googleapis.com"
resource.labels.method:("Update" OR "Create" OR "Delete")
(LOG_ID("cloudaudit.googleapis.com/activity") OR LOG_ID("cloudaudit.googleapis.com/data_access"))
protoPayload.resourceName:"projects/test-project/locations/us-east4/"
protoPayload.resourceName:"azureClusters/test-cluster"`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "audited_resource" AND resource.labels.service = "gkemulticloud.googleapis.com" AND (resource.labels.method = has_substring("Update") OR resource.labels.method = has_substring("Create") OR resource.labels.method = has_substring("Delete")) AND metric.labels.log = one_of("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")`,
			},
			wantSupportMetricsFlag: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sq := GenerateMultiCloudAPIStructuredQuery(tc.cluster)
			gotQuery := sq.GenerateCloudLoggingQuery()
			if diff := cmp.Diff(tc.wantQuery, gotQuery); diff != "" {
				t.Errorf("GenerateCloudLoggingQuery() mismatch (-want +got):\n%s", diff)
			}

			legacyQuery := generateQuery(tc.cluster)
			if diff := cmp.Diff(gotQuery, legacyQuery); diff != "" {
				t.Errorf("generateQuery() mismatch (-want +got):\n%s", diff)
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

func TestGenerateMultiCloudAPIQueryIsValid(t *testing.T) {
	testCases := []struct {
		name    string
		cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity
	}{
		{
			name: "Valid Query",
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ProjectID:   "test-project",
				ClusterName: "test-cluster",
				PrefixPolicy: googlecloudk8scommon_contract.ClusterPrefixPolicy{
					Prefix: "awsClusters/",
					RequiredUsages: []googlecloudk8scommon_contract.ClusterNameUsage{
						googlecloudk8scommon_contract.ClusterNameUsageK8sPlatformAudit,
					},
				},
				Location: "asia-northeast1",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query := generateQuery(tc.cluster)
			err := gcp_test.IsValidLogQuery(t, query)
			if err != nil {
				t.Errorf("%s", err.Error())
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
		Location:    "asia-northeast1",
		PrefixPolicy: googlecloudk8scommon_contract.ClusterPrefixPolicy{
			Prefix: "awsClusters/",
			RequiredUsages: []googlecloudk8scommon_contract.ClusterNameUsage{
				googlecloudk8scommon_contract.ClusterNameUsageK8sPlatformAudit,
			},
		},
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
		tasktest.NewTaskDependencyValuePair(googlecloudlogmulticloudapiaudit_contract.ClusterIdentityTaskID.Ref(), cluster),
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

	wantQuery := `resource.type="audited_resource"
resource.labels.service="gkemulticloud.googleapis.com"
resource.labels.method:("Update" OR "Create" OR "Delete")
(LOG_ID("cloudaudit.googleapis.com/activity") OR LOG_ID("cloudaudit.googleapis.com/data_access"))
protoPayload.resourceName:"projects/test-project/locations/asia-northeast1/"
protoPayload.resourceName:"awsClusters/test-cluster"
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`

	if diff := cmp.Diff(wantQuery, serialized[0].Query); diff != "" {
		t.Errorf("query mismatch (-want +got):\n%s", diff)
	}
	if serialized[0].Name != "Multicloud API Logs" {
		t.Errorf("query name mismatch: got %q, want %q", serialized[0].Name, "Multicloud API Logs")
	}
}
