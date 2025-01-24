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

package k8s_node

import (
	"testing"

	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
	"github.com/google/go-cmp/cmp"
)

func TestGenerateK8sNodeQueryIsValid(t *testing.T) {
	testCases := []struct {
		Name               string
		ClusterName        string
		ProjectName        string
		NodeNameSubstrings []string
	}{
		{
			Name:               "Valid query with empty node name substring",
			ClusterName:        "test-cluster",
			ProjectName:        "test-project",
			NodeNameSubstrings: []string{},
		},
		{
			Name:               "Valid query with single node name substring",
			ClusterName:        "test-cluster",
			ProjectName:        "test-project",
			NodeNameSubstrings: []string{"node-1"},
		},
		{
			Name:               "Valid query with multiple node name substrings",
			ClusterName:        "test-cluster",
			ProjectName:        "test-project",
			NodeNameSubstrings: []string{"node-1", "node-2", "node-3"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			query := GenerateK8sNodeLogQuery(tc.ProjectName, tc.ClusterName, tc.NodeNameSubstrings)
			err := gcp_test.IsValidLogQuery(query)
			if err != nil {
				t.Errorf(err.Error())
			}
		})
	}
}

func TestGenerateNodeNameSubstringLogFilter(t *testing.T) {
	tests := []struct {
		name               string
		nodeNameSubstrings []string
		want               string
	}{
		{
			name:               "empty",
			nodeNameSubstrings: []string{},
			want:               "-- No node name substring filters are specified.",
		},
		{
			name:               "single",
			nodeNameSubstrings: []string{"substring1"},
			want:               "resource.labels.node_name:(\"substring1\")",
		},
		{
			name:               "multiple",
			nodeNameSubstrings: []string{"substring1", "substring2", "substring3"},
			want:               "resource.labels.node_name:(\"substring1\" OR \"substring2\" OR \"substring3\")",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateNodeNameSubstringLogFilter(tt.nodeNameSubstrings)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("generateNodeNameSubstringLogFilter() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
