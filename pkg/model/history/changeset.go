// Copyright 2024 Google LLC
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

package history

import (
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

// history.ChangeSet is set of changes applicable to history.
// A parser ingest a log.LogEntry and returns a ChangeSet. ChangeSet contains multiple changes against the history.
// This change is applied atomically, when the parser returns an error, no partial changes would be written.
// **IMPORTANT** Note this type is not expected to be used from multi threads.
// Parsers will write its result to this type once and it will be flushed to the History type with acuireing locks.
type ChangeSet struct {
	// Log is the related log of this ChangeSet.
	Log *log.Log
	// RevisionMap is the map of a list of StagingResourceRevision staged on this ChangeSet.
	RevisionsMap map[string][]*StagingResourceRevision
	// EventsMap is the map of a list of StagingResourceEvent staged on this ChangeSet.
	EventsMap map[string][]*ResourceEvent
	// ResourceRelationshipRewrites is the map from its resourcePath to its ParentRelationship.
	// TODO: A resource can have same name subresource with different relationship in future?
	ResourceRelationshipRewrites map[string]enum.ParentRelationship
	// Annotations is the list of extra information associated to this log.
	Annotations []LogAnnotation
	// LogSummary is the summary string of this log to be rewritten with.
	LogSummary string
	// LogSeverity is the severity of this log to be rewritten with.
	LogSeverity enum.Severity
	// Aliases are the map of timeline aliases. The key is the source resource resource path and the list is the destination of its alias.
	Aliases map[string][]string
}

func NewChangeSet(l *log.Log) *ChangeSet {
	return &ChangeSet{
		Log:                          l,
		RevisionsMap:                 make(map[string][]*StagingResourceRevision),
		EventsMap:                    make(map[string][]*ResourceEvent),
		ResourceRelationshipRewrites: make(map[string]enum.ParentRelationship),
		LogSummary:                   "",
		LogSeverity:                  enum.SeverityUnknown,
		Annotations:                  []LogAnnotation{},
		Aliases:                      map[string][]string{},
	}
}

// SetLogSummary sets the summary string for the log associated with this ChangeSet.
func (cs *ChangeSet) SetLogSummary(summary string) {
	cs.LogSummary = summary
}

// SetLogSeverity sets the severity for the log associated with this ChangeSet.
func (cs *ChangeSet) SetLogSeverity(severity enum.Severity) {
	cs.LogSeverity = severity
}

// AddRevision adds a StagingResourceRevision to the ChangeSet for a given resource path.
func (cs *ChangeSet) AddRevision(resourcePath resourcepath.ResourcePath, revision *StagingResourceRevision) {
	if _, exist := cs.RevisionsMap[resourcePath.Path]; !exist {
		cs.RevisionsMap[resourcePath.Path] = make([]*StagingResourceRevision, 0)
	}
	cs.RevisionsMap[resourcePath.Path] = append(cs.RevisionsMap[resourcePath.Path], revision)
	if !revision.Inferred {
		cs.Annotations = append(cs.Annotations, NewResourceReferenceAnnotation(resourcePath.Path))
	}
	cs.addResourceRelationship(resourcePath)
}

// GetAllResourcePaths returns the all of resource paths included in this ChangeSet.
func (cs *ChangeSet) GetAllResourcePaths() []string {
	paths := []string{}
	for k := range cs.RevisionsMap {
		paths = append(paths, k)
	}
	for k := range cs.EventsMap {
		paths = append(paths, k)
	}
	return common.DedupStringArray(paths)
}

// GetRevisions returns every StagingResourceRevisions at the specified resource path.
// If no revisions exist for the given path, it returns nil.
func (cs *ChangeSet) GetRevisions(resourcePath resourcepath.ResourcePath) []*StagingResourceRevision {
	if revisions, exist := cs.RevisionsMap[resourcePath.Path]; exist {
		return revisions
	}
	return nil
}

// AddEvent adds a ResourceEvent to the ChangeSet for a given resource path.
func (cs *ChangeSet) AddEvent(resourcePath resourcepath.ResourcePath) {
	event := ResourceEvent{
		Log: cs.Log.ID,
	}
	if _, exist := cs.EventsMap[resourcePath.Path]; !exist {
		cs.EventsMap[resourcePath.Path] = make([]*ResourceEvent, 0)
	}
	cs.EventsMap[resourcePath.Path] = append(cs.EventsMap[resourcePath.Path], &event)
	cs.Annotations = append(cs.Annotations, NewResourceReferenceAnnotation(resourcePath.Path))
	cs.addResourceRelationship(resourcePath)
}

// GetEvents returns every ResourceEvents at the specified resource path.
// If no events exist for the given path, it returns nil.
func (cs *ChangeSet) GetEvents(resourcePath resourcepath.ResourcePath) []*ResourceEvent {
	if events, exist := cs.EventsMap[resourcePath.Path]; exist {
		return events
	}
	return nil
}

// AddResourceAlias adds an alias from a source resource path to a destination resource path.
func (cs *ChangeSet) AddResourceAlias(sourceResourcePath resourcepath.ResourcePath, destResourcePath resourcepath.ResourcePath) {
	if _, exist := cs.Aliases[sourceResourcePath.Path]; !exist {
		cs.Aliases[sourceResourcePath.Path] = make([]string, 0)
	}
	for _, d := range cs.Aliases[sourceResourcePath.Path] {
		if d == destResourcePath.Path {
			return
		}
	}
	cs.Aliases[sourceResourcePath.Path] = append(cs.Aliases[sourceResourcePath.Path], destResourcePath.Path)
	cs.addResourceRelationship(destResourcePath)
}

// addResourceRelationship records the parent relationship for a given resource path.
// It returns an error if the relationship for the given path has already been set to a different value.
func (cs *ChangeSet) addResourceRelationship(resourcePath resourcepath.ResourcePath) error {
	if lastRelationship, found := cs.ResourceRelationshipRewrites[resourcePath.Path]; found {
		if lastRelationship != resourcePath.ParentRelationship {
			return fmt.Errorf("failed to rewrite the parentRelationship of %s. It was already rewritten to %d", resourcePath.Path, lastRelationship)
		}
	} else {
		cs.ResourceRelationshipRewrites[resourcePath.Path] = resourcePath.ParentRelationship
	}
	return nil
}

// FlushToHistory writes the recorded changeset to the history and returns resource paths where the resource modified.
// This method applies all staged revisions, events, log properties, aliases, and resource relationships
// to the provided Builder. It returns a list of resource paths that were modified and any error encountered.
func (cs *ChangeSet) FlushToHistory(builder *Builder) ([]string, error) {
	changedPaths := []string{}
	// Write revisions in this ChangeSet
	for resourcePath, revisions := range cs.RevisionsMap {
		tb := builder.GetTimelineBuilder(resourcePath)
		for _, stagingRevision := range revisions {
			revision, err := stagingRevision.commit(builder.binaryChunk, cs.Log)
			if err != nil {
				return nil, err
			}
			tb.AddRevision(revision)
		}
		changedPaths = append(changedPaths, resourcePath)
	}
	// Write events in this ChangeSet
	for resourcePath, events := range cs.EventsMap {
		tb := builder.GetTimelineBuilder(resourcePath)
		for _, event := range events {
			tb.AddEvent(event)
		}
		changedPaths = append(changedPaths, resourcePath)
	}

	// Write log related properties
	if cs.LogSummary != "" {
		builder.setLogSummary(cs.Log.ID, cs.LogSummary)
	}
	if cs.LogSeverity != enum.SeverityUnknown {
		builder.setLogSeverity(cs.Log.ID, cs.LogSeverity)
	}
	builder.setLogAnnotations(cs.Log.ID, cs.Annotations)

	// Write the alias relationships
	for source, destinations := range cs.Aliases {
		for _, dest := range destinations {
			builder.addTimelineAlias(source, dest)
		}
	}

	// Write resource related properties
	for resourcePath, relationship := range cs.ResourceRelationshipRewrites {
		err := builder.rewriteRelationship(resourcePath, relationship)
		if err != nil {
			return nil, err
		}
	}
	return common.DedupStringArray(changedPaths), nil
}
