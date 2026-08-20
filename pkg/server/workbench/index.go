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
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
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
	Timelines   []*IndexedTimeline
	TimelineMap map[uint32]*IndexedTimeline
	Logs        []*IndexedLog
	LogMap      map[uint32]*IndexedLog
}

// BuildSearchIndex constructs an in-memory SearchIndex from the parsed Workbench chunks.
func (w *Workbench) BuildSearchIndex() (*SearchIndex, error) {
	idx := &SearchIndex{
		TimelineMap: make(map[uint32]*IndexedTimeline),
		LogMap:      make(map[uint32]*IndexedLog),
	}

	severityOrderMap := make(map[uint32]uint32)
	logTypeLabelMap := make(map[uint32]string)
	timelineTypeLabelMap := make(map[uint32]string)
	verbLabelMap := make(map[uint32]string)
	stateLabelMap := make(map[uint32]string)

	if w.styleChunk != nil {
		for _, s := range w.styleChunk.Severities {
			severityOrderMap[s.GetId()] = uint32(s.GetOrder())
		}
		for _, lt := range w.styleChunk.LogTypes {
			logTypeLabelMap[lt.GetId()] = lt.GetLabel()
		}
		for _, tt := range w.styleChunk.TimelineTypes {
			timelineTypeLabelMap[tt.GetId()] = tt.GetLabel()
		}
		for _, v := range w.styleChunk.Verbs {
			verbLabelMap[v.GetId()] = v.GetLabel()
		}
		for _, rs := range w.styleChunk.RevisionStates {
			stateLabelMap[rs.GetId()] = rs.GetLabel()
		}
	}

	serializer := &structured.YAMLNodeSerializer{}

	// Index Logs
	for _, chunk := range w.logChunks {
		for _, log := range chunk.Logs {
			summary := ""
			if w.internPool != nil {
				summary = w.internPool.ResolveStringFromID(log.GetSummaryStringId())
			}

			severityOrder := severityOrderMap[log.GetSeverityTypeId()]
			logType := logTypeLabelMap[log.GetLogTypeId()]

			var bodyMap map[string]any
			bodyYAML := ""

			if log.GetBodyStructId() != 0 && w.internPool != nil {
				if s := w.internPool.ResolveStructFromID(log.GetBodyStructId()); s != nil {
					if node, err := khifilev6model.FromInternedStruct(s, w.internPool); err == nil {
						if m, ok := nodeToValue(node).(map[string]any); ok {
							bodyMap = m
						}
						if yamlBytes, err := serializer.Serialize(node); err == nil {
							bodyYAML = string(yamlBytes)
						}
					}
				}
			}

			lData := &cel.LogData{
				ID:       log.GetId(),
				LogType:  logType,
				Severity: severityOrder,
				Summary:  summary,
				Body:     bodyMap,
				BodyYAML: bodyYAML,
			}

			indexedLog := &IndexedLog{
				ID:   log.GetId(),
				Data: lData,
			}
			idx.Logs = append(idx.Logs, indexedLog)
			idx.LogMap[log.GetId()] = indexedLog
		}
	}

	// Index Timelines and TimelineItems
	itemsMap := make(map[uint32]*pbv6.TimelineItems)
	for _, chunk := range w.timelineChunks {
		for _, item := range chunk.TimelineItems {
			itemsMap[item.GetId()] = item
		}
	}

	for _, chunk := range w.timelineChunks {
		for _, tl := range chunk.Timelines {
			tlName := ""
			if w.internPool != nil {
				tlName = w.internPool.ResolveStringFromID(tl.GetNameStringId())
			}
			tlType := timelineTypeLabelMap[tl.GetTimelineType()]

			var logIDs []uint32
			var events []cel.EventInfo
			var revisions []cel.RevisionInfo
			var maxSeverity uint32

			if item, ok := itemsMap[tl.GetTimelineItemsId()]; ok {
				for _, evt := range item.Events {
					logIDs = append(logIDs, evt.GetLogId())
					sev := uint32(0)
					if logObj, exists := idx.LogMap[evt.GetLogId()]; exists {
						sev = logObj.Data.Severity
					}
					if sev > maxSeverity {
						maxSeverity = sev
					}
					events = append(events, cel.EventInfo{
						LogID:    evt.GetLogId(),
						Severity: sev,
					})
				}

				for _, rev := range item.Revisions {
					logIDs = append(logIDs, rev.GetLogId())
					principal := ""
					if w.internPool != nil {
						principal = w.internPool.ResolveStringFromID(rev.GetPrincipalStringId())
					}
					verb := verbLabelMap[rev.GetVerbType()]
					state := stateLabelMap[rev.GetStateType()]

					sev := uint32(0)
					if logObj, exists := idx.LogMap[rev.GetLogId()]; exists {
						sev = logObj.Data.Severity
					}
					if sev > maxSeverity {
						maxSeverity = sev
					}

					var bodyMap map[string]any
					bodyYAML := ""

					if rev.GetResourceBodyStructId() != 0 && w.internPool != nil {
						if s := w.internPool.ResolveStructFromID(rev.GetResourceBodyStructId()); s != nil {
							if node, err := khifilev6model.FromInternedStruct(s, w.internPool); err == nil {
								if m, ok := nodeToValue(node).(map[string]any); ok {
									bodyMap = m
								}
								if yamlBytes, err := serializer.Serialize(node); err == nil {
									bodyYAML = string(yamlBytes)
								}
							}
						}
					}

					changedTime := int64(0)
					if rev.ChangedTime != nil {
						changedTime = rev.ChangedTime.AsTime().UnixNano()
					}

					revisions = append(revisions, cel.RevisionInfo{
						LogID:       rev.GetLogId(),
						ChangedTime: changedTime,
						Principal:   principal,
						Verb:        verb,
						State:       state,
						Body:        bodyMap,
						BodyYAML:    bodyYAML,
						Severity:    sev,
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

			idx.Timelines = append(idx.Timelines, indexedTL)
			idx.TimelineMap[tl.GetId()] = indexedTL
		}
	}

	// Link parent-child hierarchy and compute paths
	for _, tl := range idx.Timelines {
		if tl.ParentID != 0 {
			if parent, exists := idx.TimelineMap[tl.ParentID]; exists {
				parent.ChildrenIDs = append(parent.ChildrenIDs, tl.ID)
			}
		}
	}

	for _, tl := range idx.Timelines {
		tl.Path = tl.ComputePath(idx.TimelineMap)
		tl.Data.Path = tl.Path
	}

	return idx, nil
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

func nodeToValue(node structured.Node) any {
	if node == nil {
		return nil
	}
	switch node.Type() {
	case structured.ScalarNodeType:
		val, err := node.NodeScalarValue()
		if err != nil {
			return nil
		}
		return val
	case structured.MapNodeType:
		res := make(map[string]any)
		node.Children()(func(k structured.NodeChildrenKey, v structured.Node) bool {
			res[k.Key] = nodeToValue(v)
			return true
		})
		return res
	case structured.SequenceNodeType:
		var list []any
		node.Children()(func(k structured.NodeChildrenKey, v structured.Node) bool {
			list = append(list, nodeToValue(v))
			return true
		})
		return list
	}
	return nil
}
