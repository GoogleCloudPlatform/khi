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

package googlecloudlogmulticloudapiaudit_contract

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/google/go-cmp/cmp"
)

func TestExtractMulticloudAPIAuditResource(t *testing.T) {
	testCases := []struct {
		desc    string
		input   string
		want    MulticloudAPIAuditResourceFieldSet
		wantErr bool
	}{
		{
			desc: "with all parameters",
			input: `protoPayload:
  resourceName: projects/123456/locations/asia-southeast1/awsClusters/cluster-foo/awsNodePools/nodepool-bar`,
			want: MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "cluster-foo",
				NodepoolName: "nodepool-bar",
				ClusterType:  ClusterTypeAWS,
			},
		},
		{
			desc: "resourceName for cluster",
			input: `protoPayload: 
  resourceName: projects/123456/locations/asia-southeast1/azureClusters/cluster-foo`,
			want: MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "cluster-foo",
				NodepoolName: "",
				ClusterType:  ClusterTypeAzure,
			},
		},
		{
			desc: "cluster name and nodepool name are missing",
			input: `protoPayload: 
  resourceName: projects/123456/locations/asia-southeast1`,
			want: MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "unknown",
				NodepoolName: "",
				ClusterType:  ClusterTypeUnknown,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			node, err := structured.FromYAML(testCase.input)
			if err != nil {
				t.Fatalf("failed to parse test YAML data: %v", err)
			}
			nodeReader := structured.NewNodeReader(node)
			got, err := ExtractMulticloudAPIAuditResource(nodeReader)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("ExtractMulticloudAPIAuditResource() error = %v, wantErr %v", err, testCase.wantErr)
			}
			if diff := cmp.Diff(testCase.want, got); diff != "" {
				t.Errorf("ExtractMulticloudAPIAuditResource mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("mock node returns mock values", func(t *testing.T) {
		mockFS := MulticloudAPIAuditResourceFieldSet{
			ClusterName:  "mock-cluster",
			NodepoolName: "mock-nodepool",
			ClusterType:  ClusterTypeAWS,
		}
		reader := structured.NewNodeReader(structured.NewMockNode(mockFS))
		got, err := ExtractMulticloudAPIAuditResource(reader)
		if err != nil {
			t.Fatalf("ExtractMulticloudAPIAuditResource() error = %v", err)
		}
		if diff := cmp.Diff(mockFS, got); diff != "" {
			t.Errorf("ExtractMulticloudAPIAuditResource mismatch (-want +got):\n%s", diff)
		}
	})
}
