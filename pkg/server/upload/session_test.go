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
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/server/chunkedupload"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func newTestUploadManager(t *testing.T) (*UploadFileStore, UploadFileStoreProvider, *FileParameterUploadManager) {
	t.Helper()
	tempDir := t.TempDir()
	provider := NewLocalUploadFileStoreProvider(filepath.Join(tempDir, "store"))
	uploadStore := NewUploadFileStore(provider)
	chunkManager := chunkedupload.NewChunkSessionManager(filepath.Join(tempDir, "upload"))
	t.Cleanup(chunkManager.Close)
	mgr := NewFileParameterUploadManager(uploadStore, chunkManager)
	return uploadStore, provider, mgr
}

func assertUploadResult(t *testing.T, store *UploadFileStore, token UploadToken, want UploadResult) {
	t.Helper()
	got, err := store.GetResult(token, nil)
	if err != nil {
		t.Fatalf("GetResult() failed: %v", err)
	}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(UploadResult{}, "Token", "StoreProvider"), cmpopts.EquateErrors()); diff != "" {
		t.Errorf("GetResult() mismatch (-want +got):\n%s", diff)
	}
}

func TestFileParameterUploadManager_Lifecycle(t *testing.T) {
	uploadStore, provider, mgr := newTestUploadManager(t)

	token := uploadStore.GetUploadToken("field-upload-1", nil, "field-1")

	// 1. Start upload session
	session, err := mgr.StartUploadSession(token.GetID(), "sample.log", 10)
	if err != nil {
		t.Fatalf("StartUploadSession failed: %v", err)
	}
	assertUploadResult(t, uploadStore, token, UploadResult{
		Status:    UploadStatusUploading,
		FileName:  "sample.log",
		SizeBytes: 10,
	})

	// 2. Write chunks
	n1, err := mgr.WriteChunk(session.Token, 0, []byte("hello"))
	if err != nil {
		t.Fatalf("WriteChunk 0 failed: %v", err)
	}
	if n1 != 5 {
		t.Errorf("WriteChunk(0) = %d, want 5", n1)
	}

	n2, err := mgr.WriteChunk(session.Token, 5, []byte("world"))
	if err != nil {
		t.Fatalf("WriteChunk 5 failed: %v", err)
	}
	if n2 != 10 {
		t.Errorf("WriteChunk(5) = %d, want 10", n2)
	}

	// 3. Complete session
	fileSize, err := mgr.CompleteUploadSession(session.Token)
	if err != nil {
		t.Fatalf("CompleteUploadSession failed: %v", err)
	}
	if fileSize != 10 {
		t.Errorf("CompleteUploadSession() = %d, want 10", fileSize)
	}
	assertUploadResult(t, uploadStore, token, UploadResult{
		Status:    UploadStatusCompleted,
		FileName:  "sample.log",
		SizeBytes: 10,
	})

	// Verify file content in provider
	reader, err := provider.Read(token)
	if err != nil {
		t.Fatalf("provider.Read failed: %v", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll failed: %v", err)
	}
	if string(content) != "helloworld" {
		t.Errorf("file content = %q, want %q", string(content), "helloworld")
	}
}

func TestFileParameterUploadManager_Errors(t *testing.T) {
	uploadStore, _, mgr := newTestUploadManager(t)

	token := uploadStore.GetUploadToken("field-upload-2", nil, "field-2")

	testCases := []struct {
		name      string
		run       func(t *testing.T) error
		wantErrIs error
	}{
		{
			name: "start session with non-existent upload token ID",
			run: func(t *testing.T) error {
				_, err := mgr.StartUploadSession("non-existent-token-id", "test.log", 100)
				return err
			},
			wantErrIs: ErrUploadTokenNotFound,
		},
		{
			name: "write chunk with non-existent session token",
			run: func(t *testing.T) error {
				_, err := mgr.WriteChunk("non-existent-session", 0, []byte("data"))
				return err
			},
			wantErrIs: chunkedupload.ErrSessionNotFound,
		},
		{
			name: "complete session with non-existent session token",
			run: func(t *testing.T) error {
				_, err := mgr.CompleteUploadSession("non-existent-session")
				return err
			},
			wantErrIs: chunkedupload.ErrSessionNotFound,
		},
		{
			name: "abort session removes temporary chunks",
			run: func(t *testing.T) error {
				session, err := mgr.StartUploadSession(token.GetID(), "test.log", 100)
				if err != nil {
					return err
				}
				tempFile := session.TempFilePath
				if err := mgr.AbortUploadSession(session.Token); err != nil {
					return err
				}
				if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
					t.Errorf("temp file not deleted after abort: %s", tempFile)
				}
				return mgr.AbortUploadSession(session.Token)
			},
			wantErrIs: chunkedupload.ErrSessionNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run(t)
			if tc.wantErrIs != nil {
				if !errors.Is(err, tc.wantErrIs) {
					t.Errorf("err = %v, want %v", err, tc.wantErrIs)
				}
			}
		})
	}
}

func TestFileParameterUploadManager_RecordUploadFailure(t *testing.T) {
	cause := errors.New("connection reset")
	testCases := []struct {
		name          string
		uploadTokenID string
		wantErrIs     error
		wantResult    UploadResult
	}{
		{
			name:          "known upload token waits for another upload with the cause",
			uploadTokenID: "field-upload-4",
			wantResult: UploadResult{
				Status:      UploadStatusWaiting,
				UploadError: cause,
			},
		},
		{
			name:          "unknown upload token is rejected",
			uploadTokenID: "non-existent-token-id",
			wantErrIs:     ErrUploadTokenNotFound,
			wantResult: UploadResult{
				Status:    UploadStatusUploading,
				FileName:  "test.log",
				SizeBytes: 10,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			uploadStore, _, mgr := newTestUploadManager(t)
			token := uploadStore.GetUploadToken("field-upload-4", nil, "field-4")
			if _, err := mgr.StartUploadSession(token.GetID(), "test.log", 10); err != nil {
				t.Fatalf("StartUploadSession failed: %v", err)
			}

			err := mgr.RecordUploadFailure(tc.uploadTokenID, cause)

			if !errors.Is(err, tc.wantErrIs) {
				t.Errorf("RecordUploadFailure() error = %v, want %v", err, tc.wantErrIs)
			}
			assertUploadResult(t, uploadStore, token, tc.wantResult)
		})
	}
}

func TestFileParameterUploadManager_StartUploadSessionRecordsSessionStartFailure(t *testing.T) {
	tempDir := t.TempDir()
	uploadStore := NewUploadFileStore(NewLocalUploadFileStoreProvider(filepath.Join(tempDir, "store")))
	// A regular file at the upload directory path makes creating the chunk session fail.
	uploadDir := filepath.Join(tempDir, "upload")
	if err := os.WriteFile(uploadDir, nil, 0o644); err != nil {
		t.Fatalf("WriteFile() returned an unexpected error: %v", err)
	}
	chunkManager := chunkedupload.NewChunkSessionManager(uploadDir)
	defer chunkManager.Close()
	mgr := NewFileParameterUploadManager(uploadStore, chunkManager)
	token := uploadStore.GetUploadToken("field-upload-5", nil, "field-5")

	_, startErr := mgr.StartUploadSession(token.GetID(), "test.log", 10)
	if startErr == nil {
		t.Fatal("StartUploadSession() error = nil, want the session start failure")
	}

	assertUploadResult(t, uploadStore, token, UploadResult{
		Status:      UploadStatusWaiting,
		UploadError: startErr,
	})
}
