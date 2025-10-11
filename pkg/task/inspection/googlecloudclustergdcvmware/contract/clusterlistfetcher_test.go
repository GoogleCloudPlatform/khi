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

package googlecloudclustergdcvmware_contract

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestToShortClusterName(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid user cluster name",
			input: "projects/my-project/locations/us-central1/vmwareClusters/user-cluster-1",
			want:  "user-cluster-1",
		},
		{
			name:  "valid admin cluster name",
			input: "projects/my-project/locations/us-central1/vmwareAdminClusters/admin-cluster-1",
			want:  "admin-cluster-1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := toShortClusterName(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("toShortClusterName() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
