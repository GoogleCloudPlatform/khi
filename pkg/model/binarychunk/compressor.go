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
	"fmt"
	"io"
	"log/slog"
	"os"
)

type Compressor interface {
	// CompressAll reads all bytes from given reader and returns a reader for the compressed buffer.
	CompressAll(ctx context.Context, reader io.Reader) (io.Reader, error)
	// Dispose releases all allocated resource in Compressor.
	Dispose() error
}

type FileSystemGzipCompressor struct {
	temporaryFolder string
	disposed        bool
	openedFiles     []*os.File
}

var _ Compressor = (*FileSystemGzipCompressor)(nil)

func NewFileSystemGzipCompressor(temporaryFolder string) *FileSystemGzipCompressor {
	return &FileSystemGzipCompressor{
		temporaryFolder: temporaryFolder,
		disposed:        false,
		openedFiles:     make([]*os.File, 0),
	}
}

func (c *FileSystemGzipCompressor) CompressAll(ctx context.Context, reader io.Reader) (io.Reader, error) {
	if c.disposed {
		return nil, fmt.Errorf("instance is already disposed.")
	}
	readResult, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	slog.DebugContext(ctx, fmt.Sprintf("Received folder:%s", c.temporaryFolder))
	tmpfile, err := os.CreateTemp(c.temporaryFolder, "khi-c-")
	if err != nil {
		return nil, err
	}
	slog.DebugContext(ctx, fmt.Sprintf("Created a temporary file:%s", tmpfile.Name()))
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
	c.openedFiles = append(c.openedFiles, readerFile)

	return readerFile, nil
}

func (c *FileSystemGzipCompressor) Dispose() error {
	errors := make([]error, 0)
	for _, file := range c.openedFiles {
		err := file.Close()
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("one or more files returned error during closure process,%v", errors)
	}
	return nil
}
