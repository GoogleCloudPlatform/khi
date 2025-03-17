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
	"errors"
	"fmt"
	"io"
)

// UploadResult holds the result of an upload operation.
type UploadResult struct {
	// Token is an UploadToken associated with the file.
	Token UploadToken
	// StoreProvider provides the way of getting the uploaded file with the given token.
	StoreProvider UploadFileStoreProvider
	// Status is the current state of the upload.
	Status UploadStatus
	// UploadError contains any error that occurred during the upload process itself
	UploadError error
	// VerificationError contains any error returned by the UploadFileVerifier.
	VerificationError error
	// VerificationCount is the attempt count of the verification logic. This value is preventing the race condition in verification steps.
	VerificationCount int
}

// GetReader returns an io.ReadCloser for reading the uploaded file.
// The caller MUST close the returned ReadCloser.
func (r *UploadResult) GetReader() (io.ReadCloser, error) {
	if r.StoreProvider == nil {
		return nil, errors.New("store provider is not set")
	}
	if r.Status != UploadStatusCompleted {
		return nil, fmt.Errorf("upload is not completed: current status is %v", r.Status)
	}
	return r.StoreProvider.Read(r.Token)
}
