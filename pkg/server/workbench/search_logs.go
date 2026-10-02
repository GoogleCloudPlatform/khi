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
	"fmt"
	"slices"
	"time"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/RoaringBitmap/roaring/v2"
)

const (
	// DefaultMaxLogTimelines is the default maximum number of linked timelines returned by SearchLogs.
	DefaultMaxLogTimelines = 20
	// DefaultMaxSampleLogs is the default maximum number of sample logs returned by SearchLogs.
	DefaultMaxSampleLogs = 20
)

// TimelineLogGroup represents a timeline linked to matched logs, along with its per-timeline log statistics.
type TimelineLogGroup struct {
	TimelineID      uint32
	Segments        []TimelineSegment
	MatchedLogCount int
	SeverityCounts  []SeverityCount
	FirstMatchTime  time.Time
	LastMatchTime   time.Time
}

// SampleLogEntry represents a single sampled log entry and its filtered linked timeline IDs.
type SampleLogEntry struct {
	LogID       uint32
	Time        time.Time
	Severity    *khifilev6.Severity
	LogType     string
	Summary     string
	TimelineIDs []uint32
}

// LogSearchResult contains the cross-timeline log search summary, linked timeline ranking, and evenly spaced sample logs.
type LogSearchResult struct {
	MatchedLogCount      int
	MatchedTimelineCount int
	FirstMatchTime       time.Time
	LastMatchTime        time.Time
	SeverityCounts       []SeverityCount
	Applied              AppliedFilter
	TimelineGroups       []TimelineLogGroup
	SampleLogs           []SampleLogEntry
}

type timelineLogStats struct {
	timelineID     uint32
	count          int
	severityCounts severityCounter
	firstNs        int64
	lastNs         int64
}

func (s *timelineLogStats) record(severity *khifilev6.Severity, ts int64) {
	s.count++
	s.severityCounts[severity]++
	if ts <= 0 {
		return
	}
	if s.firstNs == -1 || ts < s.firstNs {
		s.firstNs = ts
	}
	if s.lastNs == -1 || ts > s.lastNs {
		s.lastNs = ts
	}
}

