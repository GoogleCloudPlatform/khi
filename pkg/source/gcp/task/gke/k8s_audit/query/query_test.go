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

package query

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
)

func TestGenerateK8sAuditQuery(t *testing.T) {
	testCases := []struct {
		ExpectedQuery        string
		InputClusterName     string
		InputKindFilter      *queryutil.SetFilterParseResult
		InputNamespaceFilter *queryutil.SetFilterParseResult
	}{
		{
			ExpectedQuery: `resource.type="k8s_cluster"
resource.labels.cluster_name="foo-cluster"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
protoPayload.methodName=~"\.(pods|deployments|jobs)\."
protoPayload.resourceName:"namespaces/"
`,
			InputClusterName: "foo-cluster",
			InputKindFilter: &queryutil.SetFilterParseResult{
				Additives: []string{
					"pods",
					"deployments",
					"jobs",
				},
			},
			InputNamespaceFilter: &queryutil.SetFilterParseResult{
				Additives: []string{
					"#namespaced",
				},
			},
		},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("testcase-%d-%s", i, testCase.ExpectedQuery), func(t *testing.T) {
			result := GenerateK8sAuditQuery(testCase.InputClusterName, testCase.InputKindFilter, testCase.InputNamespaceFilter)
			if result != testCase.ExpectedQuery {
				t.Errorf("the result query is not valid:\nInput:\n%v\nActual:\n%s\nExpected:\n%s", testCase, result, testCase.ExpectedQuery)
			}
		})
	}
}

func TestGenerateK8sAuditQueryIsValid(t *testing.T) {
	testCases := []struct {
		Name            string
		ClusterName     string
		KindFilter      *queryutil.SetFilterParseResult
		NamespaceFilter *queryutil.SetFilterParseResult
	}{
		{
			Name:            "ClusterScoped",
			ClusterName:     "foo-cluster",
			KindFilter:      &queryutil.SetFilterParseResult{Additives: []string{"pods"}},
			NamespaceFilter: &queryutil.SetFilterParseResult{Additives: []string{"#cluster-scoped"}},
		},
		{
			Name:            "Namespaced",
			ClusterName:     "foo-cluster",
			KindFilter:      &queryutil.SetFilterParseResult{Additives: []string{"pods"}},
			NamespaceFilter: &queryutil.SetFilterParseResult{Additives: []string{"#namespaced"}},
		},
		{
			Name:            "Namespaced with specific namespace",
			ClusterName:     "foo-cluster",
			KindFilter:      &queryutil.SetFilterParseResult{Additives: []string{"pods"}},
			NamespaceFilter: &queryutil.SetFilterParseResult{Additives: []string{"default"}},
		},
		{
			Name:            "Namespaced with multiple namespaces",
			ClusterName:     "foo-cluster",
			KindFilter:      &queryutil.SetFilterParseResult{Additives: []string{"pods"}},
			NamespaceFilter: &queryutil.SetFilterParseResult{Additives: []string{"default", "kube-system"}},
		},
		{
			Name:            "ClusterScoped with specific namespace",
			ClusterName:     "foo-cluster",
			KindFilter:      &queryutil.SetFilterParseResult{Additives: []string{"pods"}},
			NamespaceFilter: &queryutil.SetFilterParseResult{Additives: []string{"#cluster-scoped", "default"}},
		},
		{
			Name:            "ClusterScoped with multiple namespaces",
			ClusterName:     "foo-cluster",
			KindFilter:      &queryutil.SetFilterParseResult{Additives: []string{"pods"}},
			NamespaceFilter: &queryutil.SetFilterParseResult{Additives: []string{"#cluster-scoped", "default", "kube-system"}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			query := GenerateK8sAuditQuery(tc.ClusterName, tc.KindFilter, tc.NamespaceFilter)
			err := gcp_test.IsValidLogQuery(query)
			if err != nil {
				t.Errorf(err.Error())
			}
		})
	}
}

