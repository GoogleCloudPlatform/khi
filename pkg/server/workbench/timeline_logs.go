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

// DefaultMaxOtherTimelines is the default maximum number of other linked timelines returned by GetTimelineLogs.
const DefaultMaxOtherTimelines = 20

// TimelineLogFilter specifies log CEL and time range filter parameters for reading logs on a single timeline.
type TimelineLogFilter struct {
	LogQuery  string     `json:"logQuery,omitempty" jsonschema:"CEL expression to filter logs."`
	StartTime *time.Time `json:"startTime,omitempty" jsonschema:"Inclusive start of the time range in RFC3339 format."`
	EndTime   *time.Time `json:"endTime,omitempty" jsonschema:"Inclusive end of the time range in RFC3339 format."`
}

// OtherTimelineLogCount represents a timeline other than the queried timeline that is linked to the matched logs.
type OtherTimelineLogCount struct {
	TimelineID uint32
	Segments   []TimelineSegment
	LogCount   int
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
	OtherTimelines          []OtherTimelineLogCount
	Logs                    []TimelineLogEntry
}

// LinkedTimeline represents a timeline linked to a log entry.
type LinkedTimeline struct {
	TimelineID uint32
	Segments   []TimelineSegment
}

// LogDetail contains metadata, linked timelines, and the truncated YAML body for a single log entry.
type LogDetail struct {
	LogID           uint32
	Time            time.Time
	Severity        *khifilev6.Severity
	LogType         string
	Summary         string
	LinkedTimelines []LinkedTimeline
	Body            TruncatedBody
}

