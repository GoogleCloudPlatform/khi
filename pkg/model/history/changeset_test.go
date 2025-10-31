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
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
)

type testInsertIDTimeStampCommonFieldReader struct {
}

// FieldSetKind implements log.FieldSetReader.
func (t *testInsertIDTimeStampCommonFieldReader) FieldSetKind() string {
	return (&log.CommonFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (t *testInsertIDTimeStampCommonFieldReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	return &log.CommonFieldSet{
		Timestamp: reader.ReadTimestampOrDefault("timestamp", time.Now()),
		Severity:  enum.SeverityInfo,
		DisplayID: reader.ReadStringOrDefault("insertID", "unknown"),
	}, nil
}

var _ (log.FieldSetReader) = (*testInsertIDTimeStampCommonFieldReader)(nil)

func TestRecordLogSummary(t *testing.T) {
	log := testlog.NewEmptyLogWithID("foo")
	cs := NewChangeSet(log)
	cs.SetLogSummary("bar")
	if cs.LogSummary != "bar" {
		t.Errorf("SetLogSummary is not rewritten to the expected value")
	}
}

func TestRecordLogSeverity(t *testing.T) {
	log := testlog.NewEmptyLogWithID("foo")
	cs := NewChangeSet(log)
	cs.SetLogSeverity(enum.SeverityWarning)
	if cs.LogSeverity != enum.SeverityWarning {
		t.Errorf("SetLogSeverity is not rewritten to the expected value")
	}
}

func TestRecordEvents(t *testing.T) {
	log := testlog.NewEmptyLogWithID("foo")
	cs := NewChangeSet(log)
	cs.AddEvent(resourcepath.KindLayerGeneralItem("A", "B"))
	cs.AddEvent(resourcepath.KindLayerGeneralItem("A", "C"))
	if diff := cmp.Diff(cs.EventsMap, map[string][]*ResourceEvent{
		"A#B": {{Log: "foo"}},
		"A#C": {{Log: "foo"}},
	}); diff != "" {
		t.Errorf("AddEvent didn't modify ChangeSet as expected\n%s", diff)
	}
}

func TestGetEvents(t *testing.T) {
	log := testlog.NewEmptyLogWithID("foo")
	cs := NewChangeSet(log)
	cs.AddEvent(resourcepath.KindLayerGeneralItem("A", "B"))
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
	log := testlog.NewEmptyLogWithID("foo")
	cs := NewChangeSet(log)
	cs.AddRevision(resourcepath.KindLayerGeneralItem("A", "B"), &StagingResourceRevision{
		Inferred: true,
	})
	cs.AddRevision(resourcepath.KindLayerGeneralItem("A", "B"), &StagingResourceRevision{})
	cs.AddRevision(resourcepath.KindLayerGeneralItem("A", "C"), &StagingResourceRevision{})
	if diff := cmp.Diff(cs.RevisionsMap, map[string][]*StagingResourceRevision{
		"A#B": {{Inferred: true}, {}},
		"A#C": {{}},
	}); diff != "" {
		t.Errorf("AddRevision didn't modify ChangeSet as expected\n%s", diff)
	}

	if diff := cmp.Diff(cs.Annotations, []LogAnnotation{
		&ResourceReferenceAnnotation{Path: "A#B"},
		&ResourceReferenceAnnotation{Path: "A#C"},
	}); diff != "" {
		t.Errorf("AddRevision didn't modify log annotations in ChangeSet as expected\n%s", diff)
	}
}

func TestGetRevisions(t *testing.T) {
	log := testlog.NewEmptyLogWithID("foo")
	cs := NewChangeSet(log)
	cs.AddRevision(resourcepath.KindLayerGeneralItem("A", "B"), &StagingResourceRevision{
		Body: "AB1",
	})
	cs.AddRevision(resourcepath.KindLayerGeneralItem("A", "B"), &StagingResourceRevision{
		Body: "AB2",
	})
	cs.AddRevision(resourcepath.KindLayerGeneralItem("A", "C"), &StagingResourceRevision{
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
	builder := NewBuilder("/tmp")
	lt := testlog.New(testlog.YAML(""))
	l := [][]*log.Log{}
	allLogs := []*log.Log{}
	for i := 0; i < groupCount; i++ {
		l = append(l, make([]*log.Log, 0))
	}
	for li := 0; li < logCountPerGroup; li++ {
		for i := 0; i < groupCount; i++ {
			hour := i / 3600
			minute := (i - hour*3600) / 60
			seconds := (i - hour*3600 - minute*60) % 60
			l[i] = append(l[i], lt.With(
				testlog.StringField("insertId", fmt.Sprintf("id-group%d-%d", i, li)),
				testlog.StringField("timestamp", fmt.Sprintf("2024-01-01T%02d:%02d:%02dZ", hour, minute, seconds)),
			).MustBuildLogEntity(&testCommonFieldSetReader{}))
		}
	}
	for _, group := range l {
		allLogs = append(allLogs, group...)
	}
	err := builder.SerializeLogs(context.Background(), allLogs, func() {})
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
				cs.AddRevision(groupPath, &StagingResourceRevision{})
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
