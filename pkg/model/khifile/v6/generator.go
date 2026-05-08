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
	"io"
	"iter"

	"google.golang.org/protobuf/proto"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
)

// DefaultChunkSizeLimit is the recommended maximum byte size for a single chunk payload.
// It is set to 60MB to safely stay under the typical 64MB Protobuf parser limit.
const DefaultChunkSizeLimit = 60 * 1024 * 1024

// ChunkGenerator is an interface for generating chunks sequentially.
// Implementations must ensure that the size of the generated messages respects
// the KHI file chunk limit (typically ~64MB).
type ChunkGenerator interface {
	ChunkType() ChunkType
	// Next returns the next chunk. It returns io.EOF when done.
	Next() (proto.Message, error)
	// Close releases any resources (like goroutines from iterators) associated with the generator.
	Close() error
}

type splittingGenerator[T proto.Message, C proto.Message] struct {
	chunkType ChunkType
	next      func() (T, bool)
	stop      func()
	sizeLimit int
	wrapper   func([]T) C

	pendingItem T
	hasPending  bool
}

// NewSplittingGenerator creates a highly generic ChunkGenerator that splits an iterator
// of Protobuf messages into multiple chunks based on an exact byte limit.
//
// It uses `proto.Size()` to accurately measure each element. When the accumulated size
// exceeds `sizeLimit`, it invokes the `wrapper` function to build the final Chunk message.
//
// Type Parameters:
//   - T: The element type yielded by the iterator (e.g., *pb.Log)
//   - C: The chunk message type that holds the elements (e.g., *pb.LogChunk)
func NewSplittingGenerator[T proto.Message, C proto.Message](
	chunkType ChunkType,
	seq iter.Seq[T],
	sizeLimit int,
	wrapper func([]T) C,
) ChunkGenerator {
	next, stop := iter.Pull(seq)
	return &splittingGenerator[T, C]{
		chunkType: chunkType,
		next:      next,
		stop:      stop,
		sizeLimit: sizeLimit,
		wrapper:   wrapper,
	}
}

func (g *splittingGenerator[T, C]) ChunkType() ChunkType {
	return g.chunkType
}

func (g *splittingGenerator[T, C]) Next() (proto.Message, error) {
	var batch []T
	currentSize := 0

	for {
		var item T
		var valid bool

		if g.hasPending {
			item = g.pendingItem
			valid = true
			g.hasPending = false
		} else {
			item, valid = g.next()
		}

		if !valid {
			if len(batch) == 0 {
				return nil, io.EOF
			}
			return g.wrapper(batch), nil
		}

		// proto.Size gives the exact byte size of the serialized message.
		// We add 8 bytes as a conservative estimate for the Protobuf length-delimited field tag
		// and length prefix overhead.
		itemSize := proto.Size(item) + 8

		// If adding this item exceeds the limit AND we already have items in the batch,
		// we hold this item for the next chunk and return the current batch.
		// (If the batch is empty, we must add it anyway to avoid an infinite loop
		// in the case where a single item is larger than the limit).
		if currentSize+itemSize > g.sizeLimit && len(batch) > 0 {
			g.pendingItem = item
			g.hasPending = true
			return g.wrapper(batch), nil
		}

		batch = append(batch, item)
		currentSize += itemSize
	}
}

func (g *splittingGenerator[T, C]) Close() error {
	if g.stop != nil {
		g.stop()
	}
	return nil
}

// NewInternPoolGenerator creates a SplittingGenerator for InternPool chunks.
// It groups pb.InternString messages into pb.InterningPoolChunk respecting the size limit.
func NewInternPoolGenerator(seq iter.Seq[*pb.InternString]) ChunkGenerator {
	wrapper := func(batch []*pb.InternString) *pb.InterningPoolChunk {
		return &pb.InterningPoolChunk{Strings: batch}
	}
	return NewSplittingGenerator(ChunkTypeInternPool, seq, DefaultChunkSizeLimit, wrapper)
}

// NewInternFieldPathSetGenerator creates a SplittingGenerator for InternPool chunks containing field path sets.
// It groups pb.InternFieldPathSet messages into pb.InterningPoolChunk respecting the size limit.
func NewInternFieldPathSetGenerator(seq iter.Seq[*pb.InternFieldPathSet]) ChunkGenerator {
	wrapper := func(batch []*pb.InternFieldPathSet) *pb.InterningPoolChunk {
		return &pb.InterningPoolChunk{FieldPathSets: batch}
	}
	return NewSplittingGenerator(ChunkTypeInternPool, seq, DefaultChunkSizeLimit, wrapper)
}

// NewLogGenerator creates a SplittingGenerator for Log chunks.
// It groups pb.Log messages into pb.LogChunk respecting the size limit.
func NewLogGenerator(seq iter.Seq[*pb.Log]) ChunkGenerator {
	wrapper := func(batch []*pb.Log) *pb.LogChunk {
		return &pb.LogChunk{Logs: batch}
	}
	return NewSplittingGenerator(ChunkTypeLog, seq, DefaultChunkSizeLimit, wrapper)
}

// NewTimelineGenerator creates a SplittingGenerator for Timeline chunks.
// It groups pb.Timeline messages into pb.TimelineChunk respecting the size limit.
func NewTimelineGenerator(seq iter.Seq[*pb.Timeline]) ChunkGenerator {
	wrapper := func(batch []*pb.Timeline) *pb.TimelineChunk {
		return &pb.TimelineChunk{Timelines: batch}
	}
	return NewSplittingGenerator(ChunkTypeTimeline, seq, DefaultChunkSizeLimit, wrapper)
}

// NewTimelineItemsGenerator creates a SplittingGenerator for Timeline chunks containing timeline items.
// It groups pb.TimelineItems messages into pb.TimelineChunk respecting the size limit.
func NewTimelineItemsGenerator(seq iter.Seq[*pb.TimelineItems]) ChunkGenerator {
	wrapper := func(batch []*pb.TimelineItems) *pb.TimelineChunk {
		return &pb.TimelineChunk{TimelineItems: batch}
	}
	return NewSplittingGenerator(ChunkTypeTimeline, seq, DefaultChunkSizeLimit, wrapper)
}
