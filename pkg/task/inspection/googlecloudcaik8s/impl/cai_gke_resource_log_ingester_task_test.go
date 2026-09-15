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

package googlecloudcaik8s_impl

import (
	"testing"
	"time"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcaik8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcaik8s/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func TestGKERawLogTask(t *testing.T) {
	testTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	clusterData, _ := structpb.NewStruct(map[string]any{
		"name": "test-cluster",
	})
	snapshot := &googlecloudcaik8s_contract.GKEResourceSnapshot{
		StartTime: testTime,
		TemporalAsset: &assetpb.TemporalAsset{
			Window: &assetpb.TimeWindow{
				StartTime: timestamppb.New(testTime),
			},
			Asset: &assetpb.Asset{
				Name:      "//container.googleapis.com/projects/test-project/locations/us-central1/clusters/test-cluster",
				AssetType: googlecloudcaik8s_contract.GKEClusterAssetType,
				Resource: &assetpb.Resource{
					Data: clusterData,
				},
			},
		},
	}

	testCases := []struct {
		name      string
		taskMode  inspectioncore.InspectionTaskModeType
		snapshots []*googlecloudcaik8s_contract.GKEResourceSnapshot
		wantCount int
	}{
		{
			name:      "returns empty on DryRun mode",
			taskMode:  inspectioncore.TaskModeDryRun,
			snapshots: []*googlecloudcaik8s_contract.GKEResourceSnapshot{snapshot},
			wantCount: 0,
		},
		{
			name:      "converts snapshots to raw logs",
			taskMode:  inspectioncore.TaskModeRun,
			snapshots: []*googlecloudcaik8s_contract.GKEResourceSnapshot{snapshot},
			wantCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			ctx = khictx.WithValue(ctx, inspectioncore.IDGenerator, id.NewGenerator())

			got, _, err := inspectiontest.RunInspectionTask(ctx, GKERawLogTask, tc.taskMode, map[string]any{},
				tasktest.NewTaskDependencyValuePair(googlecloudcaik8s_contract.GKEResourceFetcherTaskID.Ref(), tc.snapshots),
			)
			if err != nil {
				t.Fatalf("GKERawLogTask error: %v", err)
			}
			if len(got) != tc.wantCount {
				t.Fatalf("len(got) = %d, want %d", len(got), tc.wantCount)
			}
			if tc.wantCount > 0 {
				if !got[0].Timestamp.Equal(testTime) {
					t.Errorf("got[0].Timestamp = %v, want %v", got[0].Timestamp, testTime)
				}
				name := got[0].NodeReader.ReadStringOrDefault(pathAssetName, "")
				if name != snapshot.TemporalAsset.Asset.Name {
					t.Errorf("got asset name = %q, want %q", name, snapshot.TemporalAsset.Asset.Name)
				}
			}
		})
	}
}

