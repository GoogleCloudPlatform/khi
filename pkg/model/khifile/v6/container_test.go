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
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestWriter_Header(t *testing.T) {
	var buf bytes.Buffer
	_, err := NewWriter(&buf)
	if err != nil {
		t.Fatalf("NewWriter returned error: %v", err)
	}

	expected := []byte{'K', 'H', 'I', 0x06}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("Header mismatch, got: %v, want: %v", buf.Bytes(), expected)
	}
}

func TestReader_ValidHeader(t *testing.T) {
	buf := bytes.NewBuffer([]byte{'K', 'H', 'I', 0x06})
	_, err := NewReader(buf)
	if err != nil {
		t.Errorf("NewReader returned unexpected error: %v", err)
	}
}

func TestReader_InvalidMagicBytes(t *testing.T) {
	buf := bytes.NewBuffer([]byte{'K', 'H', 'A', 0x06})
	_, err := NewReader(buf)
	if !errors.Is(err, ErrInvalidMagicBytes) {
		t.Errorf("Expected ErrInvalidMagicBytes, got: %v", err)
	}
}

func TestReader_UnsupportedVersion(t *testing.T) {
	buf := bytes.NewBuffer([]byte{'K', 'H', 'I', 0x07})
	_, err := NewReader(buf)
	if !errors.Is(err, ErrUnsupportedVersion) {
		t.Errorf("Expected ErrUnsupportedVersion, got: %v", err)
	}
}

func TestWriter_WriteChunk_Format(t *testing.T) {
	var buf bytes.Buffer
	w, err := NewWriter(&buf)
	if err != nil {
		t.Fatalf("NewWriter returned error: %v", err)
	}

	msg := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"test": {Kind: &structpb.Value_StringValue{StringValue: "value"}},
		},
	}
	err = w.WriteChunk(ChunkTypeMetadata, msg)
	if err != nil {
		t.Fatalf("WriteChunk returned error: %v", err)
	}

	b := buf.Bytes()
	if len(b) < 12 {
		t.Fatalf("Buffer too short")
	}

	// First 4 bytes are header
	// Next 4 bytes are size
	size := binary.LittleEndian.Uint32(b[4:8])
	if int(size) != len(b)-12 {
		t.Errorf("Size mismatch. Expected %d, got %d", len(b)-12, size)
	}

	// Next 4 bytes are type
	chunkType := binary.LittleEndian.Uint32(b[8:12])
	if chunkType != uint32(ChunkTypeMetadata) {
		t.Errorf("ChunkType mismatch. Expected %d, got %d", ChunkTypeMetadata, chunkType)
	}

	// Verify it's a valid gzip stream
	payload := b[12:]
	gr, err := gzip.NewReader(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("Failed to create gzip reader for payload: %v", err)
	}
	defer gr.Close()

	uncompressed, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("Failed to decompress payload: %v", err)
	}

	var parsed structpb.Struct
	if err := proto.Unmarshal(uncompressed, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal decompressed payload: %v", err)
	}

	if diff := cmp.Diff(msg, &parsed, protocmp.Transform()); diff != "" {
		t.Errorf("Unmarshaled proto differs from original (-want +got):\n%s", diff)
	}
}

func TestReader_EOF(t *testing.T) {
	buf := bytes.NewBuffer([]byte{'K', 'H', 'I', 0x06})
	r, _ := NewReader(buf)
	_, err := r.NextChunk()
	if !errors.Is(err, io.EOF) {
		t.Errorf("Expected io.EOF, got %v", err)
	}
}

func TestReader_IncompleteChunk(t *testing.T) {
	// Provide valid header + chunk size 10 + chunk type 1 + only 2 bytes payload
	buf := bytes.NewBuffer([]byte{'K', 'H', 'I', 0x06, 10, 0, 0, 0, 1, 0, 0, 0, 0, 0})
	r, _ := NewReader(buf)
	_, err := r.NextChunk()
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("Expected io.ErrUnexpectedEOF, got %v", err)
	}
}

func TestContainer_WriteAndRead_E2E(t *testing.T) {
	var buf bytes.Buffer
	w, err := NewWriter(&buf)
	if err != nil {
		t.Fatalf("NewWriter returned error: %v", err)
	}

	chunks := []struct {
		ctype ChunkType
		msg   proto.Message
	}{
		{
			ctype: ChunkTypeMetadata,
			msg: &structpb.Struct{
				Fields: map[string]*structpb.Value{"id": {Kind: &structpb.Value_NumberValue{NumberValue: 1}}},
			},
		},
		{
			ctype: ChunkTypeLog,
			msg: &structpb.Struct{
				Fields: map[string]*structpb.Value{"log": {Kind: &structpb.Value_StringValue{StringValue: "hello"}}},
			},
		},
	}

	for _, c := range chunks {
		if err := w.WriteChunk(c.ctype, c.msg); err != nil {
			t.Fatalf("WriteChunk failed: %v", err)
		}
	}

	r, err := NewReader(&buf)
	if err != nil {
		t.Fatalf("NewReader returned error: %v", err)
	}

	for i, c := range chunks {
		parsed, err := r.NextChunk()
		if err != nil {
			t.Fatalf("NextChunk #%d returned error: %v", i, err)
		}

		if parsed.Type != c.ctype {
			t.Errorf("Chunk #%d Type mismatch. got %d, want %d", i, parsed.Type, c.ctype)
		}

		var parsedMsg structpb.Struct
		if err := proto.Unmarshal(parsed.Data, &parsedMsg); err != nil {
			t.Fatalf("Chunk #%d Unmarshal failed: %v", i, err)
		}

		if diff := cmp.Diff(c.msg, &parsedMsg, protocmp.Transform()); diff != "" {
			t.Errorf("Chunk #%d payload differs (-want +got):\n%s", i, diff)
		}
	}

	// Should EOF now
	_, err = r.NextChunk()
	if !errors.Is(err, io.EOF) {
		t.Errorf("Expected EOF at the end, got: %v", err)
	}
}
