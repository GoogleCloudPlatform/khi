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

package googlecloudlogcsm_impl

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

func TestCSMClusterIdentifierTask(t *testing.T) {
	tests := []struct {
		name      string
		inventory googlecloudk8scommon_contract.NEGToBackendServiceMap
		want      []string
	}{
		{
			name: "single backend",
			inventory: googlecloudk8scommon_contract.NEGToBackendServiceMap{
				"neg1": "gsmrsvd-cluster1-neg1",
			},
			want: []string{"cluster1"},
		},
		{
			name: "multiple backends same cluster",
			inventory: googlecloudk8scommon_contract.NEGToBackendServiceMap{
				"neg1": "gsmrsvd-cluster1-neg1",
				"neg2": "gsmrsvd-cluster1-neg2",
			},
			want: []string{"cluster1"},
		},
		{
			name: "multiple clusters",
			inventory: googlecloudk8scommon_contract.NEGToBackendServiceMap{
				"neg1": "gsmrsvd-cluster1-neg1",
				"neg2": "gsmrsvd-cluster2-neg2",
			},
			want: []string{"cluster1", "cluster2"},
		},
		{
			name: "no gsmrsvd backends",
			inventory: googlecloudk8scommon_contract.NEGToBackendServiceMap{
				"neg1": "other-backend",
			},
			want: []string{},
		},
		{
			name: "malformed name",
			inventory: googlecloudk8scommon_contract.NEGToBackendServiceMap{
				"neg1": "gsmrsvd-clusteronly",
			},
			want: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := tasktest.RunTask(ctx, CSMClusterIdentifierTask,
				tasktest.NewTaskDependencyValuePair(googlecloudk8scommon_contract.NEGToBackendServiceInventoryTaskID.Ref(), tc.inventory),
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, result, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
