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
	"context"
	"fmt"
	"math"
	"time"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/RoaringBitmap/roaring/v2"
)

// DefaultMaxTimelineNodes is the default maximum number of timeline nodes returned by SearchTimelines.
const DefaultMaxTimelineNodes = 200

// TimelineTypeDescription describes a timeline type present in a TimelineTreeResult.
type TimelineTypeDescription struct {
	Type        string
	Description string
}

// TimelineTreeNode represents a single node in the pre-order timeline hierarchy.
type TimelineTreeNode struct {
	ID            uint32
	Depth         int
	Type          string
	Name          string
	EventCount    int
	RevisionCount int
	// SeverityCounts holds the number of matched logs per severity across the timeline and its filtered descendants, from the most severe.
	SeverityCounts  []SeverityCount
	OmittedChildren int
	FirstMatchTime  time.Time
	LastMatchTime   time.Time
}

// TimelineTreeResult contains the filtered timeline tree, summary counts, and timeline type descriptions.
type TimelineTreeResult struct {
	MatchedTimelineCount int
	ReturnedNodeCount    int
	MaxDepth             int
	MaxNodes             int
	Applied              AppliedFilter
	TimelineTypes        []TimelineTypeDescription
	Nodes                []TimelineTreeNode
}

// logSeverityResolver returns the severity definition of the log with the given ID.
type logSeverityResolver func(logID uint32) *khifilev6.Severity

type nodeMetrics struct {
	ownEv             int
	ownRev            int
	ownLogIDs         *roaring.Bitmap
	aggLogIDs         *roaring.Bitmap
	aggSeverityCounts severityCounter
	firstNs           int64
	lastNs            int64
}

func (m *nodeMetrics) updateTime(ts int64) {
	if ts <= 0 {
		return
	}
	if m.firstNs == -1 || ts < m.firstNs {
		m.firstNs = ts
	}
	if m.lastNs == -1 || ts > m.lastNs {
		m.lastNs = ts
	}
}

// recordLog adds the log linked to a matched entry to the timeline's matched log set.
func (m *nodeMetrics) recordLog(logID uint32) {
	// Entries without a linked log have no severity.
	if logID == 0 {
		return
	}
	m.ownLogIDs.Add(logID)
}