func TestCAIGKEResourceLogIngester_ProcessLog(t *testing.T) {
	testTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	generator := id.NewGenerator()

	clusterLog := newLogFromMap(t, generator, testTime, map[string]any{
		"asset": map[string]any{
			"name":      "//container.googleapis.com/projects/test-project/locations/us-central1/clusters/test-cluster",
			"assetType": googlecloudcaik8s_contract.GKEClusterAssetType,
		},
	})

	nodePoolLog := newLogFromMap(t, generator, testTime, map[string]any{
		"asset": map[string]any{
			"name":      "//container.googleapis.com/projects/test-project/locations/us-central1/clusters/test-cluster/nodePools/default-pool",
			"assetType": googlecloudcaik8s_contract.GKENodePoolAssetType,
		},
	})

	unknownLog := newLogFromMap(t, generator, testTime, map[string]any{
		"asset": map[string]any{
			"name": "//unknown",
		},
	})

	testCases := []struct {
		name      string
		inputLog  *log.Log
		assertLog func(t *testing.T, cs *khifilev6.LogChangeSet)
	}{
		{
			name:     "populates metadata for cluster log",
			inputLog: clusterLog,
			assertLog: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasTimestamp(testTime).
					HasSeverity(inspectioncore.SeverityInfo).
					HasLogType(googlecloudcaik8s_contract.LogTypeCAIResourceSnapshot).
					HasSummary("CAI resource snapshot: Cluster/test-cluster")
			},
		},
		{
			name:     "populates metadata for nodepool log",
			inputLog: nodePoolLog,
			assertLog: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasTimestamp(testTime).
					HasSeverity(inspectioncore.SeverityInfo).
					HasLogType(googlecloudcaik8s_contract.LogTypeCAIResourceSnapshot).
					HasSummary("CAI resource snapshot: NodePool/default-pool")
			},
		},
		{
			name:     "populates fallback summary for unknown asset",
			inputLog: unknownLog,
			assertLog: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasTimestamp(testTime).
					HasSeverity(inspectioncore.SeverityInfo).
					HasLogType(googlecloudcaik8s_contract.LogTypeCAIResourceSnapshot).
					HasSummary("CAI resource snapshot: GKE resource")
			},
		},
	}

	ingester := &caiGKEResourceLogIngester{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cs, err := ingester.ProcessLog(t.Context(), tc.inputLog)
			if err != nil {
				t.Fatalf("ProcessLog() unexpected error: %v", err)
			}
			tc.assertLog(t, cs)
		})
	}
}

func TestGKELogGrouperTask(t *testing.T) {
	testTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	generator := id.NewGenerator()

	clusterLog := newLogFromMap(t, generator, testTime, map[string]any{
		"asset": map[string]any{
			"name": "//container.googleapis.com/projects/test-project/locations/us-central1/clusters/test-cluster",
		},
	})
	nodepoolLog := newLogFromMap(t, generator, testTime, map[string]any{
		"asset": map[string]any{
			"name": "//container.googleapis.com/projects/test-project/locations/us-central1/clusters/test-cluster/nodePools/default-pool",
		},
	})
	unknownLog := newLogFromMap(t, generator, testTime, map[string]any{})

	testCases := []struct {
		name     string
		rawLogs  []*log.Log
		wantKeys []string
	}{
		{
			name:     "groups cluster and nodepool logs correctly",
			rawLogs:  []*log.Log{clusterLog, nodepoolLog, unknownLog},
			wantKeys: []string{"cluster/test-cluster", "nodepool/test-cluster/default-pool", "unknown"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			got, _, err := inspectiontest.RunInspectionTask(ctx, GKELogGrouperTask, inspectioncore.TaskModeRun, map[string]any{},
				tasktest.NewTaskDependencyValuePair(googlecloudcaik8s_contract.GKERawLogTaskID.Ref(), tc.rawLogs),
			)
			if err != nil {
				t.Fatalf("GKELogGrouperTask error: %v", err)
			}
			for _, wantKey := range tc.wantKeys {
				if _, ok := got[wantKey]; !ok {
					t.Errorf("missing group key %q in got", wantKey)
				}
			}
			if len(got) != len(tc.wantKeys) {
				t.Errorf("len(got) = %d, want %d", len(got), len(tc.wantKeys))
			}
			logPointerComparer := cmp.Comparer(func(a, b *log.Log) bool {
				return a == b
			})
			if diff := cmp.Diff([]*log.Log{clusterLog}, got["cluster/test-cluster"].Logs, logPointerComparer); diff != "" {
				t.Errorf("group cluster/test-cluster logs mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]*log.Log{nodepoolLog}, got["nodepool/test-cluster/default-pool"].Logs, logPointerComparer); diff != "" {
				t.Errorf("group nodepool/test-cluster/default-pool logs mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]*log.Log{unknownLog}, got["unknown"].Logs, logPointerComparer); diff != "" {
				t.Errorf("group unknown logs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
