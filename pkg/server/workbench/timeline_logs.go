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
	"cmp"
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/RoaringBitmap/roaring/v2"
)

var (
	// ErrTimelineNotFound indicates that the requested timeline ID does not exist in the search index.
	ErrTimelineNotFound = errors.New("timeline not found")
	// ErrLogNotFound indicates that the requested log ID does not exist in the search index.
	ErrLogNotFound = errors.New("log not found")
)

// maxOtherTimelines is the maximum number of other linked timelines returned by GetTimelineLogs.
const maxOtherTimelines = 20

// TimelineLogFilter specifies log CEL and time range filter parameters for reading logs on a single timeline.
type TimelineLogFilter struct {
	LogQuery  string     `json:"logQuery,omitempty" jsonschema:"CEL expression to filter logs."`
	StartTime *time.Time `json:"startTime,omitempty" jsonschema:"Inclusive start of the time range in RFC3339 format."`
	EndTime   *time.Time `json:"endTime,omitempty" jsonschema:"Inclusive end of the time range in RFC3339 format."`
}

// TimelineLogEntry represents a single log entry on a timeline along with other timeline IDs linked to it.
type TimelineLogEntry struct {
	LogID            uint32
	Time             time.Time
	Severity         *khifilev6.Severity
	LogType          string
	Summary          string
	OtherTimelineIDs []uint32
}

// TimelineLogsResult contains the filtered logs for a timeline, applied filter settings, and other linked timelines.
type TimelineLogsResult struct {
	TimelineID              uint32
	Segments                []TimelineSegment
	Applied                 AppliedFilter
	TotalOtherTimelineCount int
	OtherTimelines          []OtherTimeline
	Logs                    []TimelineLogEntry
}

// LinkedTimeline represents a timeline linked to a log entry.
type LinkedTimeline struct {
	TimelineID uint32
	Segments   []TimelineSegment
}

// OtherTimeline is a timeline other than the queried timeline that is linked to some of the matched logs.
type OtherTimeline struct {
	LinkedTimeline
	// MatchedLogCount is the number of matched logs linked to this timeline.
	MatchedLogCount int
}

// LogDetail contains metadata, linked timelines, and the truncated YAML body for a single log entry.
type LogDetail struct {
	LogID           uint32
	Time            time.Time
	Severity        *khifilev6.Severity
	LogType         string
	Summary         string
	LinkedTimelines []LinkedTimeline
	Body            BodyChunk
}

// GetTimelineLogs filters logs on the specified timeline and aggregates other timelines linked to those logs across all matched entries.
// OtherTimelines keeps at most maxOtherTimelines timelines with the most matched logs.
func (w *Workbench) GetTimelineLogs(ctx context.Context, timelineID uint32, filter TimelineLogFilter) (*TimelineLogsResult, error) {
	index, styles, err := w.readyIndex()
	if err != nil {
		return nil, err
	}
	tl, segments, err := lookupTimeline(index, timelineID)
	if err != nil {
		return nil, err
	}

	excludeNoLogs := true
	fullFilter := Filter{
		TimelineQuery:               BuildTimelineQuery(segments),
		LogQuery:                    filter.LogQuery,
		StartTime:                   filter.StartTime,
		EndTime:                     filter.EndTime,
		ExcludeTimelinesWithoutLogs: &excludeNoLogs,
	}

	filterOut, err := w.ExecuteFilter(ctx, fullFilter)
	if err != nil {
		return nil, err
	}

	matchedLogBitmap := collectTimelineMatchedLogIDs(tl, filterOut)
	matchedLogs := collectSortedMatchedLogs(index, matchedLogBitmap)

	entries, otherTimelines := buildTimelineLogEntriesAndOtherTimelines(matchedLogs, timelineID, index, styles)
	totalOther := len(otherTimelines)
	if len(otherTimelines) > maxOtherTimelines {
		otherTimelines = otherTimelines[:maxOtherTimelines]
	}
	for i := range otherTimelines {
		otherTimelines[i].Segments, _ = index.TimelineSegments(otherTimelines[i].TimelineID)
	}

	return &TimelineLogsResult{
		TimelineID:              timelineID,
		Segments:                segments,
		Applied:                 filterOut.Applied,
		TotalOtherTimelineCount: totalOther,
		OtherTimelines:          otherTimelines,
		Logs:                    entries,
	}, nil
}

// readyIndex returns the search index and the style maps, or an error when the workbench is closed or not indexed yet.
func (w *Workbench) readyIndex() (*SearchIndex, *styleMaps, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.closed {
		return nil, nil, ErrWorkbenchClosed
	}
	if w.searchIndex == nil {
		return nil, nil, fmt.Errorf("search index is not ready")
	}
	return w.searchIndex, w.styles, nil
}

