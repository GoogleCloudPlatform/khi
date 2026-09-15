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
	"strings"
	"testing"
	"time"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/caik8s"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	pathNameField = structured.CompileFieldPath("name")
)

type gkeSnapshotParams struct {
	assetName       string
	windowStartTime time.Time
	windowEndTime   time.Time
	isDeleted       bool
	resourceData    map[string]any
}

func newTestGKESnapshot(t *testing.T, params gkeSnapshotParams) *caik8s.GKEResourceSnapshot {
	t.Helper()
	var window *assetpb.TimeWindow
	if !params.windowStartTime.IsZero() || !params.windowEndTime.IsZero() {
		window = &assetpb.TimeWindow{}
		if !params.windowStartTime.IsZero() {
			window.StartTime = timestamppb.New(params.windowStartTime)
		}
		if !params.windowEndTime.IsZero() {
			window.EndTime = timestamppb.New(params.windowEndTime)
		}
	}

	var resourceData *structpb.Struct
	if params.resourceData != nil {
		data, err := structpb.NewStruct(params.resourceData)
		if err != nil {
			t.Fatalf("failed to create structpb: %v", err)
		}
		resourceData = data
	}

	assetType := caik8s.GKEClusterAssetType
	if strings.Contains(params.assetName, "/nodePools/") {
		assetType = caik8s.GKENodePoolAssetType
	}

	return &caik8s.GKEResourceSnapshot{
		TemporalAsset: &assetpb.TemporalAsset{
			Window:  window,
			Deleted: params.isDeleted,
			Asset: &assetpb.Asset{
				Name:      params.assetName,
				AssetType: assetType,
				Resource: &assetpb.Resource{
					Data: resourceData,
				},
			},
		},
	}
}

func TestCAIGKEInitialResourceStateProvider(t *testing.T) {
	queryStartTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	clusterAssetName := "//container.googleapis.com/projects/p/locations/us-central1-a/clusters/test-cluster"
	nodePoolAssetName := "//container.googleapis.com/projects/p/locations/us-central1-a/clusters/test-cluster/nodePools/pool-1"

	testCases := []struct {
		name          string
		snapshots     []gkeSnapshotParams
		queryCluster  string
		queryNodePool string
		wantCluster   bool
		wantNodePool  bool
	}{
		{
			name: "active cluster and nodepool at queryStartTime",
			snapshots: []gkeSnapshotParams{
				{
					assetName:       clusterAssetName,
					windowStartTime: queryStartTime.Add(-2 * time.Hour),
					windowEndTime:   queryStartTime.Add(2 * time.Hour),
					resourceData: map[string]any{
						"name":             "test-cluster",
						"initialNodeCount": 1,
					},
				},
				{
					assetName:       nodePoolAssetName,
					windowStartTime: queryStartTime.Add(-time.Hour),
					windowEndTime:   queryStartTime.Add(2 * time.Hour),
					resourceData: map[string]any{
						"name":             "pool-1",
						"initialNodeCount": 2,
					},
				},
			},
			queryCluster:  "test-cluster",
			queryNodePool: "pool-1",
			wantCluster:   true,
			wantNodePool:  true,
		},
		{
			name: "ignores snapshots that ended before queryStartTime",
			snapshots: []gkeSnapshotParams{
				{
					assetName:       clusterAssetName,
					windowStartTime: queryStartTime.Add(-3 * time.Hour),
					windowEndTime:   queryStartTime.Add(-time.Hour),
				},
				{
					assetName:       nodePoolAssetName,
					windowStartTime: queryStartTime.Add(-3 * time.Hour),
					windowEndTime:   queryStartTime.Add(-time.Hour),
				},
			},
			queryCluster:  "test-cluster",
			queryNodePool: "pool-1",
			wantCluster:   false,
			wantNodePool:  false,
		},
		{
			name: "ignores snapshots that started after queryStartTime",
			snapshots: []gkeSnapshotParams{
				{
					assetName:       clusterAssetName,
					windowStartTime: queryStartTime.Add(time.Hour),
					windowEndTime:   queryStartTime.Add(2 * time.Hour),
				},
				{
					assetName:       nodePoolAssetName,
					windowStartTime: queryStartTime.Add(time.Hour),
					windowEndTime:   queryStartTime.Add(2 * time.Hour),
				},
			},
			queryCluster:  "test-cluster",
			queryNodePool: "pool-1",
			wantCluster:   false,
			wantNodePool:  false,
		},
		{
			name: "ignores deleted snapshots",
			snapshots: []gkeSnapshotParams{
				{
					assetName:       clusterAssetName,
					windowStartTime: queryStartTime.Add(-time.Hour),
					windowEndTime:   queryStartTime.Add(time.Hour),
					isDeleted:       true,
				},
				{
					assetName:       nodePoolAssetName,
					windowStartTime: queryStartTime.Add(-time.Hour),
					windowEndTime:   queryStartTime.Add(time.Hour),
					isDeleted:       true,
				},
			},
			queryCluster:  "test-cluster",
			queryNodePool: "pool-1",
			wantCluster:   false,
			wantNodePool:  false,
		},
		{
			name: "ignores snapshots without resource data",
			snapshots: []gkeSnapshotParams{
				{
					assetName:       clusterAssetName,
					windowStartTime: queryStartTime.Add(-time.Hour),
					windowEndTime:   queryStartTime.Add(time.Hour),
					resourceData:    nil,
				},
				{
					assetName:       nodePoolAssetName,
					windowStartTime: queryStartTime.Add(-time.Hour),
					windowEndTime:   queryStartTime.Add(time.Hour),
					resourceData:    nil,
				},
			},
			queryCluster:  "test-cluster",
			queryNodePool: "pool-1",
			wantCluster:   false,
			wantNodePool:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			snapshots := make([]*caik8s.GKEResourceSnapshot, 0, len(tc.snapshots))
			for _, p := range tc.snapshots {
				snapshots = append(snapshots, newTestGKESnapshot(t, p))
			}

			provider := newCAIGKEInitialResourceStateProvider(snapshots, queryStartTime)

			clusterState, gotCluster := provider.ClusterInitialState(tc.queryCluster)
			if gotCluster != tc.wantCluster {
				t.Errorf("ClusterInitialState(%q) got = %v, want %v", tc.queryCluster, gotCluster, tc.wantCluster)
			}
			if tc.wantCluster {
				if !gotCluster {
					t.Fatalf("ClusterInitialState(%q) got = false, want true", tc.queryCluster)
				}
				if clusterState == nil || clusterState.ResourceBody == nil {
					t.Fatalf("ClusterInitialState(%q) returned nil state or nil ResourceBody", tc.queryCluster)
				}
				reader := structured.NewNodeReader(clusterState.ResourceBody)
				name, err := reader.ReadString(pathNameField)
				if err != nil || name != tc.queryCluster {
					t.Errorf("clusterState.ResourceBody name = %q, want %q, err = %v", name, tc.queryCluster, err)
				}
			}

			nodePoolState, gotNodePool := provider.NodePoolInitialState(tc.queryCluster, tc.queryNodePool)
			if gotNodePool != tc.wantNodePool {
				t.Errorf("NodePoolInitialState(%q, %q) got = %v, want %v", tc.queryCluster, tc.queryNodePool, gotNodePool, tc.wantNodePool)
			}
			if tc.wantNodePool {
				if !gotNodePool {
					t.Fatalf("NodePoolInitialState(%q, %q) got = false, want true", tc.queryCluster, tc.queryNodePool)
				}
				if nodePoolState == nil || nodePoolState.ResourceBody == nil {
					t.Fatalf("NodePoolInitialState(%q, %q) returned nil state or nil ResourceBody", tc.queryCluster, tc.queryNodePool)
				}
				reader := structured.NewNodeReader(nodePoolState.ResourceBody)
				name, err := reader.ReadString(pathNameField)
				if err != nil || name != tc.queryNodePool {
					t.Errorf("nodePoolState.ResourceBody name = %q, want %q, err = %v", name, tc.queryNodePool, err)
				}
			}
		})
	}
}

