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

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	"google.golang.org/protobuf/proto"
)

func TestTimelineAccumulator_Streaming(t *testing.T) {
	testCases := []struct {
		name          string
		threshold     int64
		eventsCount   int
		wantChunksMin int
	}{
		{
			name:          "streams chunks when threshold is exceeded",
			threshold:     5,
			eventsCount:   12,
			wantChunksMin: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer, err := NewWriter(&buf)
			if err != nil {
				t.Fatalf("failed to create writer: %v", err)
			}

			idGen := id.NewGenerator()
			clientPool := NewInternPool(idGen, writer)
			serverPool := NewServerInternPool(clientPool, idGen, writer)
			acc := NewTimelineAccumulator(idGen, clientPool, serverPool, writer)
			acc.SetFlushThreshold(tc.threshold)

			timelineTypeID := uint32(1)
			path := acc.GetPath(nil, PathSegment{
				Name: "test-timeline",
				Type: &pb.TimelineType{Id: &timelineTypeID},
			})
			builder := acc.GetBuilder(path)

			for i := 1; i <= tc.eventsCount; i++ {
				builder.AddEvent(pendingEvent{
					LogID:     uint32(i),
					Timestamp: time.Now(),
				})
				if err := acc.NotifyItemsAdded(1); err != nil {
					t.Fatalf("NotifyItemsAdded failed: %v", err)
				}
			}

			// Final flush for remaining items and hierarchy
			if err := acc.Flush(); err != nil {
				t.Fatalf("Flush failed: %v", err)
			}

			// Read back all chunks from the stream
			reader, err := NewReader(&buf)
			if err != nil {
				t.Fatalf("failed to create reader: %v", err)
			}

			timelineChunkCount := 0
			totalEventsRead := 0
			hasHierarchy := false

			for {
				chunk, err := reader.NextChunk()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					t.Fatalf("failed to read next chunk: %v", err)
				}

				if chunk.Type == ChunkTypeTimeline {
					timelineChunkCount++
					var protoChunk pb.TimelineChunk
					if err := proto.Unmarshal(chunk.Data, &protoChunk); err != nil {
						t.Fatalf("failed to unmarshal timeline chunk: %v", err)
					}
					if len(protoChunk.Timelines) > 0 {
						hasHierarchy = true
					}
					for _, items := range protoChunk.TimelineItems {
						totalEventsRead += len(items.Events)
					}
				}
			}

			if timelineChunkCount < tc.wantChunksMin {
				t.Errorf("got %d timeline chunks, want at least %d", timelineChunkCount, tc.wantChunksMin)
			}
			if totalEventsRead != tc.eventsCount {
				t.Errorf("got %d total events, want %d", totalEventsRead, tc.eventsCount)
			}
			if !hasHierarchy {
				t.Errorf("expected timeline hierarchy chunk to be written")
			}
		})
	}
}
