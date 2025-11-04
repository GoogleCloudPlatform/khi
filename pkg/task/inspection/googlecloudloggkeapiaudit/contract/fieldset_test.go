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

	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/google/go-cmp/cmp"
)

func TestGKEAuditLogResourceFieldSetReader(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  *GKEAuditLogResourceFieldSet
	}{
		{
			desc: "basic input",
			input: `
resource:
  labels:
    cluster_name: "test-cluster"
    nodepool_name: "test-nodepool"
`,
			want: &GKEAuditLogResourceFieldSet{
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
			want: &GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
			},
		},
		{
			desc:  "default input",
			input: "{}",
			want: &GKEAuditLogResourceFieldSet{
				ClusterName:  "unknown",
				NodepoolName: "",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l, err := log.NewLogFromYAMLString(tc.input)
			if err != nil {
				t.Fatalf("failed to parse YAML test input to log: %v", err)
			}
			err = l.SetFieldSetReader(&GKEAuditLogResourceFieldSetReader{})
			if err != nil {
				t.Errorf("failed to run GKEAuditLogResourceFieldSetReader.Read(): %v", err)
			}
			got := log.MustGetFieldSet(l, &GKEAuditLogResourceFieldSet{})
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GKEAuditLogResourceFieldSet mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
