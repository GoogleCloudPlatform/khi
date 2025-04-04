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
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

// Mock UploadFileVerifier for testing.
type MockUploadFileVerifier struct {
	VerifyFunc func(storeProvider UploadFileStoreProvider, token UploadToken) error
}

func (m *MockUploadFileVerifier) Verify(storeProvider UploadFileStoreProvider, token UploadToken) error {
	if m.VerifyFunc != nil {
		return m.VerifyFunc(storeProvider, token)
	}
	return nil
}

func TestUploadFileStore(t *testing.T) {
	// Create a temporary directory for testing.
	tempDir, err := os.MkdirTemp("", "uploadtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir) // Clean up after the test.

	// Create a new LocalUploadFileStore.
	provider := NewLocalUploadFileStoreProvider(tempDir)

	t.Run("GetUploadToken_NewToken", func(t *testing.T) {
		store := NewUploadFileStore(provider)
		verifier := &MockUploadFileVerifier{}

		token := store.GetUploadToken("test-id-1", verifier)

		result, err := store.GetResult(token)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result.Status != UploadStatusWaiting {
			t.Errorf("Expected status 'Waiting', got '%v'", result.Status)
		}
	})

	t.Run("GetResult_WithSuccessfulUploadScenario", func(t *testing.T) {
		store := NewUploadFileStore(provider)
		verifier := &MockUploadFileVerifier{}

		token := store.GetUploadToken("test-id-2", verifier)

		err := store.SetResultOnStartingUpload(token)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		result, err := store.GetResult(token)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result.Status != UploadStatusUploading {
			t.Errorf("Expected status 'Uploading', got '%v'", result.Status)
		}

		err = store.SetResultOnCompletedUpload(token, nil)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		result, err = store.GetResult(token)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result.Status != UploadStatusVerifying {
			t.Errorf("Expected status 'Verifying', got '%v'", result.Status)
		}

		<-time.After(100 * time.Microsecond)
		result, err = store.GetResult(token)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result.Status != UploadStatusCompleted {
			t.Errorf("Expected status 'Completed', got '%v'", result.Status)
		}
	})

	t.Run("SetResultOnStartingUpload_NotFound", func(t *testing.T) {
		store := NewUploadFileStore(provider)

		err := store.SetResultOnStartingUpload(&DirectUploadToken{ID: "non-existing-token"})
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "unknown upload token specifed") {
			t.Errorf("Expected error message to contain 'unknown upload token specifed', got '%v'", err)
		}
	})

	t.Run("SetResultOnCompletedUpload_NotFound", func(t *testing.T) {
		store := NewUploadFileStore(provider)

		err := store.SetResultOnCompletedUpload(&DirectUploadToken{ID: "non-existing-token"}, nil)
		if err == nil {
			t.Error("Expected an error for nonexistent token")
		}
		if !strings.Contains(err.Error(), "unknown upload token specifed") {
			t.Errorf("Expected 'unknown upload token specifed' in error, got: %v", err)
		}
	})

	t.Run("SetResultOnCompletedUpload_WithUploadError", func(t *testing.T) {
		store := NewUploadFileStore(provider)

		verifier := &MockUploadFileVerifier{
			VerifyFunc: func(storeProvider UploadFileStoreProvider, token UploadToken) error {
				t.Errorf("verify function shouldn't be called on upload fail")
				return nil // Simulate successful verification (should be ignored due to upload error).
			},
		}

		token := store.GetUploadToken("uploaderror-id", verifier)

		err := store.SetResultOnStartingUpload(token) // Set initial status
		if err != nil {
			t.Fatalf("Unexpected error on SetResultOnStartingUpload: %v", err)
		}

		uploadErr := errors.New("simulated upload error")
		err = store.SetResultOnCompletedUpload(token, uploadErr)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		result, err := store.GetResult(token)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result.Status != UploadStatusWaiting {
			t.Errorf("Expected status 'Waiting', got '%v'", result.Status)
		}
		if result.UploadError != uploadErr {
			t.Errorf("Expected UploadError to be '%v', got '%v'", uploadErr, result.UploadError)
		}
		if result.VerificationError != nil {
			t.Errorf("Expected VerificationError to be nil, got '%v'", result.VerificationError) // Should be nil
		}
	})

	t.Run("SetResultOnCompletedUpload_WithVerificationError", func(t *testing.T) {
		store := NewUploadFileStore(provider)

		verificationErr := errors.New("simulated verification error")
		verifier := &MockUploadFileVerifier{
			VerifyFunc: func(storeProvider UploadFileStoreProvider, token UploadToken) error {
				return verificationErr
			},
		}

		token := store.GetUploadToken("verifyerror-id", verifier)
		err := store.SetResultOnStartingUpload(token)
		if err != nil {
			t.Fatalf("Unexpected error on SetResultOnStartingUpload: %v", err)
		}

		err = store.SetResultOnCompletedUpload(token, nil) // No upload error.
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		time.Sleep(100 * time.Millisecond) // Allow verification goroutine to run.

		result, err := store.GetResult(token)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result.Status != UploadStatusCompleted {
			t.Errorf("Expected status 'Completed', got '%v'", result.Status)
		}
		if result.UploadError != nil {
			t.Errorf("Expected UploadError to be nil, got '%v'", result.UploadError)
		}
		if result.VerificationError != verificationErr {
			t.Errorf("Expected VerificationError to be '%v', got '%v'", verificationErr, result.VerificationError)
		}
	})
	t.Run("SetResultOnCompletedUpload_RaceVerifications", func(t *testing.T) {
		store := NewUploadFileStore(provider)

		verifier := &MockUploadFileVerifier{
			VerifyFunc: func(storeProvider UploadFileStoreProvider, token UploadToken) error {
				<-time.After(100 * time.Millisecond) // Lazy verification function
				return nil
			},
		}

		token := store.GetUploadToken("test-id-4", verifier)

		err := store.SetResultOnStartingUpload(token)
		if err != nil {
			t.Fatalf("SetResultOnStartingUpload 1 error: %v", err)
		}

		err = store.SetResultOnCompletedUpload(token, nil)
		if err != nil {
			t.Fatalf("SetResultOnCompletedUpload 1 error: %v", err)
		}

		<-time.After(50 * time.Millisecond) // Start a new upload before the verification completes

		err = store.SetResultOnStartingUpload(token)
		if err != nil {
			t.Fatalf("SetResultOnStartingUpload 2 error: %v", err)
		}

		// Verification for the first upload should be done before here, but new upload is already started
		<-time.After(70 * time.Millisecond)

		err = store.SetResultOnCompletedUpload(token, nil)
		if err != nil {
			t.Fatalf("SetResultOnCompletedUpload 2 error: %v", err)
		}

		result1, err := store.GetResult(token)
		if err != nil {
			t.Fatalf("GetResult returns error: %v", err)
		}
		if result1.Status != UploadStatusVerifying {
			t.Errorf("Want Uploading status, got %v", result1.Status)
		}

		<-time.After(500 * time.Millisecond)

		result2, err := store.GetResult(token)
		if err != nil {
			t.Fatalf("GetResult returns error: %v", err)
		}
		if result2.Status != UploadStatusCompleted {
			t.Errorf("Want Completed status, got %v", result2.Status)
		}
	})
}
