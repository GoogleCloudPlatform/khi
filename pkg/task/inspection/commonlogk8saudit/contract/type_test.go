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

package commonlogk8saudit_contract

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestResourceIdentity_SummaryTag(t *testing.T) {
	testCases := []struct {
		desc     string
		identity *ResourceIdentity
		want     string
	}{
		{
			desc: "namespaced resource",
			identity: &ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "my-pod",
			},
			want: "【my-pod (Namespace: default, APIVersion: core/v1, Kind: pod)】",
		},
		{
			desc: "cluster-scoped resource with empty namespace",
			identity: &ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "node",
				Namespace:  "",
				Name:       "node-1",
			},
			want: "【node-1 (APIVersion: core/v1, Kind: node)】",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got := tc.identity.SummaryTag()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("SummaryTag() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
