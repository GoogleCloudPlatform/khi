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
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
)

// ErrRevisionNotFound indicates that the requested revision does not exist on the timeline.
var ErrRevisionNotFound = errors.New("revision not found")

// ResourceRevision describes a single revision of a resource timeline without its manifest body.
type ResourceRevision struct {
	// Index is the 0-based position of the revision among all revisions of the timeline, oldest first.
	Index     int
	Time      time.Time
	Verb      string
	State     string
	Principal string
	LogID     uint32
}

// ResourceRevisionSummary is a ResourceRevision with the line change counts from its previous revision.
type ResourceRevisionSummary struct {
	ResourceRevision
	Changes LineChangeCount
}

// ResourceRevisionsQuery selects a window of revisions on a timeline.
type ResourceRevisionsQuery struct {
	// StartTime is the inclusive lower bound of the revision time, or nil for no bound.
	StartTime *time.Time
	// EndTime is the inclusive upper bound of the revision time, or nil for no bound.
	EndTime *time.Time
	// Offset is the number of revisions within the time range to skip.
	Offset int
	// Limit is the maximum number of revisions to return.
	Limit int
}

// ResourceRevisionsResult contains a window of revisions within the requested time range.
type ResourceRevisionsResult struct {
	TimelineID uint32
	Segments   []TimelineSegment
	// MatchedCount is the number of revisions within the time range across all windows.
	MatchedCount int
	Revisions    []ResourceRevisionSummary
}

// RevisionSelector identifies a revision either by its index or by a point in time.
// Exactly one of Index and Time must be set.
type RevisionSelector struct {
	Index *int
	// Time selects the revision effective at that time, i.e. the latest revision changed at or before it.
	Time *time.Time
}

// ResourceManifest contains the selected revision and its truncated manifest YAML.
type ResourceManifest struct {
	TimelineID uint32
	Segments   []TimelineSegment
	Revision   ResourceRevision
	Body       BodyChunk
}

// ResourceDiff contains the unified diff between a revision and its previous revision.
type ResourceDiff struct {
	TimelineID uint32
	Segments   []TimelineSegment
	Revision   ResourceRevision
	// Previous is the revision before Revision, or nil when Revision is the initial revision.
	Previous *ResourceRevision
	// Changes counts the lines over the whole diff, even when Diff is truncated.
	Changes LineChangeCount
	Diff    BodyChunk
}

// GetResourceRevisions returns the revisions of a timeline within the query time range, oldest first,
// together with the line change counts from each previous revision.
// Bodies are read only for the returned window so that long revision histories stay cheap to page through.
func (w *Workbench) GetResourceRevisions(timelineID uint32, query ResourceRevisionsQuery) (*ResourceRevisionsResult, error) {
	index, _, err := w.readyIndex()
	if err != nil {
		return nil, err
	}
	tl, segments, err := lookupTimeline(index, timelineID)
	if err != nil {
		return nil, err
	}

	matchedStart, matchedEnd := revisionIndexRange(tl.Revisions, query.StartTime, query.EndTime)
	matched := matchedEnd - matchedStart
	windowStart := matchedStart + min(max(query.Offset, 0), matched)
	windowEnd := min(windowStart+max(query.Limit, 0), matchedEnd)

	result := &ResourceRevisionsResult{
		TimelineID:   timelineID,
		Segments:     segments,
		MatchedCount: matched,
	}
	if windowStart == windowEnd {
		return result, nil
	}

	bodies, err := w.readRevisionBodies(tl.Revisions[max(windowStart-1, 0):windowEnd])
	if err != nil {
		return nil, err
	}
	result.Revisions = make([]ResourceRevisionSummary, 0, windowEnd-windowStart)
	for i := windowStart; i < windowEnd; i++ {
		var prevBody string
		if i > 0 {
			prevBody = bodies[tl.Revisions[i-1].ResourceBodyStructID]
		}
		result.Revisions = append(result.Revisions, ResourceRevisionSummary{
			ResourceRevision: toResourceRevision(index, tl.Revisions, i),
			Changes:          countLineChanges(prevBody, bodies[tl.Revisions[i].ResourceBodyStructID]),
		})
	}
	return result, nil
}

// GetResourceManifest returns the manifest YAML of the selected revision, sliced from byteOffset.
func (w *Workbench) GetResourceManifest(timelineID uint32, selector RevisionSelector, byteOffset int) (*ResourceManifest, error) {
	index, _, err := w.readyIndex()
	if err != nil {
		return nil, err
	}
	tl, segments, err := lookupTimeline(index, timelineID)
	if err != nil {
		return nil, err
	}
	revIdx, err := selectRevision(tl.Revisions, selector)
	if err != nil {
		return nil, err
	}

	bodies, err := w.readRevisionBodies(tl.Revisions[revIdx : revIdx+1])
	if err != nil {
		return nil, err
	}
	return &ResourceManifest{
		TimelineID: timelineID,
		Segments:   segments,
		Revision:   toResourceRevision(index, tl.Revisions, revIdx),
		Body:       truncateBody(bodies[tl.Revisions[revIdx].ResourceBodyStructID], byteOffset, bodyByteLimit),
	}, nil
}

