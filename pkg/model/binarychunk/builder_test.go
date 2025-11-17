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
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
)

type testCompressorWaitForSecond struct {
}

// Compress implements Compressor.
func (*testCompressorWaitForSecond) Compress(ctx context.Context, reader io.Reader) (io.ReadCloser, error) {
	<-time.After(time.Second)
	return io.NopCloser(bytes.NewBuffer([]byte{1, 2, 3, 4})), nil
}

var _ Compressor = (*testCompressorWaitForSecond)(nil)

func TestBuilder(t *testing.T) {
	t.Run("caches given string and must returns the same reference for the same input", func(t *testing.T) {
		b := NewBuilder(NewFileSystemGzipCompressor("/tmp"), "/tmp")
		_, err := b.Write([]byte("input1"))
		if err != nil {
			t.Errorf("err was not a nil:%v", err)
		}

		result1, err := b.Write([]byte("foo bar qux quux"))
		if err != nil {
			t.Errorf("err was not a nil:%v", err)
		}
		result2, err := b.Write([]byte("foo bar qux quux"))
		if err != nil {
			t.Errorf("err was not a nil:%v", err)
		}

		if diff := cmp.Diff(result1, result2); diff != "" {
			t.Errorf("Generated BinaryReferences are not identical.")
		}
		// Just for clearning up
		b.Build(context.Background(), &bytes.Buffer{}, inspectionmetadata.NewTaskProgressMetadata("foo"))
	})

	t.Run("generates binary chunks within the chunk max size and wrote as a single buffer with sizes", func(t *testing.T) {
		b := NewBuilder(NewFileSystemGzipCompressor("/tmp"), "/tmp")
		// Forcibly override the chunk size to reduce test time
		b.maxChunkSize = 1024 * 1024 * 50
		randBuf := make([]byte, 1024*1024*25)
		sizeReadBuffer := make([]byte, 4)
		for i := 0; i < 4; i++ {
			rand.Read(randBuf)
			b.Write(randBuf)
		}
		var result bytes.Buffer

		_, err := b.Build(context.Background(), &result, inspectionmetadata.NewTaskProgressMetadata("foo"))
		if err != nil {
			t.Errorf("err was not nil:%v", err)
		}
		bufferCount := 0
		gotDecompressedSizeInBytes := 0
		for {
			_, err = result.Read(sizeReadBuffer)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Errorf("err was not a nil:%v", err)
			}

			size := binary.BigEndian.Uint32(sizeReadBuffer)
			compressedBuffer := make([]byte, size)
			_, err = result.Read(compressedBuffer)
			if err != nil {
				t.Errorf("err was not a nil:%v", err)
			}

			gzipReader, err := gzip.NewReader(bytes.NewBuffer(compressedBuffer))
			if err != nil {
				t.Errorf("err was not nil:%v", err)
			}
			decompressed, err := io.ReadAll(gzipReader)
			if err != nil {
				t.Errorf("err was not nil:%v", err)
			}
			gotDecompressedSizeInBytes += len(decompressed)
			bufferCount += 1
		}
		if bufferCount != 4 {
			t.Errorf("buffer count is not matching the expected count. %d", bufferCount)
		}
		if gotDecompressedSizeInBytes != len(randBuf)*4 {
			t.Errorf("decompressed size is not matching the expected size. got:%d, want:%d", gotDecompressedSizeInBytes, len(randBuf)*4)
		}
	})
	t.Run("binarychunk.Build method must be cancellable", func(t *testing.T) {
		b := NewBuilder(&testCompressorWaitForSecond{}, "/tmp")
		b.maxChunkSize = 4
		_, err := b.Write([]byte{1, 2, 3, 4})
		if err != nil {
			t.Errorf("unexpected error\n%v", err)
		}
		_, err = b.Write([]byte{1, 2, 3, 5})
		if err != nil {
			t.Errorf("unexpected error\n%v", err)
		}
		var buf bytes.Buffer
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-time.After(time.Millisecond * 100)
			cancel()
		}()
		size, err := b.Build(ctx, &buf, inspectionmetadata.NewTaskProgressMetadata("foo"))
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Build didn't returned the Canceled error after the cancel")
		}
		if size != 0 {
			t.Errorf("b.Build() returns size=%d,want %d", size, 0)
		}
	})

	t.Run("builder should be thread safe", func(t *testing.T) {
		THREAD_COUNT := 50
		WRITE_COUNT := 10000
		builder := NewBuilder(NewFileSystemGzipCompressor("/tmp"), "/tmp")
		builder.maxChunkSize = 1024 * 1024 * 10
		wg := sync.WaitGroup{}
		for tc := 0; tc < THREAD_COUNT; tc++ {
			wg.Add(1)
			go func(tc int) {
				for c := 0; c <= WRITE_COUNT; c++ {
					data := []byte{}
					for b := 0; b < 1024*30; b++ {
						data = append(data, (byte)((b*tc*c)%256))
					}
					_, err := builder.Write(data)
					if err != nil {
						t.Errorf("unexpected error: %s", err.Error())
					}
				}
				wg.Done()
			}(tc)
		}
		wg.Wait()
	})
}