// lookupTimeline returns the timeline and its hierarchy segments, or ErrTimelineNotFound when timelineID is unknown.
func lookupTimeline(index *SearchIndex, timelineID uint32) (*cel.TimelineData, []TimelineSegment, error) {
	segments, ok := index.TimelineSegments(timelineID)
	if !ok {
		return nil, nil, fmt.Errorf("%w: %d", ErrTimelineNotFound, timelineID)
	}
	return index.TimelineMap[timelineID], segments, nil
}

// collectTimelineMatchedLogIDs returns the IDs of logs on tl that passed the filter and fall in the applied time range.
// A zero applied bound leaves that side of the range open.
func collectTimelineMatchedLogIDs(tl *cel.TimelineData, filterOut *FilterOutput) *roaring.Bitmap {
	startNs := int64(math.MinInt64)
	if !filterOut.Applied.StartTime.IsZero() {
		startNs = filterOut.Applied.StartTime.UnixNano()
	}
	endNs := int64(math.MaxInt64)
	if !filterOut.Applied.EndTime.IsZero() {
		endNs = filterOut.Applied.EndTime.UnixNano()
	}

	matched := roaring.NewBitmap()
	tl.ForEachLogIDInRange(startNs, endNs, func(logID uint32) bool {
		if logID > 0 && filterOut.LogIDs.Contains(logID) {
			matched.Add(logID)
		}
		return true
	})
	return matched
}

func buildTimelineLogEntriesAndOtherTimelines(
	matchedLogs []*cel.LogData,
	timelineID uint32,
	index *SearchIndex,
	styles *styleMaps,
) ([]TimelineLogEntry, []OtherTimeline) {
	if len(matchedLogs) == 0 {
		return nil, nil
	}

	entries := make([]TimelineLogEntry, 0, len(matchedLogs))
	otherCounts := make(map[uint32]int)
	for _, l := range matchedLogs {
		var otherIDs []uint32
		for _, linkedID := range index.GetTimelineIDsForLog(l.ID) {
			if linkedID == timelineID {
				continue
			}
			otherIDs = append(otherIDs, linkedID)
			otherCounts[linkedID]++
		}
		slices.Sort(otherIDs)

		entries = append(entries, TimelineLogEntry{
			LogID:            l.ID,
			Time:             logTime(l),
			Severity:         styles.severityMap[l.SeverityTypeID],
			LogType:          index.StyleResolver.ResolveLogType(l.LogTypeID),
			Summary:          index.InternPool.ResolveStringFromID(l.SummaryStringID),
			OtherTimelineIDs: otherIDs,
		})
	}

	otherTimelines := make([]OtherTimeline, 0, len(otherCounts))
	for id, count := range otherCounts {
		otherTimelines = append(otherTimelines, OtherTimeline{
			LinkedTimeline:  LinkedTimeline{TimelineID: id},
			MatchedLogCount: count,
		})
	}
	slices.SortFunc(otherTimelines, func(a, b OtherTimeline) int {
		if c := cmp.Compare(b.MatchedLogCount, a.MatchedLogCount); c != 0 {
			return c
		}
		return cmp.Compare(a.TimelineID, b.TimelineID)
	})

	return entries, otherTimelines
}

// logTime converts the log timestamp to UTC time, or returns the zero time when the log has no timestamp.
func logTime(l *cel.LogData) time.Time {
	if l.Timestamp <= 0 {
		return time.Time{}
	}
	return time.Unix(0, l.Timestamp).UTC()
}

// GetLogDetail returns the metadata, all linked timelines, and the YAML body sliced from byteOffset for a single log ID.
func (w *Workbench) GetLogDetail(logID uint32, byteOffset int) (*LogDetail, error) {
	index, styles, err := w.readyIndex()
	if err != nil {
		return nil, err
	}

	l := index.GetLog(logID)
	if l == nil || l.ID == 0 {
		return nil, fmt.Errorf("%w: %d", ErrLogNotFound, logID)
	}

	linkedIDs := slices.Clone(index.GetTimelineIDsForLog(l.ID))
	slices.Sort(linkedIDs)
	linked := make([]LinkedTimeline, 0, len(linkedIDs))
	for _, linkedID := range linkedIDs {
		segments, _ := index.TimelineSegments(linkedID)
		linked = append(linked, LinkedTimeline{
			TimelineID: linkedID,
			Segments:   segments,
		})
	}

	yamls, err := w.ReadStructYAMLs([]uint32{l.BodyStructID})
	if err != nil {
		return nil, err
	}

	return &LogDetail{
		LogID:           l.ID,
		Time:            logTime(l),
		Severity:        styles.severityMap[l.SeverityTypeID],
		LogType:         index.StyleResolver.ResolveLogType(l.LogTypeID),
		Summary:         index.InternPool.ResolveStringFromID(l.SummaryStringID),
		LinkedTimelines: linked,
		Body:            truncateBody(yamls[l.BodyStructID], byteOffset, bodyByteLimit),
	}, nil
}
