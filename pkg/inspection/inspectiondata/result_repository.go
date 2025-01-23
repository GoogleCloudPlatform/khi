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
	"fmt"
	"io"
	"os"
	"sync"
)

// Store persists and read the inspection result.
// The result data format is determined by Serializer, thus Store just regard them as []byte.
type Store interface {
	GetWriter() (io.Writer, error)
	GetReader() (io.Reader, error)
	GetRangeReader(start, maxLength int64) (io.Reader, error)
	Close() error
	GetInspectionResultSizeInBytes() (int, error)
}

// FileSystemStore is one of implementation of Store.
// It persist task result as a file in the data folder and read it from there.
type FileSystemStore struct {
	filePath string
	lock     sync.Mutex
	file     *os.File
}

var _ Store = (*FileSystemStore)(nil)

func NewFileSystemInspectionResultRepository(filePath string) *FileSystemStore {
	return &FileSystemStore{
		filePath: filePath,
		lock:     sync.Mutex{},
	}
}

func (r *FileSystemStore) GetWriter() (io.Writer, error) {
	r.lock.Lock()
	file, err := os.Create(r.filePath)
	if err != nil {
		return nil, err
	}
	r.file = file
	return r.file, nil
}

func (r *FileSystemStore) GetReader() (io.Reader, error) {
	r.lock.Lock()
	file, err := os.Open(r.filePath)
	if err != nil {
		return nil, err
	}
	r.file = file
	return r.file, nil
}

// GetRangeReader returns a reader only reading specified range.
func (r *FileSystemStore) GetRangeReader(start int64, maxLength int64) (io.Reader, error) {
	r.lock.Lock()
	file, err := os.Open(r.filePath)
	if err != nil {
		return nil, err
	}
	r.file = file
	return io.NewSectionReader(file, start, maxLength), nil
}

func (r *FileSystemStore) Close() error {
	if r.file != nil {
		err := r.file.Close()
		r.lock.Unlock()
		return err
	}
	return fmt.Errorf("no file open yet")
}

func (r *FileSystemStore) GetInspectionResultSizeInBytes() (int, error) {
	stat, err := os.Stat(r.filePath)
	if err != nil {
		return 0, err
	}
	return int(stat.Size()), nil
}
