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

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	googlecloudclustergdcbaremetal_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergdcbaremetal/impl"
	googlecloudclustergke_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergke/impl"
	googlecloudclustergkeonaws_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergkeonaws/impl"
	googlecloudclustergkeonazure_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergkeonazure/impl"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudk8scommon_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/impl"
	googlecloudlogcsm_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcsm/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

func TestGenerateCSMTrafficLogsStructuredQuery(t *testing.T) {
	cluster := googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ProjectID:   "test-project",
		Location:    "test-location",
		ClusterName: "test-cluster",
	}

	testCases := []struct {
		desc                   string
		cluster                googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		responseFlagsFilter    *gcpqueryutil.SetFilterParseResult
		namespaceFilter        *gcpqueryutil.SetFilterParseResult
		wantQuery              string
		wantSupportMetricsFlag bool
	}{
		{
			desc:    "basic filter with additives",
			cluster: cluster,
			responseFlagsFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"UH", "UT"},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"default"},
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.cluster_name="test-cluster"
resource.labels.namespace_name:"default"
(LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver"))
labels.response_flag:("UH" OR "UT")`,
			wantSupportMetricsFlag: false,
		},
		{
			desc:    "response flags subtractive filter",
			cluster: cluster,
			responseFlagsFilter: &gcpqueryutil.SetFilterParseResult{
				Subtractives: []string{"-"},
				SubtractMode: true,
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"default"},
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.cluster_name="test-cluster"
resource.labels.namespace_name:"default"
(LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver"))
-labels.response_flag:("-")`,
			wantSupportMetricsFlag: false,
		},
		{
			desc:    "response flags subtractive filter with empty subtractives",
			cluster: cluster,
			responseFlagsFilter: &gcpqueryutil.SetFilterParseResult{
				Subtractives: []string{},
				SubtractMode: true,
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"default"},
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.cluster_name="test-cluster"
resource.labels.namespace_name:"default"
(LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver"))`,
			wantSupportMetricsFlag: true,
		},
		{
			desc:    "response flags empty additives",
			cluster: cluster,
			responseFlagsFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"default"},
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.cluster_name="test-cluster"
resource.labels.namespace_name:"default"
(LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver"))
-- Invalid: none of the resources will be selected. Ignoring response flag filter.`,
			wantSupportMetricsFlag: true,
		},
		{
			desc:    "response flags validation error",
			cluster: cluster,
			responseFlagsFilter: &gcpqueryutil.SetFilterParseResult{
				ValidationError: "invalid response flag format",
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"default"},
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.cluster_name="test-cluster"
resource.labels.namespace_name:"default"
(LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver"))
-- Failed to generate response flags filter due to the validation error "invalid response flag format"`,
			wantSupportMetricsFlag: true,
		},
		{
			desc:    "nil response flags filter and nil namespace filter",
			cluster: cluster,
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.cluster_name="test-cluster"
(LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver"))`,
			wantSupportMetricsFlag: true,
		},
		{
			desc:    "namespace validation error",
			cluster: cluster,
			responseFlagsFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"UH"},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				ValidationError: "invalid namespace syntax",
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.cluster_name="test-cluster"
-- Failed to generate namespace filter due to the validation error "invalid namespace syntax"
(LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver"))
labels.response_flag:("UH")`,
			wantSupportMetricsFlag: false,
		},
		{
			desc:    "namespace subtractive mode unsupported",
			cluster: cluster,
			responseFlagsFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"UH"},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{"kube-system"},
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.cluster_name="test-cluster"
-- Unsupported operation
(LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver"))
labels.response_flag:("UH")`,
			wantSupportMetricsFlag: false,
		},
		{
			desc:    "namespace cluster-scoped filter",
			cluster: cluster,
			responseFlagsFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"UH"},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"#cluster-scoped"},
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.cluster_name="test-cluster"
resource.labels.namespace_name="" -- Invalid: No namespaces remain to filter for CSM traffic logs.
(LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver"))
labels.response_flag:("UH")`,
			wantSupportMetricsFlag: false,
		},
		{
			desc:    "namespace cluster-scoped and specific namespaces filter",
			cluster: cluster,
			responseFlagsFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"UH"},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"#cluster-scoped", "kube-system"},
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.cluster_name="test-cluster"
resource.labels.namespace_name:"kube-system"
(LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver"))
labels.response_flag:("UH")`,
			wantSupportMetricsFlag: false,
		},
		{
			desc:    "namespace namespaced-scoped filter",
			cluster: cluster,
			responseFlagsFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"UH"},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"#namespaced"},
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.cluster_name="test-cluster"
(LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver"))
labels.response_flag:("UH")`,
			wantSupportMetricsFlag: false,
		},
		{
			desc:    "namespace cluster-scoped and namespaced-scoped filter",
			cluster: cluster,
			responseFlagsFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"UH"},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"#cluster-scoped", "#namespaced"},
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.cluster_name="test-cluster"
(LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver"))
labels.response_flag:("UH")`,
			wantSupportMetricsFlag: false,
		},
		{
			desc:    "multiple namespaces filter",
			cluster: cluster,
			responseFlagsFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"UH"},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"default", "istio-system"},
			},
			wantQuery: `resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.cluster_name="test-cluster"
resource.labels.namespace_name:("default" OR "istio-system")
(LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver"))
labels.response_flag:("UH")`,
			wantSupportMetricsFlag: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			sq := GenerateCSMTrafficLogsStructuredQuery(tc.cluster, tc.responseFlagsFilter, tc.namespaceFilter)
			gotQuery := sq.GenerateCloudLoggingQuery()
			if diff := cmp.Diff(tc.wantQuery, gotQuery); diff != "" {
				t.Errorf("GenerateCloudLoggingQuery() mismatch (-want +got):\n%s", diff)
			}

			legacyQuery := GenerateCSMTrafficLogsQuery(tc.cluster, tc.responseFlagsFilter, tc.namespaceFilter)
			if diff := cmp.Diff(gotQuery, legacyQuery); diff != "" {
				t.Errorf("GenerateCSMTrafficLogsQuery() mismatch (-want +got):\n%s", diff)
			}

			if sq.AllFiltersSupportMetrics() != tc.wantSupportMetricsFlag {
				t.Errorf("AllFiltersSupportMetrics() = %v, want %v", sq.AllFiltersSupportMetrics(), tc.wantSupportMetricsFlag)
			}
		})
	}
}

