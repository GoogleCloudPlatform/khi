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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/server/chunkedupload"
	"github.com/google/go-cmp/cmp"
)

func TestFileParameterUploadManager_Lifecycle(t *testing.T) {
	tempDir := t.TempDir()
	storeDir := filepath.Join(tempDir, "store")
	uploadDir := filepath.Join(tempDir, "upload")

	provider := NewLocalUploadFileStoreProvider(storeDir)
	uploadStore := NewUploadFileStore(provider)
	chunkManager := chunkedupload.NewChunkSessionManager(uploadDir)
	defer chunkManager.Close()

	mgr := NewFileParameterUploadManager(uploadStore, chunkManager)

	token := uploadStore.GetUploadToken("field-upload-1", nil, "field-1")

	// 1. Start upload session
	session, err := mgr.StartUploadSession(token.GetID(), "sample.log", 10)
	if err != nil {
		t.Fatalf("StartUploadSession failed: %v", err)
	}

	// Verify store status is uploading
	res, err := uploadStore.GetResult(token, nil)
	if err != nil {
		t.Fatalf("GetResult failed: %v", err)
	}
	if res.Status != UploadStatusUploading {
		t.Errorf("status mismatch: want UploadStatusUploading, got %v", res.Status)
	}

	// 2. Write chunks
	n1, err := mgr.WriteChunk(session.Token, 0, []byte("hello"))
	if err != nil {
		t.Fatalf("WriteChunk 0 failed: %v", err)
	}
	if n1 != 5 {
		t.Errorf("n1 mismatch (-want +got):\n%s", cmp.Diff(int64(5), n1))
	}

	n2, err := mgr.WriteChunk(session.Token, 5, []byte("world"))
	if err != nil {
		t.Fatalf("WriteChunk 5 failed: %v", err)
	}
	if n2 != 10 {
		t.Errorf("n2 mismatch (-want +got):\n%s", cmp.Diff(int64(10), n2))
	}

	// 3. Complete session
	fileSize, err := mgr.CompleteUploadSession(session.Token)
	if err != nil {
		t.Fatalf("CompleteUploadSession failed: %v", err)
	}
	if fileSize != 10 {
		t.Errorf("fileSize mismatch (-want +got):\n%s", cmp.Diff(int64(10), fileSize))
	}

	// Verify store status is completed
	res, err = uploadStore.GetResult(token, nil)
	if err != nil {
		t.Fatalf("GetResult after completion failed: %v", err)
	}
	if res.Status != UploadStatusCompleted {
		t.Errorf("status mismatch: want UploadStatusCompleted, got %v", res.Status)
	}

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
	if diff := cmp.Diff("helloworld", string(content)); diff != "" {
		t.Errorf("file content mismatch (-want +got):\n%s", diff)
	}
}

func TestFileParameterUploadManager_Errors(t *testing.T) {
	tempDir := t.TempDir()
	storeDir := filepath.Join(tempDir, "store")
	uploadDir := filepath.Join(tempDir, "upload")

	provider := NewLocalUploadFileStoreProvider(storeDir)
	uploadStore := NewUploadFileStore(provider)
	chunkManager := chunkedupload.NewChunkSessionManager(uploadDir)
	defer chunkManager.Close()

	mgr := NewFileParameterUploadManager(uploadStore, chunkManager)

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
					t.Errorf("error mismatch: want %v, got %v", tc.wantErrIs, err)
				}
			}
		})
	}
}

