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
	"fmt"
	"log/slog"
	"sync"
)

// UploadStatus represents the status of an upload (Waiting or Completed).
type UploadStatus int

const (
	// UploadStatusWaiting indicates this file is not yet uploaded or in progress of upload.
	UploadStatusWaiting UploadStatus = 0

	// UploadStatusUploading indicates this file is being uploaded.
	UploadStatusUploading UploadStatus = 1

	// UploadStatusVerifying indicates the file was uploaded but this is under verifying.
	UploadStatusVerifying UploadStatus = 2

	// UploadStatusComplete indicates the file has uploaded successfully.
	UploadStatusCompleted UploadStatus = 3
)

// UploadFileStore manages file uploads.
type UploadFileStore struct {
	StoreProvider UploadFileStoreProvider
	resultLock    sync.RWMutex
	results       map[string]UploadResult
	verifierLock  sync.RWMutex
	verifiers     map[string]UploadFileVerifier
	tokenHashes   map[string]interface{}
	tokenHashLock sync.RWMutex
	providers     map[string]UploadFileStoreProvider
	fieldIDLock   sync.RWMutex
	fieldIDs      map[string]string
}

// GetUploadToken returns the token to upload it from frontend.
// The ID must be combination of a known string and random string to make it harder to guess it from outside.
// It also records the form field ID issuing this token so mode-specific stores can look up the field's request value later.
func (s *UploadFileStore) GetUploadToken(id string, verifier UploadFileVerifier, fieldID string) UploadToken {
	s.resultLock.Lock()
	s.verifierLock.Lock()
	s.tokenHashLock.Lock()
	s.fieldIDLock.Lock()
	defer s.resultLock.Unlock()
	defer s.verifierLock.Unlock()
	defer s.tokenHashLock.Unlock()
	defer s.fieldIDLock.Unlock()
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
	s.fieldIDs[token.GetID()] = fieldID
	return token
}

// GetResult returns the result of the upload with given token.
// The req parameter is unused in this implementation; it exists
// so job-mode stores can read local file paths from the request values.
func (s *UploadFileStore) GetResult(token UploadToken, req map[string]any) (UploadResult, error) {
	err := s.ensureIssuedToken(token)
	if err != nil {
		return UploadResult{}, err
	}
	s.resultLock.RLock()
	defer s.resultLock.RUnlock()
	result, ok := s.results[token.GetID()]
	if ok {
		return result, nil
	}
	return UploadResult{}, fmt.Errorf("upload result not found for token %s", token.GetID())
}

// SetResultOnStartingUpload sets the upload status to Uploading.  It returns an error if the token is not found.
func (s *UploadFileStore) SetResultOnStartingUpload(token UploadToken) error {
	err := s.ensureIssuedToken(token)
	if err != nil {
		return err
	}
	s.resultLock.Lock()
	defer s.resultLock.Unlock()
	_, ok := s.results[token.GetID()]
	if !ok {
		return fmt.Errorf("upload result not found for token %s", token.GetID())
	}
	s.results[token.GetID()] = UploadResult{
		Token:         token,
		StoreProvider: s.providerForToken(token),
		Status:        UploadStatusUploading,
	}
	return nil
}

// SetResultOnCompletedUpload notify the file upload is completed and start verifier.
func (s *UploadFileStore) SetResultOnCompletedUpload(token UploadToken, uploadError error) error {
	err := s.ensureIssuedToken(token)
	if err != nil {
		return err
	}
	s.resultLock.Lock()
	defer s.resultLock.Unlock()
	prev, ok := s.results[token.GetID()]
	if !ok {
		return fmt.Errorf("upload result not found for token %s", token.GetID())
	}
	nextVerificationIndex := prev.VerificationCount + 1
	if uploadError == nil {
		s.results[token.GetID()] = UploadResult{
			Token:             token,
			StoreProvider:     s.providerForToken(token),
			Status:            UploadStatusVerifying,
			UploadError:       uploadError,
			VerificationCount: nextVerificationIndex,
		}
	} else {
		s.results[token.GetID()] = UploadResult{
			Token:             token,
			StoreProvider:     s.providerForToken(token),
			Status:            UploadStatusWaiting,
			UploadError:       uploadError,
			VerificationCount: prev.VerificationCount,
		}
	}
	if uploadError == nil {
		s.verifierLock.RLock()
		verifier := s.verifiers[token.GetID()]
		s.verifierLock.RUnlock()
		go func() {
			err := verifier.Verify(s.providerForToken(token), token)
			s.resultLock.Lock()
			defer s.resultLock.Unlock()
			current, ok := s.results[token.GetID()]
			if !ok {
				slog.Error(fmt.Sprintf("upload result not found for token %s", token.GetID()))
				return
			}
			if current.VerificationCount != nextVerificationIndex {
				// user maybe uploaded file twice and the verification result for previous upload is ignored
				return
			}
			s.results[token.GetID()] = UploadResult{
				Token:             token,
				StoreProvider:     s.providerForToken(token),
				Status:            UploadStatusCompleted,
				UploadError:       current.UploadError,
				VerificationError: err,
				VerificationCount: nextVerificationIndex,
			}
		}()
	}
	return nil
}

// ensureIssuedToken verify given UploadToken is issued from GetUploadToken and
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
		StoreProvider: storeProvider,
		results:       make(map[string]UploadResult),
		verifiers:     make(map[string]UploadFileVerifier),
		tokenHashes:   make(map[string]interface{}),
		providers:     make(map[string]UploadFileStoreProvider),
		fieldIDs:      make(map[string]string),
	}
}

// RegisterProvider registers a provider to handle tokens of the given type.
func (s *UploadFileStore) RegisterProvider(tokenType string, provider UploadFileStoreProvider) {
	s.providers[tokenType] = provider
}

// providerForToken returns the provider registered for the token's type,
// or the default StoreProvider when no provider is registered for that type.
func (s *UploadFileStore) providerForToken(token UploadToken) UploadFileStoreProvider {
	provider, ok := s.providers[token.GetType()]
	if !ok {
		return s.StoreProvider
	}
	return provider
}
