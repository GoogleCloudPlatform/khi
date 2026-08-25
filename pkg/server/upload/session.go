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

package upload

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/server/chunkedupload"
)

var (
	// ErrUploadTokenNotFound is returned when an upload token ID is unknown to the upload store.
	ErrUploadTokenNotFound = errors.New("upload token not found")
)

// FileParameterUploadManager manages chunked upload sessions for file form parameters.
type FileParameterUploadManager struct {
	uploadStore  *UploadFileStore
	chunkManager *chunkedupload.ChunkSessionManager
	sessionMap   map[string]string // sessionToken -> uploadTokenID
	mu           sync.RWMutex
}

// NewFileParameterUploadManager creates a new FileParameterUploadManager instance.
func NewFileParameterUploadManager(
	uploadStore *UploadFileStore,
	chunkManager *chunkedupload.ChunkSessionManager,
) *FileParameterUploadManager {
	return &FileParameterUploadManager{
		uploadStore:  uploadStore,
		chunkManager: chunkManager,
		sessionMap:   make(map[string]string),
	}
}

// SuggestedChunkSize returns the recommended chunk size in bytes.
func (m *FileParameterUploadManager) SuggestedChunkSize() int64 {
	return m.chunkManager.SuggestedChunkSize()
}

// StartUploadSession starts a new chunked upload session for the given upload token ID.
func (m *FileParameterUploadManager) StartUploadSession(uploadTokenID string, fileName string, totalSizeBytes int64) (*chunkedupload.ChunkSession, error) {
	token, err := m.uploadStore.GetTokenByID(uploadTokenID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUploadTokenNotFound, uploadTokenID)
	}

	if err := m.uploadStore.SetResultOnStartingUpload(token); err != nil {
		return nil, fmt.Errorf("failed to update upload status to starting: %w", err)
	}

	session, err := m.chunkManager.StartSession(fileName, totalSizeBytes)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.sessionMap[session.Token] = uploadTokenID
	m.mu.Unlock()

	return session, nil
}

// WriteChunk writes a single chunk payload at the specified byte offset for the session.
func (m *FileParameterUploadManager) WriteChunk(sessionToken string, offsetBytes int64, data []byte) (int64, error) {
	return m.chunkManager.WriteChunk(sessionToken, offsetBytes, data)
}

// CompleteUploadSession finalizes the chunked session, persists the file into the store provider,
// and notifies the upload store that the file is ready for verification and task consumption.
func (m *FileParameterUploadManager) CompleteUploadSession(sessionToken string) (int64, error) {
	m.mu.Lock()
	uploadTokenID, exists := m.sessionMap[sessionToken]
	if exists {
		delete(m.sessionMap, sessionToken)
	}
	m.mu.Unlock()

	if !exists {
		return 0, chunkedupload.ErrSessionNotFound
	}

	token, err := m.uploadStore.GetTokenByID(uploadTokenID)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrUploadTokenNotFound, uploadTokenID)
	}

	provider := m.uploadStore.GetStoreProvider()
	var destPath string
	if writableProvider, ok := provider.(DirectWritableUploadFileStoreProvider); ok {
		path, err := writableProvider.GetDestinationPath(token)
		if err != nil {
			return 0, fmt.Errorf("failed to get upload destination path: %w", err)
		}
		destPath = path
	}

	finalPath, err := m.chunkManager.FinalizeSession(sessionToken, destPath)
	if err != nil {
		return 0, fmt.Errorf("failed to finalize upload session: %w", err)
	}

	fileInfo, err := os.Stat(finalPath)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect finalized upload file size: %w", err)
	}

	if err := m.uploadStore.NotifyFileUploaded(token); err != nil {
		return 0, fmt.Errorf("failed to notify upload file store: %w", err)
	}

	return fileInfo.Size(), nil
}

// AbortUploadSession cancels and deletes any intermediate chunk files for the session.
func (m *FileParameterUploadManager) AbortUploadSession(sessionToken string) error {
	m.mu.Lock()
	delete(m.sessionMap, sessionToken)
	m.mu.Unlock()

	return m.chunkManager.AbortSession(sessionToken)
}