func TestBuilder_Build(t *testing.T) {
	testCases := []struct {
		desc                string
		parallelWriterCount int
		writeItemeSize      int
		maxChunkSize        int
		writeCount          int
		wantBufferCount     int
	}{
		{
			desc:                "non parallel simple write",
			parallelWriterCount: 1,
			writeItemeSize:      1024,
			maxChunkSize:        1024 * 10,
			writeCount:          5,
			wantBufferCount:     1,
		},
		{
			desc:                "parallel simple write not reaching to the parallel limit",
			parallelWriterCount: 10,
			writeItemeSize:      1024,
			maxChunkSize:        1024 * 10,
			writeCount:          5,
			wantBufferCount:     5,
		},
		{
			desc:                "parallel simple write reaching to the parallel limit",
			parallelWriterCount: 5,
			writeItemeSize:      1024,
			maxChunkSize:        1024 * 10,
			writeCount:          10,
			wantBufferCount:     5,
		},
		{
			desc:                "parallel simple write reaching to the buffer limit",
			parallelWriterCount: 5,
			writeItemeSize:      1024,
			maxChunkSize:        1024 * 2,
			writeCount:          15,
			wantBufferCount:     10,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			writeData := make([][]byte, tc.writeCount)
			sizeReadBuf := make([]byte, 4)
			refs := []*BinaryReference{}
			for i := 0; i < tc.writeCount; i++ {
				writeData[i] = make([]byte, tc.writeItemeSize)
				writeData[i][0] = byte(i)
			}
			b := NewBuilder(NewFileSystemGzipCompressor("/tmp"), "/tmp")
			b.maxChunkSize = tc.maxChunkSize
			b.parallelWriterCount = tc.parallelWriterCount
			for i := 0; i < tc.writeCount; i++ {
				ref, err := b.Write(writeData[i])
				if err != nil {
					t.Fatalf("failed to write to buffer: %v", err)
				}
				refs = append(refs, ref)
			}

			var generatedBuffer bytes.Buffer
			_, err := b.Build(t.Context(), &generatedBuffer, inspectionmetadata.NewTaskProgressMetadata("foo"))
			if err != nil {
				t.Fatalf("failed to build result: %v", err)
			}
			buffers := [][]byte{}
			for {
				_, err = generatedBuffer.Read(sizeReadBuf)
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					t.Errorf("err was not a nil:%v", err)
				}

				size := binary.BigEndian.Uint32(sizeReadBuf)
				compressedBuffer := make([]byte, size)
				_, err = generatedBuffer.Read(compressedBuffer)
				if err != nil {
					t.Errorf("err was not a nil:%v", err)
				}

				gzipReader, err := gzip.NewReader(bytes.NewBuffer(compressedBuffer))
				if err != nil {
					t.Errorf("err was not nil:%v", err)
				}
				decompressed, err := io.ReadAll(gzipReader)
				if err != nil {
					t.Errorf("err was not nil:%v", err)
				}
				buffers = append(buffers, decompressed)
			}

			for i, ref := range refs {
				gotBuffer := buffers[ref.Buffer][ref.Offset : ref.Offset+ref.Length]
				if diff := cmp.Diff(writeData[i], gotBuffer); diff != "" {
					t.Errorf("read data mismatch for ref %+v (-want +got):\n%s", ref, diff)
				}
			}
			if len(buffers) != tc.wantBufferCount {
				t.Errorf("buffer count mismatch: got %d, want %d", len(buffers), tc.wantBufferCount)
			}
		})
	}
}
