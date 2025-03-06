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

package k8s_container

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
)

func TestGenerateK8sContainerQueryIsValid(t *testing.T) {
	testCases := []struct {
		Name            string
		ClusterName     string
		PodNameFilter   *queryutil.SetFilterParseResult
		NamespaceFilter *queryutil.SetFilterParseResult
	}{
		{
			Name:            "with no set filters",
			ClusterName:     "foo-cluster",
			PodNameFilter:   &queryutil.SetFilterParseResult{Additives: []string{}},
			NamespaceFilter: &queryutil.SetFilterParseResult{Additives: []string{}},
		},
		{
			Name:            "with namespace filter",
			ClusterName:     "foo-cluster",
			PodNameFilter:   &queryutil.SetFilterParseResult{Additives: []string{}},
			NamespaceFilter: &queryutil.SetFilterParseResult{Additives: []string{"kube-system"}},
		},
		{
			Name:            "with pod name filter",
			ClusterName:     "foo-cluster",
			PodNameFilter:   &queryutil.SetFilterParseResult{Additives: []string{"nginx-pod"}},
			NamespaceFilter: &queryutil.SetFilterParseResult{Additives: []string{}},
		},
		{
			Name:            "with both filters",
			ClusterName:     "foo-cluster",
			PodNameFilter:   &queryutil.SetFilterParseResult{Additives: []string{"nginx-pod"}},
			NamespaceFilter: &queryutil.SetFilterParseResult{Additives: []string{"kube-system"}},
		},
		{
			Name:            "with complex filters",
			ClusterName:     "foo-cluster",
			PodNameFilter:   &queryutil.SetFilterParseResult{Additives: []string{"nginx-pod", "apache-pod"}},
			NamespaceFilter: &queryutil.SetFilterParseResult{Additives: []string{"kube-system", "istio-system"}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			query := GenerateK8sContainerQuery(tc.ClusterName, tc.PodNameFilter, tc.NamespaceFilter)
			err := gcp_test.IsValidLogQuery(t, query)
			if err != nil {
				t.Errorf(err.Error())
			}
		})
	}
}