// SearchLogs filters logs and aggregates cross-timeline statistics, linked timeline rankings, and evenly spaced sample logs.
// If maxTimelines <= 0, DefaultMaxLogTimelines is used. If maxSampleLogs <= 0, DefaultMaxSampleLogs is used.
func (w *Workbench) SearchLogs(ctx context.Context, filter Filter, maxTimelines, maxSampleLogs int) (*LogSearchResult, error) {
	if maxTimelines <= 0 {
		maxTimelines = DefaultMaxLogTimelines
	}
	if maxSampleLogs <= 0 {
		maxSampleLogs = DefaultMaxSampleLogs
	}

	filterOut, err := w.ExecuteFilter(ctx, filter)
	if err != nil {
		return nil, err
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

	res := &LogSearchResult{
		Applied: filterOut.Applied,
	}

	matchedLogs := collectSortedMatchedLogs(index, filterOut.LogIDs)
	if len(matchedLogs) == 0 {
		return res, nil
	}

	tlStatsMap, totalSeverityCounts, firstNs, lastNs := aggregateLogSearchStats(matchedLogs, filterOut.TimelineIDs, index, styles.severityMap)
	allGroups := buildSortedTimelineGroups(tlStatsMap)

	res.MatchedLogCount = len(matchedLogs)
	res.MatchedTimelineCount = len(allGroups)
	if firstNs > 0 {
		res.FirstMatchTime = time.Unix(0, firstNs).UTC()
	}
	if lastNs > 0 {
		res.LastMatchTime = time.Unix(0, lastNs).UTC()
	}
	res.SeverityCounts = totalSeverityCounts.sorted()

	if len(allGroups) > maxTimelines {
		allGroups = allGroups[:maxTimelines]
	}
	for i := range allGroups {
		segments, _ := index.TimelineSegments(allGroups[i].TimelineID)
		allGroups[i].Segments = segments
	}
	res.TimelineGroups = allGroups
	res.SampleLogs = buildSampleLogEntries(matchedLogs, maxSampleLogs, filterOut.TimelineIDs, index, styles.severityMap)

	return res, nil
}

func collectSortedMatchedLogs(index *SearchIndex, logIDsBitmap *roaring.Bitmap) []*cel.LogData {
	logIDs := logIDsBitmap.ToArray()
	if len(logIDs) == 0 {
		return nil
	}
	matchedLogs := make([]*cel.LogData, 0, len(logIDs))
	for _, logID := range logIDs {
		if l := index.GetLog(logID); l != nil {
			matchedLogs = append(matchedLogs, l)
		}
	}
	slices.SortFunc(matchedLogs, func(a, b *cel.LogData) int {
		if c := cmp.Compare(a.Timestamp, b.Timestamp); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return matchedLogs
}

func aggregateLogSearchStats(
	matchedLogs []*cel.LogData,
	timelineIDs *roaring.Bitmap,
	index *SearchIndex,
	severityMap map[uint32]*khifilev6.Severity,
) (map[uint32]*timelineLogStats, severityCounter, int64, int64) {
	totalSeverityCounts := make(severityCounter)
	var firstNs int64 = -1
	var lastNs int64 = -1
	tlStatsMap := make(map[uint32]*timelineLogStats)

	for _, l := range matchedLogs {
		severity := severityMap[l.SeverityTypeID]
		totalSeverityCounts[severity]++

		if l.Timestamp > 0 {
			if firstNs == -1 || l.Timestamp < firstNs {
				firstNs = l.Timestamp
			}
			if lastNs == -1 || l.Timestamp > lastNs {
				lastNs = l.Timestamp
			}
		}

		for _, tlID := range index.GetTimelineIDsForLog(l.ID) {
			if !timelineIDs.Contains(tlID) {
				continue
			}
			st, ok := tlStatsMap[tlID]
			if !ok {
				st = &timelineLogStats{
					timelineID:     tlID,
					severityCounts: make(severityCounter),
					firstNs:        -1,
					lastNs:         -1,
				}
				tlStatsMap[tlID] = st
			}
			st.record(severity, l.Timestamp)
		}
	}
	return tlStatsMap, totalSeverityCounts, firstNs, lastNs
}

func buildSortedTimelineGroups(tlStatsMap map[uint32]*timelineLogStats) []TimelineLogGroup {
	allGroups := make([]TimelineLogGroup, 0, len(tlStatsMap))
	for _, st := range tlStatsMap {
		var firstTime, lastTime time.Time
		if st.firstNs > 0 {
			firstTime = time.Unix(0, st.firstNs).UTC()
		}
		if st.lastNs > 0 {
			lastTime = time.Unix(0, st.lastNs).UTC()
		}
		allGroups = append(allGroups, TimelineLogGroup{
			TimelineID:      st.timelineID,
			MatchedLogCount: st.count,
			SeverityCounts:  st.severityCounts.sorted(),
			FirstMatchTime:  firstTime,
			LastMatchTime:   lastTime,
		})
	}

	slices.SortFunc(allGroups, func(a, b TimelineLogGroup) int {
		if c := cmp.Compare(b.MatchedLogCount, a.MatchedLogCount); c != 0 {
			return c
		}
		if c := a.FirstMatchTime.Compare(b.FirstMatchTime); c != 0 {
			return c
		}
		return cmp.Compare(a.TimelineID, b.TimelineID)
	})
	return allGroups
}

func buildSampleLogEntries(
	matchedLogs []*cel.LogData,
	maxSampleLogs int,
	timelineIDs *roaring.Bitmap,
	index *SearchIndex,
	severityMap map[uint32]*khifilev6.Severity,
) []SampleLogEntry {
	sampleIndices := selectTimeSpacedSampleIndices(matchedLogs, maxSampleLogs)
	samples := make([]SampleLogEntry, 0, len(sampleIndices))
	for _, idx := range sampleIndices {
		l := matchedLogs[idx]
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

		var linkedTLs []uint32
		for _, tlID := range index.GetTimelineIDsForLog(l.ID) {
			if timelineIDs.Contains(tlID) {
				linkedTLs = append(linkedTLs, tlID)
			}
		}
		slices.Sort(linkedTLs)

		samples = append(samples, SampleLogEntry{
			LogID:       l.ID,
			Time:        logTime,
			Severity:    severityMap[l.SeverityTypeID],
			LogType:     logType,
			Summary:     summary,
			TimelineIDs: linkedTLs,
		})
	}
	return samples
}

// selectTimeSpacedSampleIndices selects up to maxSamples log indices spread evenly across the matched time range.
// matchedLogs must be sorted by timestamp ascending.
func selectTimeSpacedSampleIndices(matchedLogs []*cel.LogData, maxSamples int) []int {
	total := len(matchedLogs)
	if total == 0 || maxSamples <= 0 {
		return nil
	}
	if total <= maxSamples {
		indices := make([]int, total)
		for i := range total {
			indices[i] = i
		}
		return indices
	}
	if maxSamples == 1 {
		return []int{0}
	}

	firstTs := matchedLogs[0].Timestamp
	lastTs := matchedLogs[total-1].Timestamp
	if lastTs <= firstTs {
		indices := make([]int, maxSamples)
		for i := range maxSamples {
			indices[i] = i * (total - 1) / (maxSamples - 1)
		}
		return indices
	}

	indices := make([]int, maxSamples)
	indices[0] = 0
	indices[maxSamples-1] = total - 1
	for i := 1; i < maxSamples-1; i++ {
		targetTs := firstTs + (lastTs-firstTs)*int64(i)/int64(maxSamples-1)
		minIdx := indices[i-1] + 1
		maxIdx := total - (maxSamples - i)
		indices[i] = closestTimestampIndex(matchedLogs, minIdx, maxIdx, targetTs)
	}
	return indices
}

func closestTimestampIndex(logs []*cel.LogData, minIdx, maxIdx int, targetTs int64) int {
	pos, _ := slices.BinarySearchFunc(logs[minIdx:maxIdx+1], targetTs, func(l *cel.LogData, target int64) int {
		return cmp.Compare(l.Timestamp, target)
	})
	idx := minIdx + pos
	if idx > maxIdx {
		return maxIdx
	}
	if idx > minIdx && targetTs-logs[idx-1].Timestamp <= logs[idx].Timestamp-targetTs {
		return idx - 1
	}
	return idx
}
