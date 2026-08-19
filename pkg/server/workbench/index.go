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
	"fmt"
	"runtime"
	"strings"

	pbv6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"golang.org/x/sync/errgroup"
)

// IndexedTimeline represents an in-memory indexed timeline optimized for CEL query evaluation.
type IndexedTimeline struct {
	ID          uint32
	ParentID    uint32
	ChildrenIDs []uint32
	LogIDs      []uint32
	Path        map[string]string
	Data        *cel.TimelineData
}

// IndexedLog represents an in-memory indexed log optimized for CEL query evaluation.
type IndexedLog struct {
	ID   uint32
	Data *cel.LogData
}

// SearchIndex encapsulates the indexed timelines and logs of a Workbench session.
type SearchIndex struct {
	Timelines   []*IndexedTimeline
	TimelineMap map[uint32]*IndexedTimeline
	Logs        []*IndexedLog
	LogMap      map[uint32]*IndexedLog
	InternPool  *khifilev6model.InternPool
}

type styleMaps struct {
	severityOrderMap     map[uint32]uint32
	logTypeLabelMap      map[uint32]string
	timelineTypeLabelMap map[uint32]string
	verbLabelMap         map[uint32]string
	stateLabelMap        map[uint32]string
}

// BuildSearchIndex constructs an in-memory SearchIndex from the parsed Workbench chunks.
func (w *Workbench) BuildSearchIndex() (*SearchIndex, error) {
	styles := w.buildStyleMaps()

	logs, logMap, err := w.indexLogsParallel(styles)
	if err != nil {
		return nil, fmt.Errorf("failed to index logs: %w", err)
	}

	itemsMap := w.buildTimelineItemsMap()
	timelines, timelineMap, err := w.indexTimelinesParallel(styles, itemsMap, logMap)
	if err != nil {
		return nil, fmt.Errorf("failed to index timelines: %w", err)
	}

	w.linkTimelineHierarchy(timelines, timelineMap)
	w.computeAllTimelinePathsParallel(timelines, timelineMap)

	return &SearchIndex{
		Timelines:   timelines,
		TimelineMap: timelineMap,
		Logs:        logs,
		LogMap:      logMap,
		InternPool:  w.internPool,
	}, nil
}

func (w *Workbench) buildStyleMaps() *styleMaps {
	s := &styleMaps{
		severityOrderMap:     make(map[uint32]uint32),
		logTypeLabelMap:      make(map[uint32]string),
		timelineTypeLabelMap: make(map[uint32]string),
		verbLabelMap:         make(map[uint32]string),
		stateLabelMap:        make(map[uint32]string),
	}
	if w.styleChunk != nil {
		for _, sev := range w.styleChunk.Severities {
			s.severityOrderMap[sev.GetId()] = uint32(sev.GetOrder())
		}
		for _, lt := range w.styleChunk.LogTypes {
			s.logTypeLabelMap[lt.GetId()] = lt.GetLabel()
		}
		for _, tt := range w.styleChunk.TimelineTypes {
			s.timelineTypeLabelMap[tt.GetId()] = tt.GetLabel()
		}
		for _, v := range w.styleChunk.Verbs {
			s.verbLabelMap[v.GetId()] = v.GetLabel()
		}
		for _, rs := range w.styleChunk.RevisionStates {
			s.stateLabelMap[rs.GetId()] = rs.GetLabel()
		}
	}
	return s
}

