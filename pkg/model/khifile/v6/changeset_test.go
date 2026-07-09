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

package khifilev6_test

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestLogChangeSet_Flush(t *testing.T) {
	idGen := &khifilev6.IDGenerator{}
	pool := khifilev6.NewInternPool(idGen)
	logAcc := khifilev6.NewLogAccumulator(pool, idGen)

	node := structured.NewStandardMap(nil, nil)
	l := log.NewLog(structured.NewNodeReader(node))
	l.ID = "test-log-id"

	severityID := uint32(1)
	logTypeID := uint32(2)
	severity := &pb.Severity{Id: &severityID}
	logType := &pb.LogType{Id: &logTypeID}

	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		t.Fatalf("NewLogChangeSet() returned unexpected error: %v", err)
	}

	cs.SetSummary("test summary")
	cs.SetTimestamp(time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC))
	cs.SetSeverity(severity)
	cs.SetLogType(logType)

	// Verify fluent assertions work as expected.
	testchangeset.AssertLog(t, cs).
		HasSummary("test summary").
		HasSeverity(severity).
		HasLogType(logType)

	// Flush to accumulator.
	if err := cs.Flush(logAcc); err != nil {
		t.Fatalf("Flush() returned unexpected error: %v", err)
	}

	// Verify that the log is resolved to its serialized ID.
	resolvedID, ok := logAcc.ResolveLogID("test-log-id")
	if !ok {
		t.Fatal("expected log to be resolved in LogAccumulator")
	}
	if resolvedID != 1 {
		t.Errorf("expected resolved log ID to be 1, got %d", resolvedID)
	}
}

func TestTimelineChangeSet_Flush(t *testing.T) {
	idGen := &khifilev6.IDGenerator{}
	pool := khifilev6.NewInternPool(idGen)
	logAcc := khifilev6.NewLogAccumulator(pool, idGen)
	accumulator := khifilev6.NewTimelineAccumulator(idGen, pool, logAcc)

	node := structured.NewStandardMap(nil, nil)
	l := log.NewLog(structured.NewNodeReader(node))
	l.ID = "test-log-id"

	severityID := uint32(1)
	logTypeID := uint32(2)
	_ = logAcc.AddLog(&khifilev6.StagingLog{
		Log:       l,
		Summary:   "test summary",
		Timestamp: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
		Severity:  &pb.Severity{Id: &severityID},
		LogType:   &pb.LogType{Id: &logTypeID},
	})

	pathPool := khifilev6.NewTimelinePathPool(idGen, pool)
	timelineTypeID := uint32(3)
	timelineType := &pb.TimelineType{Id: &timelineTypeID}
	path := pathPool.Get(nil, khifilev6.PathSegment{Name: "test-path", Type: timelineType})

	cs := khifilev6.NewTimelineChangeSet(l)
	cs.AddEvent(path)

	stagingRev := &khifilev6.StagingRevision{
		ChangedTime: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
		FieldAnnotations: []*khifilev6.StagingFieldAnnotation{
			{
				FieldPath: "/spec/containers",
			},
			{
				FieldPath: "/metadata/annotations",
				MutatingWebhook: &khifilev6.StagingMutatingWebhook{
					Configuration: "test-config",
					Webhook:       "test-webhook",
				},
			},
		},
	}
	cs.AddRevision(path, stagingRev)

	// Verify fluent assertions work.
	testchangeset.AssertTimeline(t, cs).
		HasEvent(path).
		HasRevision(path, stagingRev)

	// Flush to accumulator.
	if err := cs.Flush(accumulator); err != nil {
		t.Fatalf("Flush() returned unexpected error: %v", err)
	}

	builder := accumulator.GetBuilder(path)
	if !builder.HasItems() {
		t.Error("expected TimelineBuilder to have items")
	}

	protoItems := builder.ToProto()
	revisions := protoItems.GetRevisions()
	if len(revisions) != 1 {
		t.Fatalf("expected 1 revision, got %d", len(revisions))
	}

	annotations := revisions[0].GetFieldAnnotations()
	if len(annotations) != 2 {
		t.Fatalf("expected 2 field annotations, got %d", len(annotations))
	}

	ann1 := annotations[0]
	if ann1.GetFieldPathStringId() != pool.InternString("/spec/containers").ToProto().GetId() {
		t.Errorf("expected /spec/containers ID")
	}
	if ann1.GetMutatingWebhook() != nil {
		t.Errorf("expected no mutating webhook, got %v", ann1.GetMutatingWebhook())
	}

	ann2 := annotations[1]
	if ann2.GetFieldPathStringId() != pool.InternString("/metadata/annotations").ToProto().GetId() {
		t.Errorf("expected /metadata/annotations ID")
	}
	if ann2.GetMutatingWebhook() == nil {
		t.Fatalf("expected mutating webhook, got nil")
	}
	mw := ann2.GetMutatingWebhook()
	if mw.GetConfigurationStringId() != pool.InternString("test-config").ToProto().GetId() {
		t.Errorf("expected test-config ID")
	}
	if mw.GetWebhookStringId() != pool.InternString("test-webhook").ToProto().GetId() {
		t.Errorf("expected test-webhook ID")
	}
}
