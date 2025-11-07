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

	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/google/go-cmp/cmp"
)

func TestMulticloudAPIAuditResourceFieldSetReader(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  *MulticloudAPIAuditResourceFieldSet
	}{
		{
			desc: "with all parameters",
			input: `protoPayload:
  resourceName: projects/123456/locations/asia-southeast1/awsClusters/cluster-foo/awsNodePools/nodepool-bar`,
			want: &MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "cluster-foo",
				NodepoolName: "nodepool-bar",
				ClusterType:  ClusterTypeAWS,
			},
		},
		{
			desc: "resourceName for cluster",
			input: `protoPayload: 
  resourceName: projects/123456/locations/asia-southeast1/azureClusters/cluster-foo`,
			want: &MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "cluster-foo",
				NodepoolName: "",
				ClusterType:  ClusterTypeAzure,
			},
		},
		{
			desc: "cluster name and nodepool name are missing",
			input: `protoPayload: 
  resourceName: projects/123456/locations/asia-southeast1`,
			want: &MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "unknown",
				NodepoolName: "",
				ClusterType:  ClusterTypeUnknown,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			l, err := log.NewLogFromYAMLString(testCase.input)
			if err != nil {
				t.Errorf("failed to parse test YAML data: %v", err)
			}

			err = l.SetFieldSetReader(&MulticloudAPIAuditResourceFieldSetReader{})
			if err != nil {
				t.Fatalf("MulticloudAPIAuditResourceFieldSetReader returned an unexpected error:%v", err)
			}
			fieldSet := log.MustGetFieldSet(l, &MulticloudAPIAuditResourceFieldSet{})
			if diff := cmp.Diff(testCase.want, fieldSet); diff != "" {
				t.Errorf("MulticloudAPIAuditResourceFieldSetReader mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
