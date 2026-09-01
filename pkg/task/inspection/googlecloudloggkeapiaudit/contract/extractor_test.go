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

package googlecloudloggkeapiaudit_contract

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/google/go-cmp/cmp"
)

func TestExtractGKEAuditLogResource(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  GKEAuditLogResourceFieldSet
	}{
		{
			desc: "basic input",
			input: `
resource:
  labels:
    cluster_name: "test-cluster"
    nodepool_name: "test-nodepool"
`,
			want: GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
			},
		},
		{
			desc: "nodepool name from update field",
			input: `
resource:
  labels:
    cluster_name: "test-cluster"
protoPayload:
  request:
    update:
      desiredNodePoolId: "test-nodepool"
`,
			want: GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
			},
		},
		{
			desc:  "default input",
			input: "{}",
			want: GKEAuditLogResourceFieldSet{
				ClusterName:  "unknown",
				NodepoolName: "",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			node, err := structured.FromYAML(tc.input)
			if err != nil {
				t.Fatalf("failed to parse YAML test input: %v", err)
			}
			reader := structured.NewNodeReader(node)
			got, err := ExtractGKEAuditLogResource(reader)
			if err != nil {
				t.Fatalf("ExtractGKEAuditLogResource() error = %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExtractGKEAuditLogResource() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("mock node returns mock values", func(t *testing.T) {
		mockFS := GKEAuditLogResourceFieldSet{
			ClusterName:  "mock-cluster",
			NodepoolName: "mock-nodepool",
		}
		reader := structured.NewNodeReader(structured.NewMockNode(mockFS))
		got, err := ExtractGKEAuditLogResource(reader)
		if err != nil {
			t.Fatalf("ExtractGKEAuditLogResource() error = %v", err)
		}
		if diff := cmp.Diff(mockFS, got); diff != "" {
			t.Errorf("ExtractGKEAuditLogResource() mismatch (-want +got):\n%s", diff)
		}
	})
}