func TestGKEInitialResourceStateProviderTask(t *testing.T) {
	queryStartTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	clusterAssetName := "//container.googleapis.com/projects/p/locations/us-central1-a/clusters/test-cluster"

	testCases := []struct {
		name        string
		mode        inspectioncore.InspectionTaskModeType
		snapshots   []*caik8s.GKEResourceSnapshot
		wantCluster bool
	}{
		{
			name:        "DryRun mode returns empty provider",
			mode:        inspectioncore.TaskModeDryRun,
			snapshots:   nil,
			wantCluster: false,
		},
		{
			name: "Run mode resolves snapshots and returns populated provider",
			mode: inspectioncore.TaskModeRun,
			snapshots: []*caik8s.GKEResourceSnapshot{
				newTestGKESnapshot(t, gkeSnapshotParams{
					assetName:       clusterAssetName,
					windowStartTime: queryStartTime.Add(-time.Hour),
					windowEndTime:   queryStartTime.Add(time.Hour),
					resourceData: map[string]any{
						"name": "test-cluster",
					},
				}),
			},
			wantCluster: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			provider, _, err := inspectiontest.RunInspectionTask(ctx, GKEInitialResourceStateProviderTask, tc.mode, map[string]any{},
				tasktest.NewTaskDependencyValuePair(caik8s.GKEResourceFetcherTaskID.Ref(), tc.snapshots),
				tasktest.NewTaskDependencyValuePair(gcpcommon.InputStartTimeTaskID.Ref(), queryStartTime),
			)
			if err != nil {
				t.Fatalf("unexpected error running task: %v", err)
			}

			_, gotCluster := provider.ClusterInitialState("test-cluster")
			if gotCluster != tc.wantCluster {
				t.Errorf("ClusterInitialState() got = %v, want %v", gotCluster, tc.wantCluster)
			}
		})
	}
}
