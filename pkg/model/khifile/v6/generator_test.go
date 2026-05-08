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
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
)

func TestSplittingGenerator(t *testing.T) {
	// Create a sequence of 5 string messages.
	// Each StringValue("test") is about 6 bytes serialized.
	seq := func(yield func(*wrapperspb.StringValue) bool) {
		for i := 0; i < 5; i++ {
			if !yield(wrapperspb.String("test")) {
				return
			}
		}
	}

	// Wrapper to group them into MetadataChunk (reusing for test).
	// Although MetadataChunk expects Any, we just use MetadataChunk for the structure and cast it below.
	wrapper := func(batch []*wrapperspb.StringValue) *pb.MetadataChunk {
		// Mock implementation just for checking chunk boundaries
		return &pb.MetadataChunk{}
	}

	// Case 1: Limit is large enough for all items.
	gen1 := NewSplittingGenerator(ChunkTypeMetadata, seq, 1000, wrapper)
	defer gen1.Close()

	if gen1.ChunkType() != ChunkTypeMetadata {
		t.Errorf("Expected chunk type %d, got %d", ChunkTypeMetadata, gen1.ChunkType())
	}

	_, err := gen1.Next()
	if err != nil {
		t.Fatalf("Expected 1 chunk, got error: %v", err)
	}

	_, err = gen1.Next()
	if err != io.EOF {
		t.Fatalf("Expected EOF, got: %v", err)
	}

	// Case 2: Limit is very small (10 bytes), each item is ~8 bytes.
	// So each item should get its own chunk.
	gen2 := NewSplittingGenerator(ChunkTypeMetadata, seq, 10, wrapper)
	defer gen2.Close()

	count := 0
	for {
		_, err := gen2.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		count++
	}
	if count != 5 {
		t.Errorf("Expected 5 chunks, got %d", count)
	}
}

func TestSplittingGenerator_IntegrationWithWriter(t *testing.T) {
	seq := func(yield func(*pb.InternString) bool) {
		for i := 0; i < 3; i++ {
			id := uint32(i)
			val := "a"
			if !yield(&pb.InternString{Id: &id, Value: &val}) {
				return
			}
		}
	}

	wrapper := func(batch []*pb.InternString) *pb.InterningPoolChunk {
		return &pb.InterningPoolChunk{Strings: batch}
	}

	// Set limit to 30 bytes. A single InternString is ~5 bytes + 8 overhead = 13 bytes.
	// It should fit 2 strings in the first chunk (26 bytes), and 1 in the second.
	gen := NewSplittingGenerator(ChunkTypeInternPool, seq, 30, wrapper)

	var w DummyWriter
	err := w.WriteGenerator(gen)
	if err != nil {
		t.Fatalf("WriteGenerator failed: %v", err)
	}

	if len(w.chunks) != 2 {
		t.Fatalf("Expected 2 chunks to be written, got %d", len(w.chunks))
	}

	chunk1 := w.chunks[0].(*pb.InterningPoolChunk)
	if len(chunk1.Strings) != 2 {
		t.Errorf("Expected 2 strings in chunk 1, got %d", len(chunk1.Strings))
	}

	chunk2 := w.chunks[1].(*pb.InterningPoolChunk)
	if len(chunk2.Strings) != 1 {
		t.Errorf("Expected 1 string in chunk 2, got %d", len(chunk2.Strings))
	}
}

// DummyWriter simulates the real Writer to test WriteGenerator.
type DummyWriter struct {
	chunks []proto.Message
}

func (w *DummyWriter) WriteChunk(t ChunkType, msg proto.Message) error {
	w.chunks = append(w.chunks, msg)
	return nil
}

func (w *DummyWriter) WriteGenerator(gen ChunkGenerator) error {
	defer gen.Close()
	chunkType := gen.ChunkType()
	for {
		msg, err := gen.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := w.WriteChunk(chunkType, msg); err != nil {
			return err
		}
	}
}
