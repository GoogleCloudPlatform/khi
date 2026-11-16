// Copyright 2026 Google LLC
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

package chunkedupload

import (
	"errors"
	"os"
	"sync"
	"time"
)

const (
	// DefaultSuggestedChunkSize is the recommended chunk payload size (25MB).
	DefaultSuggestedChunkSize = 25 * 1024 * 1024
	// DefaultMaxChunkSize is the maximum allowed single chunk payload (32MB).
	DefaultMaxChunkSize = 32 * 1024 * 1024
	// DefaultSessionTTL is the duration after which an inactive chunk upload session expires.
	DefaultSessionTTL = 30 * time.Minute
	// DefaultCleanupInterval is the frequency at which the cleaner checks for expired sessions.
	DefaultCleanupInterval = 1 * time.Minute
)

var (
	// ErrSessionNotFound is returned when a session token is not found or has expired.
	ErrSessionNotFound = errors.New("chunk upload session not found or expired")
	// ErrInvalidTotalSize is returned when the specified total file size is non-positive.
	ErrInvalidTotalSize = errors.New("total file size must be positive")
	// ErrEmptyChunkData is returned when an uploaded chunk contains empty data payload.
	ErrEmptyChunkData = errors.New("chunk data must not be empty")
	// ErrInvalidOffset is returned when the chunk byte offset is invalid.
	ErrInvalidOffset = errors.New("invalid chunk byte offset")
	// ErrChunkSizeTooLarge is returned when a single chunk exceeds the maximum permitted size limit.
	ErrChunkSizeTooLarge = errors.New("chunk size exceeds maximum allowed limit")
)

// ChunkSession represents an in-progress chunked upload session.
type ChunkSession struct {
	// Token is the unique session identifier issued for this upload.
	Token string
	// FileName is the original name of the file being uploaded.
	FileName string
	// TotalSize is the expected total file size in bytes.
	TotalSize int64
	// ReceivedBytes is the cumulative number of bytes written across all chunks.
	ReceivedBytes int64
	// ReceivedRanges tracks the byte intervals received so far for completeness validation.
	ReceivedRanges []ByteRange
	// TempFilePath is the filesystem path of the temporary file storing uploaded chunks.
	TempFilePath string
	// TempFile is the open file handle for writing incoming chunks.
	TempFile *os.File
	// ExpiresAt is the timestamp after which an inactive session is eligible for eviction.
	ExpiresAt time.Time
	mu        sync.Mutex
}
