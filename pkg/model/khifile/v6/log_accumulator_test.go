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
	"bytes"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"google.golang.org/protobuf/proto"
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
		logsToAdd    func(gen *id.Generator) []*StagingLog
		wantErr      bool
		wantLogCount int
		validate     func(t *testing.T, logs []*pb.Log, acc *LogAccumulator)
	}{
		{
			name: "empty accumulator",
			logsToAdd: func(gen *id.Generator) []*StagingLog {
				return nil
			},
			wantErr:      false,
			wantLogCount: 0,
			validate: func(t *testing.T, logs []*pb.Log, acc *LogAccumulator) {
				if len(logs) != 0 {
					t.Errorf("expected 0 logs, got %d", len(logs))
				}
			},
		},
		{
			name: "single log",
			logsToAdd: func(gen *id.Generator) []*StagingLog {
				return []*StagingLog{
					{
						Log:       log.NewLog(gen, structured.NewNodeReader(node1)),
						Summary:   "test summary",
						Timestamp: time.Date(2026, 4, 29, 8, 0, 0, 0, time.UTC),
						LogType:   testLogType,
						Severity:  testSeverity,
					},
				}
			},
			wantErr:      false,
			wantLogCount: 1,
			validate: func(t *testing.T, logs []*pb.Log, acc *LogAccumulator) {
				if logs[0].GetId() != 1 {
					t.Errorf("expected ID 1, got %d", logs[0].GetId())
				}
				if logs[0].GetBodyStructId() == 0 {
					t.Errorf("expected non-zero BodyStructId")
				}
				if logs[0].GetSeverityTypeId() != testSeverityID {
					t.Errorf("expected severity %d, got %d", testSeverityID, logs[0].GetSeverityTypeId())
				}
				if logs[0].GetLogTypeId() != testLogTypeID {
					t.Errorf("expected log type %d, got %d", testLogTypeID, logs[0].GetLogTypeId())
				}
			},
		},
		{
			name: "multiple logs with deduplication",
			logsToAdd: func(gen *id.Generator) []*StagingLog {
				return []*StagingLog{
					{
						Log:       log.NewLog(gen, structured.NewNodeReader(node1)),
						Summary:   "test summary 1",
						Timestamp: time.Now(),
						LogType:   testLogType,
						Severity:  testSeverity,
					},
					{
						Log:       log.NewLog(gen, structured.NewNodeReader(node2)), // Same structure
						Summary:   "test summary 2",
						Timestamp: time.Now(),
						LogType:   testLogType,
						Severity:  testSeverity,
					},
					{
						Log:       log.NewLog(gen, structured.NewNodeReader(node3)), // Different structure
						Summary:   "test summary 3",
						Timestamp: time.Now(),
						LogType:   testLogType,
						Severity:  testSeverity,
					},
				}
			},
			wantErr:      false,
			wantLogCount: 3,
			validate: func(t *testing.T, logs []*pb.Log, acc *LogAccumulator) {
				if logs[0].GetId() != 1 {
					t.Errorf("expected ID 1, got %d", logs[0].GetId())
				}
				if logs[1].GetId() != 2 {
					t.Errorf("expected ID 2, got %d", logs[1].GetId())
				}
				if logs[2].GetId() != 3 {
					t.Errorf("expected ID 3, got %d", logs[2].GetId())
				}

				// The first two logs should have exactly the same interned struct ID
				if logs[0].GetBodyStructId() != logs[1].GetBodyStructId() {
					t.Errorf("log body struct IDs should be identical due to interning (got %d and %d)", logs[0].GetBodyStructId(), logs[1].GetBodyStructId())
				}
				if logs[0].GetBodyStructId() == logs[2].GetBodyStructId() {
					t.Errorf("log 3 should have different body struct ID from log 0")
				}
			},
		},
		{
			name: "multiple logs sorted chronologically",
			logsToAdd: func(gen *id.Generator) []*StagingLog {
				return []*StagingLog{
					{
						Log:       log.NewLog(gen, structured.NewNodeReader(node1)),
						Summary:   "test summary 1",
						Timestamp: time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC),
						LogType:   testLogType,
						Severity:  testSeverity,
					},
					{
						Log:       log.NewLog(gen, structured.NewNodeReader(node2)),
						Summary:   "test summary 2",
						Timestamp: time.Date(2026, 4, 29, 8, 0, 0, 0, time.UTC),
						LogType:   testLogType,
						Severity:  testSeverity,
					},
					{
						Log:       log.NewLog(gen, structured.NewNodeReader(node3)),
						Summary:   "test summary 3",
						Timestamp: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC),
						LogType:   testLogType,
						Severity:  testSeverity,
					},
				}
			},
			wantErr:      false,
			wantLogCount: 3,
			validate: func(t *testing.T, logs []*pb.Log, acc *LogAccumulator) {
				// Expected order should be: summary 2 (8:00), summary 3 (9:00), summary 1 (10:00)
				if logs[0].GetId() != 2 {
					t.Errorf("expected first log to have ID 2, got %d", logs[0].GetId())
				}
				if logs[1].GetId() != 3 {
					t.Errorf("expected second log to have ID 3, got %d", logs[1].GetId())
				}
				if logs[2].GetId() != 1 {
					t.Errorf("expected third log to have ID 1, got %d", logs[2].GetId())
				}

				// Verify timestamps are in order
				t0 := logs[0].GetTs().AsTime()
				t1 := logs[1].GetTs().AsTime()
				t2 := logs[2].GetTs().AsTime()
				if t0.After(t1) || t1.After(t2) {
					t.Errorf("logs are not sorted chronologically: %v, %v, %v", t0, t1, t2)
				}
			},
		},
		{
			name: "missing severity error",
			logsToAdd: func(gen *id.Generator) []*StagingLog {
				return []*StagingLog{
					{
						Log:       log.NewLog(gen, structured.NewNodeReader(node1)),
						Summary:   "test summary",
						Timestamp: time.Now(),
						LogType:   testLogType,
						Severity:  nil,
					},
				}
			},
			wantErr:      true,
			wantLogCount: 0,
		},
		{
			name: "missing log type error",
			logsToAdd: func(gen *id.Generator) []*StagingLog {
				return []*StagingLog{
					{
						Log:       log.NewLog(gen, structured.NewNodeReader(node1)),
						Summary:   "test summary",
						Timestamp: time.Now(),
						LogType:   nil,
						Severity:  testSeverity,
					},
				}
			},
			wantErr:      true,
			wantLogCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idGen := id.NewGenerator()
			var buf bytes.Buffer
			writer, err := NewWriter(&buf)
			if err != nil {
				t.Fatalf("NewWriter() error = %v", err)
			}
			pool := NewInternPool(idGen, writer)
			serverPool := NewServerInternPool(pool, idGen, writer)
			acc := NewLogAccumulator(pool, serverPool, idGen, writer)

			for _, l := range tc.logsToAdd(idGen) {
				err := acc.AddLog(l)
				if (err != nil) != tc.wantErr {
					t.Fatalf("AddLog() error = %v, wantErr %v", err, tc.wantErr)
				}
			}

			if err := acc.Flush(); err != nil {
				t.Fatalf("Flush() error = %v", err)
			}
			if err := writer.Close(); err != nil {
				t.Fatalf("Close() error = %v", err)
			}

			reader, err := NewReader(&buf)
			if err != nil {
				t.Fatalf("NewReader() error = %v", err)
			}

			var logs []*pb.Log
			for {
				chunk, err := reader.NextChunk()
				if err != nil {
					break
				}
				if chunk.Type == ChunkTypeLog {
					var logChunk pb.LogChunk
					if err := proto.Unmarshal(chunk.Data, &logChunk); err != nil {
						t.Fatalf("failed to unmarshal log chunk: %v", err)
					}
					logs = append(logs, logChunk.Logs...)
				}
			}

			if len(logs) != tc.wantLogCount {
				t.Fatalf("expected %d logs, got %d", tc.wantLogCount, len(logs))
			}

			if tc.validate != nil {
				tc.validate(t, logs, acc)
			}
		})
	}
}

