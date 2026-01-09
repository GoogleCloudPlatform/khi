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

package googlecloudk8scommon_impl

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFilterAndTrimPrefixFromClusterNames(t *testing.T) {
	tests := []struct {
		name         string
		clusterNames []string
		prefix       string
		expected     []string
	}{
		{
			name:         "basic",
			clusterNames: []string{"awsClusters/cluster1", "cluster2", "awsClusters/cluster3"},
			prefix:       "awsClusters/",
			expected:     []string{"cluster1", "cluster3"},
		},
		{
			name:         "no match",
			clusterNames: []string{"cluster1", "cluster2", "cluster3"},
			prefix:       "awsClusters/",
			expected:     []string{},
		},
		{
			name:         "empty prefix(GKE)",
			clusterNames: []string{"cluster1", "awsClusters/cluster2", "cluster3"},
			prefix:       "",
			expected:     []string{"cluster1", "cluster3"},
		},
		{
			name:         "empty cluster names",
			clusterNames: []string{},
			prefix:       "awsClusters/",
			expected:     []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterAndTrimPrefixFromClusterNames(tt.clusterNames, tt.prefix)
			if diff := cmp.Diff(tt.expected, got); diff != "" {
				t.Errorf("filterAndTrimPrefixFromClusterNames() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
