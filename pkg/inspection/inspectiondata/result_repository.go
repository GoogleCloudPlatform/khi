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

package inspectiondata

import (
	"io"
	"os"
)

// Store persists and read the inspection result.
// The result data format is determined by Serializer, thus Store just regard them as []byte.
type Store interface {
	GetWriter() (io.WriteCloser, error)
	GetReader() (io.ReadCloser, error)
	GetRangeReader(start, maxLength int64) (io.ReadCloser, error)
	GetInspectionResultSizeInBytes() (int, error)
}

// FileSystemStore is one of implementation of Store.
// It persist task result as a file in the data folder and read it from there.
type FileSystemStore struct {
	filePath string
}

var _ Store = (*FileSystemStore)(nil)

func NewFileSystemInspectionResultRepository(filePath string) *FileSystemStore {
	return &FileSystemStore{
		filePath: filePath,
	}
}

func (r *FileSystemStore) GetWriter() (io.WriteCloser, error) {
	file, err := os.Create(r.filePath)
	if err != nil {
		return nil, err
	}
	return file, nil
}

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

func (r *FileSystemStore) GetInspectionResultSizeInBytes() (int, error) {
	stat, err := os.Stat(r.filePath)
	if err != nil {
		return 0, err
	}
	return int(stat.Size()), nil
}
