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
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestConvertTemporalAssetsToGKEResourceSnapshots(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	validAsset := &assetpb.TemporalAsset{
		Window: &assetpb.TimeWindow{
			StartTime: timestamppb.New(now),
		},
		Asset: &assetpb.Asset{
			Name:      "//container.googleapis.com/projects/test-project/locations/us-central1-a/clusters/test-cluster",
			AssetType: GKEClusterAssetType,
		},
	}

	testCases := []struct {
		name           string
		temporalAssets []*assetpb.TemporalAsset
		wantCount      int
		wantTimes      []time.Time
	}{
		{
			name: "filters out nil and asset-nil items and converts valid items",
			temporalAssets: []*assetpb.TemporalAsset{
				nil,
				{
					Window: &assetpb.TimeWindow{
						StartTime: timestamppb.New(now),
					},
					Asset: nil,
				},
				validAsset,
			},
			wantCount: 1,
			wantTimes: []time.Time{now},
		},
		{
			name: "handles nil window start time gracefully",
			temporalAssets: []*assetpb.TemporalAsset{
				{
					Window: nil,
					Asset: &assetpb.Asset{
						Name:      "//container.googleapis.com/projects/test-project/locations/us-central1-a/clusters/test-cluster",
						AssetType: GKEClusterAssetType,
					},
				},
			},
			wantCount: 1,
			wantTimes: []time.Time{time.Time{}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := convertTemporalAssetsToGKEResourceSnapshots(tc.temporalAssets)
			if len(got) != tc.wantCount {
				t.Fatalf("len(got) = %d, want %d", len(got), tc.wantCount)
			}
			for i, snapshot := range got {
				if !snapshot.StartTime.Equal(tc.wantTimes[i]) {
					t.Errorf("got[%d].StartTime = %v, want %v", i, snapshot.StartTime, tc.wantTimes[i])
				}
			}
		})
	}
}

