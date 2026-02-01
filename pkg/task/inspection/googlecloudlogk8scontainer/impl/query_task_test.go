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

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
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
resource.labels.namespace_name=("kube-system")
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
-- Invalid: none of the resources will be selected. Ignoring namespace filter.
resource.labels.pod_name:("nginx-pod")`,
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
resource.labels.namespace_name=("kube-system")
resource.labels.pod_name:("nginx-pod")`,
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
resource.labels.namespace_name=("kube-system" OR "istio-system")
resource.labels.pod_name:("nginx-pod" OR "apache-pod")`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			query := GenerateK8sContainerQuery(tc.Cluster, tc.NamespaceFilter, tc.PodNameFilter)
			if query != tc.ExpectedQuery {
				t.Errorf("GenerateK8sContainerQuery() = %v, want %v", query, tc.ExpectedQuery)
			}
			err := gcp_test.IsValidLogQuery(t, query)
			if err != nil {
				t.Errorf("%s", err.Error())
			}
		})
	}
}
