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
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
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

// newReadonlyInternPool copies everything interned in pool into a ReadonlyInternPool, as a loaded KHI file would provide.
func newReadonlyInternPool(pool *khifilev6model.InternPool) *khifilev6model.ReadonlyInternPool {
	readonlyPool := khifilev6model.NewReadonlyInternPool()
	var strs []*khifilev6.InternString
	for sRef := range pool.SortedStringRefs() {
		strs = append(strs, sRef.ToProto())
	}
	var fieldSets []*khifilev6.InternFieldPathSet
	for fsRef := range pool.FieldSetRefs() {
		fieldSets = append(fieldSets, fsRef.ToProto())
	}
	var structs []*pb.InternedStruct
	for sRef := range pool.StructRefs() {
		structs = append(structs, sRef.ToProto())
	}
	readonlyPool.IngestChunk(&khifilev6.InterningPoolChunk{
		Strings:       strs,
		FieldPathSets: fieldSets,
		Structs:       structs,
	})
	return readonlyPool
}

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
	readonlyPool := newReadonlyInternPool(pool)

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

	wb := NewWorkbench("wb-test", "test-inspection")
	wb.internPool = readonlyPool
	wb.searchIndex = &SearchIndex{
		InternPool: readonlyPool,
		Timelines:  []*cel.TimelineData{nsTimeline, podTimeline},
		TimelineMap: map[uint32]*cel.TimelineData{
			nsTimeline.ID:  nsTimeline,
			podTimeline.ID: podTimeline,
		},
	}
	return wb
}

var (
	testPodSegments = []TimelineSegment{
		{Type: "Namespace", Name: "default"},
		{Type: "Pod", Name: "nginx"},
	}
	testRevision0 = ResourceRevision{Index: 0, Time: testRevision0Time, Verb: "CREATE", State: "Pending", Principal: testRevisionPrincipalA, LogID: 1}
	testRevision1 = ResourceRevision{Index: 1, Time: testRevision1Time, Verb: "UPDATE", State: "Running", Principal: testRevisionPrincipalB, LogID: 2}
	testRevision2 = ResourceRevision{Index: 2, Time: testRevision2Time, Verb: "DELETE", State: "Deleted", Principal: testRevisionPrincipalA, LogID: 3}
)

func TestGetResourceRevisions(t *testing.T) {
	timePtr := func(t time.Time) *time.Time { return &t }
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

func TestGetResourceRevisions_PagingCoversEveryRevisionOnce(t *testing.T) {
	testCases := []struct {
		name     string
		pageSize int
	}{
		{name: "page size 1", pageSize: 1},
		{name: "page size 2", pageSize: 2},
		{name: "page size larger than the history", pageSize: 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupResourceHistoryTestWorkbench(t)
			var gotIndices []int
			for offset := 0; ; offset += tc.pageSize {
				res, err := wb.GetResourceRevisions(testPodTimelineID, ResourceRevisionsQuery{Offset: offset, Limit: tc.pageSize})
				if err != nil {
					t.Fatalf("GetResourceRevisions() unexpected error = %v", err)
				}
				for _, rev := range res.Revisions {
					gotIndices = append(gotIndices, rev.Index)
				}
				if offset+tc.pageSize >= res.MatchedCount {
					break
				}
			}
			if diff := cmp.Diff([]int{0, 1, 2}, gotIndices); diff != "" {
				t.Errorf("paged revision indices mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetResourceManifest(t *testing.T) {
	testCases := []struct {
		name          string
		selector      RevisionSelector
		wantRevision  ResourceRevision
		wantBody      string
		wantErrTarget error
	}{
		{
			name:         "selects the revision by index",
			selector:     RevisionAtIndex(1),
			wantRevision: testRevision1,
			wantBody:     testRevision1Body,
		},
		{
			name:         "zero value selector selects the revision at index 0",
			selector:     RevisionSelector{},
			wantRevision: testRevision0,
			wantBody:     testRevision0Body,
		},
		{
			name:         "time equal to a change selects that revision",
			selector:     RevisionAtTime(testRevision1Time),
			wantRevision: testRevision1,
			wantBody:     testRevision1Body,
		},
		{
			name:         "time between changes selects the revision effective at that time",
			selector:     RevisionAtTime(testRevision1Time.Add(500 * time.Second)),
			wantRevision: testRevision1,
			wantBody:     testRevision1Body,
		},
		{
			name:         "time after the last change selects the last revision",
			selector:     RevisionAtTime(testRevision2Time.Add(time.Hour)),
			wantRevision: testRevision2,
			wantBody:     "",
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
			got, err := wb.GetResourceManifest(testPodTimelineID, tc.selector, 0)
			if tc.wantErrTarget != nil {
				if !errors.Is(err, tc.wantErrTarget) {
					t.Fatalf("GetResourceManifest() err = %v, want %v", err, tc.wantErrTarget)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetResourceManifest() unexpected error = %v", err)
			}
			if diff := cmp.Diff(testPodSegments, got.Segments); diff != "" {
				t.Errorf("Segments mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantRevision, got.Revision); diff != "" {
				t.Errorf("Revision mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantBody, got.Body.Content); diff != "" {
				t.Errorf("Body.Content mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetResourceManifest_ByteOffset(t *testing.T) {
	testCases := []struct {
		name       string
		byteOffset int
		wantBody   BodyChunk
	}{
		{
			name:       "offset at a line boundary returns the rest of the manifest",
			byteOffset: len("kind: Pod\n"),
			wantBody: BodyChunk{
				Content:    "status:\n  phase: Running\n  ready: true\n",
				TotalBytes: len(testRevision1Body),
			},
		},
		{
			name:       "offset at the end returns no content",
			byteOffset: len(testRevision1Body),
			wantBody: BodyChunk{
				TotalBytes: len(testRevision1Body),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := setupResourceHistoryTestWorkbench(t)
			got, err := wb.GetResourceManifest(testPodTimelineID, RevisionAtIndex(1), tc.byteOffset)
			if err != nil {
				t.Fatalf("GetResourceManifest() unexpected error = %v", err)
			}
			if diff := cmp.Diff(tc.wantBody, got.Body); diff != "" {
				t.Errorf("Body mismatch (-want +got):\n%s", diff)
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

func TestGetResourceDiff_ChangesMatchRevisionList(t *testing.T) {
	wb := setupResourceHistoryTestWorkbench(t)
	list, err := wb.GetResourceRevisions(testPodTimelineID, ResourceRevisionsQuery{Limit: 50})
	if err != nil {
		t.Fatalf("GetResourceRevisions() unexpected error = %v", err)
	}
	for _, rev := range list.Revisions {
		t.Run(revisionLabel(rev.Index), func(t *testing.T) {
			// A byteOffset past the end leaves no diff text, which must not affect the change counts.
			got, err := wb.GetResourceDiff(testPodTimelineID, rev.Index, 1<<20)
			if err != nil {
				t.Fatalf("GetResourceDiff() unexpected error = %v", err)
			}
			if diff := cmp.Diff(rev.Changes, got.Changes); diff != "" {
				t.Errorf("GetResourceDiff() changes mismatch against GetResourceRevisions() (-want +got):\n%s", diff)
			}
		})
	}
}
