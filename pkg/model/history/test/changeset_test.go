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
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
)

// Testing assertion utils with failure pattern seems to be complex in Golang.
// These tests only verify the case the assertions won't fail.

func TestAssertChangeSetHasLogSummary(t *testing.T) {
	cs := &history.ChangeSet{LogSummary: "test summary"}
	AssertChangeSetHasLogSummary(t, cs, "test summary")
}

func TestAssertChangeSetHasLogSeverity(t *testing.T) {
	cs := &history.ChangeSet{LogSeverity: enum.SeverityInfo}
	AssertChangeSetHasLogSeverity(t, cs, enum.SeverityInfo)
}

func TestAssertChangeSetHasEventForResourcePath(t *testing.T) {
	rp := resourcepath.NameLayerGeneralItem("core/v1", "pods", "default", "my-pod")
	event := &history.ResourceEvent{}
	cs := &history.ChangeSet{
		EventsMap: map[string][]*history.ResourceEvent{
			rp.Path: {event},
		},
	}
	AssertChangeSetHasEventForResourcePath(t, cs, rp)
}

func TestAssertChangeSetHasRevisionForResourcePath(t *testing.T) {
	rp := resourcepath.NameLayerGeneralItem("core/v1", "pods", "default", "my-pod")
	rev1 := &history.StagingResourceRevision{Verb: enum.RevisionVerbCreate}

	cs := &history.ChangeSet{
		RevisionsMap: map[string][]*history.StagingResourceRevision{
			rp.Path: {rev1},
		},
	}
	AssertChangeSetHasRevisionForResourcePath(t, cs, rp, rev1)
}

func TestAssertChangeSetHasAliasForResourcePath(t *testing.T) {
	rp := resourcepath.NameLayerGeneralItem("core/v1", "pods", "default", "my-pod")
	rpAlias := resourcepath.NameLayerGeneralItem("app/v1", "replicasets", "default", "my-pod")
	cs := &history.ChangeSet{
		Aliases: map[string][]string{
			rp.Path: {rpAlias.Path},
		},
	}
	AssertChangeSetHasAliasForResourcePath(t, cs, rp, rpAlias)
}