func TestLogAccumulator_StreamingSplit(t *testing.T) {
	var buf bytes.Buffer
	writer, err := NewWriter(&buf)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	idGen := id.NewGenerator()
	pool := NewInternPool(idGen, writer)
	serverPool := NewServerInternPool(pool, idGen, writer)
	acc := NewLogAccumulator(pool, serverPool, idGen, writer)
	acc.SetChunkSizeLimit(50) // Small limit to trigger chunk split on AddLog

	node := structured.NewStandardMap(
		[]string{"msg"},
		[]structured.Node{structured.NewStandardScalarNode("large message to trigger batch split")},
	)
	sevID := uint32(1)
	logTypeID := uint32(2)

	for i := 0; i < 5; i++ {
		err := acc.AddLog(&StagingLog{
			Log:       log.NewLog(idGen, structured.NewNodeReader(node)),
			Summary:   "summary",
			Timestamp: time.Now(),
			Severity:  &pb.Severity{Id: &sevID},
			LogType:   &pb.LogType{Id: &logTypeID},
		})
		if err != nil {
			t.Fatalf("AddLog() error = %v", err)
		}
	}

	if err := acc.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reader, err := NewReader(&buf)
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}

	chunkCount := 0
	for {
		chunk, err := reader.NextChunk()
		if err != nil {
			break
		}
		if chunk.Type == ChunkTypeLog {
			chunkCount++
		}
	}

	if chunkCount < 2 {
		t.Errorf("expected at least 2 log chunks due to streaming split, got %d", chunkCount)
	}
}
