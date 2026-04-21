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
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"
)

var (
	// MagicBytes are the first 3 bytes of a KHI file ("KHI").
	MagicBytes = []byte{'K', 'H', 'I'}
	// Version is the supported version of the KHI file format.
	Version = byte(0x06)

	ErrInvalidMagicBytes  = errors.New("invalid magic bytes")
	ErrUnsupportedVersion = errors.New("unsupported version")
)

// ChunkType represents the type of a chunk in the KHI file.
type ChunkType uint32

const (
	ChunkTypeMetadata      ChunkType = 1
	ChunkTypeInternPool    ChunkType = 2
	ChunkTypeLog           ChunkType = 3
	ChunkTypeTimelineStyle ChunkType = 4
	ChunkTypeTimeline      ChunkType = 5
)

// Writer writes chunks to a KHI v6 file.
type Writer struct {
	w io.Writer
}

// NewWriter creates a new Writer and writes the magic bytes and version.
func NewWriter(w io.Writer) (*Writer, error) {
	header := []byte{MagicBytes[0], MagicBytes[1], MagicBytes[2], Version}
	if _, err := w.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write header: %w", err)
	}
	return &Writer{w: w}, nil
}

// WriteChunk serializes the given protobuf message, gzips it, and writes the chunk header and payload.
func (w *Writer) WriteChunk(chunkType ChunkType, message proto.Message) error {
	// 1. Serialize proto message
	b, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal proto message: %w", err)
	}

	// 2. Compress payload with gzip
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(b); err != nil {
		return fmt.Errorf("failed to compress payload: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	payload := buf.Bytes()
	const maxChunkSize = 64 * 1024 * 1024 // 64MB hard limit based on Protobuf constraints
	if len(payload) > maxChunkSize {
		return fmt.Errorf("payload size %d exceeds maximum allowed chunk size (64MB)", len(payload))
	}
	size := uint32(len(payload))

	// 3. Write chunk header (Size, Type)
	var header [8]byte
	binary.LittleEndian.PutUint32(header[0:4], size)
	binary.LittleEndian.PutUint32(header[4:8], uint32(chunkType))

	if _, err := w.w.Write(header[:]); err != nil {
		return fmt.Errorf("failed to write chunk header: %w", err)
	}

	// 4. Write payload
	if _, err := w.w.Write(payload); err != nil {
		return fmt.Errorf("failed to write chunk payload: %w", err)
	}

	return nil
}

// WriteGenerator consumes the ChunkGenerator and writes all generated chunks sequentially.
func (w *Writer) WriteGenerator(gen ChunkGenerator) error {
	defer gen.Close()
	chunkType := gen.ChunkType()
	for {
		msg, err := gen.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("generator error for type %d: %w", chunkType, err)
		}
		if err := w.WriteChunk(chunkType, msg); err != nil {
			return err
		}
	}
}

// Chunk represents a parsed chunk from a KHI v6 file.
type Chunk struct {
	Type ChunkType
	Data []byte // Uncompressed protobuf binary data
}

// Reader reads chunks from a KHI v6 file.
type Reader struct {
	r io.Reader
}

// NewReader reads and validates the file header.
func NewReader(r io.Reader) (*Reader, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	if header[0] != MagicBytes[0] || header[1] != MagicBytes[1] || header[2] != MagicBytes[2] {
		return nil, ErrInvalidMagicBytes
	}

	if header[3] != Version {
		return nil, ErrUnsupportedVersion
	}

	return &Reader{r: r}, nil
}

// NextChunk reads the next chunk size and type, decompresses the payload, and returns it.
// Returns io.EOF if no more chunks are available.
func (r *Reader) NextChunk() (*Chunk, error) {
	// 1. Read chunk header (Size, Type)
	var header [8]byte
	if _, err := io.ReadFull(r.r, header[:]); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, err // Return native EOF/UnexpectedEOF for natural termination/truncation
		}
		return nil, fmt.Errorf("failed to read chunk header: %w", err)
	}

	size := binary.LittleEndian.Uint32(header[0:4])
	chunkType := ChunkType(binary.LittleEndian.Uint32(header[4:8]))

	// 2. Read compressed payload
	const maxChunkSize = 64 * 1024 * 1024 // 64MB hard limit based on Protobuf constraints
	if size > maxChunkSize {
		return nil, fmt.Errorf("chunk size %d exceeds safety limit (64MB)", size)
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(r.r, payload); err != nil {
		return nil, fmt.Errorf("failed to read chunk payload: %w", err)
	}

	// 3. Decompress payload
	gr, err := gzip.NewReader(bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gr.Close()

	const maxUncompressedSize = 64 * 1024 * 1024 // 64MB limit to match Protobuf parser limits
	uncompressed, err := io.ReadAll(io.LimitReader(gr, maxUncompressedSize))
	if err != nil {
		return nil, fmt.Errorf("failed to decompress payload: %w", err)
	}

	return &Chunk{
		Type: chunkType,
		Data: uncompressed,
	}, nil
}
