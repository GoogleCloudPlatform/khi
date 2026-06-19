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

package khifilev6

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifile "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// StagingRevision represents a resource revision in parser-side domain model before serialization.
// It holds the raw information about resource change to be processed and interned later during serialization.
type StagingRevision struct {
	// ChangedTime is the time of the resource modification.
	ChangedTime time.Time
	// ResourceBody represents the resource content at this revision.
	ResourceBody structured.Node
	// Principal is the requestor of the modification.
	Principal string
	// VerbType is the styled representation of the change verb.
	VerbType *pb.Verb
	// StateType is the styled representation of the revision state.
	StateType *pb.RevisionState
}

// LogChangeSet accumulates metadata updates for a specific log during parsing.
// It decouples the parsing logic from serialization details and provides thread-safe accumulation.
type LogChangeSet struct {
	// Log holds the reference to the parser-side log model.
	Log *log.Log
	// Summary stores the accumulated log summary.
	Summary string
	// Timestamp represents the log timestamp.
	Timestamp time.Time
	// LogType represents the styled log type.
	LogType *pb.LogType
	// Severity represents the styled severity.
	Severity *pb.Severity
}

// NewLogChangeSet creates a new LogChangeSet for staging mutations on the given log.
func NewLogChangeSet(l *log.Log) (*LogChangeSet, error) {
	if l == nil {
		return nil, fmt.Errorf("log cannot be nil")
	}
	return &LogChangeSet{
		Log: l,
	}, nil
}

// SetSummary updates the staged summary string.
func (cs *LogChangeSet) SetSummary(summary string) {
	cs.Summary = summary
}

// SetTimestamp updates the staged timestamp.
func (cs *LogChangeSet) SetTimestamp(t time.Time) {
	cs.Timestamp = t
}

// SetLogType updates the staged log type style.
func (cs *LogChangeSet) SetLogType(lt *pb.LogType) {
	cs.LogType = lt
}

// SetSeverity updates the staged severity style.
func (cs *LogChangeSet) SetSeverity(sev *pb.Severity) {
	cs.Severity = sev
}

// Flush writes the staged log metadata into the provided LogAccumulator.
func (cs *LogChangeSet) Flush(logAcc *LogAccumulator) error {
	return logAcc.AddLog(&StagingLog{
		Log:       cs.Log,
		Summary:   cs.Summary,
		Timestamp: cs.Timestamp,
		LogType:   cs.LogType,
		Severity:  cs.Severity,
	})
}

// TimelineChangeSet stages events and revisions for multiple timeline paths.
// It acts as a localized buffer during the parsing phase of a single log.
type TimelineChangeSet struct {
	// Log holds the reference to the parser-side log model.
	Log *log.Log
	// Events tracks which paths should have an event associated with this log.
	Events map[*TimelinePath]bool
	// Revisions tracks staging revisions to be appended to specific paths.
	Revisions map[*TimelinePath][]*StagingRevision
	// Aliases tracks structural path aliases to be registered. The key is the alias path and the value is the target path.
	Aliases map[*TimelinePath]*TimelinePath
}

// NewTimelineChangeSet creates a new TimelineChangeSet to stage timeline mutations.
func NewTimelineChangeSet(l *log.Log) *TimelineChangeSet {
	return &TimelineChangeSet{
		Log:       l,
		Events:    make(map[*TimelinePath]bool),
		Revisions: make(map[*TimelinePath][]*StagingRevision),
		Aliases:   make(map[*TimelinePath]*TimelinePath),
	}
}

// AddEvent stages a timeline event on the specified path for the log associated with this changeset.
func (cs *TimelineChangeSet) AddEvent(path *TimelinePath) {
	cs.Events[path] = true
}

// AddRevision stages a resource revision on the specified path.
func (cs *TimelineChangeSet) AddRevision(path *TimelinePath, revision *StagingRevision) {
	cs.Revisions[path] = append(cs.Revisions[path], revision)
}

// AddAlias stages an alias mapping from the alias path to the target path.
func (cs *TimelineChangeSet) AddAlias(aliasPath, targetPath *TimelinePath) {
	cs.Aliases[aliasPath] = targetPath
}

// Flush converts staging events, revisions, and aliases to serialized types and writes them to the TimelineAccumulator.
func (cs *TimelineChangeSet) Flush(accumulator *TimelineAccumulator) error {
	registry := accumulator.registry
	pool := registry.internPool
	logAcc := registry.logAcc

	resolvedLogID, ok := logAcc.ResolveLogID(cs.Log.ID)
	if !ok {
		return fmt.Errorf("failed to resolve log ID for parser log %q", cs.Log.ID)
	}

	for aliasPath, targetPath := range cs.Aliases {
		if err := accumulator.SetAlias(aliasPath, targetPath); err != nil {
			return fmt.Errorf("failed to set timeline alias: %w", err)
		}
	}

	for path := range cs.Events {
		builder := registry.GetBuilder(path)
		builder.AddEvent(&pb.Event{
			LogId: &resolvedLogID,
		})
	}

	for path, revisions := range cs.Revisions {
		builder := registry.GetBuilder(path)
		for _, r := range revisions {
			var bodyRef *khifile.InternedStruct
			if r.ResourceBody != nil {
				var err error
				bodyRef, err = ToInternedStruct(r.ResourceBody, pool)
				if err != nil {
					return fmt.Errorf("failed to intern resource body for revision: %w", err)
				}
			}

			var principalID *uint32
			if r.Principal != "" {
				ref := pool.InternString(r.Principal)
				principalID = &ref.id
			}

			var verbID *uint32
			if r.VerbType != nil {
				verbID = r.VerbType.Id
			}

			var stateID *uint32
			if r.StateType != nil {
				stateID = r.StateType.Id
			}

			var changedTime *timestamppb.Timestamp
			if !r.ChangedTime.IsZero() {
				changedTime = timestamppb.New(r.ChangedTime)
			}

			builder.AddRevision(&pb.Revision{
				LogId:             &resolvedLogID,
				ChangedTime:       changedTime,
				ResourceBody:      bodyRef,
				PrincipalStringId: principalID,
				VerbType:          verbID,
				StateType:         stateID,
			})
		}
	}
	return nil
}
