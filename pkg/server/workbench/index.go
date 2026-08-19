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
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	pbv6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
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
	Timelines    []*IndexedTimeline
	TimelineMap  map[uint32]*IndexedTimeline
	Logs         []*IndexedLog
	LogMap       map[uint32]*IndexedLog
	InternPool   *khifilev6model.InternPool
	StructYAMLs  map[uint32]string
	TrigramIndex *cel.TrigramIndex
}

type styleMaps struct {
	severityOrderMap     map[uint32]uint32
	logTypeLabelMap      map[uint32]string
	timelineTypeLabelMap map[uint32]string
	verbLabelMap         map[uint32]string
	stateLabelMap        map[uint32]string
}

// BuildBaseSearchIndex constructs the base in-memory SearchIndex containing timelines, logs, and hierarchy mappings.
func (w *Workbench) BuildBaseSearchIndex() (*SearchIndex, error) {
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
		Timelines:    timelines,
		TimelineMap:  timelineMap,
		Logs:         logs,
		LogMap:       logMap,
		InternPool:   w.internPool,
		StructYAMLs:  nil,
		TrigramIndex: nil,
	}, nil
}

// serializeStructChunk converts a slice of StructIDs to their YAML representation.
func serializeStructChunk(pool *khifilev6model.InternPool, chunk []uint32, onProcessed func(int)) map[uint32]string {
	localYAML := make(map[uint32]string, len(chunk))
	serializer := &structured.YAMLNodeSerializer{}
	for _, id := range chunk {
		s := pool.ResolveStructFromID(id)
		if s != nil {
			if node, err := khifilev6model.FromInternedStruct(s, pool); err == nil {
				if yamlBytes, err := serializer.Serialize(node); err == nil {
					localYAML[id] = string(yamlBytes)
				}
			}
		}
		onProcessed(1)
	}
	return localYAML
}

