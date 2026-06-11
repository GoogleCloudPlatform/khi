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

package testchangeset

import (
	"testing"
	"time"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

// LogChangeSetAsserter provides fluent, chainable assertions for LogChangeSet.
type LogChangeSetAsserter struct {
	t  *testing.T
	cs *khifilev6.LogChangeSet
}

// AssertLog starts a fluent assertion chain for the given LogChangeSet.
func AssertLog(t *testing.T, cs *khifilev6.LogChangeSet) *LogChangeSetAsserter {
	t.Helper()
	return &LogChangeSetAsserter{t: t, cs: cs}
}

// HasSummary asserts that the LogChangeSet has the expected summary.
func (a *LogChangeSetAsserter) HasSummary(wantSummary string) *LogChangeSetAsserter {
	a.t.Helper()
	if a.cs.Summary != wantSummary {
		a.t.Errorf("LogChangeSet summary mismatch: want %q, got %q", wantSummary, a.cs.Summary)
	}
	return a
}

// HasSeverity asserts that the LogChangeSet has the expected severity type.
func (a *LogChangeSetAsserter) HasSeverity(want *pb.Severity) *LogChangeSetAsserter {
	a.t.Helper()
	if diff := cmp.Diff(want, a.cs.Severity, protocmp.Transform()); diff != "" {
		a.t.Errorf("LogChangeSet severity mismatch (-want +got):\n%s", diff)
	}
	return a
}

// HasLogType asserts that the LogChangeSet has the expected log type.
func (a *LogChangeSetAsserter) HasLogType(want *pb.LogType) *LogChangeSetAsserter {
	a.t.Helper()
	if diff := cmp.Diff(want, a.cs.LogType, protocmp.Transform()); diff != "" {
		a.t.Errorf("LogChangeSet log type mismatch (-want +got):\n%s", diff)
	}
	return a
}

// HasTimestamp asserts that the LogChangeSet has the expected timestamp.
func (a *LogChangeSetAsserter) HasTimestamp(want time.Time) *LogChangeSetAsserter {
	a.t.Helper()
	if !a.cs.Timestamp.Equal(want) {
		a.t.Errorf("LogChangeSet timestamp mismatch: want %v, got %v", want, a.cs.Timestamp)
	}
	return a
}

// TimelineChangeSetAsserter provides fluent, chainable assertions for TimelineChangeSet.
type TimelineChangeSetAsserter struct {
	t  *testing.T
	cs *khifilev6.TimelineChangeSet
}

// AssertTimeline starts a fluent assertion chain for the given TimelineChangeSet.
func AssertTimeline(t *testing.T, cs *khifilev6.TimelineChangeSet) *TimelineChangeSetAsserter {
	t.Helper()
	return &TimelineChangeSetAsserter{t: t, cs: cs}
}

// HasEvent asserts that an event was staged on the expected path.
func (a *TimelineChangeSetAsserter) HasEvent(wantPath *khifilev6.TimelinePath) *TimelineChangeSetAsserter {
	a.t.Helper()
	if !a.cs.Events[wantPath] {
		a.t.Errorf("TimelineChangeSet: expected event staged on path %v, but not found", wantPath)
	}
	return a
}

// HasNoEvent asserts that no event was staged on the path.
func (a *TimelineChangeSetAsserter) HasNoEvent(path *khifilev6.TimelinePath) *TimelineChangeSetAsserter {
	a.t.Helper()
	if a.cs.Events[path] {
		a.t.Errorf("TimelineChangeSet: expected no event staged on path %v, but found one", path)
	}
	return a
}

// HasRevision asserts that a matching StagingRevision was staged on the expected path.
func (a *TimelineChangeSetAsserter) HasRevision(wantPath *khifilev6.TimelinePath, wantRevision *khifilev6.StagingRevision, cmpOpts ...cmp.Option) *TimelineChangeSetAsserter {
	a.t.Helper()
	revisions, exist := a.cs.Revisions[wantPath]
	if !exist || len(revisions) == 0 {
		a.t.Errorf("TimelineChangeSet: no revisions found for path %v", wantPath)
		return a
	}

	for _, rev := range revisions {
		if rev.ChangedTime.Equal(wantRevision.ChangedTime) &&
			rev.Principal == wantRevision.Principal &&
			rev.VerbType == wantRevision.VerbType &&
			rev.StateType == wantRevision.StateType {

			if diff := cmp.Diff(wantRevision.ResourceBody, rev.ResourceBody, cmpOpts...); diff == "" {
				return a
			}
		}
	}

	a.t.Errorf("TimelineChangeSet: revision matching %+v not found on path %v. Staged revisions count: %d", wantRevision, wantPath, len(revisions))
	return a
}

// HasNoRevision asserts that no revision was staged on the path.
func (a *TimelineChangeSetAsserter) HasNoRevision(path *khifilev6.TimelinePath) *TimelineChangeSetAsserter {
	a.t.Helper()
	revisions, exist := a.cs.Revisions[path]
	if exist && len(revisions) > 0 {
		a.t.Errorf("TimelineChangeSet: expected no revisions staged on path %v, but found %d", path, len(revisions))
	}
	return a
}

// HasAlias asserts that an alias mapping from the alias path to the target path was staged.
func (a *TimelineChangeSetAsserter) HasAlias(wantAliasPath, wantTargetPath *khifilev6.TimelinePath) *TimelineChangeSetAsserter {
	a.t.Helper()
	targetPath, exist := a.cs.Aliases[wantAliasPath]
	if !exist {
		a.t.Errorf("TimelineChangeSet: alias from path %v not found", wantAliasPath)
		return a
	}
	if targetPath != wantTargetPath {
		a.t.Errorf("TimelineChangeSet: alias target mismatch for %v: want %v, got %v", wantAliasPath, wantTargetPath, targetPath)
	}
	return a
}

// HasNoAlias asserts that no alias was staged for the given alias path.
func (a *TimelineChangeSetAsserter) HasNoAlias(aliasPath *khifilev6.TimelinePath) *TimelineChangeSetAsserter {
	a.t.Helper()
	targetPath, exist := a.cs.Aliases[aliasPath]
	if exist {
		a.t.Errorf("TimelineChangeSet: expected no alias staged for path %v, but found one pointing to %v", aliasPath, targetPath)
	}
	return a
}
