// Copyright 2025 Google LLC
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

package history_test

import (
	"fmt"
	"maps"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	"github.com/google/go-cmp/cmp"
)

// AssertChangeSetHasLogSummary asserts that the ChangeSet's LogSummary matches
// the expected value.
func AssertChangeSetHasLogSummary(t testing.TB, cs *history.ChangeSet, wantSummary string) {
	t.Helper()
	if diff := cmp.Diff(wantSummary, cs.LogSummary); diff != "" {
		t.Errorf("LogSummary mismatch (-want +got):\n%s", diff)
	}
}

// AssertChangeSetHasLogSeverity asserts that the ChangeSet's LogSeverity matches
// the expected value.
func AssertChangeSetHasLogSeverity(t testing.TB, cs *history.ChangeSet, wantSeverity enum.Severity) {
	t.Helper()
	if diff := cmp.Diff(wantSeverity, cs.LogSeverity); diff != "" {
		t.Errorf("LogSeverity mismatch (-want +got):\n%s", diff)
	}
}

// AssertChangeSetHasEventForResourcePath asserts that the ChangeSet contains an
// event for the given resource path.
func AssertChangeSetHasEventForResourcePath(t testing.TB, cs *history.ChangeSet, resourcePath resourcepath.ResourcePath) {
	t.Helper()
	events := cs.GetEvents(resourcePath)
	if len(events) > 1 {
		// Currently ChangeSet is a log and it can only put an event on the log timing.
		// There is no reason to put 2 or more events on the same resource path. This behavior would be improved later for parsers to add multiple events on a single resource path on different timings from a single log.
		t.Errorf("Found 2 or more events for resource path '%s' in a single ChangeSet", resourcePath.Path)
	}
	if len(events) == 0 {
		availableResourcePaths := maps.Keys(cs.EventsMap)
		availableResourcePathsStr := ""
		for path := range availableResourcePaths {
			availableResourcePathsStr += fmt.Sprintf("* %s\n", path)
		}
		t.Errorf("Found no events for resource path '%s'\navailable event paths:\n %s", resourcePath.Path, availableResourcePathsStr)
	}
}

// AssertChangeSetHasRevisionForResourcePath asserts that the ChangeSet contains
// the expected revision for the given resource path.
func AssertChangeSetHasRevisionForResourcePath(t testing.TB, cs *history.ChangeSet, resourcePath resourcepath.ResourcePath, revision *history.StagingResourceRevision, opts ...cmp.Option) {
	t.Helper()
	revisions := cs.GetRevisions(resourcePath)
	if len(revisions) == 0 {
		availableResourcePaths := maps.Keys(cs.RevisionsMap)
		availableResourcePathsStr := ""
		for path := range availableResourcePaths {
			availableResourcePathsStr += fmt.Sprintf("* %s\n", path)
		}
		t.Errorf("Found no revisions for resource path '%s'\navailable revision paths:\n %s", resourcePath.Path, availableResourcePathsStr)
	}

	found := false
	for _, r := range revisions {
		if cmp.Equal(r, revision, opts...) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Did not find the expected revision for resource path '%s'.\nExpected: %+v\nFound: %+v", resourcePath.Path, revision, revisions)
	}
}

// AssertChangeSetHasCountOfRevisionsForResourcePath asserts that the ChangeSet contains
// the expected number of revisions for the given resource path.
func AssertChangeSetHasCountOfRevisionsForResourcePath(t testing.TB, cs *history.ChangeSet, resourcePath resourcepath.ResourcePath, count int) {
	t.Helper()
	revisions := cs.GetRevisions(resourcePath)
	if len(revisions) != count {
		t.Errorf("got %d revisions, want %d", len(revisions), count)
	}
}

// AssertChangeSetHasRevisionMatchingBodyGoldensForResourcePath asserts that the ChangeSet contains
// revisions whose bodies match the golden file for the given resource path.
// The `validationTargetName` is used as the suffix of golden file name.
func AssertChangeSetHasRevisionMatchingBodyGoldensForResourcePath(t testing.TB, cs *history.ChangeSet, resourcePath resourcepath.ResourcePath, validationTargetName string) {
	t.Helper()
	revisions := cs.GetRevisions(resourcePath)
	wholeBody := ""
	for _, rev := range revisions {
		if wholeBody != "" {
			wholeBody += "==================================================\n"
		}
		wholeBody += rev.Body + "\n"
	}
	testutil.VerifyWithGolden(t, validationTargetName, wholeBody)
}

// AssertChangeSetHasAliasForResourcePath asserts that the ChangeSet contains the
// expected alias for the given resource path.
func AssertChangeSetHasAliasForResourcePath(t testing.TB, cs *history.ChangeSet, resourcePath resourcepath.ResourcePath, aliasTarget resourcepath.ResourcePath) {
	t.Helper()
	aliases, ok := cs.Aliases[resourcePath.Path]
	if !ok {
		availableResourcePaths := maps.Keys(cs.Aliases)
		availableResourcePathsStr := ""
		for path := range availableResourcePaths {
			availableResourcePathsStr += fmt.Sprintf("* %s\n", path)
		}
		t.Errorf("Did not find any aliases for resource path '%s'\navailable alias paths:\n %s", resourcePath.Path, availableResourcePathsStr)
	}

	found := false
	for _, a := range aliases {
		if a == aliasTarget.Path {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Did not find the expected alias for resource path '%s'.\nExpected: %s\nFound: %+v", resourcePath.Path, aliasTarget.Path, aliases)
	}
}
