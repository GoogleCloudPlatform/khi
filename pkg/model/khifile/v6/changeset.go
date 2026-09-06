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
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
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
	// FieldAnnotations stores field-specific annotations to be displayed in the UI.
	FieldAnnotations []*StagingFieldAnnotation
}

// StagingFieldAnnotation represents a staged metadata annotation for a specific field.
type StagingFieldAnnotation struct {
	FieldPath       string
	MutatingWebhook *StagingMutatingWebhook
}

// StagingMutatingWebhook represents the payload for a mutating webhook annotation.
type StagingMutatingWebhook struct {
	Configuration string
	Webhook       string
	Round         int32
	Index         int32
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

const (
	defaultSliceCapacity     = 4
	maxRetainedSliceCapacity = 32
	maxRetainedMapEntries    = 32
)

var timelineChangeSetPool = sync.Pool{
	New: func() any {
		return &TimelineChangeSet{
			Events:    make([]*TimelinePath, 0, defaultSliceCapacity),
			Revisions: make(map[*TimelinePath][]*StagingRevision),
			Aliases:   make(map[*TimelinePath]*TimelinePath),
		}
	},
}

// TimelineChangeSet stages events and revisions for multiple timeline paths.
// It acts as a localized buffer during the parsing phase of a single log.
// Instances are pooled via sync.Pool to eliminate allocation overhead.
type TimelineChangeSet struct {
	// Log holds the reference to the parser-side log model.
	Log *log.Log

	Events    []*TimelinePath
	Revisions map[*TimelinePath][]*StagingRevision
	Aliases   map[*TimelinePath]*TimelinePath
}

// NewTimelineChangeSet retrieves a TimelineChangeSet from the pool or allocates a new one.
func NewTimelineChangeSet(l *log.Log) *TimelineChangeSet {
	cs := timelineChangeSetPool.Get().(*TimelineChangeSet)
	cs.Log = l
	return cs
}

// Release resets the changeset fields and returns it to the pool for reuse.
func (cs *TimelineChangeSet) Release() {
	if cs == nil {
		return
	}
	cs.Log = nil

	if cap(cs.Events) > maxRetainedSliceCapacity {
		cs.Events = make([]*TimelinePath, 0, defaultSliceCapacity)
	} else {
		clear(cs.Events)
		cs.Events = cs.Events[:0]
	}

	if len(cs.Revisions) > maxRetainedMapEntries {
		cs.Revisions = make(map[*TimelinePath][]*StagingRevision)
	} else {
		clear(cs.Revisions)
	}

	if len(cs.Aliases) > maxRetainedMapEntries {
		cs.Aliases = make(map[*TimelinePath]*TimelinePath)
	} else {
		clear(cs.Aliases)
	}

	timelineChangeSetPool.Put(cs)
}

// AddEvent stages a timeline event on the specified path for the log associated with this changeset.
func (cs *TimelineChangeSet) AddEvent(path *TimelinePath) {
	for _, p := range cs.Events {
		if p == path {
			return
		}
	}
	cs.Events = append(cs.Events, path)
}

// AddRevision stages a resource revision on the specified path.
func (cs *TimelineChangeSet) AddRevision(path *TimelinePath, revision *StagingRevision) {
	cs.Revisions[path] = append(cs.Revisions[path], revision)
}

// AddAlias stages an alias mapping from the alias path to the target path.
func (cs *TimelineChangeSet) AddAlias(aliasPath, targetPath *TimelinePath) {
	cs.Aliases[aliasPath] = targetPath
}

// HasEvent reports whether an event is staged on the given path.
func (cs *TimelineChangeSet) HasEvent(path *TimelinePath) bool {
	for _, p := range cs.Events {
		if p == path {
			return true
		}
	}
	return false
}

// GetRevisions returns all staging revisions for the given path.
func (cs *TimelineChangeSet) GetRevisions(path *TimelinePath) []*StagingRevision {
	return cs.Revisions[path]
}

// GetAlias returns the target path mapped by aliasPath, if staged.
func (cs *TimelineChangeSet) GetAlias(aliasPath *TimelinePath) (*TimelinePath, bool) {
	target, ok := cs.Aliases[aliasPath]
	return target, ok
}

// ForEachEvent calls fn for each staged event path.
func (cs *TimelineChangeSet) ForEachEvent(fn func(path *TimelinePath)) {
	for _, p := range cs.Events {
		fn(p)
	}
}

// ForEachRevision calls fn for each path and its staged revisions.
func (cs *TimelineChangeSet) ForEachRevision(fn func(path *TimelinePath, revs []*StagingRevision)) {
	for p, revs := range cs.Revisions {
		fn(p, revs)
	}
}

// ForEachAlias calls fn for each staged alias-target pair.
func (cs *TimelineChangeSet) ForEachAlias(fn func(aliasPath, targetPath *TimelinePath)) {
	for alias, target := range cs.Aliases {
		fn(alias, target)
	}
}

// IsEmpty reports whether the changeset contains no staged mutations.
func (cs *TimelineChangeSet) IsEmpty() bool {
	return len(cs.Events) == 0 && len(cs.Revisions) == 0 && len(cs.Aliases) == 0
}

// Flush converts staging events, revisions, and aliases to serialized types and writes them to the TimelineAccumulator.
func (cs *TimelineChangeSet) Flush(accumulator *TimelineAccumulator, logAcc *LogAccumulator) error {
	registry := accumulator.registry
	clientPool := registry.clientPool
	serverPool := registry.serverPool

	resolvedLogID, ok := logAcc.ResolveLogID(cs.Log.ID)
	if !ok {
		return fmt.Errorf("failed to resolve log ID for parser log %d", cs.Log.ID)
	}

	var err error
	cs.ForEachAlias(func(aliasPath, targetPath *TimelinePath) {
		if err == nil {
			err = accumulator.SetAlias(aliasPath, targetPath)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to set timeline alias: %w", err)
	}

	cs.ForEachEvent(func(path *TimelinePath) {
		builder := registry.GetBuilder(path)
		builder.AddEvent(pendingEvent{
			LogID:     resolvedLogID,
			Timestamp: cs.Log.Timestamp,
		})
	})

	var flushErr error
	flushRevision := func(path *TimelinePath, r *StagingRevision) error {
		var bodyStructID uint32
		if r.ResourceBody != nil {
			structRef, err := ToInternedStruct(r.ResourceBody, serverPool)
			if err != nil {
				return fmt.Errorf("failed to intern resource body for revision: %w", err)
			}
			bodyStructID = structRef.id
		}

		var principalID uint32
		if r.Principal != "" {
			ref := clientPool.InternString(r.Principal)
			principalID = ref.id
		}

		verbID := r.VerbType.GetId()
		stateID := r.StateType.GetId()

		annotations := make([]pendingFieldAnnotation, 0, len(r.FieldAnnotations))
		for _, fa := range r.FieldAnnotations {
			fieldPathRef := clientPool.InternString(fa.FieldPath)
			ann := pendingFieldAnnotation{
				FieldPathStringID: fieldPathRef.id,
			}
			if fa.MutatingWebhook != nil {
				configRef := clientPool.InternString(fa.MutatingWebhook.Configuration)
				webhookRef := clientPool.InternString(fa.MutatingWebhook.Webhook)
				ann.MutatingWebhook = &pendingMutatingWebhookInfo{
					ConfigurationStringID: configRef.id,
					WebhookStringID:       webhookRef.id,
					Round:                 fa.MutatingWebhook.Round,
					Index:                 fa.MutatingWebhook.Index,
				}
			}
			annotations = append(annotations, ann)
		}

		builder := registry.GetBuilder(path)
		builder.AddRevision(pendingRevision{
			LogID:                resolvedLogID,
			ChangedTime:          r.ChangedTime,
			ResourceBodyStructID: bodyStructID,
			PrincipalStringID:    principalID,
			VerbType:             verbID,
			StateType:            stateID,
			FieldAnnotations:     annotations,
		})
		return nil
	}

	cs.ForEachRevision(func(path *TimelinePath, revisions []*StagingRevision) {
		if flushErr != nil {
			return
		}
		for _, r := range revisions {
			if err := flushRevision(path, r); err != nil {
				flushErr = err
				return
			}
		}
	})
	if flushErr != nil {
		return flushErr
	}
	return nil
}
