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
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	pbv6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
)

// IndexedTimeline is an alias for cel.TimelineData.
type IndexedTimeline = cel.TimelineData

// IndexedLog is an alias for cel.LogData.
type IndexedLog = cel.LogData

// SearchIndex encapsulates the indexed timelines and logs of a Workbench session.
type SearchIndex struct {
	Timelines    []*cel.TimelineData
	TimelineMap  map[uint32]*cel.TimelineData
	Logs         []cel.LogData
	InternPool   *khifilev6model.InternPool
	StructYAMLs  map[uint32]string
	TrigramIndex *cel.TrigramIndex
}

// GetLog retrieves a log entry by its 1-based log ID in O(1) time.
func (s *SearchIndex) GetLog(id uint32) *cel.LogData {
	if s == nil || id == 0 || int(id) > len(s.Logs) {
		return nil
	}
	return &s.Logs[id-1]
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

	logs, err := w.indexLogsParallel(styles)
	if err != nil {
		return nil, fmt.Errorf("failed to index logs: %w", err)
	}

	itemsMap := w.buildTimelineItemsMap()
	timelines, timelineMap, err := w.indexTimelinesParallel(styles, itemsMap, logs)
	if err != nil {
		return nil, fmt.Errorf("failed to index timelines: %w", err)
	}

	w.linkTimelineHierarchy(timelines, timelineMap)

	return &SearchIndex{
		Timelines:    timelines,
		TimelineMap:  timelineMap,
		Logs:         logs,
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
func (w *Workbench) BuildStructYAMLIndexWithProgress(ctx context.Context, targetIndex *SearchIndex, onProgress ProgressCallback) (map[uint32]string, error) {
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
	for i := range targetIndex.Logs {
		l := &targetIndex.Logs[i]
		if l.BodyStructID != 0 {
			if _, ok := seenStructIDs[l.BodyStructID]; !ok {
				seenStructIDs[l.BodyStructID] = struct{}{}
				uniqueBodyStructIDs = append(uniqueBodyStructIDs, l.BodyStructID)
			}
		}
	}

	yamlResults, err := worker.ParallelChunkMap(
		ctx,
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
func (w *Workbench) BuildTrigramIndexWithProgress(ctx context.Context, structYAMLs map[uint32]string, onProgress ProgressCallback) (*cel.TrigramIndex, error) {
	trigramIndex := cel.NewTrigramIndex()
	err := trigramIndex.BuildFromStructYAMLs(ctx, structYAMLs, func(subPct float64, msg string) error {
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
func (w *Workbench) BuildAsyncIndexesWithProgress(ctx context.Context, targetIndex *SearchIndex, onProgress ProgressCallback) error {
	if targetIndex == nil {
		return fmt.Errorf("target search index is nil")
	}

	// Stage 1: Parallel Struct YAML Serialization (0% - 50%)
	structYAMLs, err := w.BuildStructYAMLIndexWithProgress(ctx, targetIndex, func(stage apiv1.OpenWorkbenchResponse_Stage, progressPercentage float64, message string) error {
		return onProgress(stage, progressPercentage*0.5, message)
	})
	if err != nil {
		return err
	}
	targetIndex.StructYAMLs = structYAMLs

	// Stage 2: Parallel Trigram Index Construction (50% - 100%)
	trigramIndex, err := w.BuildTrigramIndexWithProgress(ctx, structYAMLs, func(stage apiv1.OpenWorkbenchResponse_Stage, progressPercentage float64, message string) error {
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

func (w *Workbench) indexLogsParallel(styles *styleMaps) ([]cel.LogData, error) {
	if len(w.logChunks) == 0 {
		return nil, nil
	}

	workerResults, err := worker.ParallelChunkMap(
		context.Background(),
		w.logChunks,
		func(ctx context.Context, workerIdx int, chunk []*pbv6.LogChunk, onProcessed func(int)) ([]cel.LogData, error) {
			var localLogs []cel.LogData
			for _, logChunk := range chunk {
				for _, log := range logChunk.Logs {
					sevOrder := styles.severityOrderMap[log.GetSeverityTypeId()]
					ltLabel := styles.logTypeLabelMap[log.GetLogTypeId()]

					localLogs = append(localLogs, cel.LogData{
						ID:              log.GetId(),
						LogType:         ltLabel,
						Severity:        sevOrder,
						SummaryStringID: log.GetSummaryStringId(),
						BodyStructID:    log.GetBodyStructId(),
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
		return nil, err
	}

	maxLogID := uint32(0)
	for _, res := range workerResults {
		for _, item := range res {
			if item.ID > maxLogID {
				maxLogID = item.ID
			}
		}
	}

	logs := make([]cel.LogData, maxLogID)
	for _, res := range workerResults {
		for _, item := range res {
			if item.ID > 0 {
				logs[item.ID-1] = item
			}
		}
	}

	return logs, nil
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
	logs []cel.LogData,
) ([]*cel.TimelineData, map[uint32]*cel.TimelineData, error) {
	if len(w.timelineChunks) == 0 {
		return nil, make(map[uint32]*cel.TimelineData), nil
	}

	getLogSeverity := func(logID uint32) uint32 {
		if logID > 0 && int(logID) <= len(logs) {
			return logs[logID-1].Severity
		}
		return 0
	}

	workerResults, err := worker.ParallelChunkMap(
		context.Background(),
		w.timelineChunks,
		func(ctx context.Context, workerIdx int, chunk []*pbv6.TimelineChunk, onProcessed func(int)) ([]*cel.TimelineData, error) {
			var localTimelines []*cel.TimelineData
			for _, timelineChunk := range chunk {
				for _, tl := range timelineChunk.Timelines {
					tlName := ""
					if w.internPool != nil {
						tlName = w.internPool.ResolveStringFromID(tl.GetNameStringId())
					}
					tlType := styles.timelineTypeLabelMap[tl.GetTimelineType()]

					var events []cel.EventInfo
					var revisions []cel.RevisionInfo
					var maxSeverity uint32

					if item, ok := itemsMap[tl.GetTimelineItemsId()]; ok {
						for _, evt := range item.Events {
							logID := evt.GetLogId()
							sev := getLogSeverity(logID)
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
							verb := styles.verbLabelMap[rev.GetVerbType()]
							state := styles.stateLabelMap[rev.GetStateType()]
							sev := getLogSeverity(logID)
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
						ParentID:     tl.GetParentTimelineId(),
						Name:         tlName,
						TimelineType: tlType,
						Events:       events,
						Revisions:    revisions,
						MaxSeverity:  maxSeverity,
					}

					localTimelines = append(localTimelines, tData)
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

	timelines := make([]*cel.TimelineData, 0, totalTimelines)
	timelineMap := make(map[uint32]*cel.TimelineData, totalTimelines)

	for _, res := range workerResults {
		for _, item := range res {
			timelines = append(timelines, item)
			timelineMap[item.ID] = item
		}
	}

	return timelines, timelineMap, nil
}

func (w *Workbench) linkTimelineHierarchy(timelines []*cel.TimelineData, timelineMap map[uint32]*cel.TimelineData) {
	for _, tl := range timelines {
		if tl.ParentID != 0 {
			if parent, exists := timelineMap[tl.ParentID]; exists {
				parent.ChildrenIDs = append(parent.ChildrenIDs, tl.ID)
			}
		}
	}
}
