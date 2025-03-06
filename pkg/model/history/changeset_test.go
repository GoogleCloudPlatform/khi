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
	"context"
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/ioconfig"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	gcp_log "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/log"
	log_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/log"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestRecordLogSummary(t *testing.T) {
	log := log_test.MockLogWithId("foo")
	cs := NewChangeSet(log)
	cs.RecordLogSummary("bar")
	if cs.logSummaryRewrite != "bar" {
		t.Errorf("logSummaryRewrite is not rewritten to the expected value")
	}
}

func TestRecordLogSeverity(t *testing.T) {
	log := log_test.MockLogWithId("foo")
	cs := NewChangeSet(log)
	cs.RecordLogSeverity(enum.SeverityWarning)
	if cs.logSeverityRewrite != enum.SeverityWarning {
		t.Errorf("logSeverityRewrite is not rewritten to the expected value")
	}
}

func TestRecordEvents(t *testing.T) {
	log := log_test.MockLogWithId("foo")
	cs := NewChangeSet(log)
	cs.RecordEvent(resourcepath.KindLayerGeneralItem("A", "B"))
	cs.RecordEvent(resourcepath.KindLayerGeneralItem("A", "C"))
	if diff := cmp.Diff(cs.events, map[string][]*ResourceEvent{
		"A#B": {{Log: "foo"}},
		"A#C": {{Log: "foo"}},
	}); diff != "" {
		t.Errorf("RecordEvent didn't modify ChangeSet as expected\n%s", diff)
	}
}

func TestGetEvents(t *testing.T) {
	log := log_test.MockLogWithId("foo")
	cs := NewChangeSet(log)
	cs.RecordEvent(resourcepath.KindLayerGeneralItem("A", "B"))
	testCases := []struct {
		name           string
		resourcePath   resourcepath.ResourcePath
		expectedBodies []string
	}{
		{
			name:           "return empty array when specified resource path is not contained in the change set",
			resourcePath:   resourcepath.KindLayerGeneralItem("A", "D"),
			expectedBodies: nil,
		},
		{
			name:           "return all events when specified resource path is contained in the change set",
			resourcePath:   resourcepath.KindLayerGeneralItem("A", "B"),
			expectedBodies: []string{"foo"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			events := cs.GetEvents(tc.resourcePath)
			var eventBodies []string
			for _, event := range events {
				eventBodies = append(eventBodies, event.Log)
			}

			if diff := cmp.Diff(tc.expectedBodies, eventBodies); diff != "" {
				t.Errorf("different ResourceEvents returned:(-want,+got): %v", diff)
			}
		})
	}
}

func TestRecordRevisions(t *testing.T) {
	log := log_test.MockLogWithId("foo")
	cs := NewChangeSet(log)
	cs.RecordRevision(resourcepath.KindLayerGeneralItem("A", "B"), &StagingResourceRevision{
		Inferred: true,
	})
	cs.RecordRevision(resourcepath.KindLayerGeneralItem("A", "B"), &StagingResourceRevision{})
	cs.RecordRevision(resourcepath.KindLayerGeneralItem("A", "C"), &StagingResourceRevision{})
	if diff := cmp.Diff(cs.revisions, map[string][]*StagingResourceRevision{
		"A#B": {{Inferred: true}, {}},
		"A#C": {{}},
	}); diff != "" {
		t.Errorf("RecordRevision didn't modify ChangeSet as expected\n%s", diff)
	}

	if diff := cmp.Diff(cs.annotations, []LogAnnotation{
		&ResourceReferenceAnnotation{Path: "A#B"},
		&ResourceReferenceAnnotation{Path: "A#C"},
	}); diff != "" {
		t.Errorf("RecordRevision didn't modify log annotations in ChangeSet as expected\n%s", diff)
	}
}

