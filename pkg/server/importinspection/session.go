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
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/khi/pkg/common/idgenerator"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/server/chunkedupload"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var (
	// ErrSessionNotFound is returned when an import session token is not found or has expired.
	ErrSessionNotFound = chunkedupload.ErrSessionNotFound
	// ErrInvalidTotalSize is returned when the specified total file size is non-positive.
	ErrInvalidTotalSize = chunkedupload.ErrInvalidTotalSize
	// ErrEmptyChunkData is returned when an uploaded chunk contains empty data payload.
	ErrEmptyChunkData = chunkedupload.ErrEmptyChunkData
	// ErrInvalidOffset is returned when the chunk byte offset is invalid.
	ErrInvalidOffset = chunkedupload.ErrInvalidOffset
	// ErrChunkSizeTooLarge is returned when a single chunk exceeds the maximum permitted size limit.
	ErrChunkSizeTooLarge = chunkedupload.ErrChunkSizeTooLarge
)

const (
	// DefaultSuggestedChunkSize is the recommended chunk payload size (25MB).
	DefaultSuggestedChunkSize = chunkedupload.DefaultSuggestedChunkSize
	// DefaultMaxChunkSize is the maximum allowed single chunk payload (32MB).
	DefaultMaxChunkSize = chunkedupload.DefaultMaxChunkSize
	// DefaultSessionTTL is the duration after which an inactive import session expires.
	DefaultSessionTTL = chunkedupload.DefaultSessionTTL
)

// ByteRange represents an uploaded byte interval [Start, End).
type ByteRange = chunkedupload.ByteRange

// ImportSession represents an in-progress chunked upload session.
type ImportSession = chunkedupload.ChunkSession

// FinalizedImport represents the result of a completed and validated inspection import.
type FinalizedImport struct {
	InspectionID   string
	InspectionName string
	FileSize       int64
}

// ImportSessionManager manages the lifecycle of chunked upload sessions and registers imported inspections.
type ImportSessionManager struct {
	inspectionServer *coreinspection.InspectionTaskServer
	idGenerator      idgenerator.IDGenerator
	chunkManager     *chunkedupload.ChunkSessionManager
}

// NewImportSessionManager creates a new ImportSessionManager instance.
func NewImportSessionManager(server *coreinspection.InspectionTaskServer, ioConfig *inspectioncore_contract.IOConfig) *ImportSessionManager {
	uploadDir := os.TempDir()
	if ioConfig != nil && ioConfig.DataDestination != "" {
		uploadDir = ioConfig.DataDestination
	}

	chunkManager := chunkedupload.NewChunkSessionManager(
		uploadDir,
		chunkedupload.WithTokenGenerator(idgenerator.NewPrefixIDGenerator("import-")),
		chunkedupload.WithSessionTTL(DefaultSessionTTL),
		chunkedupload.WithMaxChunkSize(DefaultMaxChunkSize),
		chunkedupload.WithSuggestedChunkSize(DefaultSuggestedChunkSize),
	)

	return &ImportSessionManager{
		inspectionServer: server,
		idGenerator:      idgenerator.NewPrefixIDGenerator("inspection-imported-"),
		chunkManager:     chunkManager,
	}
}

// Close stops the background session cleaner and releases resources.
func (m *ImportSessionManager) Close() {
	if m.chunkManager != nil {
		m.chunkManager.Close()
	}
}

// SuggestedChunkSize returns the recommended chunk size in bytes.
func (m *ImportSessionManager) SuggestedChunkSize() int64 {
	return m.chunkManager.SuggestedChunkSize()
}

// StartSession initializes a new upload session and creates a temporary file to store incoming chunks.
func (m *ImportSessionManager) StartSession(fileName string, totalSize int64) (*ImportSession, error) {
	return m.chunkManager.StartSession(fileName, totalSize)
}

// WriteChunk writes a chunk of data at the specified byte offset to the session's temporary file.
func (m *ImportSessionManager) WriteChunk(token string, offset int64, data []byte) (int64, error) {
	return m.chunkManager.WriteChunk(token, offset, data)
}

// CompleteSession finalizes the session, validates the uploaded .khi file, moves it to the permanent destination,
// and registers the imported inspection in the InspectionTaskServer.
func (m *ImportSessionManager) CompleteSession(token string) (*FinalizedImport, error) {
	inspectionID := m.idGenerator.Generate()

	// Finalize chunks without moving yet to validate metadata
	tempFilePath, err := m.chunkManager.FinalizeSession(token, "")
	if err != nil {
		return nil, err
	}

	// Validate KHI file structure and extract metadata.
	header, metadataMap, err := ValidateAndExtractMetadata(tempFilePath)
	if err != nil {
		os.Remove(tempFilePath)
		return nil, fmt.Errorf("failed to validate KHI file: %w", err)
	}

	destinationDir := filepath.Dir(tempFilePath)
	destinationPath := filepath.Join(destinationDir, inspectionID+".khi")

	// Move file from temporary location to destination path.
	if err := os.Rename(tempFilePath, destinationPath); err != nil {
		os.Remove(tempFilePath)
		return nil, fmt.Errorf("failed to persist inspection file: %w", err)
	}

	// Invalidate any stale trigram index from a previous upload of the same ID.
	_ = os.Remove(filepath.Join(destinationDir, inspectionID+".trigram"))
	_ = os.Remove(filepath.Join(destinationDir, inspectionID+".trigram.tmp"))

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
	return m.chunkManager.AbortSession(token)
}