// GetResourceDiff returns the unified diff from the previous revision to the revision at revisionIndex, sliced from byteOffset.
// The initial revision is compared against an empty manifest, so all of its lines are reported as added.
func (w *Workbench) GetResourceDiff(timelineID uint32, revisionIndex, byteOffset int) (*ResourceDiff, error) {
	index, _, err := w.readyIndex()
	if err != nil {
		return nil, err
	}
	tl, segments, err := lookupTimeline(index, timelineID)
	if err != nil {
		return nil, err
	}
	if revisionIndex < 0 || revisionIndex >= len(tl.Revisions) {
		return nil, fmt.Errorf("%w: index %d of timeline %d", ErrRevisionNotFound, revisionIndex, timelineID)
	}

	bodies, err := w.readRevisionBodies(tl.Revisions[max(revisionIndex-1, 0) : revisionIndex+1])
	if err != nil {
		return nil, err
	}
	result := &ResourceDiff{
		TimelineID: timelineID,
		Segments:   segments,
		Revision:   toResourceRevision(index, tl.Revisions, revisionIndex),
	}
	prevBody := ""
	prevLabel := "none"
	if revisionIndex > 0 {
		prev := toResourceRevision(index, tl.Revisions, revisionIndex-1)
		result.Previous = &prev
		prevBody = bodies[tl.Revisions[revisionIndex-1].ResourceBodyStructID]
		prevLabel = revisionLabel(revisionIndex - 1)
	}
	diff := computeUnifiedDiff(prevBody, bodies[tl.Revisions[revisionIndex].ResourceBodyStructID], prevLabel, revisionLabel(revisionIndex))
	result.Changes = diff.Changes
	result.Diff = truncateBody(diff.Text, byteOffset, bodyByteLimit)
	return result, nil
}

// readRevisionBodies returns the manifest YAMLs of revs keyed by their resource body struct IDs.
func (w *Workbench) readRevisionBodies(revs []cel.RevisionInfo) (map[uint32]string, error) {
	structIDs := make([]uint32, len(revs))
	for i, rev := range revs {
		structIDs[i] = rev.ResourceBodyStructID
	}
	return w.ReadStructYAMLs(structIDs)
}

// revisionIndexRange returns the half-open index range [startIdx, endIdx) of revisions whose ChangedTime falls in [start, end].
func revisionIndexRange(revs []cel.RevisionInfo, start, end *time.Time) (int, int) {
	startIdx := 0
	if start != nil {
		startNs := start.UnixNano()
		startIdx = sort.Search(len(revs), func(i int) bool { return revs[i].ChangedTime >= startNs })
	}
	endIdx := len(revs)
	if end != nil {
		endNs := end.UnixNano()
		endIdx = sort.Search(len(revs), func(i int) bool { return revs[i].ChangedTime > endNs })
	}
	return startIdx, max(startIdx, endIdx)
}

// selectRevision resolves selector to a revision index in revs, which must be sorted by ChangedTime.
func selectRevision(revs []cel.RevisionInfo, selector RevisionSelector) (int, error) {
	if selector.Index != nil {
		idx := *selector.Index
		if idx < 0 || idx >= len(revs) {
			return 0, fmt.Errorf("%w: index %d", ErrRevisionNotFound, idx)
		}
		return idx, nil
	}
	tNs := selector.Time.UnixNano()
	// The first revision changed after t is one past the revision effective at t.
	idx := sort.Search(len(revs), func(i int) bool { return revs[i].ChangedTime > tNs }) - 1
	if idx < 0 {
		return 0, fmt.Errorf("%w: no revision at or before %s", ErrRevisionNotFound, selector.Time.UTC().Format(time.RFC3339))
	}
	return idx, nil
}

func toResourceRevision(index *SearchIndex, revs []cel.RevisionInfo, i int) ResourceRevision {
	rev := revs[i]
	var changedTime time.Time
	if rev.ChangedTime > 0 {
		changedTime = time.Unix(0, rev.ChangedTime).UTC()
	}
	return ResourceRevision{
		Index:     i,
		Time:      changedTime,
		Verb:      rev.Verb,
		State:     rev.State,
		Principal: index.InternPool.ResolveStringFromID(rev.PrincipalStringID),
		LogID:     rev.LogID,
	}
}

func revisionLabel(i int) string {
	return "revision " + strconv.Itoa(i)
}
