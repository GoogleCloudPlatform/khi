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

package googlecloudloggkeapiaudit_impl

import (
	"testing"

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

func TestEmptyInitialResourceStateProviderTask(t *testing.T) {
	testCases := []struct {
		name         string
		clusterName  string
		nodePoolName string
		wantCluster  bool
		wantNodePool bool
	}{
		{
			name:         "returns false for cluster and nodepool",
			clusterName:  "test-cluster",
			nodePoolName: "test-pool",
			wantCluster:  false,
			wantNodePool: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			provider, _, err := inspectiontest.RunInspectionTask(ctx, EmptyInitialResourceStateProviderTask, inspectioncore.TaskModeRun, map[string]any{})
			if err != nil {
				t.Fatalf("unexpected error running task: %v", err)
			}

			_, gotCluster := provider.ClusterInitialState(tc.clusterName)
			if gotCluster != tc.wantCluster {
				t.Errorf("ClusterInitialState() found = %v, want %v", gotCluster, tc.wantCluster)
			}

			_, gotNodePool := provider.NodePoolInitialState(tc.clusterName, tc.nodePoolName)
			if gotNodePool != tc.wantNodePool {
				t.Errorf("NodePoolInitialState() found = %v, want %v", gotNodePool, tc.wantNodePool)
			}
		})
	}
}
