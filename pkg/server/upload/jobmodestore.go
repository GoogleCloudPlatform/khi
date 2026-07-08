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

// JobModeUploadFileStore is an UploadFileStore variant for job mode.
// It resolves file inputs from local file paths in the inspection request
// values instead of waiting for browser uploads.
type JobModeUploadFileStore struct {
	*UploadFileStore
}

// NewJobModeUploadFileStore wraps the given store for use in job mode.
func NewJobModeUploadFileStore(base *UploadFileStore) *JobModeUploadFileStore {
	base.RegisterProvider((&LocalFileUploadToken{}).GetType(), &InPlaceUploadFileStoreProvider{})
	return &JobModeUploadFileStore{UploadFileStore: base}
}

// GetResult returns the upload result for the given token, resolving file
// inputs from local file paths in the request values when running in job mode.
func (s *JobModeUploadFileStore) GetResult(token UploadToken, req map[string]any) (UploadResult, error) {
	s.fieldIDLock.RLock()
	fieldID := s.fieldIDs[token.GetID()]
	s.fieldIDLock.RUnlock()

	pathValue, ok := req[fieldID].(string)
	// No local path in the request values: fall back to the normal upload flow.
	if !ok || pathValue == "" {
		return s.UploadFileStore.GetResult(token, req)
	}
	localToken := &LocalFileUploadToken{FilePath: pathValue}

	s.verifierLock.RLock()
	verifier, found := s.verifiers[token.GetID()]
	s.verifierLock.RUnlock()
	if !found {
		return s.UploadFileStore.GetResult(token, req)
	}
	provider := s.providerForToken(localToken)
	verificationError := verifier.Verify(provider, localToken)
	result := UploadResult{
		Token:             localToken,
		StoreProvider:     provider,
		Status:            UploadStatusCompleted,
		VerificationError: verificationError,
	}
	s.resultLock.Lock()
	s.tokenHashLock.Lock()
	defer s.resultLock.Unlock()
	defer s.tokenHashLock.Unlock()
	s.tokenHashes[localToken.GetHash()] = struct{}{}
	s.results[localToken.GetID()] = result
	return result, nil
}
