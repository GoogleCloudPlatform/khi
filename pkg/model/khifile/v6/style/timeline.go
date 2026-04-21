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

package style

import (
	"sync"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
)

var (
	mu             sync.RWMutex
	severities     []*pb.Severity
	verbs          []*pb.Verb
	logTypes       []*pb.LogType
	revisionStates []*pb.RevisionState
	timelineTypes  []*pb.TimelineType
)

// Reset clears the registry. This is primarily intended for testing to avoid test pollution
// across different packages testing plugin loading.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	severities = nil
	verbs = nil
	logTypes = nil
	revisionStates = nil
	timelineTypes = nil
}

// RegisterTimelineType registers a TimelineType, assigns a unique ID to it,
// mutates the provided pointer by setting its Id field, and returns the same pointer.
// This allows for global inline initialization in plugins.
func RegisterTimelineType(t *pb.TimelineType) *pb.TimelineType {
	mu.Lock()
	defer mu.Unlock()

	id := uint32(len(timelineTypes) + 1)
	t.Id = &id
	timelineTypes = append(timelineTypes, t)
	return t
}

// RegisterSeverity registers a Severity, assigns a unique ID to it,
// mutates the provided pointer by setting its Id field, and returns the same pointer.
func RegisterSeverity(s *pb.Severity) *pb.Severity {
	mu.Lock()
	defer mu.Unlock()

	id := uint32(len(severities) + 1)
	s.Id = &id
	severities = append(severities, s)
	return s
}

// RegisterVerb registers a Verb, assigns a unique ID to it,
// mutates the provided pointer by setting its Id field, and returns the same pointer.
func RegisterVerb(v *pb.Verb) *pb.Verb {
	mu.Lock()
	defer mu.Unlock()

	id := uint32(len(verbs) + 1)
	v.Id = &id
	verbs = append(verbs, v)
	return v
}

// RegisterLogType registers a LogType, assigns a unique ID to it,
// mutates the provided pointer by setting its Id field, and returns the same pointer.
func RegisterLogType(l *pb.LogType) *pb.LogType {
	mu.Lock()
	defer mu.Unlock()

	id := uint32(len(logTypes) + 1)
	l.Id = &id
	logTypes = append(logTypes, l)
	return l
}

// RegisterRevisionState registers a RevisionState, assigns a unique ID to it,
// mutates the provided pointer by setting its Id field, and returns the same pointer.
func RegisterRevisionState(rs *pb.RevisionState) *pb.RevisionState {
	mu.Lock()
	defer mu.Unlock()

	id := uint32(len(revisionStates) + 1)
	rs.Id = &id
	revisionStates = append(revisionStates, rs)
	return rs
}

// GenerateChunk generates the final TimelineStyleChunk containing all registered styles.
func GenerateChunk() *pb.TimelineStyleChunk {
	mu.RLock()
	defer mu.RUnlock()

	// Create shallow copies of the slices to avoid concurrent modification issues
	// if someone decides to append while another is iterating over the chunk.
	cSeverities := make([]*pb.Severity, len(severities))
	copy(cSeverities, severities)

	cVerbs := make([]*pb.Verb, len(verbs))
	copy(cVerbs, verbs)

	cLogTypes := make([]*pb.LogType, len(logTypes))
	copy(cLogTypes, logTypes)

	cRevisionStates := make([]*pb.RevisionState, len(revisionStates))
	copy(cRevisionStates, revisionStates)

	cTimelineTypes := make([]*pb.TimelineType, len(timelineTypes))
	copy(cTimelineTypes, timelineTypes)

	return &pb.TimelineStyleChunk{
		Severities:     cSeverities,
		Verbs:          cVerbs,
		LogTypes:       cLogTypes,
		RevisionStates: cRevisionStates,
		TimelineTypes:  cTimelineTypes,
	}
}