// BuildStructYAMLIndexWithProgress pre-serializes unique log struct bodies into YAML strings while streaming progress updates.
func (w *Workbench) BuildStructYAMLIndexWithProgress(targetIndex *SearchIndex, onProgress ProgressCallback) (map[uint32]string, error) {
	w.mu.RLock()
	pool := w.internPool
	w.mu.RUnlock()

	if targetIndex == nil || pool == nil {
		return nil, fmt.Errorf("search index or intern pool not initialized")
	}

	if err := onProgress(apiv1.OpenWorkbenchResponse_STAGE_INDEXING_DATA, 0.0, "Preparing unique structs for indexing..."); err != nil {
		return nil, err
	}

	var uniqueBodyStructIDs []uint32
	seenStructIDs := make(map[uint32]struct{})
	for _, l := range targetIndex.Logs {
		if l.Data != nil && l.Data.BodyStructID != 0 {
			if _, ok := seenStructIDs[l.Data.BodyStructID]; !ok {
				seenStructIDs[l.Data.BodyStructID] = struct{}{}
				uniqueBodyStructIDs = append(uniqueBodyStructIDs, l.Data.BodyStructID)
			}
		}
	}

	yamlResults, err := worker.ParallelChunkMap(
		context.Background(),
		uniqueBodyStructIDs,
		func(ctx context.Context, workerIdx int, chunk []uint32, onProcessed func(int)) (map[uint32]string, error) {
			return serializeStructChunk(pool, chunk, onProcessed), nil
		},
		func(subPct float64, msg string) error {
			return onProgress(apiv1.OpenWorkbenchResponse_STAGE_INDEXING_DATA, subPct*100.0, msg)
		},
		worker.ProgressOptions{
			Interval:    time.Second,
			MessageFmt:  "Indexing structured log data(%d/%d)...",
			MinProgress: 0.0,
			MaxProgress: 1.0,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize struct YAMLs: %w", err)
	}

	totalStructs := 0
	for _, res := range yamlResults {
		totalStructs += len(res)
	}
	structYAMLs := make(map[uint32]string, totalStructs)
	for _, res := range yamlResults {
		for id, yamlStr := range res {
			structYAMLs[id] = yamlStr
		}
	}

	if err := onProgress(apiv1.OpenWorkbenchResponse_STAGE_INDEXING_DATA, 100.0, "Struct YAML indexing complete."); err != nil {
		return nil, err
	}

	return structYAMLs, nil
}

// BuildTrigramIndexWithProgress constructs the trigram search index from pre-serialized struct YAMLs while streaming progress updates.
func (w *Workbench) BuildTrigramIndexWithProgress(structYAMLs map[uint32]string, onProgress ProgressCallback) (*cel.TrigramIndex, error) {
	trigramIndex := cel.NewTrigramIndex()
	err := trigramIndex.BuildFromStructYAMLs(structYAMLs, func(subPct float64, msg string) error {
		return onProgress(apiv1.OpenWorkbenchResponse_STAGE_INDEXING_DATA, subPct*100.0, msg)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build trigram index: %w", err)
	}

	if err := onProgress(apiv1.OpenWorkbenchResponse_STAGE_INDEXING_DATA, 100.0, "Text search index complete."); err != nil {
		return nil, err
	}

	return trigramIndex, nil
}

// BuildAsyncIndexesWithProgress populates asynchronous search indexes (such as struct YAMLs and trigram indexes) on the target SearchIndex while streaming progress updates.
func (w *Workbench) BuildAsyncIndexesWithProgress(targetIndex *SearchIndex, onProgress ProgressCallback) error {
	if targetIndex == nil {
		return fmt.Errorf("target search index is nil")
	}

	// Stage 1: Parallel Struct YAML Serialization (0% - 50%)
	structYAMLs, err := w.BuildStructYAMLIndexWithProgress(targetIndex, func(stage apiv1.OpenWorkbenchResponse_Stage, progressPercentage float64, message string) error {
		return onProgress(stage, progressPercentage*0.5, message)
	})
	if err != nil {
		return err
	}
	targetIndex.StructYAMLs = structYAMLs

	// Stage 2: Parallel Trigram Index Construction (50% - 100%)
	trigramIndex, err := w.BuildTrigramIndexWithProgress(structYAMLs, func(stage apiv1.OpenWorkbenchResponse_Stage, progressPercentage float64, message string) error {
		return onProgress(stage, 50.0+progressPercentage*0.5, message)
	})
	if err != nil {
		return err
	}
	targetIndex.TrigramIndex = trigramIndex

	if err := onProgress(apiv1.OpenWorkbenchResponse_STAGE_INDEXING_DATA, 100.0, "Search index ready."); err != nil {
		return err
	}

	return nil
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
	if len(w.logChunks) == 0 {
		return nil, make(map[uint32]*IndexedLog), nil
	}

	workerResults, err := worker.ParallelChunkMap(
		context.Background(),
		w.logChunks,
		func(ctx context.Context, workerIdx int, chunk []*pbv6.LogChunk, onProcessed func(int)) ([]*IndexedLog, error) {
			var localLogs []*IndexedLog
			for _, logChunk := range chunk {
				for _, log := range logChunk.Logs {
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
				onProcessed(1)
			}
			return localLogs, nil
		},
		nil,
		worker.ProgressOptions{},
	)
	if err != nil {
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
	if len(w.timelineChunks) == 0 {
		return nil, make(map[uint32]*IndexedTimeline), nil
	}

	workerResults, err := worker.ParallelChunkMap(
		context.Background(),
		w.timelineChunks,
		func(ctx context.Context, workerIdx int, chunk []*pbv6.TimelineChunk, onProcessed func(int)) ([]*IndexedTimeline, error) {
			var localTimelines []*IndexedTimeline
			for _, timelineChunk := range chunk {
				for _, tl := range timelineChunk.Timelines {
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
				onProcessed(1)
			}
			return localTimelines, nil
		},
		nil,
		worker.ProgressOptions{},
	)
	if err != nil {
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
	if len(timelines) == 0 {
		return
	}

	_, _ = worker.ParallelChunkMap(
		context.Background(),
		timelines,
		func(ctx context.Context, workerIdx int, chunk []*IndexedTimeline, onProcessed func(int)) (struct{}, error) {
			for _, tl := range chunk {
				tl.Path = tl.ComputePath(timelineMap)
				tl.Data.Path = tl.Path
				onProcessed(1)
			}
			return struct{}{}, nil
		},
		nil,
		worker.ProgressOptions{},
	)
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
