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
	"fmt"
	"io"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
)

const MAXIMUM_CHUNK_SIZE = 1024 * 1024 * 500

// Builder builds the list of binary data from given sequence of byte arrays.
type Builder struct {
	// Map between MD5 of given string and the reference of the buffer
	tmpFolderPath  string
	referenceCache *common.ShardingMap[*BinaryReference]
	bufferWriters  []LargeBinaryWriter
	compressor     Compressor
	maxChunkSize   int
	lock           sync.Mutex
}

func NewBuilder(compressor Compressor, tmpFolderPath string) *Builder {
	return &Builder{
		tmpFolderPath:  tmpFolderPath,
		maxChunkSize:   MAXIMUM_CHUNK_SIZE,
		referenceCache: common.NewShardingMap[*BinaryReference](common.NewSuffixShardingProvider(128, 4)),
		compressor:     compressor,
		bufferWriters:  make([]LargeBinaryWriter, 0),
		lock:           sync.Mutex{},
	}
}

// Write amends the givenBinary in some binary chunk. If same body was given previously, it will return the reference from the cache.
func (b *Builder) Write(binaryBody []byte) (*BinaryReference, error) {
	hash := b.calcStringHash(binaryBody)
	refCache := b.referenceCache.AcquireShard(hash)
	defer b.referenceCache.ReleaseShard(hash)
	if data, exists := refCache[hash]; exists {
		return data, nil
	}
	b.lock.Lock()
	targetIndex := len(b.bufferWriters)
	for i := 0; i < len(b.bufferWriters); i++ {
		if b.bufferWriters[i].CanWrite(len(binaryBody)) {
			targetIndex = i
			break
		}
	}
	if len(b.bufferWriters) <= targetIndex {
		// Due to the ArrayBuffer of Javascript limitation, each chunk must be smaller than 1GB.
		writer, err := NewFileSystemBinaryWriter(b.tmpFolderPath, len(b.bufferWriters), b.maxChunkSize)
		if err != nil {
			b.lock.Unlock()
			return nil, err
		}
		b.bufferWriters = append(b.bufferWriters, writer)
	}

	resultReference, err := b.bufferWriters[targetIndex].Write(binaryBody)
	if err != nil {
		return nil, err
	}
	b.lock.Unlock()

	refCache[hash] = resultReference
	return resultReference, nil
}

func (b *Builder) Read(ref *BinaryReference) ([]byte, error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	if ref.Buffer >= len(b.bufferWriters) {
		return nil, fmt.Errorf("buffer index %d is out of the range", ref.Buffer)
	}
	bw := b.bufferWriters[ref.Buffer]
	return bw.Read(ref)
}

// Build amends all the binary buffers to the given writer in KHI format. Returns the written byte size.
func (b *Builder) Build(ctx context.Context, writer io.Writer, progress *progress.TaskProgress) (int, error) {
	allBinarySize := 0
	b.lock.Lock()
	defer b.lock.Unlock()
	for i, binaryWriter := range b.bufferWriters {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				binaryWriter.Dispose()
				b.compressor.Dispose()
				return 0, err
			}
		default:
			progress.Update(float32(i)/float32(len(b.bufferWriters)), fmt.Sprintf("Compressing binary part... %d of %d", i, len(b.bufferWriters)))
			binaryReader, err := binaryWriter.GetBinary()
			if err != nil {
				return 0, err
			}
			compressedReader, err := b.compressor.CompressAll(ctx, binaryReader)
			if err != nil {
				return 0, err
			}
			readResult, err := io.ReadAll(compressedReader)
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
			binaryWriter.Dispose()
		}
	}
	b.compressor.Dispose()
	return allBinarySize, nil
}

func (b *Builder) calcStringHash(source []byte) string {
	return fmt.Sprintf("%x", md5.Sum(source))
}
