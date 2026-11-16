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

package inspectioncore

import (
	"io"
	"os"
)

var _ Store = (*FileSystemStore)(nil)

// Store persists and reads the inspection result.
// The result data format is determined by Serializer, thus Store just regards them as []byte.
type Store interface {
	// GetWriter returns a writer for persisting the serialized inspection result.
	GetWriter() (io.WriteCloser, error)
	// GetReader returns a reader for reading the full serialized inspection result.
	GetReader() (io.ReadCloser, error)
	// GetRangeReader returns a reader only reading the specified byte range.
	GetRangeReader(start, maxLength int64) (io.ReadCloser, error)
	// GetInspectionResultSizeInBytes returns the total size of the stored inspection result in bytes.
	GetInspectionResultSizeInBytes() (int, error)
}

// FileSystemStore is an implementation of Store backed by a local file.
// It persists the task result as a file in the data folder and reads it from there.
type FileSystemStore struct {
	filePath string
}

// NewFileSystemInspectionResultRepository creates a FileSystemStore reading and writing at filePath.
func NewFileSystemInspectionResultRepository(filePath string) *FileSystemStore {
	return &FileSystemStore{
		filePath: filePath,
	}
}

// GetWriter creates or truncates the underlying file and returns a WriteCloser to write the inspection result.
func (r *FileSystemStore) GetWriter() (io.WriteCloser, error) {
	file, err := os.Create(r.filePath)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// GetReader opens the underlying file and returns a ReadCloser to read the inspection result.
func (r *FileSystemStore) GetReader() (io.ReadCloser, error) {
	file, err := os.Open(r.filePath)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// GetRangeReader returns a reader only reading specified range.
func (r *FileSystemStore) GetRangeReader(start int64, maxLength int64) (io.ReadCloser, error) {
	f, err := os.Open(r.filePath)
	if err != nil {
		return nil, err
	}
	reader := io.NewSectionReader(f, start, maxLength)
	// io.NewSectionReader doesn't implement Close, we want close `file`.
	return struct {
		io.Reader
		io.Closer
	}{
		reader,
		f,
	}, nil
}

// GetInspectionResultSizeInBytes returns the size of the inspection result file in bytes.
func (r *FileSystemStore) GetInspectionResultSizeInBytes() (int, error) {
	stat, err := os.Stat(r.filePath)
	if err != nil {
		return 0, err
	}
	return int(stat.Size()), nil
}
