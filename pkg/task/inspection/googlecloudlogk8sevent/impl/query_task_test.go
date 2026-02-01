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
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
)

func TestGenerateK8sEventQuery(t *testing.T) {
	testCases := []struct {
		wantQuery       string
		cluster         googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		namespaceFilter *gcpqueryutil.SetFilterParseResult
		startTime       time.Time
		endTime         time.Time
	}{
		{
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "foo-cluster",
				Location:    "foo-location",
				ProjectID:   "foo-project",
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{
					"#namespaced",
				},
			},
			wantQuery: `log_id("events")
resource.labels.project_id="foo-project"
resource.labels.location="foo-location"
resource.labels.cluster_name="foo-cluster"
jsonPayload.involvedObject.namespace:"" -- ignore events in k8s object with namespace`,
		},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("testcase-%d-%s", i, testCase.wantQuery), func(t *testing.T) {
			result := GenerateK8sEventQuery(testCase.cluster, testCase.namespaceFilter)
			if result != testCase.wantQuery {
				t.Errorf("the result query is not valid:\nInput:\n%v\nActual:\n%s\nExpected:\n%s", testCase, result, testCase.wantQuery)
			}
		})
	}
}

func TestGenerateK8sEventQueryIsValid(t *testing.T) {
	testCluster := googlecloudk8scommon_contract.GoogleCloudClusterIdentity{}
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
				t.Errorf("%s", err.Error())
			}
		})
	}

}
