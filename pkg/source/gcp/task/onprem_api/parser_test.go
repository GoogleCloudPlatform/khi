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

package onprem_api

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseResourceNameOfOnPremAPI(t *testing.T) {
	// Define a struct to hold test cases
	type testCase struct {
		resourceName string
		expected     *onpremResource
	}

	// Create test cases with various input scenarios
	var testCases = []testCase{
		// Valid cases
		{
			resourceName: "projects/12345/locations/asia-northeast1/baremetalClusters/my-cluster",
			expected: &onpremResource{
				ClusterName:  "my-cluster",
				NodepoolName: "",
				ClusterType:  "baremetal",
			},
		},
		{
			resourceName: "projects/67890/locations/us-central1/vmwareClusters/dev-cluster/vmwareNodePools/pool-1",
			expected: &onpremResource{
				ClusterName:  "dev-cluster",
				NodepoolName: "pool-1",
				ClusterType:  "vmware",
			},
		},
		{ // No cluster name
			resourceName: "projects/12345/locations/asia-northeast1",
			expected: &onpremResource{
				ClusterName:  "unknown",
				NodepoolName: "",
				ClusterType:  "unknown",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.resourceName, func(t *testing.T) {
			result := parseResourceNameOfOnPremAPI(tc.resourceName)

			if diff := cmp.Diff(tc.expected, result); diff != "" {
				t.Errorf("Failed for resourceName: %s\nDifference:\n%s", tc.resourceName, diff)
			}
		})
	}

}
