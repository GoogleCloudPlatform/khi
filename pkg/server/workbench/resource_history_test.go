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

package workbench

import (
	"errors"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/google/go-cmp/cmp"
)

const (
	testPodTimelineID      uint32 = 10
	testRevision0Body             = "kind: Pod\nstatus:\n  phase: Pending\n"
	testRevision1Body             = "kind: Pod\nstatus:\n  phase: Running\n  ready: true\n"
	testRevisionPrincipalA        = "user-a"
	testRevisionPrincipalB        = "user-b"
)

var (
	testRevision0Time = time.Unix(1000, 0).UTC()
	testRevision1Time = time.Unix(2000, 0).UTC()
	testRevision2Time = time.Unix(3000, 0).UTC()
)

func internTestYAML(t *testing.T, pool *khifilev6model.InternPool, yaml string) uint32 {
	t.Helper()
	node, err := structured.FromYAML(yaml)
	if err != nil {
		t.Fatalf("failed to parse yaml: %v", err)
	}
	ref, err := khifilev6model.ToInternedStruct(node, pool)
	if err != nil {
		t.Fatalf("failed to intern struct: %v", err)
	}
	return ref.ID()
}

// setupResourceHistoryTestWorkbench builds a Pod timeline with three revisions: a creation, an update, and a deletion without a body.
func setupResourceHistoryTestWorkbench(t *testing.T) *Workbench {
	t.Helper()
	pool := khifilev6model.NewTestInternPool(id.NewGenerator())
	principalA := pool.InternString(testRevisionPrincipalA).ID()
	principalB := pool.InternString(testRevisionPrincipalB).ID()
	body0 := internTestYAML(t, pool, testRevision0Body)
	body1 := internTestYAML(t, pool, testRevision1Body)

	nsTimeline := &cel.TimelineData{
		ID:           9,
		ChildrenIDs:  []uint32{testPodTimelineID},
		Name:         "default",
		TimelineType: "Namespace",
	}
	podTimeline := &cel.TimelineData{
		ID:           testPodTimelineID,
		ParentID:     9,
		Name:         "nginx",
		TimelineType: "Pod",
		Revisions: []cel.RevisionInfo{
			{LogID: 1, ChangedTime: testRevision0Time.UnixNano(), PrincipalStringID: principalA, Verb: "CREATE", State: "Pending", ResourceBodyStructID: body0},
			{LogID: 2, ChangedTime: testRevision1Time.UnixNano(), PrincipalStringID: principalB, Verb: "UPDATE", State: "Running", ResourceBodyStructID: body1},
			{LogID: 3, ChangedTime: testRevision2Time.UnixNano(), PrincipalStringID: principalA, Verb: "DELETE", State: "Deleted"},
		},
	}

	wb := newTestIndexedWorkbench(nil, []*cel.TimelineData{nsTimeline, podTimeline}, nil)
	ingestTestPool(wb, pool)
	return wb
}

var (
	testPodSegments = []TimelineSegment{
		{Type: "Namespace", Name: "default"},
		{Type: "Pod", Name: "nginx"},
	}
	testRevision0 = ResourceRevision{Index: 0, Time: testRevision0Time, Verb: "CREATE", State: "Pending", Principal: testRevisionPrincipalA, LogID: 1, HasBody: true}
	testRevision1 = ResourceRevision{Index: 1, Time: testRevision1Time, Verb: "UPDATE", State: "Running", Principal: testRevisionPrincipalB, LogID: 2, HasBody: true}
	testRevision2 = ResourceRevision{Index: 2, Time: testRevision2Time, Verb: "DELETE", State: "Deleted", Principal: testRevisionPrincipalA, LogID: 3, HasBody: false}
)

