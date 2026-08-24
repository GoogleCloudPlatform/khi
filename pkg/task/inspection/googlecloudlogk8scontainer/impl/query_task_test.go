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

package googlecloudlogk8scontainer_impl

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
	googlecloudlogk8scontainer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontainer/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
	"github.com/google/go-cmp/cmp"
)

func TestGenerateK8sContainerQueryIsValid(t *testing.T) {
	testCases := []struct {
		Name            string
		Cluster         googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		PodNameFilter   *gcpqueryutil.SetFilterParseResult
		NamespaceFilter *gcpqueryutil.SetFilterParseResult
		ExpectedQuery   string
	}{
		{
			Name: "with no set filters",
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			PodNameFilter:   &gcpqueryutil.SetFilterParseResult{Additives: []string{}},
			NamespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{}},
			ExpectedQuery: `resource.type="k8s_container"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-LOG_ID("server-accesslog-stackdriver")
-LOG_ID("client-accesslog-stackdriver")
-- Invalid: none of the resources will be selected. Ignoring namespace filter.
-- Invalid: none of the resources will be selected. Ignoring pod name filter.`,
		},
		{
			Name: "with namespace filter",
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			PodNameFilter:   &gcpqueryutil.SetFilterParseResult{Additives: []string{}},
			NamespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"kube-system"}},
			ExpectedQuery: `resource.type="k8s_container"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-LOG_ID("server-accesslog-stackdriver")
-LOG_ID("client-accesslog-stackdriver")
resource.labels.namespace_name="kube-system"
-- Invalid: none of the resources will be selected. Ignoring pod name filter.`,
		},
		{
			Name: "with pod name filter",
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			PodNameFilter:   &gcpqueryutil.SetFilterParseResult{Additives: []string{"nginx-pod"}},
			NamespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{}},
			ExpectedQuery: `resource.type="k8s_container"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-LOG_ID("server-accesslog-stackdriver")
-LOG_ID("client-accesslog-stackdriver")
-- Invalid: none of the resources will be selected. Ignoring namespace filter.
resource.labels.pod_name:"nginx-pod"`,
		},
		{
			Name: "with both filters",
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			PodNameFilter:   &gcpqueryutil.SetFilterParseResult{Additives: []string{"nginx-pod"}},
			NamespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"kube-system"}},
			ExpectedQuery: `resource.type="k8s_container"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-LOG_ID("server-accesslog-stackdriver")
-LOG_ID("client-accesslog-stackdriver")
resource.labels.namespace_name="kube-system"
resource.labels.pod_name:"nginx-pod"`,
		},
		{
			Name: "with complex filters",
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			PodNameFilter:   &gcpqueryutil.SetFilterParseResult{Additives: []string{"nginx-pod", "apache-pod"}},
			NamespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"kube-system", "istio-system"}},
			ExpectedQuery: `resource.type="k8s_container"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-LOG_ID("server-accesslog-stackdriver")
-LOG_ID("client-accesslog-stackdriver")
resource.labels.namespace_name=("kube-system" OR "istio-system")
resource.labels.pod_name:("nginx-pod" OR "apache-pod")`,
		},
		{
			Name: "with subtractive namespace filter",
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			PodNameFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: true, Subtractives: []string{}},
			NamespaceFilter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{"kube-system", "istio-system"},
			},
			ExpectedQuery: `resource.type="k8s_container"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-LOG_ID("server-accesslog-stackdriver")
-LOG_ID("client-accesslog-stackdriver")
-resource.labels.namespace_name=("kube-system" OR "istio-system")`,
		},
		{
			Name: "with subtractive pod name filter",
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			PodNameFilter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{"nginx-", "redis"},
			},
			NamespaceFilter: &gcpqueryutil.SetFilterParseResult{Additives: []string{"default"}},
			ExpectedQuery: `resource.type="k8s_container"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-LOG_ID("server-accesslog-stackdriver")
-LOG_ID("client-accesslog-stackdriver")
resource.labels.namespace_name="default"
-resource.labels.pod_name:("nginx-" OR "redis")`,
		},
		{
			Name: "with empty subtractive namespace filter",
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			PodNameFilter: &gcpqueryutil.SetFilterParseResult{SubtractMode: true, Subtractives: []string{}},
			NamespaceFilter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{},
			},
			ExpectedQuery: `resource.type="k8s_container"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-LOG_ID("server-accesslog-stackdriver")
-LOG_ID("client-accesslog-stackdriver")`,
		},
		{
			Name: "with validation error on namespace and pod name",
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			PodNameFilter: &gcpqueryutil.SetFilterParseResult{
				ValidationError: "invalid pod regex",
			},
			NamespaceFilter: &gcpqueryutil.SetFilterParseResult{
				ValidationError: "invalid namespace regex",
			},
			ExpectedQuery: `resource.type="k8s_container"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-LOG_ID("server-accesslog-stackdriver")
-LOG_ID("client-accesslog-stackdriver")
-- Failed to generate namespaces filter due to the validation error "invalid namespace regex"
-- Failed to generate pod name filter due to the validation error "invalid pod regex"`,
		},
		{
			Name: "with nil namespace and pod name filters",
			Cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				ProjectID:   "foo-project",
				Location:    "foo-location",
			},
			PodNameFilter:   nil,
			NamespaceFilter: nil,
			ExpectedQuery: `resource.type="k8s_container"
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
-LOG_ID("server-accesslog-stackdriver")
-LOG_ID("client-accesslog-stackdriver")`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			query := GenerateK8sContainerQuery(tc.Cluster, tc.NamespaceFilter, tc.PodNameFilter)
			if diff := cmp.Diff(tc.ExpectedQuery, query); diff != "" {
				t.Errorf("GenerateK8sContainerQuery() mismatch (-want +got):\n%s", diff)
			}
			err := gcp_test.IsValidLogQuery(t, query)
			if err != nil {
				t.Errorf("%s", err.Error())
			}
		})
	}
}

