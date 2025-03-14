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

package k8s_event

import (
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
)

func TestGenerateK8sEventQuery(t *testing.T) {
	testCases := []struct {
		ExpectedQuery        string
		InputClusterName     string
		InputProjectName     string
		InputNamespaceFilter *queryutil.SetFilterParseResult
		InputStartTime       time.Time
		InputEndTime         time.Time
	}{
		{
			InputClusterName: "foo-cluster",
			InputProjectName: "foo-project",
			InputNamespaceFilter: &queryutil.SetFilterParseResult{
				Additives: []string{
					"#namespaced",
				},
			},
			ExpectedQuery: `logName="projects/foo-project/logs/events"
resource.labels.cluster_name="foo-cluster"
jsonPayload.involvedObject.namespace:"" -- ignore events in k8s object with namespace`,
		},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("testcase-%d-%s", i, testCase.ExpectedQuery), func(t *testing.T) {
			result := GenerateK8sEventQuery(testCase.InputClusterName, testCase.InputProjectName, testCase.InputNamespaceFilter)
			if result != testCase.ExpectedQuery {
				t.Errorf("the result query is not valid:\nInput:\n%v\nActual:\n%s\nExpected:\n%s", testCase, result, testCase.ExpectedQuery)
			}
		})
	}
}

func TestGenerateK8sEventQueryIsValid(t *testing.T) {
	testCases := []struct {
		Name            string
		ClusterName     string
		ProjectName     string
		NamespaceFilter *queryutil.SetFilterParseResult
	}{
		{
			Name:            "ClusterScoped",
			ClusterName:     "foo-cluster",
			ProjectName:     "foo-project",
			NamespaceFilter: &queryutil.SetFilterParseResult{Additives: []string{"#cluster-scoped"}},
		},
		{
			Name:            "Namespaced",
			ClusterName:     "foo-cluster",
			ProjectName:     "foo-project",
			NamespaceFilter: &queryutil.SetFilterParseResult{Additives: []string{"#namespaced"}},
		},
		{
			Name:            "Namespaced with specific namespace",
			ClusterName:     "foo-cluster",
			ProjectName:     "foo-project",
			NamespaceFilter: &queryutil.SetFilterParseResult{Additives: []string{"default"}},
		},
		{
			Name:            "Namespaced with multiple namespaces",
			ClusterName:     "foo-cluster",
			ProjectName:     "foo-project",
			NamespaceFilter: &queryutil.SetFilterParseResult{Additives: []string{"default", "kube-system"}},
		},
		{
			Name:            "ClusterScoped with specific namespace",
			ClusterName:     "foo-cluster",
			ProjectName:     "foo-project",
			NamespaceFilter: &queryutil.SetFilterParseResult{Additives: []string{"#cluster-scoped", "default"}},
		},
		{
			Name:            "ClusterScoped with multiple namespaces",
			ClusterName:     "foo-cluster",
			ProjectName:     "foo-project",
			NamespaceFilter: &queryutil.SetFilterParseResult{Additives: []string{"#cluster-scoped", "default", "kube-system"}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			query := GenerateK8sEventQuery(tc.ClusterName, tc.ProjectName, tc.NamespaceFilter)
			err := gcp_test.IsValidLogQuery(t, query)
			if err != nil {
				t.Errorf("%s", err.Error())
			}
		})
	}

}