func TestFetchGKEResourceSnapshots(t *testing.T) {
	startTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)

	cluster := googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ProjectID:   "test-project",
		ClusterName: "test-cluster",
		Location:    "us-central1-a",
	}

	clusterCandidates := clusterParentCandidates(cluster)
	expectedClusterQuery := fmt.Sprintf("name=%q OR name=%q", clusterCandidates[0], clusterCandidates[1])
	matchedClusterName := clusterCandidates[0]
	expectedNodePoolQuery := fmt.Sprintf("parentFullResourceName=%q", matchedClusterName)
	nodePoolName := matchedClusterName + "/nodePools/default-pool"

	testCases := []struct {
		name                 string
		searchResultsPerCall [][]*assetpb.ResourceSearchResult
		searchErr            error
		searchErrPerCall     []error
		batchAssets          []*assetpb.TemporalAsset
		batchErr             error
		wantCount            int
		wantErr              bool
		wantSearchQueries    []string
		wantBatchAssetNames  []string
	}{
		{
			name: "successfully fetches cluster and nodepool snapshots",
			searchResultsPerCall: [][]*assetpb.ResourceSearchResult{
				{
					{Name: matchedClusterName, AssetType: GKEClusterAssetType},
				},
				{
					{Name: nodePoolName, AssetType: GKENodePoolAssetType},
				},
			},
			batchAssets: []*assetpb.TemporalAsset{
				{
					Window: &assetpb.TimeWindow{StartTime: timestamppb.New(startTime)},
					Asset:  &assetpb.Asset{Name: matchedClusterName, AssetType: GKEClusterAssetType},
				},
				{
					Window: &assetpb.TimeWindow{StartTime: timestamppb.New(startTime)},
					Asset:  &assetpb.Asset{Name: nodePoolName, AssetType: GKENodePoolAssetType},
				},
			},
			wantCount:           2,
			wantErr:             false,
			wantSearchQueries:   []string{expectedClusterQuery, expectedNodePoolQuery},
			wantBatchAssetNames: []string{matchedClusterName, nodePoolName},
		},
		{
			name:                 "returns empty when cluster is not found",
			searchResultsPerCall: [][]*assetpb.ResourceSearchResult{{}},
			wantCount:            0,
			wantErr:              false,
			wantSearchQueries:    []string{expectedClusterQuery},
			wantBatchAssetNames:  nil,
		},
		{
			name:              "returns error when cluster search fails",
			searchErr:         errors.New("cluster search failed"),
			wantErr:           true,
			wantSearchQueries: []string{expectedClusterQuery},
		},
		{
			name: "returns error when nodepool search fails",
			searchResultsPerCall: [][]*assetpb.ResourceSearchResult{
				{
					{Name: matchedClusterName, AssetType: GKEClusterAssetType},
				},
			},
			searchErrPerCall:  []error{nil, errors.New("nodepool search failed")},
			wantErr:           true,
			wantSearchQueries: []string{expectedClusterQuery, expectedNodePoolQuery},
		},
		{
			name: "returns error when batch get assets history fails",
			searchResultsPerCall: [][]*assetpb.ResourceSearchResult{
				{
					{Name: matchedClusterName, AssetType: GKEClusterAssetType},
				},
				{
					{Name: nodePoolName, AssetType: GKENodePoolAssetType},
				},
			},
			batchErr:            errors.New("batch get failed"),
			wantErr:             true,
			wantSearchQueries:   []string{expectedClusterQuery, expectedNodePoolQuery},
			wantBatchAssetNames: []string{matchedClusterName, nodePoolName},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fetcher := &mockCAIFetcher{
				searchResultsPerCall: tc.searchResultsPerCall,
				searchErr:            tc.searchErr,
				searchErrPerCall:     tc.searchErrPerCall,
				batchAssets:          tc.batchAssets,
				batchErr:             tc.batchErr,
			}
			progress := &inspectionmetadata.TaskProgressMetadata{}

			got, err := fetchGKEResourceSnapshots(context.Background(), fetcher, cluster, startTime, endTime, progress)
			if (err != nil) != tc.wantErr {
				t.Fatalf("fetchGKEResourceSnapshots() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr {
				if len(got) != tc.wantCount {
					t.Errorf("len(got) = %d, want %d", len(got), tc.wantCount)
				}
			}

			if diff := cmp.Diff(tc.wantSearchQueries, fetcher.recordedSearchQueries()); diff != "" {
				t.Errorf("search queries mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantBatchAssetNames, fetcher.gotBatchAssetNames, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("batch asset names mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGKEResourceFetcherTask(t *testing.T) {
	startTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)

	completeCluster := googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ProjectID:   "test-project",
		ClusterName: "test-cluster",
		Location:    "us-central1-a",
	}

	incompleteCluster := googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ProjectID:   "test-project",
		ClusterName: "",
		Location:    "us-central1-a",
	}

	clusterCandidates := clusterParentCandidates(completeCluster)
	clusterQuery := fmt.Sprintf("name=%q OR name=%q", clusterCandidates[0], clusterCandidates[1])
	clusterAssetName := clusterCandidates[0]
	nodePoolQuery := fmt.Sprintf("parentFullResourceName=%q", clusterAssetName)
	nodePoolAssetName := clusterAssetName + "/nodePools/default-pool"

	clusterData, _ := structpb.NewStruct(map[string]any{
		"name": "test-cluster",
	})
	nodePoolData, _ := structpb.NewStruct(map[string]any{
		"name": "default-pool",
	})

	testCases := []struct {
		name                 string
		taskMode             inspectioncore_contract.InspectionTaskModeType
		cluster              googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		searchErr            error
		wantCount            int
		wantSearchQueries    []string
		wantSearchAssetTypes [][]string
	}{
		{
			name:      "returns empty on DryRun mode",
			taskMode:  inspectioncore_contract.TaskModeDryRun,
			cluster:   completeCluster,
			wantCount: 0,
		},
		{
			name:      "returns empty when cluster identity is incomplete",
			taskMode:  inspectioncore_contract.TaskModeRun,
			cluster:   incompleteCluster,
			wantCount: 0,
		},
		{
			name:      "fetches cluster and nodepool temporal assets successfully",
			taskMode:  inspectioncore_contract.TaskModeRun,
			cluster:   completeCluster,
			wantCount: 2,
			wantSearchQueries: []string{
				clusterQuery,
				nodePoolQuery,
			},
			wantSearchAssetTypes: [][]string{
				{GKEClusterAssetType},
				{GKENodePoolAssetType},
			},
		},
		{
			name:      "returns empty without error when CAI search fails",
			taskMode:  inspectioncore_contract.TaskModeRun,
			cluster:   completeCluster,
			searchErr: errors.New("permission denied"),
			wantCount: 0,
			wantSearchQueries: []string{
				clusterQuery,
			},
			wantSearchAssetTypes: [][]string{
				{GKEClusterAssetType},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := &mockAssetServer{
				searchResultsByQuery: map[string][]*assetpb.ResourceSearchResult{
					clusterQuery: {
						{Name: clusterAssetName, AssetType: GKEClusterAssetType},
					},
					nodePoolQuery: {
						{Name: nodePoolAssetName, AssetType: GKENodePoolAssetType},
					},
				},
				searchErr: tc.searchErr,
				batchAssets: []*assetpb.TemporalAsset{
					{
						Window: &assetpb.TimeWindow{StartTime: timestamppb.New(startTime)},
						Asset: &assetpb.Asset{
							Name:      clusterAssetName,
							AssetType: GKEClusterAssetType,
							Resource:  &assetpb.Resource{Data: clusterData},
						},
					},
					{
						Window: &assetpb.TimeWindow{StartTime: timestamppb.New(startTime)},
						Asset: &assetpb.Asset{
							Name:      nodePoolAssetName,
							AssetType: GKENodePoolAssetType,
							Resource:  &assetpb.Resource{Data: nodePoolData},
						},
					},
				},
			}
			factory := setupMockServer(t, mockServer)

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			got, _, err := inspectiontest.RunInspectionTask(ctx, GKEResourceFetcherTask, tc.taskMode, map[string]any{},
				tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputStartTimeTaskID.Ref(), startTime),
				tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.InputEndTimeTaskID.Ref(), endTime),
				tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.APIClientFactoryTaskID.Ref(), factory),
				tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref(), googlecloud.NewCallOptionInjector()),
				tasktest.NewTaskDependencyValuePair(googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(), tc.cluster),
			)
			if err != nil {
				t.Fatalf("GKEResourceFetcherTask unexpected error: %v", err)
			}

			if len(got) != tc.wantCount {
				t.Errorf("len(got) = %d, want %d", len(got), tc.wantCount)
			}

			var gotQueries []string
			var gotAssetTypes [][]string
			for _, searchRequest := range mockServer.recordedSearchRequests() {
				gotQueries = append(gotQueries, searchRequest.Query)
				gotAssetTypes = append(gotAssetTypes, searchRequest.AssetTypes)
				if searchRequest.Scope != "projects/test-project" {
					t.Errorf("search scope = %s, want projects/test-project", searchRequest.Scope)
				}
			}
			if diff := cmp.Diff(tc.wantSearchQueries, gotQueries, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("search queries mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantSearchAssetTypes, gotAssetTypes, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("search asset types mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
