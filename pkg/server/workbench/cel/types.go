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
	"context"
	"strings"
)

type contextKey string

const (
	timelineCtxKey contextKey = "khi_cel_timeline"
	logCtxKey      contextKey = "khi_cel_log"
)

// EventInfo represents lightweight event metadata associated with a timeline for CEL evaluation.
type EventInfo struct {
	LogID    uint32
	Severity uint32
}

// RevisionInfo represents lightweight revision history metadata associated with a timeline for CEL evaluation.
type RevisionInfo struct {
	LogID                uint32
	ChangedTime          int64
	PrincipalStringID    uint32
	Verb                 string
	State                string
	ResourceBodyStructID uint32
	Severity             uint32
}

// TimelineData encapsulates the indexed timeline attributes and nested items required for CEL evaluation.
type TimelineData struct {
	ID           uint32
	ParentID     uint32
	ChildrenIDs  []uint32
	Name         string
	TimelineType string
	Events       []EventInfo
	Revisions    []RevisionInfo
	MaxSeverity  uint32
	SeverityMask uint8
}

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

// StyleResolver resolves style attributes (e.g. log type label, severity order) from their IDs.
type StyleResolver interface {
	ResolveLogType(id uint32) string
	ResolveSeverity(id uint32) uint32
}

// SimpleStyleResolver provides a map-based StyleResolver implementation for evaluation and testing.
type SimpleStyleResolver struct {
	LogTypes   map[uint32]string
	Severities map[uint32]uint32
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

var _ StyleResolver = (*SimpleStyleResolver)(nil)

// LogData encapsulates the indexed log attributes and struct ID required for CEL evaluation.
// It holds primitive IDs to ensure zero pointers and minimal memory footprint.
type LogData struct {
	ID              uint32
	LogTypeID       uint32
	SeverityTypeID  uint32
	SummaryStringID uint32
	BodyStructID    uint32
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