// GetTimelineLogs filters logs on the specified timeline and aggregates other timelines linked to those logs across all matched entries.
// If maxOtherTimelines <= 0, DefaultMaxOtherTimelines is used.
func (w *Workbench) GetTimelineLogs(ctx context.Context, timelineID uint32, filter TimelineLogFilter, maxOtherTimelines int) (*TimelineLogsResult, error) {
	if maxOtherTimelines <= 0 {
		maxOtherTimelines = DefaultMaxOtherTimelines
	}

	w.mu.RLock()
	if w.closed {
		w.mu.RUnlock()
		return nil, ErrWorkbenchClosed
	}
	if w.searchIndex == nil {
		w.mu.RUnlock()
		return nil, fmt.Errorf("search index is not ready")
	}
	index := w.searchIndex
	styles := w.styles
	w.mu.RUnlock()

	tl := index.TimelineMap[timelineID]
	segments, ok := index.TimelineSegments(timelineID)
	if !ok || tl == nil {
		return nil, fmt.Errorf("%w: %d", ErrTimelineNotFound, timelineID)
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

	entries, allOther := buildTimelineLogEntriesAndOtherTimelines(matchedLogs, timelineID, index, styles)
	totalOther := len(allOther)
	if len(allOther) > maxOtherTimelines {
		allOther = allOther[:maxOtherTimelines]
	}
	for i := range allOther {
		otherSegs, _ := index.TimelineSegments(allOther[i].TimelineID)
		allOther[i].Segments = otherSegs
	}

	return &TimelineLogsResult{
		TimelineID:              timelineID,
		Segments:                segments,
		Applied:                 filterOut.Applied,
		TotalOtherTimelineCount: totalOther,
		OtherTimelines:          allOther,
		Logs:                    entries,
	}, nil
}

func collectTimelineMatchedLogIDs(tl *cel.TimelineData, filterOut *FilterOutput) *roaring.Bitmap {
	matched := roaring.NewBitmap()
	hasRange := !filterOut.Applied.StartTime.IsZero() || !filterOut.Applied.EndTime.IsZero()
	if hasRange {
		startNs := int64(math.MinInt64)
		if !filterOut.Applied.StartTime.IsZero() {
			startNs = filterOut.Applied.StartTime.UnixNano()
		}
		endNs := int64(math.MaxInt64)
		if !filterOut.Applied.EndTime.IsZero() {
			endNs = filterOut.Applied.EndTime.UnixNano()
		}
		tl.ForEachLogIDInRange(startNs, endNs, func(logID uint32) bool {
			if logID > 0 && filterOut.LogIDs.Contains(logID) {
				matched.Add(logID)
			}
			return true
		})
		return matched
	}

	tl.ForEachLogID(func(logID uint32) bool {
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
) ([]TimelineLogEntry, []OtherTimelineLogCount) {
	if len(matchedLogs) == 0 {
		return nil, nil
	}

	entries := make([]TimelineLogEntry, 0, len(matchedLogs))
	otherCounts := make(map[uint32]int)

	var severityMap map[uint32]*khifilev6.Severity
	if styles != nil {
		severityMap = styles.severityMap
	}

	for _, l := range matchedLogs {
		var logType string
		if index.StyleResolver != nil {
			logType = index.StyleResolver.ResolveLogType(l.LogTypeID)
		}
		var summary string
		if index.InternPool != nil {
			summary = index.InternPool.ResolveStringFromID(l.SummaryStringID)
		}
		var logTime time.Time
		if l.Timestamp > 0 {
			logTime = time.Unix(0, l.Timestamp).UTC()
		}

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
			Time:             logTime,
			Severity:         severityMap[l.SeverityTypeID],
			LogType:          logType,
			Summary:          summary,
			OtherTimelineIDs: otherIDs,
		})
	}

	allOther := make([]OtherTimelineLogCount, 0, len(otherCounts))
	for id, count := range otherCounts {
		allOther = append(allOther, OtherTimelineLogCount{
			TimelineID: id,
			LogCount:   count,
		})
	}
	slices.SortFunc(allOther, func(a, b OtherTimelineLogCount) int {
		if c := cmp.Compare(b.LogCount, a.LogCount); c != 0 {
			return c
		}
		return cmp.Compare(a.TimelineID, b.TimelineID)
	})

	return entries, allOther
}

// GetLogDetail returns the metadata, all linked timelines, and the truncated YAML body for a single log ID.
// If byteLimit <= 0, DefaultBodyByteLimit is used.
func (w *Workbench) GetLogDetail(logID uint32, byteOffset, byteLimit int) (*LogDetail, error) {
	w.mu.RLock()
	if w.closed {
		w.mu.RUnlock()
		return nil, ErrWorkbenchClosed
	}
	if w.searchIndex == nil {
		w.mu.RUnlock()
		return nil, fmt.Errorf("search index is not ready")
	}
	index := w.searchIndex
	styles := w.styles
	w.mu.RUnlock()

	l := index.GetLog(logID)
	if l == nil || l.ID == 0 {
		return nil, fmt.Errorf("%w: %d", ErrLogNotFound, logID)
	}

	var logType string
	if index.StyleResolver != nil {
		logType = index.StyleResolver.ResolveLogType(l.LogTypeID)
	}
	var summary string
	if index.InternPool != nil {
		summary = index.InternPool.ResolveStringFromID(l.SummaryStringID)
	}
	var logTime time.Time
	if l.Timestamp > 0 {
		logTime = time.Unix(0, l.Timestamp).UTC()
	}
	var severity *khifilev6.Severity
	if styles != nil {
		severity = styles.severityMap[l.SeverityTypeID]
	}

	rawTLIDs := slices.Clone(index.GetTimelineIDsForLog(l.ID))
	slices.Sort(rawTLIDs)
	linked := make([]LinkedTimeline, 0, len(rawTLIDs))
	for _, tlID := range rawTLIDs {
		segments, ok := index.TimelineSegments(tlID)
		if !ok {
			continue
		}
		linked = append(linked, LinkedTimeline{
			TimelineID: tlID,
			Segments:   segments,
		})
	}

	yamls, err := w.ReadStructYAMLs([]uint32{l.BodyStructID})
	if err != nil {
		return nil, err
	}
	bodyYAML := yamls[l.BodyStructID]

	return &LogDetail{
		LogID:           l.ID,
		Time:            logTime,
		Severity:        severity,
		LogType:         logType,
		Summary:         summary,
		LinkedTimelines: linked,
		Body:            TruncateBody(bodyYAML, byteOffset, byteLimit),
	}, nil
}