func TestCSMQueryTaskPrefixResolution(t *testing.T) {
	testCases := []struct {
		name       string
		prefixTask coretask.Task[googlecloudk8scommon_contract.ClusterPrefixPolicy]
		want       string
	}{
		{
			name:       "GKE side task generates no prefix",
			prefixTask: googlecloudclustergke_impl.GKEClusterNamePrefixTask,
			want: `resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.cluster_name="test-cluster"
resource.labels.namespace_name:"default"
(LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver"))
labels.response_flag:("UH")`,
		},
		{
			name:       "baremetal side task adds prefix",
			prefixTask: googlecloudclustergdcbaremetal_impl.GDCVForBaremetalClusterNamePrefixTask,
			want: `resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.cluster_name="baremetalClusters/test-cluster"
resource.labels.namespace_name:"default"
(LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver"))
labels.response_flag:("UH")`,
		},
		{
			name:       "aws side task adds prefix",
			prefixTask: googlecloudclustergkeonaws_impl.AnthosOnAWSClusterNamePrefixTask,
			want: `resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.cluster_name="awsClusters/test-cluster"
resource.labels.namespace_name:"default"
(LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver"))
labels.response_flag:("UH")`,
		},
		{
			name:       "azure side task adds prefix",
			prefixTask: googlecloudclustergkeonazure_impl.AnthosOnAzureClusterNamePrefixTask,
			want: `resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.cluster_name="azureClusters/test-cluster"
resource.labels.namespace_name:"default"
(LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver"))
labels.response_flag:("UH")`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = khictx.WithValue(ctx, inspectioncore_contract.InspectionTaskMode, inspectioncore_contract.TaskModeRun)
			prefixPolicy, err := tasktest.RunTask(ctx, tc.prefixTask)
			if err != nil {
				t.Fatalf("unexpected error running prefix task: %v", err)
			}

			idRes, err := tasktest.RunTask(ctx, googlecloudk8scommon_impl.ClusterIdentityTask,
				tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputProjectIdTaskID.Ref(), "test-project"),
				tasktest.NewTaskDependencyValuePair(googlecloudk8scommon_contract.InputClusterNameTaskID.Ref(), "test-cluster"),
				tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputLocationsTaskID.Ref(), "test-location"),
				tasktest.NewTaskDependencyValuePair(googlecloudk8scommon_contract.ClusterNamePrefixTaskRef, prefixPolicy),
			)
			if err != nil {
				t.Fatalf("unexpected error running cluster identity task: %v", err)
			}

			got := GenerateCSMTrafficLogsQuery(idRes, &gcpqueryutil.SetFilterParseResult{Additives: []string{"UH"}}, &gcpqueryutil.SetFilterParseResult{Additives: []string{"default"}})

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateCSMTrafficLogsQuery() mismatch (-want +got):\n%s", diff)
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
		Location:    "test-location",
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
		tasktest.NewTaskDependencyValuePair(googlecloudlogcsm_contract.ClusterIdentityTaskID.Ref(), cluster),
		tasktest.NewTaskDependencyValuePair(googlecloudk8scommon_contract.InputNamespaceFilterTaskID.Ref(), &gcpqueryutil.SetFilterParseResult{Additives: []string{"default"}}),
		tasktest.NewTaskDependencyValuePair(googlecloudlogcsm_contract.InputCSMResponseFlagsTaskID.Ref(), &gcpqueryutil.SetFilterParseResult{Additives: []string{"UH"}}),
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
resource.labels.location="test-location"
resource.labels.cluster_name="test-cluster"
resource.labels.namespace_name:"default"
(LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver"))
labels.response_flag:("UH")
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`

	if diff := cmp.Diff(wantQuery, serialized[0].Query); diff != "" {
		t.Errorf("query mismatch (-want +got):\n%s", diff)
	}
	if serialized[0].Name != "CSM Traffic logs" {
		t.Errorf("query name mismatch: got %q, want %q", serialized[0].Name, "CSM Traffic logs")
	}
}
