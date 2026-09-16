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
)

func newTestClusterCreateTimeLog(t *testing.T, assetType string, createTimeStr string) *log.Log {
	t.Helper()
	var data map[string]any
	if createTimeStr != "" {
		data = map[string]any{
			"createTime": createTimeStr,
		}
	}
	m := map[string]any{
		"asset": map[string]any{
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
	return log.NewLogWithTimestamp(id.NewGenerator(), structured.NewNodeReader(node), time.Time{})
}

func TestExtractClusterCreateTimeFromLogs(t *testing.T) {
	createTime := time.Date(2025, 5, 1, 10, 0, 0, 0, time.UTC)
	validClusterLog := newTestClusterCreateTimeLog(t, caik8s.GKEClusterAssetType, createTime.Format(time.RFC3339))
	nonClusterLog := newTestClusterCreateTimeLog(t, caik8s.GKENodePoolAssetType, createTime.Format(time.RFC3339))
	invalidTimestampLog := newTestClusterCreateTimeLog(t, caik8s.GKEClusterAssetType, "not-a-valid-timestamp")

	testCases := []struct {
		name string
		logs []*log.Log
		want time.Time
	}{
		{
			name: "extracts createTime from GKECluster log",
			logs: []*log.Log{validClusterLog},
			want: createTime,
		},
		{
			name: "skips non-cluster and invalid timestamp logs and extracts createTime from subsequent valid log",
			logs: []*log.Log{nonClusterLog, invalidTimestampLog, validClusterLog},
			want: createTime,
		},
		{
			name: "returns zero time when no cluster log",
			logs: []*log.Log{nonClusterLog},
			want: time.Time{},
		},
		{
			name: "returns zero time for empty logs",
			logs: []*log.Log{},
			want: time.Time{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractClusterCreateTimeFromLogs(tc.logs)
			if !got.Equal(tc.want) {
				t.Errorf("extractClusterCreateTimeFromLogs() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMapGKEResourceInitialRevision(t *testing.T) {
	builder := khifilev6.NewTestBuilder(id.NewGenerator())
	clusterName := "test-cluster"
	projectID := "test-project"
	queryStartTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

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
	nodePoolAssetName := clusterAssetName + "/nodePools/default-pool"

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

	nodePoolEarliestTime := queryStartTime.Add(-10 * time.Hour)
	nodePoolInitialLog := newGKECAILog(
		nodePoolAssetName,
		caik8s.GKENodePoolAssetType,
		nodePoolEarliestTime,
		time.Time{},
		false,
		map[string]any{
			"name": "default-pool",
		},
	)

	nodePoolCreatedWithClusterLog := newGKECAILog(
		nodePoolAssetName,
		caik8s.GKENodePoolAssetType,
		clusterCreateTime.Add(200*time.Millisecond),
		time.Time{},
		false,
		map[string]any{
			"name": "default-pool",
		},
	)

	nodePoolZeroStartTimeLog := newGKECAILog(
		nodePoolAssetName,
		caik8s.GKENodePoolAssetType,
		time.Time{},
		time.Time{},
		false,
		map[string]any{
			"name": "default-pool",
		},
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
		name            string
		inputLog        *log.Log
		observedTime    time.Time
		omitClusterLogs bool
		assertResult    func(t *testing.T, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name:         "creates 2 revisions for existing cluster with prior createTime",
			inputLog:     clusterInitialLog,
			observedTime: queryStartTime.Add(-2 * time.Hour),
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
						ResourceBody: extractGKEResourceBody(clusterInitialLog.NodeReader),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUpdate,
						StateType:    caik8s.RevisionStateGKEClusterSnapshotFromCAI,
					}, nodeCmpOpt)
			},
		},
		{
			name:         "creates 1 revision for cluster created within skew tolerance",
			inputLog:     clusterCreatedRecentlyLog,
			observedTime: queryStartTime,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				revs := cs.GetRevisions(clusterTimeline)
				if len(revs) != 1 {
					t.Fatalf("len(revs) = %d, want 1", len(revs))
				}
				testchangeset.AssertTimeline(t, cs).
					HasRevision(clusterTimeline, &khifilev6.StagingRevision{
						ChangedTime:  queryStartTime,
						ResourceBody: extractGKEResourceBody(clusterCreatedRecentlyLog.NodeReader),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbCreate,
						StateType:    caik8s.RevisionStateGKEClusterSnapshotFromCAI,
					}, nodeCmpOpt)
			},
		},
		{
			name:         "creates 2 revisions for nodepool with undetermined existence period",
			inputLog:     nodePoolInitialLog,
			observedTime: nodePoolEarliestTime,
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
						ChangedTime:  nodePoolEarliestTime,
						ResourceBody: extractGKEResourceBody(nodePoolInitialLog.NodeReader),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUpdate,
						StateType:    caik8s.RevisionStateGKENodePoolSnapshotFromCAI,
					}, nodeCmpOpt)
			},
		},
		{
			name:         "creates 1 revision for nodepool created alongside cluster",
			inputLog:     nodePoolCreatedWithClusterLog,
			observedTime: clusterCreateTime.Add(200 * time.Millisecond),
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				revs := cs.GetRevisions(nodePoolTimeline)
				if len(revs) != 1 {
					t.Fatalf("len(revs) = %d, want 1", len(revs))
				}
				testchangeset.AssertTimeline(t, cs).
					HasRevision(nodePoolTimeline, &khifilev6.StagingRevision{
						ChangedTime:  clusterCreateTime.Add(200 * time.Millisecond),
						ResourceBody: extractGKEResourceBody(nodePoolCreatedWithClusterLog.NodeReader),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbCreate,
						StateType:    caik8s.RevisionStateGKENodePoolSnapshotFromCAI,
					}, nodeCmpOpt)
			},
		},
		{
			name:         "creates 2 revisions with undetermined existence for nodepool when window startTime is zero",
			inputLog:     nodePoolZeroStartTimeLog,
			observedTime: queryStartTime,
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
						ResourceBody: extractGKEResourceBody(nodePoolZeroStartTimeLog.NodeReader),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbUpdate,
						StateType:    caik8s.RevisionStateGKENodePoolSnapshotFromCAI,
					}, nodeCmpOpt)
			},
		},
		{
			name:            "creates 1 revision for nodepool when cluster createTime is unavailable",
			inputLog:        nodePoolInitialLog,
			observedTime:    nodePoolEarliestTime,
			omitClusterLogs: true,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				revs := cs.GetRevisions(nodePoolTimeline)
				if len(revs) != 1 {
					t.Fatalf("len(revs) = %d, want 1", len(revs))
				}
				testchangeset.AssertTimeline(t, cs).
					HasRevision(nodePoolTimeline, &khifilev6.StagingRevision{
						ChangedTime:  nodePoolEarliestTime,
						ResourceBody: extractGKEResourceBody(nodePoolInitialLog.NodeReader),
						Principal:    "N/A",
						VerbType:     k8saudit.VerbCreate,
						StateType:    caik8s.RevisionStateGKENodePoolSnapshotFromCAI,
					}, nodeCmpOpt)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)
			ctx = tasktest.WithTaskResult(ctx, k8scommon.ClusterIdentityTaskID.Ref(), clusterIdentity)
			clusterLogs := []*log.Log{clusterInitialLog}
			if tc.omitClusterLogs {
				clusterLogs = []*log.Log{}
			}
			ctx = tasktest.WithTaskResult(ctx, caik8s.GKEResourceTaskIDs.RawLog.Ref(), clusterLogs)

			identity, ok := extractGKEIdentity(tc.inputLog.NodeReader)
			if !ok {
				t.Fatal("extractGKEIdentity() = false, want true")
			}

			spec, skip, err := mapGKEResourceInitialRevision(ctx, tc.inputLog, identity, tc.observedTime)
			if err != nil {
				t.Fatalf("mapGKEResourceInitialRevision() unexpected error: %v", err)
			}
			if skip {
				t.Fatal("mapGKEResourceInitialRevision() returned skip=true, want false")
			}

			cs := khifilev6.NewTimelineChangeSet(tc.inputLog)
			gcpcommon.StageCAIInitialSnapshotRevisions(cs, spec)
			tc.assertResult(t, cs)
		})
	}
}
