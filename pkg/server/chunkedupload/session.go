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

// ChunkSession represents an in-progress chunked upload session.
type ChunkSession struct {
	Token          string
	FileName       string
	TotalSize      int64
	ReceivedBytes  int64
	ReceivedRanges []ByteRange
	TempFilePath   string
	TempFile       *os.File
	ExpiresAt      time.Time
	mu             sync.Mutex
}
