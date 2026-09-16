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

package caik8s_impl

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/google/go-cmp/cmp"
)

func TestParseGKEAssetName(t *testing.T) {
	testCases := []struct {
		name           string
		assetName      string
		want           gkeResourceIdentity
		wantIsNodePool bool
		wantIsCluster  bool
	}{
		{
			name:      "regional cluster",
			assetName: "//container.googleapis.com/projects/test-project/locations/us-central1/clusters/test-cluster",
			want: gkeResourceIdentity{
				ClusterName:  "test-cluster",
				NodePoolName: "",
			},
			wantIsNodePool: false,
			wantIsCluster:  true,
		},
		{
			name:      "zonal cluster",
			assetName: "//container.googleapis.com/projects/test-project/zones/us-central1-a/clusters/test-cluster",
			want: gkeResourceIdentity{
				ClusterName:  "test-cluster",
				NodePoolName: "",
			},
			wantIsNodePool: false,
			wantIsCluster:  true,
		},
		{
			name:      "regional nodepool",
			assetName: "//container.googleapis.com/projects/test-project/locations/us-central1/clusters/test-cluster/nodePools/default-pool",
			want: gkeResourceIdentity{
				ClusterName:  "test-cluster",
				NodePoolName: "default-pool",
			},
			wantIsNodePool: true,
			wantIsCluster:  false,
		},
		{
			name:      "zonal nodepool",
			assetName: "//container.googleapis.com/projects/test-project/zones/us-central1-a/clusters/test-cluster/nodePools/default-pool",
			want: gkeResourceIdentity{
				ClusterName:  "test-cluster",
				NodePoolName: "default-pool",
			},
			wantIsNodePool: true,
			wantIsCluster:  false,
		},
		{
			name:      "non-gke asset name",
			assetName: "//compute.googleapis.com/projects/test-project/zones/us-central1-a/instances/test-instance",
			want: gkeResourceIdentity{
				ClusterName:  "",
				NodePoolName: "",
			},
			wantIsNodePool: false,
			wantIsCluster:  false,
		},
		{
			name:      "empty asset name",
			assetName: "",
			want: gkeResourceIdentity{
				ClusterName:  "",
				NodePoolName: "",
			},
			wantIsNodePool: false,
			wantIsCluster:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseGKEAssetName(tc.assetName)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("parseGKEAssetName(%q) mismatch (-want +got):\n%s", tc.assetName, diff)
			}
			if got.IsNodePool() != tc.wantIsNodePool {
				t.Errorf("parseGKEAssetName(%q).IsNodePool() = %v, want %v", tc.assetName, got.IsNodePool(), tc.wantIsNodePool)
			}
			if got.IsCluster() != tc.wantIsCluster {
				t.Errorf("parseGKEAssetName(%q).IsCluster() = %v, want %v", tc.assetName, got.IsCluster(), tc.wantIsCluster)
			}
		})
	}
}

func TestExtractGKEIdentity(t *testing.T) {
	testCases := []struct {
		name      string
		inputData map[string]any
		want      gkeResourceIdentity
		wantOK    bool
	}{
		{
			name: "cluster asset",
			inputData: map[string]any{
				"asset": map[string]any{
					"name": "//container.googleapis.com/projects/p/locations/l/clusters/test-cluster",
				},
			},
			want: gkeResourceIdentity{
				ClusterName:  "test-cluster",
				NodePoolName: "",
			},
			wantOK: true,
		},
		{
			name: "node pool asset",
			inputData: map[string]any{
				"asset": map[string]any{
					"name": "//container.googleapis.com/projects/p/locations/l/clusters/test-cluster/nodePools/default-pool",
				},
			},
			want: gkeResourceIdentity{
				ClusterName:  "test-cluster",
				NodePoolName: "default-pool",
			},
			wantOK: true,
		},
		{
			name: "node pool without cluster returns ok false",
			inputData: map[string]any{
				"asset": map[string]any{
					"name": "//container.googleapis.com/projects/p/locations/l/nodePools/pool-1",
				},
			},
			want:   gkeResourceIdentity{},
			wantOK: false,
		},
		{
			name: "empty asset name returns ok false",
			inputData: map[string]any{
				"asset": map[string]any{
					"name": "",
				},
			},
			want:   gkeResourceIdentity{},
			wantOK: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := structured.FromGoValue(tc.inputData, &structured.AlphabeticalGoMapKeyOrderProvider{})
			if err != nil {
				t.Fatalf("failed to create structured node: %v", err)
			}
			reader := structured.NewNodeReader(node)
			got, ok := extractGKEIdentity(reader)
			if ok != tc.wantOK {
				t.Errorf("extractGKEIdentity() ok = %v, want %v", ok, tc.wantOK)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("extractGKEIdentity() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatGKEResourceLogSummary(t *testing.T) {
	testCases := []struct {
		name     string
		identity gkeResourceIdentity
		want     string
	}{
		{
			name: "cluster snapshot",
			identity: gkeResourceIdentity{
				ClusterName:  "test-cluster",
				NodePoolName: "",
			},
			want: "CAI resource snapshot: Cluster/test-cluster",
		},
		{
			name: "node pool snapshot",
			identity: gkeResourceIdentity{
				ClusterName:  "test-cluster",
				NodePoolName: "default-pool",
			},
			want: "CAI resource snapshot: NodePool/default-pool",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatGKEResourceLogSummary(tc.identity)
			if got != tc.want {
				t.Errorf("formatGKEResourceLogSummary(%+v) = %q, want %q", tc.identity, got, tc.want)
			}
		})
	}
}
