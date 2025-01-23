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

package rawlogs

import (
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"
)

// FilesystemRepository stores retrieved raw log data with timestamp.
// This can sort data won't fit the entire log data on memory.
type FilesystemRepository struct {
	writeLock        *sync.Mutex
	file             *os.File
	exported         bool
	workspaceEntries []workspaceEntry
	lastOffset       uint64
}

type fileSystemRepositoryIterator struct {
	ownerRepository *FilesystemRepository
	entries         []workspaceEntry
	currentIndex    int
}

type workspaceEntry struct {
	TimestampInNanos uint64
	Offset           uint64
	Size             int
}

var _ Repository = (*FilesystemRepository)(nil)
var _ LogIterator = (*fileSystemRepositoryIterator)(nil)

func NewFilesystemRepository() (*FilesystemRepository, error) {
	tmpFile, err := os.CreateTemp("/tmp", "khi-rawlog-")
	if err != nil {
		return nil, err
	}
	return &FilesystemRepository{
		writeLock:        &sync.Mutex{},
		file:             tmpFile,
		exported:         false,
		workspaceEntries: make([]workspaceEntry, 0),
		lastOffset:       0,
	}, nil
}

func (r *FilesystemRepository) Write(timestamp time.Time, data []byte) error {
	if r.exported {
		return fmt.Errorf("unsupported operation. data is already sorted and exported")
	}
	r.writeLock.Lock()
	defer r.writeLock.Unlock()

	err := r.writeToWorkspace(timestamp, data)
	if err != nil {
		return err
	}

	return nil
}

func (r *FilesystemRepository) IterateInSortedOrder() LogIterator {
	r.exported = true
	sort.Slice(r.workspaceEntries, func(i, j int) bool {
		return r.workspaceEntries[i].TimestampInNanos < r.workspaceEntries[j].TimestampInNanos
	})
	return &fileSystemRepositoryIterator{
		ownerRepository: r,
		entries:         r.workspaceEntries,
		currentIndex:    0,
	}
}

func (r *FilesystemRepository) writeToWorkspace(timestamp time.Time, data []byte) error {
	_, err := r.file.Seek(0, 2) // go to end of the workspace file
	if err != nil {
		return err
	}
	_, err = r.file.Write(data)
	if err != nil {
		return err
	}
	r.workspaceEntries = append(r.workspaceEntries, workspaceEntry{
		TimestampInNanos: uint64(timestamp.UnixNano()),
		Offset:           r.lastOffset,
		Size:             len(data),
	})
	r.lastOffset += uint64(len(data))
	return nil
}

func (r *FilesystemRepository) Dispose() error {
	return os.Remove(r.file.Name())
}

// Implementations for fileSystemRepositoryIterator

func (i *fileSystemRepositoryIterator) HasNext() bool {
	return i.currentIndex < len(i.entries)
}

func (i *fileSystemRepositoryIterator) Next() ([]byte, error) {
	if !i.HasNext() {
		return nil, io.EOF
	}
	result := make([]byte, i.entries[i.currentIndex].Size)
	if _, err := i.ownerRepository.file.ReadAt(result, int64(i.entries[i.currentIndex].Offset)); err != nil {
		return nil, err
	}
	i.currentIndex += 1
	return result, nil
}

func (i *fileSystemRepositoryIterator) Reset() {
	i.currentIndex = 0
}
