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

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	"github.com/GoogleCloudPlatform/khi/pkg/model/k8s"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	googlecloudcaik8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcaik8s/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

// caiLogParams describes the CAI temporal asset log a test case feeds to the mapper.
type caiLogParams struct {
	windowStartTime   time.Time
	windowEndTime     time.Time
	isDeleted         bool
	hasResourceData   bool
	creationTimestamp time.Time
}

func TestCAIClusterResourceTimelineMapper_ProcessLogByGroup(t *testing.T) {
	builder := khifilev6.NewTestBuilder(id.NewGenerator())
	clusterName := "test-cluster"
	queryStartTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	podIdent := &k8saudit.ResourceIdentity{
		APIVersion: "core/v1",
		Kind:       "pod",
		Name:       "pod-1",
		Namespace:  "default",
	}

	manifestNode, err := structured.FromGoValue(map[string]any{
		"apiVersion": "core/v1",
		"kind":       "Pod",
		"metadata": map[string]any{
			"name":      "pod-1",
			"namespace": "default",
		},
	}, &structured.AlphabeticalGoMapKeyOrderProvider{})
	if err != nil {
		t.Fatalf("failed to build manifest node: %v", err)
	}
	manifestNode = structured.WithKeyOrder(manifestNode, k8s.K8sManifestKeyOrder...)

	ctxWithBuilder := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)
	targetPath := k8saudit.MustResourceTimeline(ctxWithBuilder, clusterName, podIdent)

	// The generator is shared by every log the closure builds so that each log gets a distinct
	// log ID; a generator created per call would restart the counter and hand out duplicates.
	generator := id.NewGenerator()
	newCAILog := func(params caiLogParams) *log.Log {
		window := map[string]any{}
		if !params.windowStartTime.IsZero() {
			window["startTime"] = params.windowStartTime.Format(time.RFC3339Nano)
		}
		if !params.windowEndTime.IsZero() {
			window["endTime"] = params.windowEndTime.Format(time.RFC3339Nano)
		}
		m := map[string]any{
			"window":  window,
			"deleted": params.isDeleted,
		}
		if params.hasResourceData {
			metadata := map[string]any{
				"name":      "pod-1",
				"namespace": "default",
			}
			if !params.creationTimestamp.IsZero() {
				metadata["creationTimestamp"] = params.creationTimestamp.Format(time.RFC3339Nano)
			}
			m["asset"] = map[string]any{
				"resource": map[string]any{
					"data": map[string]any{
						"apiVersion": "core/v1",
						"kind":       "Pod",
						"metadata":   metadata,
					},
				},
			}
		}
		node, err := structured.FromGoValue(m, &structured.AlphabeticalGoMapKeyOrderProvider{})
		if err != nil {
			t.Fatalf("failed to build node: %v", err)
		}
		return log.NewLogWithTimestamp(generator, structured.NewNodeReader(node), params.windowStartTime)
	}

	assetWindowStartTime := queryStartTime.Add(-1 * time.Hour)
	activeLog := newCAILog(caiLogParams{windowStartTime: assetWindowStartTime, windowEndTime: queryStartTime.Add(1 * time.Hour), hasResourceData: true})
	windowStartAtQueryStartLog := newCAILog(caiLogParams{windowStartTime: queryStartTime, windowEndTime: queryStartTime.Add(1 * time.Hour), hasResourceData: true})
	openEndedLog := newCAILog(caiLogParams{windowStartTime: assetWindowStartTime, hasResourceData: true})
	deletedLog := newCAILog(caiLogParams{windowStartTime: assetWindowStartTime, windowEndTime: queryStartTime.Add(1 * time.Hour), isDeleted: true, hasResourceData: true})
	futureLog := newCAILog(caiLogParams{windowStartTime: queryStartTime.Add(10 * time.Minute), windowEndTime: queryStartTime.Add(1 * time.Hour), hasResourceData: true})
	endedLog := newCAILog(caiLogParams{windowStartTime: assetWindowStartTime, windowEndTime: queryStartTime.Add(-10 * time.Minute), hasResourceData: true})
	windowEndAtQueryStartLog := newCAILog(caiLogParams{windowStartTime: assetWindowStartTime, windowEndTime: queryStartTime, hasResourceData: true})
	noResourceDataLog := newCAILog(caiLogParams{windowStartTime: queryStartTime, windowEndTime: queryStartTime.Add(1 * time.Hour)})
	creationTime := assetWindowStartTime.Add(-24 * time.Hour)
	createdLongBeforeLog := newCAILog(caiLogParams{windowStartTime: assetWindowStartTime, hasResourceData: true, creationTimestamp: creationTime})
	createdWithinToleranceLog := newCAILog(caiLogParams{windowStartTime: assetWindowStartTime, hasResourceData: true, creationTimestamp: assetWindowStartTime.Add(-500 * time.Millisecond)})
	createdAfterWindowStartLog := newCAILog(caiLogParams{windowStartTime: assetWindowStartTime, hasResourceData: true, creationTimestamp: assetWindowStartTime.Add(time.Minute)})
	noWindowStartLog := newCAILog(caiLogParams{windowEndTime: queryStartTime.Add(1 * time.Hour), hasResourceData: true})

	testCases := []struct {
		name         string
		inputLog     *log.Log
		wantNil      bool
		assertResult func(t *testing.T, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name:     "stages the snapshot at the time the asset version became current",
			inputLog: activeLog,
			wantNil:  false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				nodeCmpOpt := cmp.AllowUnexported(
					structured.StandardMapNode{},
					structured.StandardScalarNode[string]{},
					structured.StandardScalarNode[any]{},
					structured.StandardSequenceNode{},
					structured.OrderedMapNode{},
					unique.Handle[string]{},
				)
				testchangeset.AssertTimeline(t, cs).
					HasEventCount(0).
					HasNoEvent(targetPath).
					HasRevision(targetPath, &khifilev6.StagingRevision{
						ChangedTime:  assetWindowStartTime,
						ResourceBody: manifestNode,
						Principal:    "N/A",
						VerbType:     k8saudit.VerbCreate,
						StateType:    googlecloudcaik8s_contract.RevisionStateK8sResourceExistingFromCAI,
					}, nodeCmpOpt)

				revs := cs.GetRevisions(targetPath)
				if len(revs) != 1 {
					t.Errorf("len(revs) = %d, want 1", len(revs))
				}
				if diff := cmp.Diff(manifestNode, revs[0].ResourceBody, nodeCmpOpt); diff != "" {
					t.Errorf("ResourceBody mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			name:     "creates revision when snapshot starts exactly at query window start boundary",
			inputLog: windowStartAtQueryStartLog,
			wantNil:  false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEventCount(0).
					HasNoEvent(targetPath)
				revs := cs.GetRevisions(targetPath)
				if len(revs) != 1 {
					t.Errorf("len(revs) = %d, want 1", len(revs))
				}
			},
		},
		{
			name:     "creates revision for open-ended active resource snapshot without endTime",
			inputLog: openEndedLog,
			wantNil:  false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEventCount(0).
					HasNoEvent(targetPath)
				revs := cs.GetRevisions(targetPath)
				if len(revs) != 1 {
					t.Errorf("len(revs) = %d, want 1", len(revs))
				}
			},
		},
		{
			name:     "prepends a body-less revision from the creation timestamp",
			inputLog: createdLongBeforeLog,
			wantNil:  false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				revs := cs.GetRevisions(targetPath)
				if len(revs) != 2 {
					t.Fatalf("len(revs) = %d, want 2", len(revs))
				}
				if revs[0].ChangedTime != creationTime {
					t.Errorf("revs[0].ChangedTime = %v, want %v", revs[0].ChangedTime, creationTime)
				}
				if revs[0].StateType != k8saudit.RevisionStateK8sResourceExistingLogNotFound {
					t.Errorf("revs[0].StateType = %v, want %v", revs[0].StateType, k8saudit.RevisionStateK8sResourceExistingLogNotFound)
				}
				if revs[0].VerbType != k8saudit.VerbCreate {
					t.Errorf("revs[0].VerbType = %v, want %v", revs[0].VerbType, k8saudit.VerbCreate)
				}
				if revs[0].ResourceBody != nil {
					t.Errorf("revs[0].ResourceBody = %v, want nil", revs[0].ResourceBody)
				}
				if revs[1].ChangedTime != assetWindowStartTime {
					t.Errorf("revs[1].ChangedTime = %v, want %v", revs[1].ChangedTime, assetWindowStartTime)
				}
				if revs[1].StateType != googlecloudcaik8s_contract.RevisionStateK8sResourceExistingFromCAI {
					t.Errorf("revs[1].StateType = %v, want %v", revs[1].StateType, googlecloudcaik8s_contract.RevisionStateK8sResourceExistingFromCAI)
				}
				if revs[1].VerbType != k8saudit.VerbUpdate {
					t.Errorf("revs[1].VerbType = %v, want %v", revs[1].VerbType, k8saudit.VerbUpdate)
				}
			},
		},
		{
			name:     "keeps a creation timestamp gap under the skew tolerance unrendered",
			inputLog: createdWithinToleranceLog,
			wantNil:  false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				revs := cs.GetRevisions(targetPath)
				if len(revs) != 1 {
					t.Errorf("len(revs) = %d, want 1", len(revs))
				}
				if revs[0].VerbType != k8saudit.VerbCreate {
					t.Errorf("revs[0].VerbType = %v, want %v", revs[0].VerbType, k8saudit.VerbCreate)
				}
			},
		},
		{
			name:     "does not prepend when the creation timestamp is after the window start",
			inputLog: createdAfterWindowStartLog,
			wantNil:  false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				revs := cs.GetRevisions(targetPath)
				if len(revs) != 1 {
					t.Errorf("len(revs) = %d, want 1", len(revs))
				}
			},
		},
		{
			name:     "falls back to the inspection start time when the window start is missing",
			inputLog: noWindowStartLog,
			wantNil:  false,
			assertResult: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				revs := cs.GetRevisions(targetPath)
				if len(revs) != 1 {
					t.Fatalf("len(revs) = %d, want 1", len(revs))
				}
				if revs[0].ChangedTime != queryStartTime {
					t.Errorf("revs[0].ChangedTime = %v, want %v", revs[0].ChangedTime, queryStartTime)
				}
			},
		},
		{
			name:     "skips deleted tombstone snapshot",
			inputLog: deletedLog,
			wantNil:  true,
		},
		{
			name:     "skips snapshot that starts in the future",
			inputLog: futureLog,
			wantNil:  true,
		},
		{
			name:     "skips snapshot that ended before query window",
			inputLog: endedLog,
			wantNil:  true,
		},
		{
			name:     "skips snapshot that ended at query window start boundary",
			inputLog: windowEndAtQueryStartLog,
			wantNil:  true,
		},
		{
			name:     "skips log without resource identity",
			inputLog: noResourceDataLog,
			wantNil:  true,
		},
	}

	mapper := &caiClusterResourceTimelineMapper{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tasktest.WithTaskResult(ctxWithBuilder, k8scommon.ClusterIdentityTaskID.Ref(), k8scommon.GoogleCloudClusterIdentity{
				ProjectID:   "test-project",
				ClusterName: clusterName,
				Location:    "us-central1-a",
			})
			ctx = tasktest.WithTaskResult(ctx, gcpcommon.InputStartTimeTaskID.Ref(), queryStartTime)

			cs, _, err := mapper.ProcessLogByGroup(ctx, tc.inputLog, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() returned error: %v", err)
			}
			if tc.wantNil {
				if cs != nil {
					t.Errorf("ProcessLogByGroup() = %v, want nil", cs)
				}
				return
			}
			if cs == nil {
				t.Fatal("ProcessLogByGroup() = nil, want non-nil changeset")
			}
			tc.assertResult(t, cs)
		})
	}
}
