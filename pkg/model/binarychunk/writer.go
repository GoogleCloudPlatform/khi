// Copyright 2024 Google LLC
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

package binarychunk

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"weak"
)

var ErrBufferChunkFilled = errors.New("this buffer is already full")

type BinaryReference struct {
	Offset int `json:"offset"`
	Length int `json:"len"`
	Buffer int `json:"buffer"`
}

// LargeBinaryWriter stores text as a large binary chunk and returns BinaryReference points the buffer location.
type LargeBinaryWriter interface {
	// Write the specified text and returns the BinaryReference
	Write(data []byte) (*BinaryReference, error)
	// Read buffer from a BinaryReference
	Read(ref *BinaryReference) ([]byte, error)
	// ChunkReader returns a read closer for the result binary chunk.
	ChunkReader() (io.ReadCloser, error)
	// Seal marks this writer done writing.
	Seal() error
}

// FileSystemBinaryWriter is a basic implementation of the LargeTextWriter.
type FileSystemBinaryWriter struct {
	bufferIndex       int
	maximumBufferSize int
	currentLength     int
	buffer            *bytes.Buffer
	bufferWeak        weak.Pointer[bytes.Buffer]
	mu                sync.RWMutex
	sealed            bool
	tmpFolderPath     string
	sealedFilePath    string
}

// Seal implements LargeBinaryWriter.
func (w *FileSystemBinaryWriter) Seal() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	file, err := os.CreateTemp(w.tmpFolderPath, "khi-")
	if err != nil {
		return err
	}
	w.sealedFilePath = file.Name()
	size, err := w.buffer.WriteTo(file)
	if err != nil {
		return err
	}
	slog.Debug("sealed binary chunk", "path", w.sealedFilePath, "size", size)
	w.bufferWeak = weak.Make(w.buffer)
	w.buffer = nil
	w.sealed = true
	return nil
}

var _ LargeBinaryWriter = (*FileSystemBinaryWriter)(nil)

func NewFileSystemBinaryWriter(tmpPath string, bufferIndex int, maxSize int) (*FileSystemBinaryWriter, error) {
	buf := new(bytes.Buffer)
	buf.Grow(maxSize)
	return &FileSystemBinaryWriter{
		bufferIndex:       bufferIndex,
		maximumBufferSize: maxSize,
		currentLength:     0,
		buffer:            buf,
		mu:                sync.RWMutex{},
		sealedFilePath:    "",
		tmpFolderPath:     tmpPath,
	}, nil
}

func (w *FileSystemBinaryWriter) canWrite(size int) bool {
	return !w.sealed && w.currentLength+size <= w.maximumBufferSize
}

func (w *FileSystemBinaryWriter) Write(data []byte) (*BinaryReference, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.canWrite(len(data)) {
		return nil, fmt.Errorf("buffer can't write the specified length %d (current:%d,maximum:%d): %w", len(data), w.currentLength, w.maximumBufferSize, ErrBufferChunkFilled)
	}
	size, err := w.buffer.Write(data)
	if err != nil {
		return nil, err
	}
	reference := &BinaryReference{
		Buffer: w.bufferIndex,
		Length: size,
		Offset: w.currentLength,
	}
	w.currentLength += size
	return reference, nil
}

func (w *FileSystemBinaryWriter) Read(ref *BinaryReference) ([]byte, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if ref.Buffer != w.bufferIndex {
		return nil, fmt.Errorf("invalid buffer index. it's not current buffer index")
	}
	if w.buffer != nil {
		return w.buffer.Bytes()[ref.Offset : ref.Offset+ref.Length], nil
	} else if weakBufPtr := w.bufferWeak.Value(); weakBufPtr != nil {
		return weakBufPtr.Bytes()[ref.Offset : ref.Offset+ref.Length], nil
	} else {
		slog.Debug("cache miss. Loading the chunk buffer again", "bufferIndex", w.bufferIndex)
		file, err := os.Open(w.sealedFilePath)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		buf, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}
		w.bufferWeak = weak.Make(w.buffer) // This writer must be Sealed when w.buffer = nil, thus we can expect no Write calls after that. Thus it's safe to write bufferWeak with RLock
		return buf[ref.Offset : ref.Offset+ref.Length], nil
	}
}

func (w *FileSystemBinaryWriter) ChunkReader() (io.ReadCloser, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.sealedFilePath == "" {
		return nil, fmt.Errorf("this writer has no associated sealed file")
	}
	file, err := os.Open(w.sealedFilePath)
	if err != nil {
		return nil, err
	}
	return file, nil
}
