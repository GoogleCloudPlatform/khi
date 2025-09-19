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

package recorderutil_test

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/binarychunk"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/impl/recorder"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/impl/recorder/recorderutil"
)

func TestParentDeletionHandler_BeforeRecordingCurrent(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	historyBuilder := history.NewBuilder(t.TempDir())
	requestorBytes := []byte("test-requestor")
	requestorRef, err := historyBuilder.BinaryBuilder.Write(requestorBytes)
	if err != nil {
		t.Fatalf("failed to write requestor: %v", err)
	}

	childPath := resourcepath.ResourcePath{Path: "child/resource"}
	currentPaths := []resourcepath.ResourcePath{childPath}
	log := log.NewLogWithFieldSetsForTest(&log.CommonFieldSet{
		Timestamp: baseTime,
	})

	testCases := []struct {
		name                            string
		handler                         *recorderutil.ParentDeletionHandler
		req                             *recorder.RecorderRequest
		expectedRecordedCount           int
		expectedRequestor               string
		expectedProcessedParentRevIndex int
	}{
		{
			name:    "should not record deletion if no parent deletion revision exists",
			handler: &recorderutil.ParentDeletionHandler{},
			req: &recorder.RecorderRequest{
				LogParseResult: &commonlogk8saudit_contract.AuditLogParserInput{Log: log},
				ParentTimelineRevisions: []*history.ResourceRevision{
					{ChangeTime: baseTime.Add(-2 * time.Minute), State: enum.RevisionStateExisting},
				},
			},
			expectedRecordedCount:           0,
			expectedProcessedParentRevIndex: 1,
		},
		{
			name:    "should record deletion if parent was deleted before the current log",
			handler: &recorderutil.ParentDeletionHandler{},
			req: &recorder.RecorderRequest{
				LogParseResult: &commonlogk8saudit_contract.AuditLogParserInput{Log: log},
				ParentTimelineRevisions: []*history.ResourceRevision{
					{ChangeTime: baseTime.Add(-2 * time.Minute), State: enum.RevisionStateExisting},
					{ChangeTime: baseTime.Add(-1 * time.Minute), State: enum.RevisionStateDeleted, Requestor: requestorRef, Verb: enum.RevisionVerbDelete},
				},
			},
			expectedRecordedCount:           1,
			expectedRequestor:               string(requestorBytes),
			expectedProcessedParentRevIndex: 2,
		},
		{
			name:    "should record deletion only once even if multiple parent deletions exist",
			handler: &recorderutil.ParentDeletionHandler{},
			req: &recorder.RecorderRequest{
				LogParseResult: &commonlogk8saudit_contract.AuditLogParserInput{Log: log},
				ParentTimelineRevisions: []*history.ResourceRevision{
					{ChangeTime: baseTime.Add(-3 * time.Minute), State: enum.RevisionStateDeleted, Requestor: requestorRef, Verb: enum.RevisionVerbDelete},
					{ChangeTime: baseTime.Add(-2 * time.Minute), State: enum.RevisionStateDeleted, Requestor: requestorRef, Verb: enum.RevisionVerbDelete},
				},
			},
			expectedRecordedCount:           1,
			expectedRequestor:               string(requestorBytes),
			expectedProcessedParentRevIndex: 2,
		},
		{
			name:    "should not record deletion if parent deletion is after the current log",
			handler: &recorderutil.ParentDeletionHandler{},
			req: &recorder.RecorderRequest{
				LogParseResult: &commonlogk8saudit_contract.AuditLogParserInput{Log: log},
				ParentTimelineRevisions: []*history.ResourceRevision{
					{ChangeTime: baseTime.Add(1 * time.Minute), State: enum.RevisionStateDeleted, Requestor: requestorRef},
				},
			},
			expectedRecordedCount:           0,
			expectedProcessedParentRevIndex: 0,
		},
		{
			name:    "should update parentRevisionIndex correctly",
			handler: &recorderutil.ParentDeletionHandler{},
			req: &recorder.RecorderRequest{
				LogParseResult: &commonlogk8saudit_contract.AuditLogParserInput{Log: log},
				ParentTimelineRevisions: []*history.ResourceRevision{
					{ChangeTime: baseTime.Add(-1 * time.Minute), State: enum.RevisionStateExisting},
					{ChangeTime: baseTime.Add(1 * time.Minute), State: enum.RevisionStateExisting}, // Index should point here
					{ChangeTime: baseTime.Add(2 * time.Minute), State: enum.RevisionStateDeleted},
				},
			},
			expectedRecordedCount:           0,
			expectedProcessedParentRevIndex: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			changeSet := history.NewChangeSet(tc.req.LogParseResult.Log)
			tc.req.ChangeSet = changeSet
			tc.req.Builder = historyBuilder

			tc.handler.BeforeRecordingCurrent(tc.req, currentPaths)

			revisions := changeSet.GetRevisions(childPath)
			if len(revisions) != tc.expectedRecordedCount {
				t.Errorf("expected %d recorded revisions, but got %d", tc.expectedRecordedCount, len(revisions))
			}
			if tc.handler.ProcessedParentRevisionIndex != tc.expectedProcessedParentRevIndex {
				t.Errorf("expected ProcessedParentRevisionIndex to be %d, but got %d", tc.expectedProcessedParentRevIndex, tc.handler.ProcessedParentRevisionIndex)
			}

			if tc.expectedRecordedCount > 0 {
				rev := revisions[0]
				if rev.State != enum.RevisionStateDeleted {
					t.Errorf("expected revision state to be Deleted, but got %v", rev.State)
				}
				if rev.Requestor != tc.expectedRequestor {
					t.Errorf("expected requestor to be '%s', but got '%s'", tc.expectedRequestor, rev.Requestor)
				}
			}
		})
	}
}

