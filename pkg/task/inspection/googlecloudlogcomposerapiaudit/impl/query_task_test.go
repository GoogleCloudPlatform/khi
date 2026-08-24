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

package googlecloudlogcomposerapiaudit_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogcomposerapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcomposerapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
	"github.com/google/go-cmp/cmp"
)

func TestGenerateComposerAuditStructuredQuery(t *testing.T) {
	testCases := []struct {
		name                   string
		projectID              string
		location               string
		environmentName        string
		wantQuery              string
		wantMetricFilters      []string
		wantSupportMetricsFlag bool
	}{
		{
			name:            "with location and environment name",
			projectID:       "my-project",
			location:        "us-central1",
			environmentName: "env-1",
			wantQuery: `resource.type="cloud_composer_environment"
resource.labels.project_id="my-project"
resource.labels.location="us-central1"
resource.labels.environment_name="env-1"
(LOG_ID("cloudaudit.googleapis.com/activity") OR LOG_ID("cloudaudit.googleapis.com/data_access"))
protoPayload.serviceName="composer.googleapis.com"`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "cloud_composer_environment" AND resource.labels.project_id = "my-project" AND resource.labels.location = "us-central1" AND resource.labels.environment_name = "env-1" AND metric.labels.log = one_of("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:            "empty location and environment name",
			projectID:       "my-project",
			location:        "",
			environmentName: "",
			wantQuery: `resource.type="cloud_composer_environment"
resource.labels.project_id="my-project"
(LOG_ID("cloudaudit.googleapis.com/activity") OR LOG_ID("cloudaudit.googleapis.com/data_access"))
protoPayload.serviceName="composer.googleapis.com"`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "cloud_composer_environment" AND resource.labels.project_id = "my-project" AND metric.labels.log = one_of("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")`,
			},
			wantSupportMetricsFlag: false,
		},
		{
			name:            "location all",
			projectID:       "my-project",
			location:        "all",
			environmentName: "env-1",
			wantQuery: `resource.type="cloud_composer_environment"
resource.labels.project_id="my-project"
resource.labels.environment_name="env-1"
(LOG_ID("cloudaudit.googleapis.com/activity") OR LOG_ID("cloudaudit.googleapis.com/data_access"))
protoPayload.serviceName="composer.googleapis.com"`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "cloud_composer_environment" AND resource.labels.project_id = "my-project" AND resource.labels.environment_name = "env-1" AND metric.labels.log = one_of("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")`,
			},
			wantSupportMetricsFlag: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sq := GenerateComposerAuditStructuredQuery(tc.projectID, tc.location, tc.environmentName)
			gotQuery := sq.GenerateCloudLoggingQuery()
			if diff := cmp.Diff(tc.wantQuery, gotQuery); diff != "" {
				t.Errorf("GenerateCloudLoggingQuery() mismatch (-want +got):\n%s", diff)
			}

			legacyQuery := GenerateComposerAuditQuery(tc.projectID, tc.location, tc.environmentName)
			if diff := cmp.Diff(gotQuery, legacyQuery); diff != "" {
				t.Errorf("GenerateComposerAuditQuery() mismatch (-want +got):\n%s", diff)
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

func TestGenerateComposerAuditQueryIsValid(t *testing.T) {
	testCases := []struct {
		name            string
		projectID       string
		location        string
		environmentName string
	}{
		{
			name:            "Valid Query with location and env",
			projectID:       "my-project",
			location:        "us-central1",
			environmentName: "env-1",
		},
		{
			name:            "Valid Query with empty location",
			projectID:       "my-project",
			location:        "",
			environmentName: "env-1",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query := GenerateComposerAuditQuery(tc.projectID, tc.location, tc.environmentName)
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
		Location:    "us-central1",
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
		tasktest.NewTaskDependencyValuePair(googlecloudlogcomposerapiaudit_contract.ClusterIdentityTaskID.Ref(), cluster),
		tasktest.NewTaskDependencyValuePair(googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref(), "test-env"),
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

	wantQuery := `resource.type="cloud_composer_environment"
resource.labels.project_id="test-project"
resource.labels.location="us-central1"
resource.labels.environment_name="test-env"
(LOG_ID("cloudaudit.googleapis.com/activity") OR LOG_ID("cloudaudit.googleapis.com/data_access"))
protoPayload.serviceName="composer.googleapis.com"
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`

	if diff := cmp.Diff(wantQuery, serialized[0].Query); diff != "" {
		t.Errorf("query mismatch (-want +got):\n%s", diff)
	}
	if serialized[0].Name != "Composer API Audit logs" {
		t.Errorf("query name mismatch: got %q, want %q", serialized[0].Name, "Composer API Audit logs")
	}
}
