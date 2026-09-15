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
	"time"
	"unique"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/caik8s"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
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
		snapshots []*caik8s.GKEResourceSnapshot
		want      time.Time
	}{
		{
			name: "extracts createTime from GKECluster snapshot",
			snapshots: []*caik8s.GKEResourceSnapshot{
				{
					TemporalAsset: &assetpb.TemporalAsset{
						Asset: &assetpb.Asset{
							AssetType: caik8s.GKEClusterAssetType,
							Resource:  &assetpb.Resource{Data: data},
						},
					},
				},
			},
			want: createTime,
		},
		{
			name: "returns zero time when no cluster snapshot",
			snapshots: []*caik8s.GKEResourceSnapshot{
				{
					TemporalAsset: &assetpb.TemporalAsset{
						Asset: &assetpb.Asset{
							AssetType: caik8s.GKENodePoolAssetType,
							Resource:  &assetpb.Resource{Data: data},
						},
					},
				},
			},
			want: time.Time{},
		},
		{
			name:      "returns zero time for empty snapshots",
			snapshots: []*caik8s.GKEResourceSnapshot{},
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

func TestCAIGKEResourceTimelineMapper_ProcessLogByGroup(t *testing.T) {
	builder := khifilev6.NewTestBuilder(id.NewGenerator())
	clusterName := "test-cluster"
	projectID := "test-project"
	queryStartTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	queryEndTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	clusterIdentity := k8scommon.GoogleCloudClusterIdentity{
		ProjectID:   projectID,
		ClusterName: clusterName,
		Location:    "us-central1-a",
	}

	ctxWithBuilder := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)
	projectTimeline := gcpcommon.MustGCPProjectTimeline(ctxWithBuilder, projectID)
	clusterTimeline := gcpcommon.MustGKEClusterTimeline(ctxWithBuilder, projectTimeline, clusterName)
	nodePoolTimeline := gcpcommon.MustGKENodePoolTimeline(ctxWithBuilder, clusterTimeline, "default-pool")

	clusterCreateTime := time.Date(2025, 5, 1, 10, 0, 0, 0, time.UTC)
	clusterData, _ := structpb.NewStruct(map[string]any{
		"name":       clusterName,
		"createTime": clusterCreateTime.Format(time.RFC3339),
	})
	clusterSnapshot := &caik8s.GKEResourceSnapshot{
		TemporalAsset: &assetpb.TemporalAsset{
			Asset: &assetpb.Asset{
				Name:      "//container.googleapis.com/projects/test-project/locations/us-central1-a/clusters/test-cluster",
				AssetType: caik8s.GKEClusterAssetType,
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
		caik8s.GKEClusterAssetType,
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
		caik8s.GKEClusterAssetType,
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
		caik8s.GKEClusterAssetType,
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
		caik8s.GKENodePoolAssetType,
		nodepoolEarliestTime,
		time.Time{},
		false,
		map[string]any{
			"name": "default-pool",
		},
	)

	nodepoolCreatedWithClusterLog := newGKECAILog(
		nodepoolAssetName,
		caik8s.GKENodePoolAssetType,
		clusterCreateTime.Add(200*time.Millisecond),
		time.Time{},
		false,
		map[string]any{
			"name": "default-pool",
		},
	)

	nodepoolZeroStartTimeLog := newGKECAILog(
		nodepoolAssetName,
		caik8s.GKENodePoolAssetType,
		time.Time{},
		time.Time{},
		false,
		map[string]any{
			"name": "default-pool",
		},
	)

	nodepoolUpdateLog := newGKECAILog(
		nodepoolAssetName,
		caik8s.GKENodePoolAssetType,
		queryStartTime.Add(45*time.Minute),
		time.Time{},
		false,
		map[string]any{
			"name": "default-pool",
		},
	)

	nodepoolCreatedDuringWindowLog := newGKECAILog(
		nodepoolAssetName,
		caik8s.GKENodePoolAssetType,
		queryStartTime.Add(30*time.Minute),
		time.Time{},
		false,
		map[string]any{
			"name": "default-pool",
		},
	)

	deletedLog := newGKECAILog(
		clusterAssetName,
		caik8s.GKEClusterAssetType,
		queryStartTime.Add(-2*time.Hour),
		time.Time{},
		true,
		map[string]any{},
	)

	pastLog := newGKECAILog(
		clusterAssetName,
		caik8s.GKEClusterAssetType,
		queryStartTime.Add(-5*time.Hour),
		queryStartTime.Add(-2*time.Hour),
		false,
		map[string]any{},
	)

	futureLog := newGKECAILog(
		clusterAssetName,
		caik8s.GKEClusterAssetType,
		queryEndTime.Add(1*time.Hour),
		time.Time{},
		false,
		map[string]any{},
	)

	unrecognizedLog := newGKECAILog(
		"//compute.googleapis.com/projects/test-project/zones/us-central1-a/instances/test-instance",
		"compute.googleapis.com/Instance",
		queryStartTime.Add(-1*time.Hour),
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
		name                 string
		inputLog             *log.Log
		omitClusterSnapshots bool
		wantNil              bool
		assertResult         func(t *testing.T, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name:     "creates 2 revisions for existing cluster with prior createTime",
			inputLog: clusterInitialLog,
			wantNil:  false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				revs := cs.GetRevisions(clusterTimeline)
				if len(revs) != 2 {
					t.Fatalf("len(revs) = %d, want 2", len(revs))
				}
				testchangeset.AssertTimeline(t, cs).
					HasRevision(clusterTimeline, &khifilev6.StagingRevision{
						ChangedTime:  clusterCreateTime,
						ResourceBody: nil,
						Principal:    "N/A",
						VerbType:     k8saudit.VerbCreate,
						StateType:    k8saudit.RevisionStateK8sClusterExistingLogNotFound,
					}, nodeCmpOpt).
					HasRevision(clusterTimeline, &khifilev6.StagingRevision{
						ChangedTime:  queryStartTime.Add(-2 * time.Hour),
						ResourceBody: extractResourceBody(clusterInitialLog.NodeReader),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUpdate,
						StateType:    caik8s.RevisionStateGKEClusterSnapshotFromCAI,
					}, nodeCmpOpt)
			},
		},
		{
			name:     "creates 1 revision for cluster created within skew tolerance",
			inputLog: clusterCreatedRecentlyLog,
			wantNil:  false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				revs := cs.GetRevisions(clusterTimeline)
				if len(revs) != 1 {
					t.Fatalf("len(revs) = %d, want 1", len(revs))
				}
				testchangeset.AssertTimeline(t, cs).
					HasRevision(clusterTimeline, &khifilev6.StagingRevision{
						ChangedTime:  queryStartTime,
						ResourceBody: extractResourceBody(clusterCreatedRecentlyLog.NodeReader),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbCreate,
						StateType:    caik8s.RevisionStateGKEClusterSnapshotFromCAI,
					}, nodeCmpOpt)
			},
		},
		{
			name:     "skips secondary cluster snapshot not active at query start",
			inputLog: clusterUpdateLog,
			wantNil:  true,
		},
		{
			name:     "creates 2 revisions for nodepool with undetermined existence period",
			inputLog: nodepoolInitialLog,
			wantNil:  false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				revs := cs.GetRevisions(nodePoolTimeline)
				if len(revs) != 2 {
					t.Fatalf("len(revs) = %d, want 2", len(revs))
				}
				testchangeset.AssertTimeline(t, cs).
					HasRevision(nodePoolTimeline, &khifilev6.StagingRevision{
						ChangedTime:  clusterCreateTime,
						ResourceBody: nil,
						Principal:    "N/A",
						VerbType:     k8saudit.VerbCreate,
						StateType:    caik8s.RevisionStateGKENodePoolExistenceUndetermined,
					}, nodeCmpOpt).
					HasRevision(nodePoolTimeline, &khifilev6.StagingRevision{
						ChangedTime:  nodepoolEarliestTime,
						ResourceBody: extractResourceBody(nodepoolInitialLog.NodeReader),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUpdate,
						StateType:    caik8s.RevisionStateGKENodePoolSnapshotFromCAI,
					}, nodeCmpOpt)
			},
		},
		{
			name:     "creates 1 revision for nodepool created alongside cluster",
			inputLog: nodepoolCreatedWithClusterLog,
			wantNil:  false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				revs := cs.GetRevisions(nodePoolTimeline)
				if len(revs) != 1 {
					t.Fatalf("len(revs) = %d, want 1", len(revs))
				}
				testchangeset.AssertTimeline(t, cs).
					HasRevision(nodePoolTimeline, &khifilev6.StagingRevision{
						ChangedTime:  clusterCreateTime.Add(200 * time.Millisecond),
						ResourceBody: extractResourceBody(nodepoolCreatedWithClusterLog.NodeReader),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbCreate,
						StateType:    caik8s.RevisionStateGKENodePoolSnapshotFromCAI,
					}, nodeCmpOpt)
			},
		},
		{
			name:     "creates 2 revisions with undetermined existence for nodepool when window startTime is zero",
			inputLog: nodepoolZeroStartTimeLog,
			wantNil:  false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				revs := cs.GetRevisions(nodePoolTimeline)
				if len(revs) != 2 {
					t.Fatalf("len(revs) = %d, want 2", len(revs))
				}
				testchangeset.AssertTimeline(t, cs).
					HasRevision(nodePoolTimeline, &khifilev6.StagingRevision{
						ChangedTime:  clusterCreateTime,
						ResourceBody: nil,
						Principal:    "N/A",
						VerbType:     k8saudit.VerbCreate,
						StateType:    caik8s.RevisionStateGKENodePoolExistenceUndetermined,
					}, nodeCmpOpt).
					HasRevision(nodePoolTimeline, &khifilev6.StagingRevision{
						ChangedTime:  queryStartTime,
						ResourceBody: extractResourceBody(nodepoolZeroStartTimeLog.NodeReader),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUpdate,
						StateType:    caik8s.RevisionStateGKENodePoolSnapshotFromCAI,
					}, nodeCmpOpt)
			},
		},
		{
			name:                 "creates 1 revision for nodepool when cluster createTime is unavailable",
			inputLog:             nodepoolInitialLog,
			omitClusterSnapshots: true,
			wantNil:              false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				revs := cs.GetRevisions(nodePoolTimeline)
				if len(revs) != 1 {
					t.Fatalf("len(revs) = %d, want 1", len(revs))
				}
				testchangeset.AssertTimeline(t, cs).
					HasRevision(nodePoolTimeline, &khifilev6.StagingRevision{
						ChangedTime:  nodepoolEarliestTime,
						ResourceBody: extractResourceBody(nodepoolInitialLog.NodeReader),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbCreate,
						StateType:    caik8s.RevisionStateGKENodePoolSnapshotFromCAI,
					}, nodeCmpOpt)
			},
		},
		{
			name:     "skips nodepool created during inspection window",
			inputLog: nodepoolCreatedDuringWindowLog,
			wantNil:  true,
		},
		{
			name:     "skips secondary nodepool snapshot not active at query start",
			inputLog: nodepoolUpdateLog,
			wantNil:  true,
		},
		{
			name:     "skips deleted snapshot",
			inputLog: deletedLog,
			wantNil:  true,
		},
		{
			name:     "skips snapshot that ended before query window",
			inputLog: pastLog,
			wantNil:  true,
		},
		{
			name:     "skips snapshot that started after query window",
			inputLog: futureLog,
			wantNil:  true,
		},
		{
			name:     "skips unrecognized non-GKE log asset",
			inputLog: unrecognizedLog,
			wantNil:  true,
		},
	}

	mapper := &caiGKEResourceTimelineMapper{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)
			ctx = tasktest.WithTaskResult(ctx, k8scommon.ClusterIdentityTaskID.Ref(), clusterIdentity)
			ctx = tasktest.WithTaskResult(ctx, gcpcommon.InputStartTimeTaskID.Ref(), queryStartTime)
			clusterSnapshots := []*caik8s.GKEResourceSnapshot{clusterSnapshot}
			if tc.omitClusterSnapshots {
				clusterSnapshots = []*caik8s.GKEResourceSnapshot{}
			}
			ctx = tasktest.WithTaskResult(ctx, caik8s.GKEResourceFetcherTaskID.Ref(), clusterSnapshots)

			cs, _, err := mapper.ProcessLogByGroup(ctx, tc.inputLog, struct{}{})
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
			tc.assertResult(t, cs)
		})
	}
}