func TestGenerateNamespaceFilter(t *testing.T) {
	testCases := []struct {
		ExpectedQuery string
		Input         *queryutil.SetFilterParseResult
	}{
		{
			ExpectedQuery: `-- Invalid: none of the resources will be selected. Ignoreing namespace filter.`,
			Input: &queryutil.SetFilterParseResult{
				Additives: []string{},
			},
		},
		{
			ExpectedQuery: `-- Failed to generate namespace filter due to the validation error "test error"`,
			Input: &queryutil.SetFilterParseResult{
				ValidationError: "test error",
			},
		},
		{
			ExpectedQuery: `protoPayload.resourceName:("/namespaces/kube-system" OR "/namespaces/default")`,
			Input: &queryutil.SetFilterParseResult{
				Additives: []string{
					"kube-system",
					"default",
				},
			},
		},
		{
			ExpectedQuery: `protoPayload.resourceName:("/namespaces/kube-system" OR "/namespaces/default")`,
			Input: &queryutil.SetFilterParseResult{
				Additives: []string{
					"kube-system",
					"default",
				},
			},
		},
		{
			ExpectedQuery: `-- No namespace filter`,
			Input: &queryutil.SetFilterParseResult{
				Additives: []string{
					"#cluster-scoped",
					"#namespaced",
				},
			},
		},
		{
			ExpectedQuery: `protoPayload.resourceName:"namespaces/"`,
			Input: &queryutil.SetFilterParseResult{
				Additives: []string{
					"#namespaced",
				},
			},
		},
		{
			ExpectedQuery: `(protoPayload.resourceName:("/namespaces/kube-system" OR "/namespaces/default") OR NOT (protoPayload.resourceName:"/namespaces/"))`,
			Input: &queryutil.SetFilterParseResult{
				Additives: []string{
					"#cluster-scoped",
					"kube-system",
					"default",
				},
			},
		},
		{
			ExpectedQuery: `-protoPayload.resourceName:"/namespaces/"`,
			Input: &queryutil.SetFilterParseResult{
				Additives: []string{
					"#cluster-scoped",
				},
			},
		},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("testcase-%d-%s", i, testCase.ExpectedQuery), func(t *testing.T) {
			result := generateK8sAuditNamespaceFilter(testCase.Input)
			if result != testCase.ExpectedQuery {
				t.Errorf("the result query is not valid:\nInput:\n%v\nActual:\n%s\nExpected:\n%s", testCase.Input, result, testCase.ExpectedQuery)
			}
		})
	}
}

func TestKindNameFilter(t *testing.T) {
	testCases := []struct {
		ExpectedQuery string
		Input         *queryutil.SetFilterParseResult
	}{
		{
			ExpectedQuery: `-- Failed to generate kind filter due to the validation error "test error"`,
			Input: &queryutil.SetFilterParseResult{
				ValidationError: "test error",
			},
		},
		{
			ExpectedQuery: `-- Invalid: none of the resources will be selected. Ignoreing kind filter.`,
			Input: &queryutil.SetFilterParseResult{
				Additives: []string{},
			},
		},
		{
			ExpectedQuery: `protoPayload.methodName=~"\.(pods)\."`,
			Input: &queryutil.SetFilterParseResult{
				Additives: []string{"pods"},
			},
		},
		{
			ExpectedQuery: `protoPayload.methodName=~"\.(pods|deployments)\."`,
			Input: &queryutil.SetFilterParseResult{
				Additives: []string{"pods", "deployments"},
			},
		},
		{
			ExpectedQuery: `-protoPayload.methodName=~"\.(pods|deployments)\."`,
			Input: &queryutil.SetFilterParseResult{
				Subtractives: []string{"pods", "deployments"},
				SubtractMode: true,
			},
		},
		{
			ExpectedQuery: `-- No kind filter`,
			Input: &queryutil.SetFilterParseResult{
				Subtractives: []string{},
				SubtractMode: true,
			},
		},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("testcase-%d-%s", i, testCase.ExpectedQuery), func(t *testing.T) {
			result := generateAuditKindFilter(testCase.Input)
			if result != testCase.ExpectedQuery {
				t.Errorf("the result query is not valid:\nInput:\n%v\nActual:\n%s\nExpected:\n%s", testCase.Input, result, testCase.ExpectedQuery)
			}
		})
	}
}