func (w *Workbench) indexLogsParallel(styles *styleMaps) ([]*IndexedLog, map[uint32]*IndexedLog, error) {
	numChunks := len(w.logChunks)
	if numChunks == 0 {
		return nil, make(map[uint32]*IndexedLog), nil
	}

	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > numChunks {
		numWorkers = numChunks
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	chunkSliceSize := (numChunks + numWorkers - 1) / numWorkers
	workerResults := make([][]*IndexedLog, numWorkers)

	g := new(errgroup.Group)
	for wIdx := 0; wIdx < numWorkers; wIdx++ {
		workerIndex := wIdx
		startChunk := workerIndex * chunkSliceSize
		endChunk := startChunk + chunkSliceSize
		if endChunk > numChunks {
			endChunk = numChunks
		}
		if startChunk >= endChunk {
			continue
		}

		g.Go(func() error {
			var localLogs []*IndexedLog
			for c := startChunk; c < endChunk; c++ {
				chunk := w.logChunks[c]
				for _, log := range chunk.Logs {
					sevOrder := styles.severityOrderMap[log.GetSeverityTypeId()]
					ltLabel := styles.logTypeLabelMap[log.GetLogTypeId()]

					lData := &cel.LogData{
						ID:              log.GetId(),
						LogType:         ltLabel,
						Severity:        sevOrder,
						SummaryStringID: log.GetSummaryStringId(),
						BodyStructID:    log.GetBodyStructId(),
					}

					localLogs = append(localLogs, &IndexedLog{
						ID:   log.GetId(),
						Data: lData,
					})
				}
			}
			workerResults[workerIndex] = localLogs
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, nil, err
	}

	totalLogs := 0
	for _, res := range workerResults {
		totalLogs += len(res)
	}

	logs := make([]*IndexedLog, 0, totalLogs)
	logMap := make(map[uint32]*IndexedLog, totalLogs)

	for _, res := range workerResults {
		for _, item := range res {
			logs = append(logs, item)
			logMap[item.ID] = item
		}
	}

	return logs, logMap, nil
}

func (w *Workbench) buildTimelineItemsMap() map[uint32]*pbv6.TimelineItems {
	itemsMap := make(map[uint32]*pbv6.TimelineItems)
	for _, chunk := range w.timelineChunks {
		for _, item := range chunk.TimelineItems {
			itemsMap[item.GetId()] = item
		}
	}
	return itemsMap
}

func (w *Workbench) indexTimelinesParallel(
	styles *styleMaps,
	itemsMap map[uint32]*pbv6.TimelineItems,
	logMap map[uint32]*IndexedLog,
) ([]*IndexedTimeline, map[uint32]*IndexedTimeline, error) {
	numChunks := len(w.timelineChunks)
	if numChunks == 0 {
		return nil, make(map[uint32]*IndexedTimeline), nil
	}

	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > numChunks {
		numWorkers = numChunks
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	chunkSliceSize := (numChunks + numWorkers - 1) / numWorkers
	workerResults := make([][]*IndexedTimeline, numWorkers)

	g := new(errgroup.Group)
	for wIdx := 0; wIdx < numWorkers; wIdx++ {
		workerIndex := wIdx
		startChunk := workerIndex * chunkSliceSize
		endChunk := startChunk + chunkSliceSize
		if endChunk > numChunks {
			endChunk = numChunks
		}
		if startChunk >= endChunk {
			continue
		}

		g.Go(func() error {
			var localTimelines []*IndexedTimeline
			for c := startChunk; c < endChunk; c++ {
				chunk := w.timelineChunks[c]
				for _, tl := range chunk.Timelines {
					tlName := ""
					if w.internPool != nil {
						tlName = w.internPool.ResolveStringFromID(tl.GetNameStringId())
					}
					tlType := styles.timelineTypeLabelMap[tl.GetTimelineType()]

					var logIDs []uint32
					var events []cel.EventInfo
					var revisions []cel.RevisionInfo
					var maxSeverity uint32

					if item, ok := itemsMap[tl.GetTimelineItemsId()]; ok {
						for _, evt := range item.Events {
							logID := evt.GetLogId()
							logIDs = append(logIDs, logID)
							sev := uint32(0)
							if logObj, exists := logMap[logID]; exists {
								sev = logObj.Data.Severity
							}
							if sev > maxSeverity {
								maxSeverity = sev
							}
							events = append(events, cel.EventInfo{
								LogID:    logID,
								Severity: sev,
							})
						}

						for _, rev := range item.Revisions {
							logID := rev.GetLogId()
							logIDs = append(logIDs, logID)
							verb := styles.verbLabelMap[rev.GetVerbType()]
							state := styles.stateLabelMap[rev.GetStateType()]

							sev := uint32(0)
							if logObj, exists := logMap[logID]; exists {
								sev = logObj.Data.Severity
							}
							if sev > maxSeverity {
								maxSeverity = sev
							}

							changedTime := int64(0)
							if rev.ChangedTime != nil {
								changedTime = rev.ChangedTime.AsTime().UnixNano()
							}

							revisions = append(revisions, cel.RevisionInfo{
								LogID:                logID,
								ChangedTime:          changedTime,
								PrincipalStringID:    rev.GetPrincipalStringId(),
								Verb:                 verb,
								State:                state,
								ResourceBodyStructID: rev.GetResourceBodyStructId(),
								Severity:             sev,
							})
						}
					}

					tData := &cel.TimelineData{
						ID:           tl.GetId(),
						Name:         tlName,
						TimelineType: tlType,
						Path:         make(map[string]string),
						Events:       events,
						Revisions:    revisions,
						MaxSeverity:  maxSeverity,
					}

					indexedTL := &IndexedTimeline{
						ID:       tl.GetId(),
						ParentID: tl.GetParentTimelineId(),
						LogIDs:   logIDs,
						Data:     tData,
					}

					localTimelines = append(localTimelines, indexedTL)
				}
			}
			workerResults[workerIndex] = localTimelines
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, nil, err
	}

	totalTimelines := 0
	for _, res := range workerResults {
		totalTimelines += len(res)
	}

	timelines := make([]*IndexedTimeline, 0, totalTimelines)
	timelineMap := make(map[uint32]*IndexedTimeline, totalTimelines)

	for _, res := range workerResults {
		for _, item := range res {
			timelines = append(timelines, item)
			timelineMap[item.ID] = item
		}
	}

	return timelines, timelineMap, nil
}

func (w *Workbench) linkTimelineHierarchy(timelines []*IndexedTimeline, timelineMap map[uint32]*IndexedTimeline) {
	for _, tl := range timelines {
		if tl.ParentID != 0 {
			if parent, exists := timelineMap[tl.ParentID]; exists {
				parent.ChildrenIDs = append(parent.ChildrenIDs, tl.ID)
			}
		}
	}
}

func (w *Workbench) computeAllTimelinePathsParallel(timelines []*IndexedTimeline, timelineMap map[uint32]*IndexedTimeline) {
	total := len(timelines)
	if total == 0 {
		return
	}
	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > total {
		numWorkers = total
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	sliceSize := (total + numWorkers - 1) / numWorkers
	g := new(errgroup.Group)

	for wIdx := 0; wIdx < numWorkers; wIdx++ {
		start := wIdx * sliceSize
		end := start + sliceSize
		if end > total {
			end = total
		}
		if start >= end {
			continue
		}

		g.Go(func() error {
			for i := start; i < end; i++ {
				tl := timelines[i]
				tl.Path = tl.ComputePath(timelineMap)
				tl.Data.Path = tl.Path
			}
			return nil
		})
	}

	_ = g.Wait()
}

// ComputePath resolves the timeline hierarchy path map for this timeline segment and all parent segments.
func (tl *IndexedTimeline) ComputePath(tlMap map[uint32]*IndexedTimeline) map[string]string {
	path := make(map[string]string)
	visited := make(map[uint32]struct{})
	curr := tl
	for curr != nil {
		if _, seen := visited[curr.ID]; seen {
			break
		}
		visited[curr.ID] = struct{}{}

		if curr.Data != nil {
			typeKey := strings.ToLower(curr.Data.TimelineType)
			if typeKey != "" {
				path[typeKey] = curr.Data.Name
			}
		}
		if curr.ParentID == 0 {
			break
		}
		curr = tlMap[curr.ParentID]
	}
	return path
}
