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
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
)

// history.ChangeSet is set of changes applicable to history.
// A parser ingest a log.LogEntry and returns a ChangeSet. ChangeSet contains multiple changes against the history.
// This change is applied atomically, when the parser returns an error, no partial changes would be written.
type ChangeSet struct {
	associatedLog                *log.Log
	revisions                    map[string][]*StagingResourceRevision
	events                       map[string][]*ResourceEvent
	resourceRelationshipRewrites map[string]enum.ParentRelationship
	annotations                  []LogAnnotation
	logSummaryRewrite            string
	logSeverityRewrite           enum.Severity
	aliases                      map[string][]string
}

func NewChangeSet(l *log.Log) *ChangeSet {
	return &ChangeSet{
		associatedLog:                l,
		revisions:                    make(map[string][]*StagingResourceRevision),
		events:                       make(map[string][]*ResourceEvent),
		resourceRelationshipRewrites: make(map[string]enum.ParentRelationship),
		logSummaryRewrite:            "",
		logSeverityRewrite:           enum.SeverityUnknown,
		annotations:                  []LogAnnotation{},
		aliases:                      map[string][]string{},
	}
}

func (cs *ChangeSet) RecordLogSummary(summary string) {
	cs.logSummaryRewrite = summary
}

// GetLogSummary returns the summary of log to be written with the log.
func (cs *ChangeSet) GetLogSummary() string {
	return cs.logSummaryRewrite
}

func (cs *ChangeSet) RecordLogSeverity(severity enum.Severity) {
	cs.logSeverityRewrite = severity
}

func (cs *ChangeSet) RecordRevision(resourcePath resourcepath.ResourcePath, revision *StagingResourceRevision) {
	if _, exist := cs.revisions[resourcePath.Path]; !exist {
		cs.revisions[resourcePath.Path] = make([]*StagingResourceRevision, 0)
	}
	cs.revisions[resourcePath.Path] = append(cs.revisions[resourcePath.Path], revision)
	if !revision.Inferred {
		cs.annotations = append(cs.annotations, NewResourceReferenceAnnotation(resourcePath.Path))
	}
	cs.recordResourceRelationship(resourcePath)
}

// GetAllResourcePaths returns the all of resource paths included in this ChangeSet.
func (cs *ChangeSet) GetAllResourcePaths() []string {
	paths := []string{}
	for k := range cs.revisions {
		paths = append(paths, k)
	}
	for k := range cs.events {
		paths = append(paths, k)
	}
	return common.DedupStringArray(paths)

}

// GetRevisions returns every StagingResourceRevisions at the specified resource path.
func (cs *ChangeSet) GetRevisions(resourcePath resourcepath.ResourcePath) []*StagingResourceRevision {
	if revisions, exist := cs.revisions[resourcePath.Path]; exist {
		return revisions
	}
	return nil
}

func (cs *ChangeSet) RecordEvent(resourcePath resourcepath.ResourcePath) {
	event := ResourceEvent{
		Log: cs.associatedLog.ID,
	}
	if _, exist := cs.events[resourcePath.Path]; !exist {
		cs.events[resourcePath.Path] = make([]*ResourceEvent, 0)
	}
	cs.events[resourcePath.Path] = append(cs.events[resourcePath.Path], &event)
	cs.annotations = append(cs.annotations, NewResourceReferenceAnnotation(resourcePath.Path))
	cs.recordResourceRelationship(resourcePath)
}

// GetEvents returns every ResourceEvents at the specified resource path.
func (cs *ChangeSet) GetEvents(resourcePath resourcepath.ResourcePath) []*ResourceEvent {
	if events, exist := cs.events[resourcePath.Path]; exist {
		return events
	}
	return nil
}

func (cs *ChangeSet) RecordResourceAlias(sourceResourcePath resourcepath.ResourcePath, destResourcePath resourcepath.ResourcePath) {
	if _, exist := cs.aliases[sourceResourcePath.Path]; !exist {
		cs.aliases[sourceResourcePath.Path] = make([]string, 0)
	}
	for _, d := range cs.aliases[sourceResourcePath.Path] {
		if d == destResourcePath.Path {
			return
		}
	}
	cs.aliases[sourceResourcePath.Path] = append(cs.aliases[sourceResourcePath.Path], destResourcePath.Path)
	cs.recordResourceRelationship(destResourcePath)
}

func (cs *ChangeSet) recordResourceRelationship(resourcePath resourcepath.ResourcePath) error {
	if lastRelationship, found := cs.resourceRelationshipRewrites[resourcePath.Path]; found {
		if lastRelationship != resourcePath.ParentRelationship {
			return fmt.Errorf("failed to rewrite the parentRelationship of %s. It was already rewritten to %d", resourcePath.Path, lastRelationship)
		}
	} else {
		cs.resourceRelationshipRewrites[resourcePath.Path] = resourcePath.ParentRelationship
	}
	return nil
}

// FlushToHistory writes the recorded changeset to the history and returns resource paths where the resource modified.
func (cs *ChangeSet) FlushToHistory(builder *Builder) ([]string, error) {
	changedPaths := []string{}
	// Write revisions in this ChangeSet
	for resourcePath, revisions := range cs.revisions {
		tb := builder.GetTimelineBuilder(resourcePath)
		for _, stagingRevision := range revisions {
			revision, err := stagingRevision.commit(builder.binaryChunk, cs.associatedLog)
			if err != nil {
				return nil, err
			}
			tb.AddRevision(revision)
		}
		changedPaths = append(changedPaths, resourcePath)
	}
	// Write events in this ChangeSet
	for resourcePath, events := range cs.events {
		tb := builder.GetTimelineBuilder(resourcePath)
		for _, event := range events {
			tb.AddEvent(event)
		}
		changedPaths = append(changedPaths, resourcePath)
	}

	// Write log related properties
	if cs.logSummaryRewrite != "" {
		builder.setLogSummary(cs.associatedLog.ID, cs.logSummaryRewrite)
	}
	if cs.logSeverityRewrite != enum.SeverityUnknown {
		builder.setLogSeverity(cs.associatedLog.ID, cs.logSeverityRewrite)
	}
	builder.setLogAnnotations(cs.associatedLog.ID, cs.annotations)

	// Write the alias relationships
	for source, destinations := range cs.aliases {
		for _, dest := range destinations {
			builder.addTimelineAlias(source, dest)
		}
	}

	// Write resource related properties
	for resourcePath, relationship := range cs.resourceRelationshipRewrites {
		err := builder.rewriteRelationship(resourcePath, relationship)
		if err != nil {
			return nil, err
		}
	}
	return common.DedupStringArray(changedPaths), nil
}
