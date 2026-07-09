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
	"sync"
)

// JobModeStore is a Store implementation for job mode. Job mode does not run
// the web server, so files cannot be uploaded from a browser; instead this
// store resolves file form inputs from local file paths given in the
// inspection request values.
type JobModeStore struct {
	provider  UploadFileStoreProvider
	lock      sync.RWMutex
	fieldIDs  map[string]string
	verifiers map[string]UploadFileVerifier
}

// NewJobModeStore creates a new JobModeStore.
func NewJobModeStore() *JobModeStore {
	return &JobModeStore{
		verifiers: make(map[string]UploadFileVerifier),
		fieldIDs:  make(map[string]string),
		provider:  &InPlaceUploadFileStoreProvider{},
	}
}

// GetUploadToken issues a token for the given ID and records the form field ID
// and verifier so that GetResult can resolve the file later.
func (s *JobModeStore) GetUploadToken(id string, verifier UploadFileVerifier, fieldID string) UploadToken {
	s.lock.Lock()
	defer s.lock.Unlock()
	token := s.provider.GetUploadToken(id)
	s.verifiers[token.GetID()] = verifier
	s.fieldIDs[token.GetID()] = fieldID
	return token
}

// GetResult resolves the file for the given token from the local file path in
// the request values, verifies it, and returns a completed UploadResult.
// It returns an error when the token is unknown or no path was provided.
func (s *JobModeStore) GetResult(token UploadToken, req map[string]any) (UploadResult, error) {
	s.lock.RLock()
	verifier, found := s.verifiers[token.GetID()]
	fieldID := s.fieldIDs[token.GetID()]
	s.lock.RUnlock()
	if !found {
		return UploadResult{}, fmt.Errorf("unknown upload token specified: %s", token.GetID())
	}
	path, ok := req[fieldID].(string)
	if !ok || path == "" {
		return UploadResult{}, fmt.Errorf("no local file path was provided for the form field %q in job mode", fieldID)
	}
	localToken := &LocalFileUploadToken{FilePath: path}
	verificationError := verifier.Verify(s.provider, localToken)
	return UploadResult{
		Token:             localToken,
		StoreProvider:     s.provider,
		Status:            UploadStatusCompleted,
		VerificationError: verificationError,
	}, nil
}

var _ Store = (*JobModeStore)(nil)