func TestGetRevisions(t *testing.T) {
	log := log_test.MockLogWithId("foo")
	cs := NewChangeSet(log)
	cs.RecordRevision(resourcepath.KindLayerGeneralItem("A", "B"), &StagingResourceRevision{
		Body: "AB1",
	})
	cs.RecordRevision(resourcepath.KindLayerGeneralItem("A", "B"), &StagingResourceRevision{
		Body: "AB2",
	})
	cs.RecordRevision(resourcepath.KindLayerGeneralItem("A", "C"), &StagingResourceRevision{
		Body: "AC1",
	})
	testCases := []struct {
		name           string
		resourcePath   resourcepath.ResourcePath
		expectedBodies []string
	}{
		{
			name:           "return empty array when specified resource path is not contained in the change set",
			resourcePath:   resourcepath.KindLayerGeneralItem("A", "D"),
			expectedBodies: nil,
		},
		{
			name:           "return all revisions when specified resource path is contained in the change set(multiple)",
			resourcePath:   resourcepath.KindLayerGeneralItem("A", "B"),
			expectedBodies: []string{"AB1", "AB2"},
		},
		{
			name:           "return all revisions when specified resource path is contained in the change set(single)",
			resourcePath:   resourcepath.KindLayerGeneralItem("A", "C"),
			expectedBodies: []string{"AC1"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			revisions := cs.GetRevisions(tc.resourcePath)
			var revisionBodies []string
			for _, revision := range revisions {
				revisionBodies = append(revisionBodies, revision.Body)
			}

			if diff := cmp.Diff(tc.expectedBodies, revisionBodies); diff != "" {
				t.Errorf("different StagingResourceRevisions returned:(-want,+got): %v", diff)
			}
		})
	}
}

func TestChangesetFlushIsThreadSafe(t *testing.T) {
	groupCount := 100
	logCountPerGroup := 100
	builder := NewBuilder(&ioconfig.IOConfig{})
	lt := testlog.New(testlog.BaseYaml(""))
	l := [][]*log.LogEntity{}
	allLogs := []*log.LogEntity{}
	for i := 0; i < groupCount; i++ {
		l = append(l, make([]*log.LogEntity, 0))
	}
	for li := 0; li < logCountPerGroup; li++ {
		for i := 0; i < groupCount; i++ {
			hour := i / 3600
			minute := (i - hour*3600) / 60
			seconds := (i - hour*3600 - minute*60) % 60
			l[i] = append(l[i], lt.With(
				testlog.StringField("insertId", fmt.Sprintf("id-group%d-%d", i, li)),
				testlog.StringField("timestamp", fmt.Sprintf("2024-01-01T%02d:%02d:%02dZ", hour, minute, seconds)),
			).MustBuildLogEntity(gcp_log.GCPCommonFieldExtractor{}))
		}
	}
	for _, group := range l {
		allLogs = append(allLogs, group...)
	}
	err := builder.PrepareParseLogs(context.Background(), allLogs, func() {})
	if err != nil {
		t.Fatal(err.Error())
	}
	pool := worker.NewPool(groupCount)
	for i := 0; i < groupCount; i++ {
		currentGroup := l[i]
		groupPath := resourcepath.KindLayerGeneralItem("grp", fmt.Sprintf("%d", i))
		pool.Run(func() {
			for _, l := range currentGroup {
				cs := NewChangeSet(l)
				cs.RecordRevision(groupPath, &StagingResourceRevision{})
				paths, err := cs.FlushToHistory(builder)

				for _, path := range paths {
					tb := builder.GetTimelineBuilder(path)
					tb.Sort()
				}
				if err != nil {
					t.Fatal(err.Error())
				}
			}
		})
	}

	pool.Wait()
	for i := 0; i < groupCount; i++ {
		grpPath := fmt.Sprintf("grp#%d", i)
		tb := builder.GetTimelineBuilder(grpPath)
		if len(tb.timeline.Revisions) != logCountPerGroup {
			t.Errorf("revision count mismatch: expected %d, actual %d", logCountPerGroup, len(tb.timeline.Revisions))
		}
		for li := 0; li < logCountPerGroup; li++ {
			rev := tb.timeline.Revisions[li]
			sl, err := tb.builder.GetLog(rev.Log)
			expectedId := fmt.Sprintf("id-group%d-%d", i, li)
			if err != nil {
				t.Errorf("log %s not found!", rev.Log)
				continue
			}
			if sl.DisplayId != expectedId {
				t.Errorf("log id mismatch: expected %s, actual %s", expectedId, sl.DisplayId)
			}
		}
	}
}