func TestFileParameterUploadManager_KeepsFileNameAndSizeThroughCompletion(t *testing.T) {
	tempDir := t.TempDir()
	uploadStore := NewUploadFileStore(NewLocalUploadFileStoreProvider(filepath.Join(tempDir, "store")))
	chunkManager := chunkedupload.NewChunkSessionManager(filepath.Join(tempDir, "upload"))
	defer chunkManager.Close()
	mgr := NewFileParameterUploadManager(uploadStore, chunkManager)
	token := uploadStore.GetUploadToken("field-upload-3", nil, "field-3")

	session, err := mgr.StartUploadSession(token.GetID(), "audit.jsonl", 5)
	if err != nil {
		t.Fatalf("StartUploadSession failed: %v", err)
	}
	assertFileNameAndSize(t, uploadStore, token, "audit.jsonl", 5)

	if _, err := mgr.WriteChunk(session.Token, 0, []byte("hello")); err != nil {
		t.Fatalf("WriteChunk failed: %v", err)
	}
	if _, err := mgr.CompleteUploadSession(session.Token); err != nil {
		t.Fatalf("CompleteUploadSession failed: %v", err)
	}
	assertFileNameAndSize(t, uploadStore, token, "audit.jsonl", 5)
}

func assertFileNameAndSize(t *testing.T, store *UploadFileStore, token UploadToken, wantFileName string, wantSizeBytes int64) {
	t.Helper()
	res, err := store.GetResult(token, nil)
	if err != nil {
		t.Fatalf("GetResult failed: %v", err)
	}
	if res.FileName != wantFileName {
		t.Errorf("FileName = %q, want %q", res.FileName, wantFileName)
	}
	if res.SizeBytes != wantSizeBytes {
		t.Errorf("SizeBytes = %d, want %d", res.SizeBytes, wantSizeBytes)
	}
}

func TestFileParameterUploadManager_RecordUploadFailure(t *testing.T) {
	cause := errors.New("connection reset")
	testCases := []struct {
		name            string
		uploadTokenID   string
		wantErrIs       error
		wantStatus      UploadStatus
		wantUploadError error
	}{
		{
			name:            "known upload token waits for another upload with the cause",
			uploadTokenID:   "field-upload-4",
			wantStatus:      UploadStatusWaiting,
			wantUploadError: cause,
		},
		{
			name:          "unknown upload token is rejected",
			uploadTokenID: "non-existent-token-id",
			wantErrIs:     ErrUploadTokenNotFound,
			wantStatus:    UploadStatusUploading,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			uploadStore := NewUploadFileStore(NewLocalUploadFileStoreProvider(filepath.Join(tempDir, "store")))
			chunkManager := chunkedupload.NewChunkSessionManager(filepath.Join(tempDir, "upload"))
			defer chunkManager.Close()
			mgr := NewFileParameterUploadManager(uploadStore, chunkManager)
			token := uploadStore.GetUploadToken("field-upload-4", nil, "field-4")
			if _, err := mgr.StartUploadSession(token.GetID(), "test.log", 10); err != nil {
				t.Fatalf("StartUploadSession failed: %v", err)
			}

			err := mgr.RecordUploadFailure(tc.uploadTokenID, cause)

			if !errors.Is(err, tc.wantErrIs) {
				t.Errorf("RecordUploadFailure() error = %v, want %v", err, tc.wantErrIs)
			}
			res, err := uploadStore.GetResult(token, nil)
			if err != nil {
				t.Fatalf("GetResult failed: %v", err)
			}
			if res.Status != tc.wantStatus {
				t.Errorf("Status = %v, want %v", res.Status, tc.wantStatus)
			}
			if res.UploadError != tc.wantUploadError {
				t.Errorf("UploadError = %v, want %v", res.UploadError, tc.wantUploadError)
			}
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

	if _, err := mgr.StartUploadSession(token.GetID(), "test.log", 10); err == nil {
		t.Fatal("StartUploadSession() error = nil, want the session start failure")
	}

	res, err := uploadStore.GetResult(token, nil)
	if err != nil {
		t.Fatalf("GetResult failed: %v", err)
	}
	if res.Status != UploadStatusWaiting {
		t.Errorf("Status = %v, want %v", res.Status, UploadStatusWaiting)
	}
	if res.UploadError == nil || !strings.Contains(res.UploadError.Error(), "failed to create upload directory") {
		t.Errorf("UploadError = %v, want the session start failure", res.UploadError)
	}
}
