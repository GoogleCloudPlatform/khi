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

package chunkedupload

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestChunkSessionManager_Lifecycle(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewChunkSessionManager(tempDir, WithSessionTTL(time.Minute))
	defer manager.Close()

	totalSize := int64(10)
	session, err := manager.StartSession("test.log", totalSize)
	if err != nil {
		t.Fatalf("StartSession() failed: %v", err)
	}

	// 1. Write first chunk: bytes [0, 5)
	received, err := manager.WriteChunk(session.Token, 0, []byte("hello"))
	if err != nil {
		t.Fatalf("WriteChunk(0) failed: %v", err)
	}
	if received != 5 {
		t.Errorf("WriteChunk(0) received mismatch (-want +got):\n%s", cmp.Diff(int64(5), received))
	}

	// 2. Write second chunk: bytes [5, 10)
	received, err = manager.WriteChunk(session.Token, 5, []byte("world"))
	if err != nil {
		t.Fatalf("WriteChunk(5) failed: %v", err)
	}
	if received != 10 {
		t.Errorf("WriteChunk(5) received mismatch (-want +got):\n%s", cmp.Diff(int64(10), received))
	}

	// 3. Finalize session
	destPath := filepath.Join(tempDir, "final.log")
	finalPath, err := manager.FinalizeSession(session.Token, destPath)
	if err != nil {
		t.Fatalf("FinalizeSession() failed: %v", err)
	}

	if finalPath != destPath {
		t.Errorf("FinalizeSession() path mismatch (-want +got):\n%s", cmp.Diff(destPath, finalPath))
	}

	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read finalized file: %v", err)
	}
	if diff := cmp.Diff("helloworld", string(content)); diff != "" {
		t.Errorf("finalized content mismatch (-want +got):\n%s", diff)
	}

	// Finalized session should no longer exist
	_, err = manager.WriteChunk(session.Token, 0, []byte("extra"))
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("expected ErrSessionNotFound after finalization, got %v", err)
	}
}

func TestChunkSessionManager_WriteChunkErrors(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewChunkSessionManager(tempDir, WithMaxChunkSize(10))
	defer manager.Close()

	session, err := manager.StartSession("test.log", 20)
	if err != nil {
		t.Fatalf("StartSession() failed: %v", err)
	}

	testCases := []struct {
		name    string
		token   string
		offset  int64
		data    []byte
		wantErr error
	}{
		{
			name:    "unknown session",
			token:   "non-existent-token",
			offset:  0,
			data:    []byte("a"),
			wantErr: ErrSessionNotFound,
		},
		{
			name:    "chunk too large",
			token:   session.Token,
			offset:  0,
			data:    []byte("12345678901"), // 11 bytes > max 10
			wantErr: ErrChunkSizeTooLarge,
		},
		{
			name:    "negative offset",
			token:   session.Token,
			offset:  -1,
			data:    []byte("a"),
			wantErr: ErrInvalidOffset,
		},
		{
			name:    "empty data",
			token:   session.Token,
			offset:  0,
			data:    []byte{},
			wantErr: ErrEmptyChunkData,
		},
		{
			name:    "offset exceeds total size",
			token:   session.Token,
			offset:  25,
			data:    []byte("a"),
			wantErr: ErrInvalidOffset,
		},
		{
			name:    "offset near math.MaxInt64 does not overflow",
			token:   session.Token,
			offset:  math.MaxInt64 - 1,
			data:    []byte("a"),
			wantErr: ErrInvalidOffset,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := manager.WriteChunk(tc.token, tc.offset, tc.data)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("WriteChunk() error mismatch: want error is %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestChunkSessionManager_AbortSession(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewChunkSessionManager(tempDir)
	defer manager.Close()

	session, err := manager.StartSession("test.log", 10)
	if err != nil {
		t.Fatalf("StartSession() failed: %v", err)
	}

	tempFile := session.TempFilePath
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Fatalf("expected temp file to exist: %s", tempFile)
	}

	err = manager.AbortSession(session.Token)
	if err != nil {
		t.Fatalf("AbortSession() failed: %v", err)
	}

	if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
		t.Errorf("expected temp file to be deleted after abort, but it exists")
	}

	// Aborting again should return ErrSessionNotFound
	err = manager.AbortSession(session.Token)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("expected ErrSessionNotFound on second abort, got %v", err)
	}
}