// SearchTimelines filters timelines and builds a breadth-first truncated, pre-order rendered timeline hierarchy tree.
// Depth is 0-based from root timelines. If maxDepth <= 0, depth is unlimited. If maxNodes <= 0, DefaultMaxTimelineNodes is used.
func (w *Workbench) SearchTimelines(ctx context.Context, filter Filter, maxDepth, maxNodes int) (*TimelineTreeResult, error) {
	if maxNodes <= 0 {
		maxNodes = DefaultMaxTimelineNodes
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

	matchedCount := int(filterOut.TimelineIDs.GetCardinality())
	res := &TimelineTreeResult{
		MatchedTimelineCount: matchedCount,
		MaxDepth:             maxDepth,
		MaxNodes:             maxNodes,
		Applied:              filterOut.Applied,
	}
	if matchedCount == 0 {
		return res, nil
	}

	// Matched entries only link to logs that exist in the index, so GetLog never returns nil here.
	resolveLogSeverity := func(logID uint32) *khifilev6.Severity {
		return styles.severityMap[index.GetLog(logID).SeverityTypeID]
	}
	roots, filteredChildren, metricsMap := collectFilteredTimelineHierarchy(index, filterOut)
	aggregateTimelineSeverities(roots, filteredChildren, metricsMap, resolveLogSeverity)
	selected, nodeDepth := selectBFSNodes(roots, filteredChildren, maxDepth, maxNodes, matchedCount)
	nodes, timelineTypes := emitTimelineTreeNodes(roots, filteredChildren, selected, nodeDepth, metricsMap, index, styles.timelineTypeDescriptionMap)

	res.ReturnedNodeCount = len(nodes)
	res.TimelineTypes = timelineTypes
	res.Nodes = nodes
	return res, nil
}

func collectFilteredTimelineHierarchy(index *SearchIndex, filterOut *FilterOutput) ([]uint32, map[uint32][]uint32, map[uint32]*nodeMetrics) {
	matchedCount := int(filterOut.TimelineIDs.GetCardinality())
	startNs := int64(math.MinInt64)
	if !filterOut.Applied.StartTime.IsZero() {
		startNs = filterOut.Applied.StartTime.UnixNano()
	}
	endNs := int64(math.MaxInt64)
	if !filterOut.Applied.EndTime.IsZero() {
		endNs = filterOut.Applied.EndTime.UnixNano()
	}

	metricsMap := make(map[uint32]*nodeMetrics, matchedCount)
	filteredChildren := make(map[uint32][]uint32, matchedCount)
	var roots []uint32

	it := filterOut.TimelineIDs.Iterator()
	for it.HasNext() {
		id := it.Next()
		tl := index.TimelineMap[id]
		if tl == nil {
			continue
		}
		if tl.ParentID == 0 || !filterOut.TimelineIDs.Contains(tl.ParentID) {
			roots = append(roots, tl.ID)
		}
		for _, childID := range tl.ChildrenIDs {
			if filterOut.TimelineIDs.Contains(childID) {
				filteredChildren[tl.ID] = append(filteredChildren[tl.ID], childID)
			}
		}
		metricsMap[tl.ID] = computeNodeMetrics(tl, filterOut, startNs, endNs)
	}
	return roots, filteredChildren, metricsMap
}

func computeNodeMetrics(tl *cel.TimelineData, filterOut *FilterOutput, startNs, endNs int64) *nodeMetrics {
	m := &nodeMetrics{
		ownLogIDs: roaring.New(),
		firstNs:   -1,
		lastNs:    -1,
	}

	for _, evt := range tl.Events {
		if !isTimelineEntryMatched(evt.LogID, evt.Timestamp, filterOut, startNs, endNs) {
			continue
		}
		m.ownEv++
		m.updateTime(evt.Timestamp)
		m.recordLog(evt.LogID)
	}

	for _, rev := range tl.Revisions {
		if !isTimelineEntryMatched(rev.LogID, rev.ChangedTime, filterOut, startNs, endNs) {
			continue
		}
		m.ownRev++
		m.updateTime(rev.ChangedTime)
		m.recordLog(rev.LogID)
	}
	return m
}

func isTimelineEntryMatched(logID uint32, ts int64, filterOut *FilterOutput, startNs, endNs int64) bool {
	if logID > 0 {
		return filterOut.LogIDs.Contains(logID)
	}
	return ts >= startNs && ts <= endNs
}

func aggregateTimelineSeverities(roots []uint32, filteredChildren map[uint32][]uint32, metricsMap map[uint32]*nodeMetrics, resolveLogSeverity logSeverityResolver) {
	visited := make(map[uint32]bool, len(metricsMap))
	var dfs func(id uint32) *roaring.Bitmap
	dfs = func(id uint32) *roaring.Bitmap {
		m := metricsMap[id]
		if m == nil {
			return nil
		}
		if visited[id] {
			return m.aggLogIDs
		}
		visited[id] = true

		aggLogIDs := m.ownLogIDs.Clone()
		for _, childID := range filteredChildren[id] {
			if childLogIDs := dfs(childID); childLogIDs != nil {
				aggLogIDs.Or(childLogIDs)
			}
		}
		m.aggLogIDs = aggLogIDs

		counts := make(severityCounter)
		it := aggLogIDs.Iterator()
		for it.HasNext() {
			counts[resolveLogSeverity(it.Next())]++
		}
		m.aggSeverityCounts = counts
		return aggLogIDs
	}
	for _, rootID := range roots {
		dfs(rootID)
	}
}

func selectBFSNodes(roots []uint32, filteredChildren map[uint32][]uint32, maxDepth, maxNodes, matchedCount int) (map[uint32]bool, map[uint32]int) {
	type bfsItem struct {
		id    uint32
		depth int
	}
	queue := make([]bfsItem, 0, len(roots))
	for _, rootID := range roots {
		queue = append(queue, bfsItem{id: rootID, depth: 0})
	}

	capHint := min(matchedCount, maxNodes)
	selected := make(map[uint32]bool, capHint)
	nodeDepth := make(map[uint32]int, capHint)
	for len(queue) > 0 && len(selected) < maxNodes {
		curr := queue[0]
		queue = queue[1:]
		if selected[curr.id] {
			continue
		}
		selected[curr.id] = true
		nodeDepth[curr.id] = curr.depth

		if maxDepth > 0 && curr.depth >= maxDepth {
			continue
		}
		for _, childID := range filteredChildren[curr.id] {
			if !selected[childID] {
				queue = append(queue, bfsItem{id: childID, depth: curr.depth + 1})
			}
		}
	}
	return selected, nodeDepth
}

func emitTimelineTreeNodes(
	roots []uint32,
	filteredChildren map[uint32][]uint32,
	selected map[uint32]bool,
	nodeDepth map[uint32]int,
	metricsMap map[uint32]*nodeMetrics,
	index *SearchIndex,
	timelineTypeDescriptionMap map[string]string,
) ([]TimelineTreeNode, []TimelineTypeDescription) {
	seenTypes := make(map[string]bool)
	var timelineTypes []TimelineTypeDescription
	nodes := make([]TimelineTreeNode, 0, len(selected))

	visited := make(map[uint32]bool, len(selected))
	var dfs func(id uint32)
	dfs = func(id uint32) {
		if !selected[id] || visited[id] {
			return
		}
		visited[id] = true
		tl := index.TimelineMap[id]
		if tl == nil {
			return
		}
		nodes = append(nodes, buildTimelineTreeNode(tl, nodeDepth[id], filteredChildren[id], selected, metricsMap[id]))

		if tl.TimelineType != "" && !seenTypes[tl.TimelineType] {
			seenTypes[tl.TimelineType] = true
			if desc := timelineTypeDescriptionMap[tl.TimelineType]; desc != "" {
				timelineTypes = append(timelineTypes, TimelineTypeDescription{
					Type:        tl.TimelineType,
					Description: desc,
				})
			}
		}

		for _, childID := range filteredChildren[id] {
			if selected[childID] {
				dfs(childID)
			}
		}
	}
	for _, rootID := range roots {
		dfs(rootID)
	}
	return nodes, timelineTypes
}

func buildTimelineTreeNode(tl *cel.TimelineData, depth int, children []uint32, selected map[uint32]bool, m *nodeMetrics) TimelineTreeNode {
	selectedChildren := 0
	for _, childID := range children {
		if selected[childID] {
			selectedChildren++
		}
	}
	var firstTime, lastTime time.Time
	if m.firstNs > 0 {
		firstTime = time.Unix(0, m.firstNs).UTC()
	}
	if m.lastNs > 0 {
		lastTime = time.Unix(0, m.lastNs).UTC()
	}
	return TimelineTreeNode{
		ID:              tl.ID,
		Depth:           depth,
		Type:            tl.TimelineType,
		Name:            tl.Name,
		EventCount:      m.ownEv,
		RevisionCount:   m.ownRev,
		SeverityCounts:  m.aggSeverityCounts.sorted(),
		OmittedChildren: len(children) - selectedChildren,
		FirstMatchTime:  firstTime,
		LastMatchTime:   lastTime,
	}
}
