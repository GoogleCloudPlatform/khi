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

package multicloud_api

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestParseResourceNameOfMulticloudAPI(t *testing.T) {
	testCases := []struct {
		Input    string
		Expected *multiCloudResource
	}{
		{
			Input: "projects/123456/locations/asia-southeast1/awsClusters/cluster-foo/awsNodePools/nodepool-bar",
			Expected: &multiCloudResource{
				ClusterName:  "cluster-foo",
				NodepoolName: "nodepool-bar",
				ClusterType:  "aws",
			},
		},
		{
			Input: "projects/123456/locations/asia-southeast1/azureClusters/cluster-foo",
			Expected: &multiCloudResource{
				ClusterName:  "cluster-foo",
				NodepoolName: "",
				ClusterType:  "azure",
			},
		},
		{
			Input: "projects/123456/locations/asia-southeast1",
			Expected: &multiCloudResource{
				ClusterName:  "unknown",
				NodepoolName: "",
				ClusterType:  "unknown",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Input, func(t *testing.T) {
			actual := parseResourceNameOfMulticloudAPI(testCase.Input)
			if diff := cmp.Diff(testCase.Expected, actual); diff != "" {
				t.Errorf("The generated result is not matching with the expected\n%s", diff)
			}
		})
	}
}

func TestFilterMethodNameOperation(t *testing.T) {
	testCases := []struct {
		Input     string
		Operation string
		Operand   string
		Expected  bool
	}{
		{
			Input:     "google.cloud.gkemulticloud.v1.AwsClusters.CreateAwsNodePool",
			Operation: "Create",
			Operand:   "NodePool",
			Expected:  true,
		},
		{
			Input:     "google.cloud.gkemulticloud.v1.AwsClusters.CreateAwsNodePool",
			Operation: "Delete",
			Operand:   "NodePool",
			Expected:  false,
		},
		{
			Input:     "google.cloud.gkemulticloud.v1.AwsClusters.CreateAzureNodePool",
			Operation: "Create",
			Operand:   "Cluster",
			Expected:  false,
		},
		{
			Input:     "google.cloud.gkemulticloud.v1.AwsClusters.CreateAzureNodePool",
			Operation: "Create",
			Operand:   "NodePool",
			Expected:  true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.Input, func(t *testing.T) {
			actual := filterMethodNameOperation(testCase.Input, testCase.Operation, testCase.Operand)
			if diff := cmp.Diff(testCase.Expected, actual); diff != "" {
				t.Errorf("The generated result is not matching with the expected\n%s", diff)
			}
		})
	}
}
