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

package googlecloudlogonpremapiaudit_contract

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/google/go-cmp/cmp"
)

func TestOnPremAPIAuditResourceFieldSetReader(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  *OnPremAPIAuditResourceFieldSet
	}{
		{
			desc: "with all parameters",
			input: `resource:
  labels:
    project_id: "123456"
protoPayload:
  resourceName: projects/123456/locations/asia-southeast1/baremetalAdminClusters/cluster-foo/baremetalAdminNodepools/nodepool-bar`,
			want: &OnPremAPIAuditResourceFieldSet{
				Project:      "123456",
				ClusterName:  "cluster-foo",
				NodepoolName: "nodepool-bar",
				ClusterType:  ClusterTypeBaremetalAdmin,
			},
		},
		{
			desc: "resourceName for cluster",
			input: `resource:
  labels:
    project_id: "123456"
protoPayload: 
  resourceName: projects/123456/locations/asia-southeast1/baremetalStandaloneClusters/cluster-foo`,
			want: &OnPremAPIAuditResourceFieldSet{
				Project:      "123456",
				ClusterName:  "cluster-foo",
				NodepoolName: "",
				ClusterType:  ClusterTypeBaremetalStandalone,
			},
		},
		{
			desc: "cluster name and nodepool name are missing",
			input: `resource:
  labels:
    project_id: "123456"
protoPayload: 
  resourceName: projects/123456/locations/asia-southeast1`,
			want: &OnPremAPIAuditResourceFieldSet{
				Project:      "123456",
				ClusterName:  "unknown",
				NodepoolName: "",
				ClusterType:  ClusterTypeUnknown,
			},
		},
		{
			desc: "with project_id in resource labels",
			input: `resource:
  labels:
    project_id: "my-project-from-labels"
protoPayload:
  resourceName: projects/123456/locations/asia-southeast1/baremetalAdminClusters/cluster-foo/baremetalAdminNodepools/nodepool-bar`,
			want: &OnPremAPIAuditResourceFieldSet{
				Project:      "my-project-from-labels",
				ClusterName:  "cluster-foo",
				NodepoolName: "nodepool-bar",
				ClusterType:  ClusterTypeBaremetalAdmin,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			l, err := log.NewLogFromYAMLString(testCase.input)
			if err != nil {
				t.Errorf("failed to parse test YAML data: %v", err)
			}

			err = l.SetFieldSetReader(&OnPremAPIAuditResourceFieldSetReader{})
			if err != nil {
				t.Fatalf("OnPremAPIAuditResourceFieldSetReader returned an unexpected error:%v", err)
			}
			fieldSet := log.MustGetFieldSet(l, &OnPremAPIAuditResourceFieldSet{})
			if diff := cmp.Diff(testCase.want, fieldSet); diff != "" {
				t.Errorf("OnPremAPIAuditResourceFieldSetReader mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
