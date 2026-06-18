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
	"errors"
	"io"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestBuilder_Build verifies that the Builder successfully generates a KHI file.
// It tests both an empty build and a full build with metadata, logs, and timelines.
func TestBuilder_Build(t *testing.T) {
	testCases := []struct {
		name   string
		setup  func(b *Builder)
		verify func(t *testing.T, reader *Reader)
	}{
		{
			name:  "empty build writes only style chunk",
			setup: func(b *Builder) {},
			verify: func(t *testing.T, reader *Reader) {
				chunk, err := reader.NextChunk()
				if err != nil {
					t.Fatalf("Failed to read next chunk: %v.", err)
				}
				if chunk.Type != ChunkTypeTimelineStyle {
					t.Errorf("Expected style chunk, got %v.", chunk.Type)
				}

				_, err = reader.NextChunk()
				if !errors.Is(err, io.EOF) {
					t.Errorf("Expected EOF, got %v.", err)
				}
			},
		},
		{
			name: "full build writes metadata, log, timeline, style, and intern chunks",
			setup: func(b *Builder) {
				_ = b.MetadataAccumulator.AddMetadata(&inspectionmetadata.HeaderMetadata{
					InspectionType: "test-inspection-type",
					InspectionName: "test-inspection-name",
					FileSize:       1024,
				})

				node := structured.NewStandardMap(
					[]string{"message"},
					[]structured.Node{
						structured.NewStandardScalarNode("hello log"),
					},
				)
				severityID := uint32(1)
				logTypeID := uint32(2)
				_ = b.LogAccumulator.AddLog(&StagingLog{
					Log:       log.NewLog(structured.NewNodeReader(node)),
					Summary:   "hello summary",
					Timestamp: time.Date(2026, 4, 29, 8, 0, 0, 0, time.UTC),
					Severity:  &pb.Severity{Id: &severityID},
					LogType:   &pb.LogType{Id: &logTypeID},
				})

				timelineTypeID := uint32(3)
				timelineType := &pb.TimelineType{Id: &timelineTypeID}
				path := b.TimelineAccumulator.GetPath(nil, PathSegment{
					Name: "test-timeline-path",
					Type: timelineType,
				})
				tb := b.TimelineAccumulator.GetBuilder(path)
				logID := uint32(1)
				tb.AddRevision(&pb.Revision{
					LogId:       &logID,
					ChangedTime: timestamppb.New(time.Date(2026, 4, 29, 8, 0, 0, 0, time.UTC)),
				})
			},
			verify: func(t *testing.T, reader *Reader) {
				var chunks []*Chunk
				for {
					chunk, err := reader.NextChunk()
					if errors.Is(err, io.EOF) {
						break
					}
					if err != nil {
						t.Fatalf("Failed to read next chunk: %v.", err)
					}
					chunks = append(chunks, chunk)
				}

				expectedTypes := []ChunkType{
					ChunkTypeTimelineStyle,
					ChunkTypeMetadata,
					ChunkTypeLog,
					ChunkTypeTimeline,
					ChunkTypeTimeline,
					ChunkTypeInternPool,
					ChunkTypeInternPool,
				}

				var gotTypes []ChunkType
				for _, c := range chunks {
					gotTypes = append(gotTypes, c.Type)
				}

				if diff := cmp.Diff(expectedTypes, gotTypes); diff != "" {
					t.Errorf("Chunk types mismatch (-want +got):\n%s", diff)
				}

				for _, c := range chunks {
					switch c.Type {
					case ChunkTypeTimelineStyle:
						var styleChunk pb.TimelineStyleChunk
						if err := proto.Unmarshal(c.Data, &styleChunk); err != nil {
							t.Fatalf("Failed to unmarshal style chunk: %v.", err)
						}
						if styleChunk.IconAtlas == nil {
							t.Error("Expected style chunk to have non-nil IconAtlas.")
						}

					case ChunkTypeMetadata:
						var metadataChunk pb.MetadataChunk
						if err := proto.Unmarshal(c.Data, &metadataChunk); err != nil {
							t.Fatalf("Failed to unmarshal metadata chunk: %v.", err)
						}
						if len(metadataChunk.Metadata) != 1 {
							t.Errorf("Expected 1 metadata item, got %d.", len(metadataChunk.Metadata))
							break
						}
						header := metadataChunk.Metadata[0].GetHeader()
						if header == nil {
							t.Error("Expected header metadata.")
							break
						}
						if header.GetInspectionType() != "test-inspection-type" || header.GetInspectionName() != "test-inspection-name" {
							t.Errorf("Unexpected header: %+v.", header)
						}

					case ChunkTypeLog:
						var logChunk pb.LogChunk
						if err := proto.Unmarshal(c.Data, &logChunk); err != nil {
							t.Fatalf("Failed to unmarshal log chunk: %v.", err)
						}
						if len(logChunk.Logs) != 1 {
							t.Errorf("Expected 1 log, got %d.", len(logChunk.Logs))
							break
						}
						logItem := logChunk.Logs[0]
						if logItem.GetId() != 1 {
							t.Errorf("Expected log ID 1, got %d.", logItem.GetId())
						}
						if logItem.GetSeverityTypeId() != 1 {
							t.Errorf("Expected severity ID 1, got %d.", logItem.GetSeverityTypeId())
						}
						if logItem.GetLogTypeId() != 2 {
							t.Errorf("Expected log type ID 2, got %d.", logItem.GetLogTypeId())
						}

					case ChunkTypeTimeline:
						var timelineChunk pb.TimelineChunk
						if err := proto.Unmarshal(c.Data, &timelineChunk); err != nil {
							t.Fatalf("Failed to unmarshal timeline chunk: %v.", err)
						}
						switch {
						case len(timelineChunk.Timelines) > 0:
							if len(timelineChunk.Timelines) != 1 {
								t.Errorf("Expected 1 timeline, got %d.", len(timelineChunk.Timelines))
							} else {
								tl := timelineChunk.Timelines[0]
								if tl.GetId() != 1 {
									t.Errorf("Expected timeline ID 1, got %d.", tl.GetId())
								}
							}
						case len(timelineChunk.TimelineItems) > 0:
							if len(timelineChunk.TimelineItems) != 1 {
								t.Errorf("Expected 1 timeline item collection, got %d.", len(timelineChunk.TimelineItems))
							} else {
								ti := timelineChunk.TimelineItems[0]
								if ti.GetId() != 1 {
									t.Errorf("Expected timeline items ID 1, got %d.", ti.GetId())
								}
								if len(ti.Revisions) != 1 {
									t.Errorf("Expected 1 revision, got %d.", len(ti.Revisions))
								} else {
									rev := ti.Revisions[0]
									if rev.GetLogId() != 1 {
										t.Errorf("Expected log ID 1 in revision, got %d.", rev.GetLogId())
									}
								}
							}
						default:
							t.Error("Expected either Timelines or TimelineItems in TimelineChunk.")
						}

					case ChunkTypeInternPool:
						var internPool pb.InterningPoolChunk
						if err := proto.Unmarshal(c.Data, &internPool); err != nil {
							t.Fatalf("Failed to unmarshal intern pool chunk: %v.", err)
						}
						if len(internPool.Strings) > 0 {
							foundSummary := false
							for _, s := range internPool.Strings {
								if s.GetValue() == "hello summary" {
									foundSummary = true
									break
								}
							}
							if !foundSummary {
								t.Error("Expected 'hello summary' to be interned in InternPool.")
							}
						}
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBuilder()
			tc.setup(b)

			var buf bytes.Buffer
			if err := b.Build(&buf, nil); err != nil {
				t.Fatalf("Build() failed: %v.", err)
			}

			reader, err := NewReader(&buf)
			if err != nil {
				t.Fatalf("NewReader() failed: %v.", err)
			}

			tc.verify(t, reader)
		})
	}
}