func TestGenerateK8sContainerStructuredQuery_MetricSupport(t *testing.T) {
	cluster := googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ClusterName: "foo-cluster",
		ProjectID:   "foo-project",
		Location:    "foo-location",
	}

	testCases := []struct {
		desc                     string
		namespaceFilter          *gcpqueryutil.SetFilterParseResult
		podNameFilter            *gcpqueryutil.SetFilterParseResult
		wantAllSupported         bool
		wantMonitoringFilterBody string
	}{
		{
			desc: "all supported when single namespace and no pod name filter",
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"kube-system"},
			},
			podNameFilter:            &gcpqueryutil.SetFilterParseResult{SubtractMode: true, Subtractives: []string{}},
			wantAllSupported:         true,
			wantMonitoringFilterBody: `resource.type = "k8s_container" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster" AND metric.labels.log != "server-accesslog-stackdriver" AND metric.labels.log != "client-accesslog-stackdriver" AND resource.labels.namespace_name = "kube-system"`,
		},
		{
			desc: "all supported when multiple namespaces with one_of",
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"kube-system", "istio-system"},
			},
			podNameFilter:            &gcpqueryutil.SetFilterParseResult{SubtractMode: true, Subtractives: []string{}},
			wantAllSupported:         true,
			wantMonitoringFilterBody: `resource.type = "k8s_container" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster" AND metric.labels.log != "server-accesslog-stackdriver" AND metric.labels.log != "client-accesslog-stackdriver" AND resource.labels.namespace_name = one_of("kube-system", "istio-system")`,
		},
		{
			desc: "all supported when subtractive namespace filter",
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				SubtractMode: true,
				Subtractives: []string{"kube-system"},
			},
			podNameFilter:            &gcpqueryutil.SetFilterParseResult{SubtractMode: true, Subtractives: []string{}},
			wantAllSupported:         true,
			wantMonitoringFilterBody: `resource.type = "k8s_container" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster" AND metric.labels.log != "server-accesslog-stackdriver" AND metric.labels.log != "client-accesslog-stackdriver" AND resource.labels.namespace_name != "kube-system"`,
		},
		{
			desc: "all supported when pod name filter is present with has_substring",
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"kube-system"},
			},
			podNameFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"nginx-pod"},
			},
			wantAllSupported:         true,
			wantMonitoringFilterBody: `resource.type = "k8s_container" AND resource.labels.project_id = "foo-project" AND resource.labels.location = "foo-location" AND resource.labels.cluster_name = "foo-cluster" AND metric.labels.log != "server-accesslog-stackdriver" AND metric.labels.log != "client-accesslog-stackdriver" AND resource.labels.namespace_name = "kube-system" AND resource.labels.pod_name = has_substring("nginx-pod")`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			sq := GenerateK8sContainerStructuredQuery(cluster, tc.namespaceFilter, tc.podNameFilter)
			if got := sq.AllFiltersSupportMetrics(); got != tc.wantAllSupported {
				t.Errorf("AllFiltersSupportMetrics() = %v, want %v", got, tc.wantAllSupported)
			}
			filters := sq.GenerateMonitoringMetricFilters()
			if len(filters) != 1 {
				t.Fatalf("expected 1 metric filter, got %d", len(filters))
			}
			wantFilter := "metric.type = \"logging.googleapis.com/log_entry_count\" AND " + tc.wantMonitoringFilterBody
			if diff := cmp.Diff(wantFilter, filters[0]); diff != "" {
				t.Errorf("Monitoring metric filter mismatch (-want +got):\n%s", diff)
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
		tasktest.NewTaskDependencyValuePair(googlecloudlogk8scontainer_contract.ClusterIdentityTaskID.Ref(), cluster),
		tasktest.NewTaskDependencyValuePair(googlecloudlogk8scontainer_contract.InputContainerQueryNamespacesTaskID.Ref(), &gcpqueryutil.SetFilterParseResult{Additives: []string{"default"}}),
		tasktest.NewTaskDependencyValuePair(googlecloudlogk8scontainer_contract.InputContainerQueryPodNamesTaskID.Ref(), &gcpqueryutil.SetFilterParseResult{SubtractMode: true, Subtractives: []string{}}),
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

	wantQuery := `resource.type="k8s_container"
resource.labels.project_id="test-project"
resource.labels.location="us-central1-a"
resource.labels.cluster_name="test-cluster"
-LOG_ID("server-accesslog-stackdriver")
-LOG_ID("client-accesslog-stackdriver")
resource.labels.namespace_name="default"
timestamp >= "2025-01-01T01:00:00+0000"
timestamp <= "2025-01-01T01:01:00+0000"`

	if diff := cmp.Diff(wantQuery, serialized[0].Query); diff != "" {
		t.Errorf("Query mismatch (-want +got):\n%s", diff)
	}
	if serialized[0].Name != "K8s container logs" {
		t.Errorf("Query Name mismatch: got %q, want %q", serialized[0].Name, "K8s container logs")
	}
}
