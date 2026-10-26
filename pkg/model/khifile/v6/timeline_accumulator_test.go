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

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	"github.com/google/go-cmp/cmp"
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

func TestTimelineAccumulator_TimelineTypeVocabulary(t *testing.T) {
	kindType := &pb.TimelineType{
		Id:           proto.Uint32(1),
		Label:        proto.String("kind"),
		SortPriority: proto.Int32(200),
		SortPolicyConfig: &pb.TimelineType_AlphabeticalPolicy{
			AlphabeticalPolicy: &pb.AlphabeticalSortPolicy{
				PrioritizedNames: []string{"node", "pod", "service"},
			},
		},
	}
	namespaceType := &pb.TimelineType{
		Id:           proto.Uint32(2),
		Label:        proto.String("namespace"),
		SortPriority: proto.Int32(300),
		SortPolicyConfig: &pb.TimelineType_AlphabeticalPolicy{
			AlphabeticalPolicy: &pb.AlphabeticalSortPolicy{
				PrioritizedNames: []string{"kube-system", "default"},
			},
		},
	}
	resourceType := &pb.TimelineType{
		Id:           proto.Uint32(3),
		Label:        proto.String("resource"),
		SortPriority: proto.Int32(1000),
	}
	apiVersionType := &pb.TimelineType{
		Id:           proto.Uint32(4),
		Label:        proto.String("apiVersion"),
		SortPriority: proto.Int32(100),
	}
	samePriorityAType := &pb.TimelineType{
		Id:           proto.Uint32(5),
		Label:        proto.String("alpha"),
		SortPriority: proto.Int32(500),
	}
	samePriorityBType := &pb.TimelineType{
		Id:           proto.Uint32(6),
		Label:        proto.String("beta"),
		SortPriority: proto.Int32(500),
	}

	testCases := []struct {
		name  string
		limit int
		setup func(acc *TimelineAccumulator)
		want  []inspectionmetadata.TimelineTypeVocabulary
	}{
		{
			name:  "orders types by priority and label, prioritizes configured names, deduplicates across parents, and caps examples",
			limit: 3,
			setup: func(acc *TimelineAccumulator) {
				v1 := acc.GetPath(nil, PathSegment{Name: "core/v1", Type: apiVersionType})
				podKind := acc.GetPath(v1, PathSegment{Name: "pod", Type: kindType})
				svcKind := acc.GetPath(v1, PathSegment{Name: "service", Type: kindType})
				nodeKind := acc.GetPath(v1, PathSegment{Name: "node", Type: kindType})
				_ = acc.GetPath(v1, PathSegment{Name: "configmap", Type: kindType})
				_ = acc.GetPath(v1, PathSegment{Name: "daemonset", Type: kindType})

				// Same namespace names under two different kind parents should be counted once.
				nsDefaultUnderPod := acc.GetPath(podKind, PathSegment{Name: "default", Type: namespaceType})
				_ = acc.GetPath(svcKind, PathSegment{Name: "default", Type: namespaceType})
				_ = acc.GetPath(podKind, PathSegment{Name: "kube-system", Type: namespaceType})
				_ = acc.GetPath(podKind, PathSegment{Name: "app-ns", Type: namespaceType})

				// Empty names are ignored.
				_ = acc.GetPath(nodeKind, PathSegment{Name: "", Type: resourceType})
				_ = acc.GetPath(nsDefaultUnderPod, PathSegment{Name: "zeta", Type: resourceType})
				_ = acc.GetPath(nsDefaultUnderPod, PathSegment{Name: "alpha", Type: resourceType})

				// Tie-breaker by label when sort priority is equal.
				_ = acc.GetPath(nil, PathSegment{Name: "b1", Type: samePriorityBType})
				_ = acc.GetPath(nil, PathSegment{Name: "a1", Type: samePriorityAType})
			},
			want: []inspectionmetadata.TimelineTypeVocabulary{
				{
					Label:        "apiVersion",
					NameCount:    1,
					ExampleNames: []string{"core/v1"},
				},
				{
					Label:        "kind",
					NameCount:    5,
					ExampleNames: []string{"node", "pod", "service"},
				},
				{
					Label:        "namespace",
					NameCount:    3,
					ExampleNames: []string{"kube-system", "default", "app-ns"},
				},
				{
					Label:        "alpha",
					NameCount:    1,
					ExampleNames: []string{"a1"},
				},
				{
					Label:        "beta",
					NameCount:    1,
					ExampleNames: []string{"b1"},
				},
				{
					Label:        "resource",
					NameCount:    2,
					ExampleNames: []string{"alpha", "zeta"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idGen := id.NewGenerator()
			clientPool := NewTestInternPool(idGen)
			serverPool := NewTestServerInternPool(clientPool, idGen)
			acc := NewTestTimelineAccumulator(idGen, clientPool, serverPool)

			tc.setup(acc)

			got := acc.TimelineTypeVocabulary(tc.limit)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TimelineTypeVocabulary(%d) mismatch (-want +got):\n%s", tc.limit, diff)
			}
		})
	}
}
