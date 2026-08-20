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

func TestWorkbench_NewFromReader(t *testing.T) {
	testKhiData := createTestKhiFileData(t)

	testCases := []struct {
		name      string
		ctx       func() context.Context
		reader    func() *bytes.Reader
		totalSize int64
		wantErr   bool
	}{
		{
			name:      "successfully parses chunks from reader",
			ctx:       context.Background,
			reader:    func() *bytes.Reader { return bytes.NewReader(testKhiData) },
			totalSize: int64(len(testKhiData)),
			wantErr:   false,
		},
		{
			name:      "fails on invalid stream",
			ctx:       context.Background,
			reader:    func() *bytes.Reader { return bytes.NewReader([]byte("not a khi file")) },
			totalSize: 14,
			wantErr:   true,
		},
		{
			name: "fails on canceled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			reader:    func() *bytes.Reader { return bytes.NewReader(testKhiData) },
			totalSize: int64(len(testKhiData)),
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

			wb, err := NewFromReader(
				tc.ctx(),
				"wb-test-1",
				"inspection-1",
				tc.reader(),
				tc.totalSize,
				progressCb,
			)

			if (err != nil) != tc.wantErr {
				t.Fatalf("NewFromReader() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if wb.ID() != "wb-test-1" {
				t.Errorf("ID() = %q, want %q", wb.ID(), "wb-test-1")
			}
			if wb.searchIndex == nil {
				t.Fatalf("searchIndex is nil, want initialized search index")
			}
			if len(wb.searchIndex.Logs) != 2 {
				t.Errorf("len(searchIndex.Logs) = %d, want 2", len(wb.searchIndex.Logs))
			}
			if len(wb.searchIndex.Timelines) != 1 {
				t.Errorf("len(searchIndex.Timelines) = %d, want 1", len(wb.searchIndex.Timelines))
			}
			if len(wb.logChunks) != 0 {
				t.Errorf("len(logChunks) = %d, want 0 (released after indexing)", len(wb.logChunks))
			}
			if len(wb.timelineChunks) != 0 {
				t.Errorf("len(timelineChunks) = %d, want 0 (released after indexing)", len(wb.timelineChunks))
			}
			hasParsing := false
			hasIndexing := false
			for _, s := range capturedStages {
				if s == apiv1.OpenWorkbenchResponse_STAGE_PARSING_CHUNKS {
					hasParsing = true
				}
				if s == apiv1.OpenWorkbenchResponse_STAGE_INDEXING_DATA {
					hasIndexing = true
				}
			}
			if !hasParsing {
				t.Errorf("expected STAGE_PARSING_CHUNKS in captured stages: %v", capturedStages)
			}
			if !hasIndexing {
				t.Errorf("expected STAGE_INDEXING_DATA in captured stages: %v", capturedStages)
			}
		})
	}
}

func TestFormatByteSize(t *testing.T) {
	testCases := []struct {
		name  string
		input int64
		want  string
	}{
		{
			name:  "bytes below 1 KB",
			input: 512,
			want:  "512 B",
		},
		{
			name:  "exact 1 KB",
			input: 1024,
			want:  "1.0 KB",
		},
		{
			name:  "kilobytes with decimal",
			input: 1536,
			want:  "1.5 KB",
		},
		{
			name:  "megabytes",
			input: 15 * 1024 * 1024,
			want:  "15.0 MB",
		},
		{
			name:  "gigabytes",
			input: 2 * 1024 * 1024 * 1024,
			want:  "2.0 GB",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatByteSize(tc.input)
			if got != tc.want {
				t.Errorf("formatByteSize(%d) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
