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

package workbench

import (
	"bytes"
	"context"
	"testing"

	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"google.golang.org/protobuf/proto"
)

func createTestKhiFileData(t *testing.T) []byte {
	var buf bytes.Buffer
	writer, err := khifilev6model.NewWriter(&buf)
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}

	metadataChunk := &khifilev6.MetadataChunk{
		Metadata: []*khifilev6.MetadataItem{
			{
				Payload: &khifilev6.MetadataItem_Header{
					Header: &khifilev6.HeaderMetadata{
						InspectionName: proto.String("test-inspection"),
					},
				},
			},
		},
	}
	if err := writer.WriteChunk(khifilev6model.ChunkTypeMetadata, metadataChunk); err != nil {
		t.Fatalf("failed to write metadata chunk: %v", err)
	}

	logChunk := &khifilev6.LogChunk{
		Logs: []*khifilev6.Log{
			{Id: proto.Uint32(1), SummaryStringId: proto.Uint32(1)},
			{Id: proto.Uint32(2), SummaryStringId: proto.Uint32(2)},
		},
	}
	if err := writer.WriteChunk(khifilev6model.ChunkTypeLog, logChunk); err != nil {
		t.Fatalf("failed to write log chunk: %v", err)
	}

	timelineChunk := &khifilev6.TimelineChunk{
		Timelines: []*khifilev6.Timeline{
			{Id: proto.Uint32(10), NameStringId: proto.Uint32(10)},
		},
		TimelineItems: []*khifilev6.TimelineItems{
			{
				Id: proto.Uint32(100),
				Revisions: []*khifilev6.Revision{
					{LogId: proto.Uint32(1)},
					{LogId: proto.Uint32(2)},
				},
			},
		},
	}
	if err := writer.WriteChunk(khifilev6model.ChunkTypeTimeline, timelineChunk); err != nil {
		t.Fatalf("failed to write timeline chunk: %v", err)
	}

	return buf.Bytes()
}

func TestWorkbench_NewWorkbenchFromReader(t *testing.T) {
	testKhiData := createTestKhiFileData(t)

	testCases := []struct {
		name      string
		reader    func() *bytes.Reader
		totalSize int64
		wantErr   bool
	}{
		{
			name:      "successfully parses chunks from reader",
			reader:    func() *bytes.Reader { return bytes.NewReader(testKhiData) },
			totalSize: int64(len(testKhiData)),
			wantErr:   false,
		},
		{
			name:      "fails on invalid stream",
			reader:    func() *bytes.Reader { return bytes.NewReader([]byte("not a khi file")) },
			totalSize: 14,
			wantErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var capturedStages []apiv1.OpenWorkbenchResponse_Stage
			progressCb := func(stage apiv1.OpenWorkbenchResponse_Stage, pct float64, msg string) error {
				capturedStages = append(capturedStages, stage)
				return nil
			}

			wb, err := NewWorkbenchFromReader(
				context.Background(),
				"wb-test-1",
				"inspection-1",
				tc.reader(),
				tc.totalSize,
				progressCb,
			)

			if (err != nil) != tc.wantErr {
				t.Fatalf("NewWorkbenchFromReader() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if wb.ID() != "wb-test-1" {
				t.Errorf("ID() = %q, want %q", wb.ID(), "wb-test-1")
			}
			if len(wb.logChunks) != 1 {
				t.Errorf("len(logChunks) = %d, want 1", len(wb.logChunks))
			}
			if len(wb.timelineChunks) != 1 {
				t.Errorf("len(timelineChunks) = %d, want 1", len(wb.timelineChunks))
			}
			if len(capturedStages) == 0 {
				t.Errorf("expected captured progress stages")
			}
		})
	}
}
