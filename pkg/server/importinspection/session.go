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

package importinspection

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/idgenerator"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var (
	// ErrSessionNotFound is returned when an import session token is not found or has expired.
	ErrSessionNotFound = errors.New("import session not found or expired")
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
	// DefaultSuggestedChunkSize is the recommended chunk payload size (10MB).
	DefaultSuggestedChunkSize = 25 * 1024 * 1024
	// DefaultMaxChunkSize is the maximum allowed single chunk payload (32MB).
	DefaultMaxChunkSize = 32 * 1024 * 1024
	// DefaultSessionTTL is the duration after which an inactive import session expires.
	DefaultSessionTTL = 30 * time.Minute
)

// ByteRange represents an uploaded byte interval [Start, End).
type ByteRange struct {
	Start int64
	End   int64
}

// ImportSession represents an in-progress chunked upload session.
type ImportSession struct {
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

// FinalizedImport represents the result of a completed and validated inspection import.
type FinalizedImport struct {
	InspectionID   string
	InspectionName string
	FileSize       int64
}

// ImportSessionManager manages the lifecycle of chunked upload sessions and registers imported inspections.
type ImportSessionManager struct {
	inspectionServer   *coreinspection.InspectionTaskServer
	ioConfig           *inspectioncore_contract.IOConfig
	tokenGenerator     idgenerator.IDGenerator
	idGenerator        idgenerator.IDGenerator
	sessions           map[string]*ImportSession
	ttl                time.Duration
	maxChunkSize       int64
	suggestedChunkSize int64
	cleaner            *ImportSessionCleaner
	mu                 sync.RWMutex
}

// NewImportSessionManager creates a new ImportSessionManager instance.
func NewImportSessionManager(server *coreinspection.InspectionTaskServer, ioConfig *inspectioncore_contract.IOConfig) *ImportSessionManager {
	m := &ImportSessionManager{
		inspectionServer:   server,
		ioConfig:           ioConfig,
		tokenGenerator:     idgenerator.NewPrefixIDGenerator("import-"),
		idGenerator:        idgenerator.NewPrefixIDGenerator("inspection-imported-"),
		sessions:           make(map[string]*ImportSession),
		ttl:                DefaultSessionTTL,
		maxChunkSize:       DefaultMaxChunkSize,
		suggestedChunkSize: DefaultSuggestedChunkSize,
	}
	m.cleaner = NewImportSessionCleaner(m, DefaultCleanupInterval)
	m.cleaner.Start()
	return m
}

// Close stops the background session cleaner and releases resources.
func (m *ImportSessionManager) Close() {
	if m.cleaner != nil {
		m.cleaner.Stop()
	}
}

// GetActiveSessions returns a snapshot list of all currently registered import sessions.
func (m *ImportSessionManager) GetActiveSessions() []*ImportSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*ImportSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// SuggestedChunkSize returns the recommended chunk size in bytes.
func (m *ImportSessionManager) SuggestedChunkSize() int64 {
	return m.suggestedChunkSize
}

// StartSession initializes a new upload session and creates a temporary file to store incoming chunks.
func (m *ImportSessionManager) StartSession(fileName string, totalSize int64) (*ImportSession, error) {
	if totalSize <= 0 {
		return nil, ErrInvalidTotalSize
	}

	uploadDir := os.TempDir()
	if m.ioConfig != nil && m.ioConfig.DataDestination != "" {
		uploadDir = m.ioConfig.DataDestination
	}

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	tempFile, err := os.CreateTemp(uploadDir, "khi-import-*.part")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary upload file: %w", err)
	}

	token := m.tokenGenerator.Generate()
	session := &ImportSession{
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
func (m *ImportSessionManager) WriteChunk(token string, offset int64, data []byte) (int64, error) {
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

	if offset+int64(len(data)) > session.TotalSize {
		return 0, fmt.Errorf("%w: chunk end %d exceeds total size %d", ErrInvalidOffset, offset+int64(len(data)), session.TotalSize)
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

// CompleteSession finalizes the session, validates the uploaded .khi file, moves it to the permanent destination,
// and registers the imported inspection in the InspectionTaskServer.
func (m *ImportSessionManager) CompleteSession(token string) (*FinalizedImport, error) {
	m.mu.Lock()
	session, exists := m.sessions[token]
	if exists {
		delete(m.sessions, token)
	}
	m.mu.Unlock()

	if !exists {
		return nil, ErrSessionNotFound
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Verify that the file upload is complete without missing byte ranges or overlaps.
	if err := validateReceivedRanges(session.ReceivedRanges, session.TotalSize); err != nil {
		session.TempFile.Close()
		os.Remove(session.TempFilePath)
		return nil, err
	}

	// Close temporary file handle before reading and moving it.
	if err := session.TempFile.Close(); err != nil {
		os.Remove(session.TempFilePath)
		return nil, fmt.Errorf("failed to close temporary upload file: %w", err)
	}

	// Validate KHI file structure and extract metadata.
	header, metadataMap, err := ValidateAndExtractMetadata(session.TempFilePath)
	if err != nil {
		os.Remove(session.TempFilePath)
		return nil, fmt.Errorf("failed to validate KHI file: %w", err)
	}

	inspectionID := m.idGenerator.Generate()
	destinationDir := filepath.Dir(session.TempFilePath)
	destinationPath := filepath.Join(destinationDir, inspectionID+".khi")

	// Move file from temporary location to destination path.
	if err := os.Rename(session.TempFilePath, destinationPath); err != nil {
		os.Remove(session.TempFilePath)
		return nil, fmt.Errorf("failed to persist inspection file: %w", err)
	}

	store := inspectioncore_contract.NewFileSystemInspectionResultRepository(destinationPath)
	fileSize, err := store.GetInspectionResultSizeInBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain final inspection file size: %w", err)
	}
	header.FileSize = fileSize

	// Register in InspectionTaskServer.
	m.inspectionServer.RegisterImportedInspection(inspectionID, store, metadataMap)

	return &FinalizedImport{
		InspectionID:   inspectionID,
		InspectionName: header.InspectionName,
		FileSize:       int64(fileSize),
	}, nil
}

// AbortSession closes and deletes the temporary upload file and removes the session.
func (m *ImportSessionManager) AbortSession(token string) error {
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

// validateReceivedRanges validates that the given byte ranges completely cover [0, expectedTotalSize)
// without gaps or overlaps after sorting by start offset.
func validateReceivedRanges(ranges []ByteRange, expectedTotalSize int64) error {
	if len(ranges) == 0 {
		return fmt.Errorf("incomplete upload: no data received")
	}

	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Start < ranges[j].Start
	})

	if ranges[0].Start != 0 {
		return fmt.Errorf("incomplete upload: first chunk starts at offset %d, expected 0", ranges[0].Start)
	}

	for i := 0; i < len(ranges)-1; i++ {
		if ranges[i].End != ranges[i+1].Start {
			return fmt.Errorf("incomplete or overlapping upload: chunk %d ends at %d but next chunk starts at %d",
				i, ranges[i].End, ranges[i+1].Start)
		}
	}

	lastEnd := ranges[len(ranges)-1].End
	if lastEnd != expectedTotalSize {
		return fmt.Errorf("incomplete upload: received up to byte %d, expected %d", lastEnd, expectedTotalSize)
	}

	return nil
}
