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

package importinspection

import (
	"os"
	"testing"
	"time"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

func TestImportSessionCleaner_Cleanup(t *testing.T) {
	testCases := []struct {
		name          string
		setupSessions func(t *testing.T, m *ImportSessionManager) (activeToken, expiredToken string, expiredTempPath string)
		wantCleaned   int
	}{
		{
			name: "cleans up only expired sessions and deletes temporary files",
			setupSessions: func(t *testing.T, m *ImportSessionManager) (string, string, string) {
				activeSession, err := m.StartSession("active.khi", 100)
				if err != nil {
					t.Fatalf("StartSession active failed: %v", err)
				}
				expiredSession, err := m.StartSession("expired.khi", 100)
				if err != nil {
					t.Fatalf("StartSession expired failed: %v", err)
				}
				// Force expiration for expiredSession
				expiredSession.ExpiresAt = time.Now().Add(-1 * time.Hour)
				return activeSession.Token, expiredSession.Token, expiredSession.TempFilePath
			},
			wantCleaned: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			destDir := t.TempDir()
			ioConfig := &inspectioncore_contract.IOConfig{
				TemporaryFolder: tempDir,
				DataDestination: destDir,
			}
			server, err := coreinspection.NewServer(ioConfig)
			if err != nil {
				t.Fatalf("NewServer failed: %v", err)
			}

			manager := NewImportSessionManager(server, ioConfig)
			defer manager.Close()

			activeToken, expiredToken, expiredTempPath := tc.setupSessions(t, manager)

			cleaner := NewImportSessionCleaner(manager, 100*time.Millisecond)
			cleanedTokens := cleaner.Cleanup(time.Now())

			if diff := cmp.Diff(tc.wantCleaned, len(cleanedTokens)); diff != "" {
				t.Errorf("cleaned tokens count mismatch (-want +got):\n%s", diff)
			}

			if len(cleanedTokens) > 0 && cleanedTokens[0] != expiredToken {
				t.Errorf("expected cleaned token %s, got %s", expiredToken, cleanedTokens[0])
			}

			// Verify expired temporary file was deleted
			if _, err := os.Stat(expiredTempPath); !os.IsNotExist(err) {
				t.Errorf("temporary file for expired session was not removed: %s", expiredTempPath)
			}

			// Verify active session remains accessible
			if _, err := manager.WriteChunk(activeToken, 0, []byte("active-data")); err != nil {
				t.Errorf("active session should remain writable: %v", err)
			}
		})
	}
}

func TestImportSessionCleaner_BackgroundExecution(t *testing.T) {
	tempDir := t.TempDir()
	destDir := t.TempDir()
	ioConfig := &inspectioncore_contract.IOConfig{
		TemporaryFolder: tempDir,
		DataDestination: destDir,
	}
	server, err := coreinspection.NewServer(ioConfig)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	manager := NewImportSessionManager(server, ioConfig)
	defer manager.Close()

	// Step 1: Create an import session.
	session, err := manager.StartSession("expired.khi", 100)
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}
	tempPath := session.TempFilePath

	// Step 2: Simulate session expiration by setting its expiry timestamp into the past.
	session.mu.Lock()
	session.ExpiresAt = time.Now().Add(-10 * time.Millisecond)
	session.mu.Unlock()

	// Step 3: Start the background cleaner with a fast tick interval (10ms).
	cleaner := NewImportSessionCleaner(manager, 10*time.Millisecond)
	cleaner.Start()
	defer cleaner.Stop()

	// Step 4: Wait for the cleaner's background ticker to trigger a cleanup cycle.
	// We expect the expired session to be automatically aborted and removed from the manager,
	// and its associated temporary upload file to be deleted from disk within the deadline.
	deadline := time.Now().Add(1 * time.Second)
	cleaned := false
	for time.Now().Before(deadline) {
		sessions := manager.GetActiveSessions()
		_, statErr := os.Stat(tempPath)
		if len(sessions) == 0 && os.IsNotExist(statErr) {
			cleaned = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !cleaned {
		t.Errorf("background cleaner did not remove expired session and temp file within deadline")
	}
}
