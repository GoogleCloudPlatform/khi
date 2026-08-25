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
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/idgenerator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/ttlcleaner"
)

// ChunkSessionOption is a functional option for configuring ChunkSessionManager.
type ChunkSessionOption func(*ChunkSessionManager)

// WithSessionTTL configures the session expiration TTL.
func WithSessionTTL(ttl time.Duration) ChunkSessionOption {
	return func(m *ChunkSessionManager) {
		m.ttl = ttl
	}
}

// WithMaxChunkSize configures the maximum permitted single chunk size in bytes.
func WithMaxChunkSize(maxChunkSize int64) ChunkSessionOption {
	return func(m *ChunkSessionManager) {
		m.maxChunkSize = maxChunkSize
	}
}

// WithSuggestedChunkSize configures the recommended chunk size in bytes.
func WithSuggestedChunkSize(suggestedChunkSize int64) ChunkSessionOption {
	return func(m *ChunkSessionManager) {
		m.suggestedChunkSize = suggestedChunkSize
	}
}

// WithTokenGenerator configures the token generator for session tokens.
func WithTokenGenerator(generator idgenerator.IDGenerator) ChunkSessionOption {
	return func(m *ChunkSessionManager) {
		m.tokenGenerator = generator
	}
}

// ChunkSessionManager manages chunked upload sessions and temporary storage.
type ChunkSessionManager struct {
	uploadDir          string
	tokenGenerator     idgenerator.IDGenerator
	sessions           map[string]*ChunkSession
	ttl                time.Duration
	maxChunkSize       int64
	suggestedChunkSize int64
	cleaner            *ttlcleaner.TTLCleaner[string]
	mu                 sync.RWMutex
}

var _ ttlcleaner.ExpirableTarget[string] = (*ChunkSessionManager)(nil)

// NewChunkSessionManager creates a new ChunkSessionManager instance.
func NewChunkSessionManager(uploadDir string, opts ...ChunkSessionOption) *ChunkSessionManager {
	if uploadDir == "" {
		uploadDir = os.TempDir()
	}
	m := &ChunkSessionManager{
		uploadDir:          uploadDir,
		tokenGenerator:     idgenerator.NewPrefixIDGenerator("chunk-upload-"),
		sessions:           make(map[string]*ChunkSession),
		ttl:                DefaultSessionTTL,
		maxChunkSize:       DefaultMaxChunkSize,
		suggestedChunkSize: DefaultSuggestedChunkSize,
	}
	for _, opt := range opts {
		opt(m)
	}
	m.cleaner = ttlcleaner.NewTTLCleaner[string](m, DefaultCleanupInterval)
	m.cleaner.Start()
	return m
}

// Close stops the background cleaner and cleans up active sessions.
func (m *ChunkSessionManager) Close() {
	if m.cleaner != nil {
		m.cleaner.Stop()
	}
}

// SuggestedChunkSize returns the recommended chunk size in bytes.
func (m *ChunkSessionManager) SuggestedChunkSize() int64 {
	return m.suggestedChunkSize
}

// StartSession initializes a new chunk upload session and creates a temporary file to store chunks.
func (m *ChunkSessionManager) StartSession(fileName string, totalSize int64) (*ChunkSession, error) {
	if totalSize <= 0 {
		return nil, ErrInvalidTotalSize
	}

	if err := os.MkdirAll(m.uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	tempFile, err := os.CreateTemp(m.uploadDir, "khi-chunk-*.part")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary chunk file: %w", err)
	}

	token := m.tokenGenerator.Generate()
	session := &ChunkSession{
		Token:          token,
		FileName:       fileName,
		TotalSize:      totalSize,
		ReceivedBytes:  0,
		ReceivedRanges: make([]ByteRange, 0),
		TempFilePath:   tempFile.Name(),
		TempFile:       tempFile,
		ExpiresAt:      time.Now().Add(m.ttl),
	}

	m.mu.Lock()
	m.sessions[token] = session
	m.mu.Unlock()

	return session, nil
}

// WriteChunk writes a chunk of data at the specified byte offset to the session's temporary file.
func (m *ChunkSessionManager) WriteChunk(token string, offset int64, data []byte) (int64, error) {
	m.mu.RLock()
	session, exists := m.sessions[token]
	m.mu.RUnlock()

	if !exists {
		return 0, ErrSessionNotFound
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if int64(len(data)) > m.maxChunkSize {
		return 0, ErrChunkSizeTooLarge
	}

	if offset < 0 {
		return 0, fmt.Errorf("%w: offset must be non-negative", ErrInvalidOffset)
	}

	if len(data) == 0 {
		return 0, ErrEmptyChunkData
	}

	if offset > session.TotalSize-int64(len(data)) {
		return 0, fmt.Errorf("%w: chunk exceeds total size %d", ErrInvalidOffset, session.TotalSize)
	}

	n, err := session.TempFile.WriteAt(data, offset)
	if err != nil {
		return 0, fmt.Errorf("failed to write chunk to temporary file: %w", err)
	}
	session.ReceivedRanges = append(session.ReceivedRanges, ByteRange{
		Start: offset,
		End:   offset + int64(n),
	})
	session.ReceivedBytes += int64(n)
	session.ExpiresAt = time.Now().Add(m.ttl)

	return session.ReceivedBytes, nil
}

// FinalizeSession validates chunk completeness, closes the temporary file, moves it to destinationPath (if provided),
// and removes the session from the manager. If destinationPath is empty, it closes the file and returns the temp file path.
func (m *ChunkSessionManager) FinalizeSession(token string, destinationPath string) (string, error) {
	m.mu.Lock()
	session, exists := m.sessions[token]
	if exists {
		delete(m.sessions, token)
	}
	m.mu.Unlock()

	if !exists {
		return "", ErrSessionNotFound
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if err := ValidateReceivedRanges(session.ReceivedRanges, session.TotalSize); err != nil {
		session.TempFile.Close()
		os.Remove(session.TempFilePath)
		return "", err
	}

	if err := session.TempFile.Close(); err != nil {
		os.Remove(session.TempFilePath)
		return "", fmt.Errorf("failed to close temporary upload file: %w", err)
	}

	if destinationPath != "" {
		if err := os.MkdirAll(filepath.Dir(destinationPath), 0755); err != nil {
			os.Remove(session.TempFilePath)
			return "", fmt.Errorf("failed to create destination directory: %w", err)
		}
		if err := os.Rename(session.TempFilePath, destinationPath); err != nil {
			os.Remove(session.TempFilePath)
			return "", fmt.Errorf("failed to persist uploaded file: %w", err)
		}
		return destinationPath, nil
	}

	return session.TempFilePath, nil
}

// AbortSession closes and deletes the temporary file and removes the session from the manager.
func (m *ChunkSessionManager) AbortSession(token string) error {
	m.mu.Lock()
	session, exists := m.sessions[token]
	if exists {
		delete(m.sessions, token)
	}
	m.mu.Unlock()

	if !exists {
		return ErrSessionNotFound
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.TempFile.Close()
	os.Remove(session.TempFilePath)
	return nil
}

// Expirations implements ttlcleaner.ExpirableTarget.
func (m *ChunkSessionManager) Expirations() map[string]time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	expirations := make(map[string]time.Time, len(m.sessions))
	for token, s := range m.sessions {
		s.mu.Lock()
		expirations[token] = s.ExpiresAt
		s.mu.Unlock()
	}
	return expirations
}

// Evict implements ttlcleaner.ExpirableTarget.
func (m *ChunkSessionManager) Evict(token string) error {
	return m.AbortSession(token)
}
