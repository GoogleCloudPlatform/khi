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

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

type testCompressorWaitForSecond struct {
}

// CompressAll implements Compressor.
func (*testCompressorWaitForSecond) CompressAll(ctx context.Context, reader io.Reader) (io.Reader, error) {
	<-time.After(time.Second)
	return bytes.NewBuffer([]byte{1, 2, 3, 4}), nil
}

// Dispose implements Compressor.
func (*testCompressorWaitForSecond) Dispose() error {
	return nil
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
		b.Build(context.Background(), &bytes.Buffer{}, progress.NewTaskProgress("foo"))
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

		_, err := b.Build(context.Background(), &result, progress.NewTaskProgress("foo"))
		bufferCount := 0
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
				t.Errorf("err was not a nil:%v", err)
			}
			decompressed, err := io.ReadAll(gzipReader)

			if len(decompressed) != 1024*1024*50 {
				t.Errorf("decompressed buffer size is not matching the source buffer size: %d != %d", len(decompressed), 1024*1024*50)
			}
			bufferCount += 1
		}
		if bufferCount != 2 {
			t.Errorf("buffer count is not matching the expected count. %d", bufferCount)
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
		size, err := b.Build(ctx, &buf, progress.NewTaskProgress("foo"))
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
