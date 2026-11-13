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

package cel

import (
	"cmp"
	"context"
	"slices"
	"strings"
)

const (
	timelineCtxKey contextKey = "khi_cel_timeline"
	logCtxKey      contextKey = "khi_cel_log"
)

var _ StyleResolver = (*SimpleStyleResolver)(nil)

// EventInfo represents lightweight event metadata associated with a timeline for CEL evaluation.
type EventInfo struct {
	// LogID is the 1-based identifier of the associated log entry.
	LogID uint32
	// Timestamp is the event occurrence time in Unix nanoseconds.
	Timestamp int64
}

// RevisionInfo represents lightweight revision history metadata associated with a timeline for CEL evaluation.
type RevisionInfo struct {
	// LogID is the 1-based identifier of the associated log entry.
	LogID uint32
	// ChangedTime is the revision timestamp in Unix nanoseconds.
	ChangedTime int64
	// PrincipalStringID is the intern pool ID of the acting principal string.
	PrincipalStringID uint32
	// Verb is the resolved action verb label for the revision.
	Verb string
	// State is the resolved resource state label for the revision.
	State string
	// ResourceBodyStructID is the intern pool ID of the resource manifest struct.
	ResourceBodyStructID uint32
}

// TimelineData encapsulates the indexed timeline attributes and nested items required for CEL evaluation.
type TimelineData struct {
	// ID is the unique timeline identifier.
	ID uint32
	// ParentID is the parent timeline identifier, or 0 for root timelines.
	ParentID uint32
	// ChildrenIDs lists the direct child timeline identifiers.
	ChildrenIDs []uint32
	// Name is the display name of the timeline.
	Name string
	// TimelineType is the resolved timeline type label.
	TimelineType string
	// Events holds the sorted events associated with this timeline.
	Events []EventInfo
	// Revisions holds the sorted revisions associated with this timeline.
	Revisions []RevisionInfo
	// MaxSeverity is the highest severity order among logs on this timeline.
	MaxSeverity uint32
	// SeverityMask is a bitmask of severity orders present on this timeline.
	SeverityMask uint8
}

// StyleResolver resolves style attributes (e.g. log type label, severity order) from their IDs.
type StyleResolver interface {
	// ResolveLogType returns the log type label corresponding to the given ID.
	ResolveLogType(id uint32) string
	// ResolveSeverity returns the severity order value corresponding to the given ID.
	ResolveSeverity(id uint32) uint32
}

// SimpleStyleResolver provides a map-based StyleResolver implementation for evaluation and testing.
type SimpleStyleResolver struct {
	// LogTypes maps log type IDs to their string labels.
	LogTypes map[uint32]string
	// Severities maps severity IDs to their numeric severity orders.
	Severities map[uint32]uint32
}

// LogData encapsulates the indexed log attributes and struct ID required for CEL evaluation.
// It holds primitive IDs to ensure zero pointers and minimal memory footprint.
type LogData struct {
	// ID is the unique 1-based log identifier.
	ID uint32
	// Timestamp is the log entry occurrence time in Unix nanoseconds.
	Timestamp int64
	// LogTypeID is the style chunk ID of the log type.
	LogTypeID uint32
	// SeverityTypeID is the style chunk ID of the log severity.
	SeverityTypeID uint32
	// SummaryStringID is the intern pool ID of the log summary string.
	SummaryStringID uint32
	// BodyStructID is the intern pool ID of the structured log body.
	BodyStructID uint32
}

type contextKey string

// ForEachLogID iterates over all log IDs associated with this timeline's events and revisions.
// If the callback returns false, iteration stops early.
func (t *TimelineData) ForEachLogID(cb func(logID uint32) bool) {
	if t == nil {
		return
	}
	for _, evt := range t.Events {
		if !cb(evt.LogID) {
			return
		}
	}
	for _, rev := range t.Revisions {
		if !cb(rev.LogID) {
			return
		}
	}
}