func TestParentDeletionHandler_AfterRecordingCurrent(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	historyBuilder := history.NewBuilder(t.TempDir())
	requestorBytes := []byte("test-requestor")
	requestorRef, err := historyBuilder.BinaryBuilder.Write(requestorBytes)
	if err != nil {
		t.Fatalf("failed to write requestor: %v", err)
	}

	childPath := resourcepath.ResourcePath{Path: "child/resource"}
	currentPaths := []resourcepath.ResourcePath{childPath}

	testCases := []struct {
		name                  string
		handler               *recorderutil.ParentDeletionHandler
		req                   *recorder.RecorderRequest
		expectedRecordedCount int
		expectedRequestor     string
	}{
		{
			name:    "should not record if IsLastLog is false",
			handler: &recorderutil.ParentDeletionHandler{},
			req: &recorder.RecorderRequest{
				IsLastLog: false,
				ParentTimelineRevisions: []*history.ResourceRevision{
					{ChangeTime: baseTime.Add(1 * time.Minute), State: enum.RevisionStateDeleted, Requestor: requestorRef},
				},
			},
			expectedRecordedCount: 0,
		},
		{
			name:    "should record deletion if IsLastLog is true and parent is deleted",
			handler: &recorderutil.ParentDeletionHandler{},
			req: &recorder.RecorderRequest{
				IsLastLog: true,
				ParentTimelineRevisions: []*history.ResourceRevision{
					{ChangeTime: baseTime.Add(-1 * time.Minute), State: enum.RevisionStateExisting, Requestor: requestorRef},
					{ChangeTime: baseTime.Add(1 * time.Minute), State: enum.RevisionStateDeleted, Requestor: requestorRef, Verb: enum.RevisionVerbDelete},
				},
			},
			expectedRecordedCount: 1,
			expectedRequestor:     string(requestorBytes),
		},
		{
			name:    "should record deletion only once if multiple parent deletions exist",
			handler: &recorderutil.ParentDeletionHandler{},
			req: &recorder.RecorderRequest{
				IsLastLog: true,
				ParentTimelineRevisions: []*history.ResourceRevision{
					{ChangeTime: baseTime.Add(1 * time.Minute), State: enum.RevisionStateDeleted, Requestor: requestorRef, Verb: enum.RevisionVerbDelete},
					{ChangeTime: baseTime.Add(2 * time.Minute), State: enum.RevisionStateDeleted, Requestor: requestorRef, Verb: enum.RevisionVerbDelete},
				},
			},
			expectedRecordedCount: 1,
			expectedRequestor:     string(requestorBytes),
		},
		{
			name:    "should not record if no parent deletion revision exists",
			handler: &recorderutil.ParentDeletionHandler{},
			req: &recorder.RecorderRequest{
				IsLastLog: true,
				ParentTimelineRevisions: []*history.ResourceRevision{
					{ChangeTime: baseTime.Add(1 * time.Minute), State: enum.RevisionStateExisting, Requestor: requestorRef},
				},
			},
			expectedRecordedCount: 0,
		},
		{
			name:    "should use error string for requestor if binary read fails",
			handler: &recorderutil.ParentDeletionHandler{},
			req: &recorder.RecorderRequest{
				IsLastLog: true,
				ParentTimelineRevisions: []*history.ResourceRevision{
					{ChangeTime: baseTime.Add(1 * time.Minute), State: enum.RevisionStateDeleted, Requestor: &binarychunk.BinaryReference{Buffer: 999}, Verb: enum.RevisionVerbDelete}, // Invalid ref
				},
			},
			expectedRecordedCount: 1,
			expectedRequestor:     "error: failed to read requestor",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// For AfterRecordingCurrent, LogParseResult can be minimal as it's not directly used.
			logEntry := &log.Log{}
			changeSet := history.NewChangeSet(logEntry)
			tc.req.ChangeSet = changeSet
			tc.req.Builder = historyBuilder

			tc.handler.AfterRecordingCurrent(tc.req, currentPaths)

			revisions := changeSet.GetRevisions(childPath)
			if len(revisions) != tc.expectedRecordedCount {
				t.Errorf("expected %d recorded revisions, but got %d", tc.expectedRecordedCount, len(revisions))
			}

			if tc.expectedRecordedCount > 0 {
				rev := revisions[0]
				if rev.State != enum.RevisionStateDeleted {
					t.Errorf("expected revision state to be Deleted, but got %v", rev.State)
				}
				if rev.Requestor != tc.expectedRequestor {
					t.Errorf("expected requestor to be '%s', but got '%s'", tc.expectedRequestor, rev.Requestor)
				}
			}
		})
	}
}
