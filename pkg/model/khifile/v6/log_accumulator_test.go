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

package khifilev6

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestLogAccumulator(t *testing.T) {
	node1 := structured.NewStandardMap(
		[]string{"message", "level"},
		[]structured.Node{
			structured.NewStandardScalarNode("test message 1"),
			structured.NewStandardScalarNode("info"),
		},
	)
	node2 := structured.NewStandardMap(
		[]string{"message", "level"},
		[]structured.Node{
			structured.NewStandardScalarNode("test message 1"), // Same structure and content
			structured.NewStandardScalarNode("info"),
		},
	)
	node3 := structured.NewStandardMap(
		[]string{"error_code"},
		[]structured.Node{
			structured.NewStandardScalarNode(500),
		},
	)

	testSeverityID := uint32(1)
	testLogTypeID := uint32(2)
	testSeverity := &pb.Severity{Id: &testSeverityID}
	testLogType := &pb.LogType{Id: &testLogTypeID}

	testCases := []struct {
		name         string
		logsToAdd    []*StagingLog
		wantErr      bool
		wantLogCount int
		validate     func(t *testing.T, acc *LogAccumulator)
	}{
		{
			name:         "empty accumulator",
			logsToAdd:    []*StagingLog{},
			wantErr:      false,
			wantLogCount: 0,
			validate: func(t *testing.T, acc *LogAccumulator) {
				if gotLog := acc.GetLog(1); gotLog != nil {
					t.Errorf("GetLog(1) returned a log on empty accumulator instead of nil")
				}
			},
		},
		{
			name: "single log",
			logsToAdd: []*StagingLog{
				{
					Log:       log.NewLog(structured.NewNodeReader(node1)),
					Summary:   "test summary",
					Timestamp: time.Date(2026, 4, 29, 8, 0, 0, 0, time.UTC),
					LogType:   testLogType,
					Severity:  testSeverity,
				},
			},
			wantErr:      false,
			wantLogCount: 1,
			validate: func(t *testing.T, acc *LogAccumulator) {
				logs := acc.Accumulate()
				if logs[0].GetId() != 1 {
					t.Errorf("expected ID 1, got %d", logs[0].GetId())
				}
				if logs[0].Body == nil {
					t.Errorf("expected non-nil Body")
				}
				if logs[0].GetSeverityTypeId() != testSeverityID {
					t.Errorf("expected severity %d, got %d", testSeverityID, logs[0].GetSeverityTypeId())
				}
				if logs[0].GetLogTypeId() != testLogTypeID {
					t.Errorf("expected log type %d, got %d", testLogTypeID, logs[0].GetLogTypeId())
				}

				// Verify GetLog
				if gotLog := acc.GetLog(1); gotLog == nil || gotLog.GetId() != 1 {
					t.Errorf("GetLog(1) failed to return the correct log")
				}
				if gotLog := acc.GetLog(999); gotLog != nil {
					t.Errorf("GetLog(999) returned a log instead of nil")
				}
			},
		},
		{
			name: "multiple logs with deduplication",
			logsToAdd: []*StagingLog{
				{
					Log:       log.NewLog(structured.NewNodeReader(node1)),
					Summary:   "test summary 1",
					Timestamp: time.Now(),
					LogType:   testLogType,
					Severity:  testSeverity,
				},
				{
					Log:       log.NewLog(structured.NewNodeReader(node2)), // Same structure
					Summary:   "test summary 2",
					Timestamp: time.Now(),
					LogType:   testLogType,
					Severity:  testSeverity,
				},
				{
					Log:       log.NewLog(structured.NewNodeReader(node3)), // Different structure
					Summary:   "test summary 3",
					Timestamp: time.Now(),
					LogType:   testLogType,
					Severity:  testSeverity,
				},
			},
			wantErr:      false,
			wantLogCount: 3,
			validate: func(t *testing.T, acc *LogAccumulator) {
				logs := acc.Accumulate()
				if logs[0].GetId() != 1 {
					t.Errorf("expected ID 1, got %d", logs[0].GetId())
				}
				if logs[1].GetId() != 2 {
					t.Errorf("expected ID 2, got %d", logs[1].GetId())
				}
				if logs[2].GetId() != 3 {
					t.Errorf("expected ID 3, got %d", logs[2].GetId())
				}

				// The first two logs should have exactly the same interned structure
				if diff := cmp.Diff(logs[0].Body, logs[1].Body, protocmp.Transform()); diff != "" {
					t.Errorf("log bodies should be identical due to interning (-want +got):\n%s", diff)
				}

				// Verify GetLog
				if gotLog := acc.GetLog(2); gotLog == nil || gotLog.GetId() != 2 {
					t.Errorf("GetLog(2) failed to return the correct log")
				}
			},
		},
		{
			name: "missing severity error",
			logsToAdd: []*StagingLog{
				{
					Log:       log.NewLog(structured.NewNodeReader(node1)),
					Summary:   "test summary",
					Timestamp: time.Now(),
					LogType:   testLogType,
					Severity:  nil,
				},
			},
			wantErr:      true,
			wantLogCount: 0,
		},
		{
			name: "missing log type error",
			logsToAdd: []*StagingLog{
				{
					Log:       log.NewLog(structured.NewNodeReader(node1)),
					Summary:   "test summary",
					Timestamp: time.Now(),
					LogType:   nil,
					Severity:  testSeverity,
				},
			},
			wantErr:      true,
			wantLogCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idGen := &IDGenerator{}
			pool := NewInternPool(idGen)
			acc := NewLogAccumulator(pool, idGen)

			for _, l := range tc.logsToAdd {
				err := acc.AddLog(l)
				if (err != nil) != tc.wantErr {
					t.Fatalf("AddLog() error = %v, wantErr %v", err, tc.wantErr)
				}
			}

			logs := acc.Accumulate()
			if len(logs) != tc.wantLogCount {
				t.Fatalf("expected %d logs, got %d", tc.wantLogCount, len(logs))
			}

			if tc.validate != nil {
				tc.validate(t, acc)
			}
		})
	}
}
