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
	"unique"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcaik8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcaik8s/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestExtractClusterCreateTimeFromSnapshots(t *testing.T) {
	createTime := time.Date(2025, 5, 1, 10, 0, 0, 0, time.UTC)
	data, _ := structpb.NewStruct(map[string]any{
		"createTime": createTime.Format(time.RFC3339),
	})

	testCases := []struct {
		name      string
		snapshots []*googlecloudcaik8s_contract.GKEResourceSnapshot
		want      time.Time
	}{
		{
			name: "extracts createTime from GKECluster snapshot",
			snapshots: []*googlecloudcaik8s_contract.GKEResourceSnapshot{
				{
					TemporalAsset: &assetpb.TemporalAsset{
						Asset: &assetpb.Asset{
							AssetType: GKEClusterAssetType,
							Resource:  &assetpb.Resource{Data: data},
						},
					},
				},
			},
			want: createTime,
		},
		{
			name: "returns zero time when no cluster snapshot",
			snapshots: []*googlecloudcaik8s_contract.GKEResourceSnapshot{
				{
					TemporalAsset: &assetpb.TemporalAsset{
						Asset: &assetpb.Asset{
							AssetType: GKENodePoolAssetType,
							Resource:  &assetpb.Resource{Data: data},
						},
					},
				},
			},
			want: time.Time{},
		},
		{
			name:      "returns zero time for empty snapshots",
			snapshots: []*googlecloudcaik8s_contract.GKEResourceSnapshot{},
			want:      time.Time{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractClusterCreateTimeFromSnapshots(tc.snapshots)
			if !got.Equal(tc.want) {
				t.Errorf("extractClusterCreateTimeFromSnapshots() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCaiGKEResourceTimelineMapper_ProcessLogByGroup(t *testing.T) {
	builder := khifilev6.NewTestBuilder(id.NewGenerator())
	clusterName := "test-cluster"
	projectID := "test-project"
	queryStartTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	queryEndTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	clusterIdentity := googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ProjectID:   projectID,
		ClusterName: clusterName,
		Location:    "us-central1-a",
	}

	ctxWithBuilder := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
	projectTimeline := googlecloudcommon_contract.MustGCPProjectTimeline(ctxWithBuilder, projectID)
	clusterTimeline := googlecloudcommon_contract.MustGKEClusterTimeline(ctxWithBuilder, projectTimeline, clusterName)
	nodepoolTimeline := googlecloudcommon_contract.MustGKENodePoolTimeline(ctxWithBuilder, clusterTimeline, "default-pool")

	clusterCreateTime := time.Date(2025, 5, 1, 10, 0, 0, 0, time.UTC)
	clusterData, _ := structpb.NewStruct(map[string]any{
		"name":       clusterName,
		"createTime": clusterCreateTime.Format(time.RFC3339),
	})
	clusterSnapshot := &googlecloudcaik8s_contract.GKEResourceSnapshot{
		TemporalAsset: &assetpb.TemporalAsset{
			Asset: &assetpb.Asset{
				Name:      "//container.googleapis.com/projects/test-project/locations/us-central1-a/clusters/test-cluster",
				AssetType: GKEClusterAssetType,
				Resource:  &assetpb.Resource{Data: clusterData},
			},
		},
	}

	generator := id.NewGenerator()
	newGKECAILog := func(assetName, assetType string, windowStartTime, windowEndTime time.Time, isDeleted bool, data map[string]any) *log.Log {
		window := map[string]any{}
		if !windowStartTime.IsZero() {
			window["startTime"] = windowStartTime.Format(time.RFC3339Nano)
		}
		if !windowEndTime.IsZero() {
			window["endTime"] = windowEndTime.Format(time.RFC3339Nano)
		}
		m := map[string]any{
			"window":  window,
			"deleted": isDeleted,
			"asset": map[string]any{
				"name":      assetName,
				"assetType": assetType,
				"resource": map[string]any{
					"data": data,
				},
			},
		}
		node, err := structured.FromGoValue(m, &structured.AlphabeticalGoMapKeyOrderProvider{})
		if err != nil {
			t.Fatalf("failed to build node: %v", err)
		}
		return log.NewLogWithTimestamp(generator, structured.NewNodeReader(node), windowStartTime)
	}

	clusterAssetName := "//container.googleapis.com/projects/test-project/locations/us-central1-a/clusters/test-cluster"
	nodepoolAssetName := clusterAssetName + "/nodePools/default-pool"

	clusterInitialLog := newGKECAILog(
		clusterAssetName,
		GKEClusterAssetType,
		queryStartTime.Add(-2*time.Hour),
		time.Time{},
		false,
		map[string]any{
			"name":       clusterName,
			"createTime": clusterCreateTime.Format(time.RFC3339),
		},
	)

	clusterCreatedRecentlyLog := newGKECAILog(
		clusterAssetName,
		GKEClusterAssetType,
		queryStartTime,
		time.Time{},
		false,
		map[string]any{
			"name":       clusterName,
			"createTime": queryStartTime.Format(time.RFC3339),
		},
	)

	clusterUpdateLog := newGKECAILog(
		clusterAssetName,
		GKEClusterAssetType,
		queryStartTime.Add(30*time.Minute),
		time.Time{},
		false,
		map[string]any{
			"name": clusterName,
		},
	)

	nodepoolEarliestTime := queryStartTime.Add(-10 * time.Hour)
	nodepoolInitialLog := newGKECAILog(
		nodepoolAssetName,
		GKENodePoolAssetType,
		nodepoolEarliestTime,
		time.Time{},
		false,
		map[string]any{
			"name": "default-pool",
		},
	)

	nodepoolCreatedWithClusterLog := newGKECAILog(
		nodepoolAssetName,
		GKENodePoolAssetType,
		clusterCreateTime.Add(200*time.Millisecond),
		time.Time{},
		false,
		map[string]any{
			"name": "default-pool",
		},
	)

	nodepoolUpdateLog := newGKECAILog(
		nodepoolAssetName,
		GKENodePoolAssetType,
		queryStartTime.Add(45*time.Minute),
		time.Time{},
		false,
		map[string]any{
			"name": "default-pool",
		},
	)

	deletedLog := newGKECAILog(
		clusterAssetName,
		GKEClusterAssetType,
		queryStartTime.Add(-2*time.Hour),
		time.Time{},
		true,
		map[string]any{},
	)

	pastLog := newGKECAILog(
		clusterAssetName,
		GKEClusterAssetType,
		queryStartTime.Add(-5*time.Hour),
		queryStartTime.Add(-2*time.Hour),
		false,
		map[string]any{},
	)

	futureLog := newGKECAILog(
		clusterAssetName,
		GKEClusterAssetType,
		queryEndTime.Add(1*time.Hour),
		time.Time{},
		false,
		map[string]any{},
	)

	nodeCmpOpt := cmp.AllowUnexported(
		structured.StandardMapNode{},
		structured.StandardScalarNode[string]{},
		structured.StandardScalarNode[any]{},
		structured.StandardSequenceNode{},
		structured.OrderedMapNode{},
		unique.Handle[string]{},
	)

	testCases := []struct {
		name         string
		inputLog     *log.Log
		initialState gkeTimelineMapperState
		wantNil      bool
		assertResult func(t *testing.T, cs *khifilev6.TimelineChangeSet, finalState gkeTimelineMapperState)
	}{
		{
			name:         "creates 2 revisions for existing cluster with prior createTime",
			inputLog:     clusterInitialLog,
			initialState: gkeTimelineMapperState{hasProcessedInitialSnapshot: false},
			wantNil:      false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet, finalState gkeTimelineMapperState) {
				if !finalState.hasProcessedInitialSnapshot {
					t.Errorf("hasProcessedInitialSnapshot = false, want true")
				}
				revs := cs.GetRevisions(clusterTimeline)
				if len(revs) != 2 {
					t.Fatalf("len(revs) = %d, want 2", len(revs))
				}
				testchangeset.AssertTimeline(t, cs).
					HasRevision(clusterTimeline, &khifilev6.StagingRevision{
						ChangedTime:  clusterCreateTime,
						ResourceBody: nil,
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sClusterExistingLogNotFound,
					}, nodeCmpOpt).
					HasRevision(clusterTimeline, &khifilev6.StagingRevision{
						ChangedTime:  queryStartTime,
						ResourceBody: extractResourceBody(clusterInitialLog.NodeReader),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    googlecloudcaik8s_contract.RevisionStateGKEClusterExistingFromCAI,
					}, nodeCmpOpt)
			},
		},
		{
			name:         "creates 1 revision for cluster created within skew tolerance",
			inputLog:     clusterCreatedRecentlyLog,
			initialState: gkeTimelineMapperState{hasProcessedInitialSnapshot: false},
			wantNil:      false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet, finalState gkeTimelineMapperState) {
				revs := cs.GetRevisions(clusterTimeline)
				if len(revs) != 1 {
					t.Fatalf("len(revs) = %d, want 1", len(revs))
				}
				testchangeset.AssertTimeline(t, cs).
					HasRevision(clusterTimeline, &khifilev6.StagingRevision{
						ChangedTime:  queryStartTime,
						ResourceBody: extractResourceBody(clusterCreatedRecentlyLog.NodeReader),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    googlecloudcaik8s_contract.RevisionStateGKEClusterExistingFromCAI,
					}, nodeCmpOpt)
			},
		},
		{
			name:         "stages update revision for secondary cluster snapshot",
			inputLog:     clusterUpdateLog,
			initialState: gkeTimelineMapperState{hasProcessedInitialSnapshot: true},
			wantNil:      false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet, finalState gkeTimelineMapperState) {
				revs := cs.GetRevisions(clusterTimeline)
				if len(revs) != 1 {
					t.Fatalf("len(revs) = %d, want 1", len(revs))
				}
				testchangeset.AssertTimeline(t, cs).
					HasRevision(clusterTimeline, &khifilev6.StagingRevision{
						ChangedTime:  queryStartTime.Add(30 * time.Minute),
						ResourceBody: extractResourceBody(clusterUpdateLog.NodeReader),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    googlecloudcaik8s_contract.RevisionStateGKEClusterExistingFromCAI,
					}, nodeCmpOpt)
			},
		},
		{
			name:         "creates 3 revisions for nodepool with undetermined existence period",
			inputLog:     nodepoolInitialLog,
			initialState: gkeTimelineMapperState{hasProcessedInitialSnapshot: false},
			wantNil:      false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet, finalState gkeTimelineMapperState) {
				if !finalState.hasProcessedInitialSnapshot {
					t.Errorf("hasProcessedInitialSnapshot = false, want true")
				}
				revs := cs.GetRevisions(nodepoolTimeline)
				if len(revs) != 3 {
					t.Fatalf("len(revs) = %d, want 3", len(revs))
				}
				testchangeset.AssertTimeline(t, cs).
					HasRevision(nodepoolTimeline, &khifilev6.StagingRevision{
						ChangedTime:  clusterCreateTime,
						ResourceBody: nil,
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    googlecloudcaik8s_contract.RevisionStateGKENodePoolExistenceUnknown,
					}, nodeCmpOpt).
					HasRevision(nodepoolTimeline, &khifilev6.StagingRevision{
						ChangedTime:  nodepoolEarliestTime,
						ResourceBody: nil,
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sNodepoolExistingLogNotFound,
					}, nodeCmpOpt).
					HasRevision(nodepoolTimeline, &khifilev6.StagingRevision{
						ChangedTime:  queryStartTime,
						ResourceBody: extractResourceBody(nodepoolInitialLog.NodeReader),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    googlecloudcaik8s_contract.RevisionStateGKENodePoolExistingFromCAI,
					}, nodeCmpOpt)
			},
		},
		{
			name:         "creates 2 revisions for nodepool created alongside cluster",
			inputLog:     nodepoolCreatedWithClusterLog,
			initialState: gkeTimelineMapperState{hasProcessedInitialSnapshot: false},
			wantNil:      false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet, finalState gkeTimelineMapperState) {
				revs := cs.GetRevisions(nodepoolTimeline)
				if len(revs) != 2 {
					t.Fatalf("len(revs) = %d, want 2", len(revs))
				}
				testchangeset.AssertTimeline(t, cs).
					HasRevision(nodepoolTimeline, &khifilev6.StagingRevision{
						ChangedTime:  clusterCreateTime,
						ResourceBody: nil,
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sNodepoolExistingLogNotFound,
					}, nodeCmpOpt).
					HasRevision(nodepoolTimeline, &khifilev6.StagingRevision{
						ChangedTime:  queryStartTime,
						ResourceBody: extractResourceBody(nodepoolCreatedWithClusterLog.NodeReader),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    googlecloudcaik8s_contract.RevisionStateGKENodePoolExistingFromCAI,
					}, nodeCmpOpt)
			},
		},
		{
			name:         "stages update revision for secondary nodepool snapshot",
			inputLog:     nodepoolUpdateLog,
			initialState: gkeTimelineMapperState{hasProcessedInitialSnapshot: true},
			wantNil:      false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet, finalState gkeTimelineMapperState) {
				revs := cs.GetRevisions(nodepoolTimeline)
				if len(revs) != 1 {
					t.Fatalf("len(revs) = %d, want 1", len(revs))
				}
				testchangeset.AssertTimeline(t, cs).
					HasRevision(nodepoolTimeline, &khifilev6.StagingRevision{
						ChangedTime:  queryStartTime.Add(45 * time.Minute),
						ResourceBody: extractResourceBody(nodepoolUpdateLog.NodeReader),
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    googlecloudcaik8s_contract.RevisionStateGKENodePoolExistingFromCAI,
					}, nodeCmpOpt)
			},
		},
		{
			name:         "skips deleted snapshot",
			inputLog:     deletedLog,
			initialState: gkeTimelineMapperState{},
			wantNil:      true,
		},
		{
			name:         "skips snapshot that ended before query window",
			inputLog:     pastLog,
			initialState: gkeTimelineMapperState{},
			wantNil:      true,
		},
		{
			name:         "skips snapshot that started after query window",
			inputLog:     futureLog,
			initialState: gkeTimelineMapperState{},
			wantNil:      true,
		},
	}

	mapper := &caiGKEResourceTimelineMapper{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			ctx = tasktest.WithTaskResult(ctx, googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(), clusterIdentity)
			ctx = tasktest.WithTaskResult(ctx, googlecloudcommon_contract.InputStartTimeTaskID.Ref(), queryStartTime)
			ctx = tasktest.WithTaskResult(ctx, googlecloudcommon_contract.InputEndTimeTaskID.Ref(), queryEndTime)
			ctx = tasktest.WithTaskResult(ctx, googlecloudcaik8s_contract.GKEResourceFetcherTaskID.Ref(), []*googlecloudcaik8s_contract.GKEResourceSnapshot{clusterSnapshot})

			cs, finalState, err := mapper.ProcessLogByGroup(ctx, tc.inputLog, tc.initialState)
			if err != nil {
				t.Fatalf("ProcessLogByGroup() unexpected error: %v", err)
			}
			if tc.wantNil {
				if cs != nil {
					t.Errorf("ProcessLogByGroup() returned cs, want nil")
				}
				return
			}
			if cs == nil {
				t.Fatalf("ProcessLogByGroup() returned nil cs, want non-nil")
			}
			tc.assertResult(t, cs, finalState)
		})
	}
}
