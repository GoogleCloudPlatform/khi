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

package testchangeset

import (
	"slices"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/google/go-cmp/cmp"
)

// ChangeSetAsserter is an interface to assert if the given changeset with rules of implementations.
type ChangeSetAsserter interface {
	Assert(t *testing.T, cs *history.ChangeSet)
}

type HasRevision struct {
	ResourcePath string
	WantRevision history.StagingResourceRevision
	CmpOpts      []cmp.Option
}

// Assert implements ChangeSetAsserter.
func (r *HasRevision) Assert(t *testing.T, cs *history.ChangeSet) {
	t.Helper()
	revisions := cs.GetRevisions(resourcepath.ResourcePath{
		Path: r.ResourcePath,
	})
	if len(revisions) == 0 {
		t.Errorf("no revisions found for %s. available revisions are %v", r.ResourcePath, cs.GetAllResourcePaths())
		return
	}
	for _, rev := range revisions {
		if rev.ChangeTime == r.WantRevision.ChangeTime {
			if diff := cmp.Diff(r.WantRevision, *rev, r.CmpOpts...); diff != "" {
				t.Errorf("stored revision is not matching with the expected:(-want,+got)\n%s", diff)
			}
			return
		}
	}

	times := []time.Time{}
	for _, rev := range revisions {
		times = append(times, rev.ChangeTime)
	}
	t.Errorf("no revision found for %s at %s. available times are %v", r.ResourcePath, r.WantRevision.ChangeTime, times)
}

var _ ChangeSetAsserter = (*HasRevision)(nil)

type MatchResourcePathSet struct {
	WantResourcePaths []string
}

func (r *MatchResourcePathSet) Assert(t *testing.T, cs *history.ChangeSet) {
	t.Helper()
	gotResourcePaths := cs.GetAllResourcePaths()
	slices.Sort(r.WantResourcePaths)
	slices.Sort(gotResourcePaths)
	if diff := cmp.Diff(r.WantResourcePaths, gotResourcePaths); diff != "" {
		t.Errorf("resource paths are different: (-want, +got) = %s", diff)
	}
}

var _ ChangeSetAsserter = (*MatchResourcePathSet)(nil)

type HasEvent struct {
	ResourcePath string
}

// Assert implements ChangeSetAsserter.
func (h *HasEvent) Assert(t *testing.T, cs *history.ChangeSet) {
	t.Helper()
	events := cs.GetEvents(resourcepath.ResourcePath{
		Path: h.ResourcePath,
	})
	if len(events) == 0 {
		t.Errorf("no events found for %s. available paths are: %v", h.ResourcePath, cs.GetAllResourcePaths())
	}
}

var _ ChangeSetAsserter = (*HasEvent)(nil)

type HasLogSummary struct {
	WantLogSummary string
}

// Assert implements ChangeSetAsserter.
func (h *HasLogSummary) Assert(t *testing.T, cs *history.ChangeSet) {
	t.Helper()
	if cs.LogSummary != h.WantLogSummary {
		t.Errorf("log summary is not matching with the expected: want=%s, got=%s", h.WantLogSummary, cs.LogSummary)

	}
}

var _ ChangeSetAsserter = (*HasLogSummary)(nil)
