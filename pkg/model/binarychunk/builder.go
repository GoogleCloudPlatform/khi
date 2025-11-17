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
	"context"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	"golang.org/x/sync/errgroup"
)

const MAXIMUM_CHUNK_SIZE = 1024 * 1024 * 50

// defaultParallelWriterCount is the default count of binary chunk count.
// Each writers must lock when it's used and builder uses them in rotation to avoid lock time.
const defaultParallelWriterCount = 10

// Builder builds the list of binary data from given sequence of byte arrays.
type Builder struct {
	// Map between MD5 of given string and the reference of the buffer
	tmpFolderPath        string
	referenceCache       *common.ShardingMap[*BinaryReference]
	bufferWriters        []LargeBinaryWriter
	availableWritersChan chan LargeBinaryWriter
	compressor           Compressor
	maxChunkSize         int
	parallelWriterCount  int
	lock                 sync.RWMutex
	onceWrite            sync.Once
}

func NewBuilder(compressor Compressor, tmpFolderPath string) *Builder {
	builder := &Builder{
		tmpFolderPath:       tmpFolderPath,
		maxChunkSize:        MAXIMUM_CHUNK_SIZE,
		referenceCache:      common.NewShardingMap[*BinaryReference](common.NewSuffixShardingProvider(128, 4)),
		compressor:          compressor,
		lock:                sync.RWMutex{},
		parallelWriterCount: defaultParallelWriterCount,
		onceWrite:           sync.Once{},
	}
	return builder
}

// Write amends the givenBinary in some binary chunk. If same body was given previously, it will return the reference from the cache.
func (b *Builder) Write(binaryBody []byte) (*BinaryReference, error) {
	var err error
	hash := b.calcStringHash(binaryBody)
	refCache := b.referenceCache.AcquireShard(hash)
	defer b.referenceCache.ReleaseShard(hash)
	if data, exists := refCache[hash]; exists {
		return data, nil
	}
	b.onceWrite.Do(func() {
		b.availableWritersChan = make(chan LargeBinaryWriter, b.parallelWriterCount)
		b.bufferWriters = make([]LargeBinaryWriter, 0, b.parallelWriterCount)
		// fill nil up to the count of parallel writer count for the next write calls to instanciate writers.
		for i := 0; i < b.parallelWriterCount; i++ {
			b.availableWritersChan <- nil
		}
	})
	writer := <-b.availableWritersChan

	if writer == nil {
		writer, err = b.nextNewBinaryWriter()
		if err != nil {
			return nil, err
		}
	}

	ref, err := writer.Write(binaryBody)
	if err != nil {
		if !errors.Is(err, ErrBufferChunkFilled) {
			return nil, err
		}
		err = writer.Seal()
		if err != nil {
			return nil, err
		}
		writer, err = b.nextNewBinaryWriter()
		if err != nil {
			return nil, err
		}
		ref, err = writer.Write(binaryBody)
		if err != nil {
			return nil, err
		}
	}

	refCache[hash] = ref
	b.availableWritersChan <- writer
	return ref, nil
}

func (b *Builder) nextNewBinaryWriter() (*FileSystemBinaryWriter, error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	writer, err := NewFileSystemBinaryWriter(b.tmpFolderPath, len(b.bufferWriters), b.maxChunkSize)
	if err != nil {
		return nil, err
	}
	slog.Debug("instanciated a new binary writer", "currentBinaryWriterCount", len(b.bufferWriters))
	b.bufferWriters = append(b.bufferWriters, writer)
	return writer, nil
}

func (b *Builder) Read(ref *BinaryReference) ([]byte, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	if ref.Buffer >= len(b.bufferWriters) {
		return nil, fmt.Errorf("buffer index %d is out of the range", ref.Buffer)
	}
	bw := b.bufferWriters[ref.Buffer]
	return bw.Read(ref)
}

func (b *Builder) sealAllBuffers() error {
	if b.availableWritersChan == nil {
		return nil
	}
	close(b.availableWritersChan)
	for bw := range b.availableWritersChan {
		if bw == nil {
			continue
		}
		err := bw.Seal()
		if err != nil {
			return err
		}
	}
	return nil
}

// Build amends all the binary buffers to the given writer in KHI format. Returns the written byte size.
func (b *Builder) Build(ctx context.Context, writer io.Writer, progress *inspectionmetadata.TaskProgressMetadata) (int, error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	err := b.sealAllBuffers()
	if err != nil {
		return 0, err
	}
	allBinarySize := 0

	compressedBufferCount := atomic.Int32{}
	updater := progressutil.NewProgressUpdator(progress, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
		completedCount := compressedBufferCount.Load()
		tp.Update(float32(completedCount)/float32(len(b.bufferWriters)), fmt.Sprintf("Compressing binary part... %d of %d", completedCount, len(b.bufferWriters)))
	})
	err = updater.Start(ctx)
	if err != nil {
		return 0, err
	}

	defer updater.Done()
	compressedBuffers := make([]io.ReadCloser, len(b.bufferWriters))
	errgrp, childCtx := errgroup.WithContext(ctx)
	for i, binaryWriter := range b.bufferWriters {
		i, binaryWriter := i, binaryWriter
		errgrp.Go(func() error {
			binaryReader, err := binaryWriter.ChunkReader()
			if err != nil {
				return err
			}
			defer binaryReader.Close()
			compressedReader, err := b.compressor.Compress(childCtx, binaryReader)
			if err != nil {
				return err
			}
			compressedBuffers[i] = compressedReader
			compressedBufferCount.Add(1)
			return nil
		})
	}
	err = errgrp.Wait()
	if err != nil {
		return 0, err
	}

	for i := range b.bufferWriters {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				return 0, err
			}
		default:
			readResult, err := io.ReadAll(compressedBuffers[i])
			if err != nil {
				return 0, err
			}
			sizeInBytesBinary := make([]byte, 4)
			binary.BigEndian.PutUint32(sizeInBytesBinary, uint32(len(readResult)))
			if writtenSize, err := writer.Write(sizeInBytesBinary); err != nil {
				return 0, err
			} else {
				allBinarySize += writtenSize
			}
			if writtenSize, err := writer.Write(readResult); err != nil {
				return 0, err
			} else {
				allBinarySize += writtenSize
			}
		}
	}
	return allBinarySize, nil
}

func (b *Builder) calcStringHash(source []byte) string {
	return fmt.Sprintf("%x", md5.Sum(source))
}