func TestGetResourceRevisions(t *testing.T) {
	testCases := []struct {
		name          string
		timelineID    uint32
		query         ResourceRevisionsQuery
		want          *ResourceRevisionsResult
		wantErrTarget error
	}{
		{
			name:       "returns all revisions with change counts from each previous revision",
			timelineID: testPodTimelineID,
			query:      ResourceRevisionsQuery{Limit: 50},
			want: &ResourceRevisionsResult{
				TimelineID:   testPodTimelineID,
				Segments:     testPodSegments,
				MatchedCount: 3,
				Revisions: []ResourceRevisionSummary{
					{ResourceRevision: testRevision0, Changes: LineChangeCount{Added: 3}},
					{ResourceRevision: testRevision1, Changes: LineChangeCount{Added: 2, Deleted: 1}},
					{ResourceRevision: testRevision2, Changes: LineChangeCount{Deleted: 4}},
				},
			},
		},
		{
			name:       "time range keeps global indices and compares against the revision before the range",
			timelineID: testPodTimelineID,
			query: ResourceRevisionsQuery{
				StartTime: timePtr(testRevision0Time.Add(time.Second)),
				EndTime:   timePtr(testRevision2Time),
				Limit:     50,
			},
			want: &ResourceRevisionsResult{
				TimelineID:   testPodTimelineID,
				Segments:     testPodSegments,
				MatchedCount: 2,
				Revisions: []ResourceRevisionSummary{
					{ResourceRevision: testRevision1, Changes: LineChangeCount{Added: 2, Deleted: 1}},
					{ResourceRevision: testRevision2, Changes: LineChangeCount{Deleted: 4}},
				},
			},
		},
		{
			name:       "start time equal to a change includes that revision",
			timelineID: testPodTimelineID,
			query: ResourceRevisionsQuery{
				StartTime: timePtr(testRevision1Time),
				Limit:     50,
			},
			want: &ResourceRevisionsResult{
				TimelineID:   testPodTimelineID,
				Segments:     testPodSegments,
				MatchedCount: 2,
				Revisions: []ResourceRevisionSummary{
					{ResourceRevision: testRevision1, Changes: LineChangeCount{Added: 2, Deleted: 1}},
					{ResourceRevision: testRevision2, Changes: LineChangeCount{Deleted: 4}},
				},
			},
		},
		{
			name:       "offset counts from the first revision in the time range",
			timelineID: testPodTimelineID,
			query: ResourceRevisionsQuery{
				StartTime: timePtr(testRevision0Time.Add(time.Second)),
				Offset:    1,
				Limit:     1,
			},
			want: &ResourceRevisionsResult{
				TimelineID:   testPodTimelineID,
				Segments:     testPodSegments,
				MatchedCount: 2,
				Revisions: []ResourceRevisionSummary{
					{ResourceRevision: testRevision2, Changes: LineChangeCount{Deleted: 4}},
				},
			},
		},
		{
			name:       "offset and limit select a window within the matched revisions",
			timelineID: testPodTimelineID,
			query:      ResourceRevisionsQuery{Offset: 1, Limit: 1},
			want: &ResourceRevisionsResult{
				TimelineID:   testPodTimelineID,
				Segments:     testPodSegments,
				MatchedCount: 3,
				Revisions: []ResourceRevisionSummary{
					{ResourceRevision: testRevision1, Changes: LineChangeCount{Added: 2, Deleted: 1}},
				},
			},
		},
		{
			name:       "offset past the matched revisions returns no revisions",
			timelineID: testPodTimelineID,
			query:      ResourceRevisionsQuery{Offset: 5, Limit: 50},
			want: &ResourceRevisionsResult{
				TimelineID:   testPodTimelineID,
				Segments:     testPodSegments,
				MatchedCount: 3,
			},
		},
		{
			name:       "timeline without revisions returns no revisions",
			timelineID: 9,
			query:      ResourceRevisionsQuery{Limit: 50},
			want: &ResourceRevisionsResult{
				TimelineID: 9,
				Segments:   testPodSegments[:1],
			},
		},
		{
			name:          "non-existent timeline returns ErrTimelineNotFound",
			timelineID:    999,
			query:         ResourceRevisionsQuery{Limit: 50},
			wantErrTarget: ErrTimelineNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupResourceHistoryTestWorkbench(t)
			got, err := wb.GetResourceRevisions(tc.timelineID, tc.query)
			if tc.wantErrTarget != nil {
				if !errors.Is(err, tc.wantErrTarget) {
					t.Fatalf("GetResourceRevisions() err = %v, want %v", err, tc.wantErrTarget)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetResourceRevisions() unexpected error = %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GetResourceRevisions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetResourceManifest(t *testing.T) {
	testCases := []struct {
		name          string
		selector      RevisionSelector
		byteOffset    int
		want          *ResourceManifest
		wantErrTarget error
	}{
		{
			name:     "selects the revision by index",
			selector: RevisionAtIndex(1),
			want: &ResourceManifest{
				TimelineID: testPodTimelineID,
				Segments:   testPodSegments,
				Revision:   testRevision1,
				Body:       BodyChunk{Content: testRevision1Body, TotalBytes: len(testRevision1Body)},
			},
		},
		{
			name:     "zero value selector selects the revision at index 0",
			selector: RevisionSelector{},
			want: &ResourceManifest{
				TimelineID: testPodTimelineID,
				Segments:   testPodSegments,
				Revision:   testRevision0,
				Body:       BodyChunk{Content: testRevision0Body, TotalBytes: len(testRevision0Body)},
			},
		},
		{
			name:     "time equal to a change selects that revision",
			selector: RevisionAtTime(testRevision1Time),
			want: &ResourceManifest{
				TimelineID: testPodTimelineID,
				Segments:   testPodSegments,
				Revision:   testRevision1,
				Body:       BodyChunk{Content: testRevision1Body, TotalBytes: len(testRevision1Body)},
			},
		},
		{
			name:     "time between changes selects the revision effective at that time",
			selector: RevisionAtTime(testRevision1Time.Add(500 * time.Second)),
			want: &ResourceManifest{
				TimelineID: testPodTimelineID,
				Segments:   testPodSegments,
				Revision:   testRevision1,
				Body:       BodyChunk{Content: testRevision1Body, TotalBytes: len(testRevision1Body)},
			},
		},
		{
			name:     "time after the last change selects the last revision",
			selector: RevisionAtTime(testRevision2Time.Add(time.Hour)),
			want: &ResourceManifest{
				TimelineID: testPodTimelineID,
				Segments:   testPodSegments,
				Revision:   testRevision2,
			},
		},
		{
			name:       "byteOffset at a line boundary returns the rest of the manifest",
			selector:   RevisionAtIndex(1),
			byteOffset: len("kind: Pod\n"),
			want: &ResourceManifest{
				TimelineID: testPodTimelineID,
				Segments:   testPodSegments,
				Revision:   testRevision1,
				Body: BodyChunk{
					Content:    "status:\n  phase: Running\n  ready: true\n",
					TotalBytes: len(testRevision1Body),
				},
			},
		},
		{
			name:       "byteOffset at the end returns no content",
			selector:   RevisionAtIndex(1),
			byteOffset: len(testRevision1Body),
			want: &ResourceManifest{
				TimelineID: testPodTimelineID,
				Segments:   testPodSegments,
				Revision:   testRevision1,
				Body: BodyChunk{
					TotalBytes: len(testRevision1Body),
				},
			},
		},
		{
			name:          "time before the first change returns ErrRevisionNotFound",
			selector:      RevisionAtTime(testRevision0Time.Add(-time.Second)),
			wantErrTarget: ErrRevisionNotFound,
		},
		{
			name:          "out of range index returns ErrRevisionNotFound",
			selector:      RevisionAtIndex(3),
			wantErrTarget: ErrRevisionNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupResourceHistoryTestWorkbench(t)
			got, err := wb.GetResourceManifest(testPodTimelineID, tc.selector, tc.byteOffset, false)
			if tc.wantErrTarget != nil {
				if !errors.Is(err, tc.wantErrTarget) {
					t.Fatalf("GetResourceManifest() err = %v, want %v", err, tc.wantErrTarget)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetResourceManifest() unexpected error = %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GetResourceManifest() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetResourceManifest_IncludeManagedFields(t *testing.T) {
	const fullManifest = "apiVersion: v1\nkind: Pod\nmetadata:\n  managedFields: \n  - apiVersion: v1\n    fieldsType: FieldsV1\n    manager: kubelet\n  name: nginx\n  namespace: default\nstatus:\n  phase: Running\n"
	const strippedManifest = "apiVersion: v1\nkind: Pod\nmetadata:\n  name: nginx\n  namespace: default\nstatus:\n  phase: Running\n"

	pool := khifilev6model.NewTestInternPool(id.NewGenerator())
	principalA := pool.InternString(testRevisionPrincipalA).ID()
	bodyID := internTestYAML(t, pool, fullManifest)

	podTimeline := &cel.TimelineData{
		ID:           testPodTimelineID,
		Name:         "nginx",
		TimelineType: "Pod",
		Revisions: []cel.RevisionInfo{
			{LogID: 1, ChangedTime: testRevision0Time.UnixNano(), PrincipalStringID: principalA, Verb: "CREATE", State: "Running", ResourceBodyStructID: bodyID},
		},
	}
	wb := newTestIndexedWorkbench(nil, []*cel.TimelineData{podTimeline}, nil)
	ingestTestPool(wb, pool)

	wantRevision := ResourceRevision{
		Index:     0,
		Time:      testRevision0Time,
		Verb:      "CREATE",
		State:     "Running",
		Principal: testRevisionPrincipalA,
		LogID:     1,
		HasBody:   true,
	}
	wantSegments := []TimelineSegment{{Type: "Pod", Name: "nginx"}}

	testCases := []struct {
		name                 string
		includeManagedFields bool
		byteOffset           int
		wantBody             BodyChunk
	}{
		{
			name:                 "omits metadata.managedFields by default",
			includeManagedFields: false,
			wantBody: BodyChunk{
				Content:    strippedManifest,
				TotalBytes: len(strippedManifest),
			},
		},
		{
			name:                 "includes metadata.managedFields when requested",
			includeManagedFields: true,
			wantBody: BodyChunk{
				Content:    fullManifest,
				TotalBytes: len(fullManifest),
			},
		},
		{
			name:                 "byteOffset applies to the stripped manifest when includeManagedFields is false",
			includeManagedFields: false,
			byteOffset:           len("apiVersion: v1\nkind: Pod\n"),
			wantBody: BodyChunk{
				Content:    "metadata:\n  name: nginx\n  namespace: default\nstatus:\n  phase: Running\n",
				TotalBytes: len(strippedManifest),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := wb.GetResourceManifest(testPodTimelineID, RevisionAtIndex(0), tc.byteOffset, tc.includeManagedFields)
			if err != nil {
				t.Fatalf("GetResourceManifest() unexpected error = %v", err)
			}
			want := &ResourceManifest{
				TimelineID: testPodTimelineID,
				Segments:   wantSegments,
				Revision:   wantRevision,
				Body:       tc.wantBody,
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("GetResourceManifest() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestStripManagedFields(t *testing.T) {
	testCases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "removes metadata.managedFields before sibling metadata keys",
			in:   "kind: Pod\nmetadata:\n  managedFields:\n  - manager: kubectl\n    operation: Apply\n  name: pod-1\nspec: {}\n",
			want: "kind: Pod\nmetadata:\n  name: pod-1\nspec: {}\n",
		},
		{
			name: "removes metadata.managedFields at the end of metadata",
			in:   "kind: Pod\nmetadata:\n  name: pod-1\n  managedFields:\n  - manager: kubectl\nspec: {}\n",
			want: "kind: Pod\nmetadata:\n  name: pod-1\nspec: {}\n",
		},
		{
			name: "removes empty metadata block when managedFields was its only child",
			in:   "kind: Pod\nmetadata:\n  managedFields:\n  - manager: kubectl\nstatus:\n  phase: Running\n",
			want: "kind: Pod\nstatus:\n  phase: Running\n",
		},
		{
			name: "removes empty metadata block at EOF when managedFields was its only child",
			in:   "kind: Pod\nmetadata:\n  managedFields:\n  - manager: kubectl\n",
			want: "kind: Pod\n",
		},
		{
			name: "preserves non-top-level managedFields under spec",
			in:   "kind: Custom\nmetadata:\n  name: c1\nspec:\n  managedFields:\n  - keep: true\n",
			want: "kind: Custom\nmetadata:\n  name: c1\nspec:\n  managedFields:\n  - keep: true\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := stripManagedFields(tc.in)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("stripManagedFields() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetResourceDiff(t *testing.T) {
	testCases := []struct {
		name          string
		revisionIndex int
		byteOffset    int
		want          *ResourceDiff
		wantErrTarget error
	}{
		{
			name:          "initial revision is compared against none and reports every line as added",
			revisionIndex: 0,
			want: &ResourceDiff{
				TimelineID: testPodTimelineID,
				Segments:   testPodSegments,
				Revision:   testRevision0,
				Changes:    LineChangeCount{Added: 3},
				Diff: BodyChunk{
					Content:    "--- none\n+++ revision 0\n@@ -0,0 +1,3 @@\n+kind: Pod\n+status:\n+  phase: Pending\n",
					TotalBytes: 78,
				},
			},
		},
		{
			name:          "later revision is compared against the previous revision",
			revisionIndex: 1,
			want: &ResourceDiff{
				TimelineID: testPodTimelineID,
				Segments:   testPodSegments,
				Revision:   testRevision1,
				Previous:   &testRevision0,
				Changes:    LineChangeCount{Added: 2, Deleted: 1},
				Diff: BodyChunk{
					Content:    "--- revision 0\n+++ revision 1\n@@ -1,3 +1,4 @@\n kind: Pod\n status:\n-  phase: Pending\n+  phase: Running\n+  ready: true\n",
					TotalBytes: 117,
				},
			},
		},
		{
			name:          "byteOffset slices the diff text but keeps the change counts of the whole diff",
			revisionIndex: 1,
			byteOffset:    len("--- revision 0\n+++ revision 1\n"),
			want: &ResourceDiff{
				TimelineID: testPodTimelineID,
				Segments:   testPodSegments,
				Revision:   testRevision1,
				Previous:   &testRevision0,
				Changes:    LineChangeCount{Added: 2, Deleted: 1},
				Diff: BodyChunk{
					Content:    "@@ -1,3 +1,4 @@\n kind: Pod\n status:\n-  phase: Pending\n+  phase: Running\n+  ready: true\n",
					TotalBytes: 117,
				},
			},
		},
		{
			name:          "deletion revision with no body reports every previous line as deleted",
			revisionIndex: 2,
			want: &ResourceDiff{
				TimelineID: testPodTimelineID,
				Segments:   testPodSegments,
				Revision:   testRevision2,
				Previous:   &testRevision1,
				Changes:    LineChangeCount{Deleted: 4},
				Diff: BodyChunk{
					Content:    "--- revision 1\n+++ revision 2\n@@ -1,4 +0,0 @@\n-kind: Pod\n-status:\n-  phase: Running\n-  ready: true\n",
					TotalBytes: 99,
				},
			},
		},
		{
			name:          "out of range index returns ErrRevisionNotFound",
			revisionIndex: 3,
			wantErrTarget: ErrRevisionNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupResourceHistoryTestWorkbench(t)
			got, err := wb.GetResourceDiff(testPodTimelineID, tc.revisionIndex, tc.byteOffset)
			if tc.wantErrTarget != nil {
				if !errors.Is(err, tc.wantErrTarget) {
					t.Fatalf("GetResourceDiff() err = %v, want %v", err, tc.wantErrTarget)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetResourceDiff() unexpected error = %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GetResourceDiff() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
