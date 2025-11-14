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
	"compress/gzip"
	"context"
	"io"
	"os"
)

type Compressor interface {
	// Compress reads all bytes from given reader and returns a reader for the compressed buffer.
	Compress(ctx context.Context, reader io.Reader) (io.ReadCloser, error)
}

type FileSystemGzipCompressor struct {
	temporaryFolder string
}

var _ Compressor = (*FileSystemGzipCompressor)(nil)

func NewFileSystemGzipCompressor(temporaryFolder string) *FileSystemGzipCompressor {
	return &FileSystemGzipCompressor{
		temporaryFolder: temporaryFolder,
	}
}

func (c *FileSystemGzipCompressor) Compress(ctx context.Context, reader io.Reader) (io.ReadCloser, error) {
	readResult, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	tmpfile, err := os.CreateTemp(c.temporaryFolder, "khi-c-")
	if err != nil {
		return nil, err
	}
	defer tmpfile.Close()

	gzipWriter := gzip.NewWriter(tmpfile)

	_, err = gzipWriter.Write(readResult)
	if err != nil {
		return nil, err
	}

	err = gzipWriter.Flush()
	if err != nil {
		return nil, err
	}

	err = gzipWriter.Close()
	if err != nil {
		return nil, err
	}
	readerFile, err := os.Open(tmpfile.Name())
	if err != nil {
		return nil, err
	}

	return readerFile, nil
}
