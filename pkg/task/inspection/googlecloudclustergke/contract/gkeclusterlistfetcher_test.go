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

package googlecloudclustergke_contract

import (
	"testing"

	"cloud.google.com/go/container/apiv1/containerpb"
	"github.com/google/go-cmp/cmp"
)

func TestApiResponseToClusterNameList(t *testing.T) {
	testCases := []struct {
		desc     string
		response *containerpb.ListClustersResponse
		want     []string
	}{
		{
			desc:     "empty response",
			response: &containerpb.ListClustersResponse{},
			want:     []string{},
		},
		{
			desc: "standard response",
			response: &containerpb.ListClustersResponse{
				Clusters: []*containerpb.Cluster{
					{Name: "foo"},
					{Name: "bar"},
				},
			},
			want: []string{"foo", "bar"},
		},
		{
			desc:     "nil response",
			response: nil,
			want:     []string{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got := apiResponseToClusterNameList(tc.response)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("apiResponseToClusterNameList() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