// HasActivityInRange checks whether at least one event or revision falls within [startTimeNs, endTimeNs] (inclusive).
// It searches t.Events (by Timestamp) and t.Revisions (by ChangedTime) using binary search in O(log E + log R).
func (t *TimelineData) HasActivityInRange(startTimeNs, endTimeNs int64) bool {
	if t == nil || startTimeNs > endTimeNs {
		return false
	}
	evtIdx, _ := slices.BinarySearchFunc(t.Events, startTimeNs, func(e EventInfo, target int64) int {
		return cmp.Compare(e.Timestamp, target)
	})
	if evtIdx < len(t.Events) && t.Events[evtIdx].Timestamp <= endTimeNs {
		return true
	}
	revIdx, _ := slices.BinarySearchFunc(t.Revisions, startTimeNs, func(r RevisionInfo, target int64) int {
		return cmp.Compare(r.ChangedTime, target)
	})
	return revIdx < len(t.Revisions) && t.Revisions[revIdx].ChangedTime <= endTimeNs
}

// ForEachLogIDInRange iterates over log IDs associated with this timeline's events and revisions
// that fall within [startTimeNs, endTimeNs] (inclusive).
// If the callback returns false, iteration stops early.
func (t *TimelineData) ForEachLogIDInRange(startTimeNs, endTimeNs int64, cb func(logID uint32) bool) {
	if t == nil || startTimeNs > endTimeNs {
		return
	}
	evtIdx, _ := slices.BinarySearchFunc(t.Events, startTimeNs, func(e EventInfo, target int64) int {
		return cmp.Compare(e.Timestamp, target)
	})
	for i := evtIdx; i < len(t.Events) && t.Events[i].Timestamp <= endTimeNs; i++ {
		if !cb(t.Events[i].LogID) {
			return
		}
	}
	revIdx, _ := slices.BinarySearchFunc(t.Revisions, startTimeNs, func(r RevisionInfo, target int64) int {
		return cmp.Compare(r.ChangedTime, target)
	})
	for i := revIdx; i < len(t.Revisions) && t.Revisions[i].ChangedTime <= endTimeNs; i++ {
		if !cb(t.Revisions[i].LogID) {
			return
		}
	}
}

// ComputePath resolves the timeline hierarchy path map on demand by traversing parent timelines.
func (t *TimelineData) ComputePath(tlMap map[uint32]*TimelineData) map[string]string {
	if t == nil {
		return nil
	}
	path := make(map[string]string)
	visited := make(map[uint32]struct{})
	curr := t
	for curr != nil {
		if _, seen := visited[curr.ID]; seen {
			break
		}
		visited[curr.ID] = struct{}{}

		typeKey := strings.ToLower(curr.TimelineType)
		if typeKey != "" {
			path[typeKey] = curr.Name
		}
		if curr.ParentID == 0 || tlMap == nil {
			break
		}
		curr = tlMap[curr.ParentID]
	}
	return path
}

// ResolveLogType returns the log type label corresponding to the given ID.
func (s *SimpleStyleResolver) ResolveLogType(id uint32) string {
	if s != nil && s.LogTypes != nil {
		return s.LogTypes[id]
	}
	return ""
}

// ResolveSeverity returns the severity order value corresponding to the given ID.
func (s *SimpleStyleResolver) ResolveSeverity(id uint32) uint32 {
	if s != nil && s.Severities != nil {
		return s.Severities[id]
	}
	return 0
}

// WithTimelineContext binds the given TimelineData to the context for CEL evaluation.
func WithTimelineContext(ctx context.Context, t *TimelineData) context.Context {
	return context.WithValue(ctx, timelineCtxKey, t)
}

// TimelineFromContext retrieves the TimelineData bound to the context.
func TimelineFromContext(ctx context.Context) *TimelineData {
	if val, ok := ctx.Value(timelineCtxKey).(*TimelineData); ok {
		return val
	}
	return nil
}

// WithLogContext binds the given LogData to the context for CEL evaluation.
func WithLogContext(ctx context.Context, l *LogData) context.Context {
	return context.WithValue(ctx, logCtxKey, l)
}

// LogFromContext retrieves the LogData bound to the context.
func LogFromContext(ctx context.Context) *LogData {
	if val, ok := ctx.Value(logCtxKey).(*LogData); ok {
		return val
	}
	return nil
}
