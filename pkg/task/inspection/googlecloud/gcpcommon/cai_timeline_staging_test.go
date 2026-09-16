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

package gcpcommon

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestStageCAIInitialSnapshotRevisions(t *testing.T) {
	creationTime := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	observedTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	creationState := k8saudit.RevisionStateK8sResourceExistingLogNotFound
	snapshotState := k8saudit.RevisionStateK8sClusterExistingLogNotFound
	sampleBody, err := structured.FromGoValue(map[string]any{"name": "sample"}, &structured.AlphabeticalGoMapKeyOrderProvider{})
	if err != nil {
		t.Fatalf("failed to create sample body: %v", err)
	}

	testCases := []struct {
		name          string
		spec          CAIInitialSnapshotRevisionSpec
		wantRevisions []*khifilev6.StagingRevision
	}{
		{
			name: "stages two revisions when creation time is earlier than skew tolerance",
			spec: CAIInitialSnapshotRevisionSpec{
				CreationTime:      creationTime,
				ObservedTime:      observedTime,
				ResourceBody:      sampleBody,
				CreationStateType: creationState,
				SnapshotStateType: snapshotState,
			},
			wantRevisions: []*khifilev6.StagingRevision{
				{
					ChangedTime:  creationTime,
					ResourceBody: nil,
					Principal:    "N/A",
					VerbType:     k8saudit.VerbCreate,
					StateType:    creationState,
				},
				{
					ChangedTime:  observedTime,
					ResourceBody: sampleBody,
					Principal:    "N/A",
					VerbType:     k8saudit.VerbUpdate,
					StateType:    snapshotState,
				},
			},
		},
		{
			name: "stages two revisions when difference equals skew tolerance",
			spec: CAIInitialSnapshotRevisionSpec{
				CreationTime:      observedTime.Add(-CAICreationTimestampSkewTolerance),
				ObservedTime:      observedTime,
				ResourceBody:      sampleBody,
				CreationStateType: creationState,
				SnapshotStateType: snapshotState,
			},
			wantRevisions: []*khifilev6.StagingRevision{
				{
					ChangedTime:  observedTime.Add(-CAICreationTimestampSkewTolerance),
					ResourceBody: nil,
					Principal:    "N/A",
					VerbType:     k8saudit.VerbCreate,
					StateType:    creationState,
				},
				{
					ChangedTime:  observedTime,
					ResourceBody: sampleBody,
					Principal:    "N/A",
					VerbType:     k8saudit.VerbUpdate,
					StateType:    snapshotState,
				},
			},
		},
		{
			name: "stages single create revision when creation time is zero",
			spec: CAIInitialSnapshotRevisionSpec{
				CreationTime:      time.Time{},
				ObservedTime:      observedTime,
				ResourceBody:      sampleBody,
				CreationStateType: creationState,
				SnapshotStateType: snapshotState,
			},
			wantRevisions: []*khifilev6.StagingRevision{
				{
					ChangedTime:  observedTime,
					ResourceBody: sampleBody,
					Principal:    "N/A",
					VerbType:     k8saudit.VerbCreate,
					StateType:    snapshotState,
				},
			},
		},
		{
			name: "stages single create revision when difference is smaller than skew tolerance",
			spec: CAIInitialSnapshotRevisionSpec{
				CreationTime:      observedTime.Add(-500 * time.Millisecond),
				ObservedTime:      observedTime,
				ResourceBody:      sampleBody,
				CreationStateType: creationState,
				SnapshotStateType: snapshotState,
			},
			wantRevisions: []*khifilev6.StagingRevision{
				{
					ChangedTime:  observedTime,
					ResourceBody: sampleBody,
					Principal:    "N/A",
					VerbType:     k8saudit.VerbCreate,
					StateType:    snapshotState,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetPath := &khifilev6.TimelinePath{ID: 1}
			tc.spec.TargetTimeline = targetPath

			dummyLog := &log.Log{}
			cs := khifilev6.NewTimelineChangeSet(dummyLog)
			StageCAIInitialSnapshotRevisions(cs, tc.spec)

			revisions := cs.GetRevisions(targetPath)
			if len(revisions) != len(tc.wantRevisions) {
				t.Fatalf("StageCAIInitialSnapshotRevisions() revision count = %d, want %d", len(revisions), len(tc.wantRevisions))
			}
			for i, wantRev := range tc.wantRevisions {
				gotRev := revisions[i]
				if !gotRev.ChangedTime.Equal(wantRev.ChangedTime) {
					t.Errorf("revision[%d].ChangedTime = %v, want %v", i, gotRev.ChangedTime, wantRev.ChangedTime)
				}
				if diff := cmp.Diff(wantRev.VerbType, gotRev.VerbType, protocmp.Transform()); diff != "" {
					t.Errorf("revision[%d].VerbType mismatch (-want +got):\n%s", i, diff)
				}
				if diff := cmp.Diff(wantRev.StateType, gotRev.StateType, protocmp.Transform()); diff != "" {
					t.Errorf("revision[%d].StateType mismatch (-want +got):\n%s", i, diff)
				}
				if (gotRev.ResourceBody == nil) != (wantRev.ResourceBody == nil) {
					t.Errorf("revision[%d].ResourceBody nil = %v, want %v", i, gotRev.ResourceBody == nil, wantRev.ResourceBody == nil)
				}
			}
		})
	}
}
