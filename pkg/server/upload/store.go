// Copyright 2025 Google LLC
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
	"context"
	"fmt"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/asynctask"
)

// UploadStatus represents the status of an upload (Waiting, Uploading, Verifying, or Completed).
type UploadStatus int

const (
	// UploadStatusWaiting indicates this file is not yet uploaded or in progress of upload.
	UploadStatusWaiting UploadStatus = 0

	// UploadStatusUploading indicates this file is being uploaded.
	UploadStatusUploading UploadStatus = 1

	// UploadStatusVerifying indicates the file was uploaded and is currently being verified.
	UploadStatusVerifying UploadStatus = 2

	// UploadStatusCompleted indicates the file upload and verification process has finished.
	UploadStatusCompleted UploadStatus = 3
)

// UploadFileStore manages file uploads and coordinates non-blocking verification.
type UploadFileStore struct {
	StoreProvider        UploadFileStoreProvider
	resultLock           sync.RWMutex
	results              map[string]UploadResult
	verifierLock         sync.RWMutex
	verifiers            map[string]UploadFileVerifier
	tokenHashes          map[string]struct{}
	tokenHashLock        sync.RWMutex
	asyncVerifierManager *asynctask.AsyncTaskManager[string, struct{}]
}

// GetUploadToken returns the token to upload a file from the frontend.
func (s *UploadFileStore) GetUploadToken(id string, verifier UploadFileVerifier, fieldID string) UploadToken {
	s.resultLock.Lock()
	s.verifierLock.Lock()
	s.tokenHashLock.Lock()
	defer s.resultLock.Unlock()
	defer s.verifierLock.Unlock()
	defer s.tokenHashLock.Unlock()

	token := s.StoreProvider.GetUploadToken(id)
	s.tokenHashes[token.GetHash()] = struct{}{}
	_, ok := s.results[token.GetID()]
	if !ok {
		s.results[token.GetID()] = UploadResult{
			Token:  token,
			Status: UploadStatusWaiting,
		}
	}
	s.verifiers[token.GetID()] = verifier
	return token
}

// GetResult returns the result of the upload with the given token.
func (s *UploadFileStore) GetResult(token UploadToken, req map[string]any) (UploadResult, error) {
	err := s.ensureIssuedToken(token)
	if err != nil {
		return UploadResult{}, err
	}

	s.resultLock.RLock()
	result, ok := s.results[token.GetID()]
	s.resultLock.RUnlock()
	if !ok {
		return UploadResult{}, fmt.Errorf("upload result not found for token %s", token.GetID())
	}

	if result.Status == UploadStatusWaiting || result.Status == UploadStatusUploading {
		return result, nil
	}

	s.verifierLock.RLock()
	verifier := s.verifiers[token.GetID()]
	s.verifierLock.RUnlock()

	if verifier == nil {
		return UploadResult{
			Token:         token,
			StoreProvider: s.StoreProvider,
			Status:        UploadStatusCompleted,
		}, nil
	}

	// Non-blockingly query or launch the async verifier
	asyncRes := s.asyncVerifierManager.DoAsyncOrGet(token.GetID(), token.GetHash(), func(ctx context.Context) (struct{}, error) {
		return struct{}{}, verifier.Verify(s.StoreProvider, token)
	})

	switch asyncRes.Status {
	case asynctask.StatusRunning, asynctask.StatusPending:
		return UploadResult{
			Token:         token,
			StoreProvider: s.StoreProvider,
			Status:        UploadStatusVerifying,
		}, nil
	case asynctask.StatusFailed:
		return UploadResult{
			Token:             token,
			StoreProvider:     s.StoreProvider,
			Status:            UploadStatusCompleted,
			VerificationError: asyncRes.Error,
		}, nil
	default:
		return UploadResult{
			Token:         token,
			StoreProvider: s.StoreProvider,
			Status:        UploadStatusCompleted,
		}, nil
	}
}

// NotifyFileUploaded records that a file upload has completed and resets any cached verification.
func (s *UploadFileStore) NotifyFileUploaded(token UploadToken) error {
	err := s.ensureIssuedToken(token)
	if err != nil {
		return err
	}

	s.resultLock.Lock()
	defer s.resultLock.Unlock()

	s.results[token.GetID()] = UploadResult{
		Token:         token,
		StoreProvider: s.StoreProvider,
		Status:        UploadStatusCompleted,
	}
	s.asyncVerifierManager.Reset(token.GetID())
	return nil
}

// SetResultOnStartingUpload sets the upload status to Uploading.
func (s *UploadFileStore) SetResultOnStartingUpload(token UploadToken) error {
	err := s.ensureIssuedToken(token)
	if err != nil {
		return err
	}

	s.resultLock.Lock()
	defer s.resultLock.Unlock()

	s.results[token.GetID()] = UploadResult{
		Token:         token,
		StoreProvider: s.StoreProvider,
		Status:        UploadStatusUploading,
	}
	return nil
}

// SetResultOnCompletedUpload records the upload completion status and error.
func (s *UploadFileStore) SetResultOnCompletedUpload(token UploadToken, uploadError error) error {
	if uploadError != nil {
		s.resultLock.Lock()
		defer s.resultLock.Unlock()
		s.results[token.GetID()] = UploadResult{
			Token:         token,
			StoreProvider: s.StoreProvider,
			Status:        UploadStatusWaiting,
			UploadError:   uploadError,
		}
		return nil
	}
	return s.NotifyFileUploaded(token)
}

// GetTokenByID looks up an issued UploadToken by its string ID.
func (s *UploadFileStore) GetTokenByID(id string) (UploadToken, error) {
	s.resultLock.RLock()
	defer s.resultLock.RUnlock()
	res, ok := s.results[id]
	if !ok {
		return nil, fmt.Errorf("upload token %s not found", id)
	}
	return res.Token, nil
}

// GetStoreProvider returns the underlying UploadFileStoreProvider.
func (s *UploadFileStore) GetStoreProvider() UploadFileStoreProvider {
	return s.StoreProvider
}

// ensureIssuedToken verifies that the given UploadToken was issued from GetUploadToken.
func (s *UploadFileStore) ensureIssuedToken(token UploadToken) error {
	s.tokenHashLock.RLock()
	defer s.tokenHashLock.RUnlock()
	_, found := s.tokenHashes[token.GetHash()]
	if found {
		return nil
	}
	return fmt.Errorf("unknown upload token specified")
}

// NewUploadFileStore creates a new UploadFileStore.
func NewUploadFileStore(storeProvider UploadFileStoreProvider) *UploadFileStore {
	return &UploadFileStore{
		StoreProvider:        storeProvider,
		results:              make(map[string]UploadResult),
		verifiers:            make(map[string]UploadFileVerifier),
		tokenHashes:          make(map[string]struct{}),
		asyncVerifierManager: asynctask.NewAsyncTaskManager[string, struct{}](),
	}
}
