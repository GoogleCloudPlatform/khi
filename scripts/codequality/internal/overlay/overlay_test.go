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

package overlay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestWriteOverlayJSON verifies that WriteOverlayJSON creates the directory, writes valid Replace JSON, and surfaces directory errors.
func TestWriteOverlayJSON(t *testing.T) {
	t.Run("writes valid overlay json inside nested directory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "nested", "ov")
		wantReplace := map[string]string{
			"/repo/a.go": "/tmp/a.go",
			"/repo/b.go": "/tmp/b.go",
		}

		outPath, err := WriteOverlayJSON(dir, "overlay.json", wantReplace)
		if err != nil {
			t.Fatalf("WriteOverlayJSON() unexpected error: %v", err)
		}

		data, readErr := os.ReadFile(outPath)
		if readErr != nil {
			t.Fatalf("ReadFile(%s) failed: %v", outPath, readErr)
		}
		var parsed overlayJSON
		if unmarshalErr := json.Unmarshal(data, &parsed); unmarshalErr != nil {
			t.Fatalf("json.Unmarshal failed: %v", unmarshalErr)
		}
		if diff := cmp.Diff(wantReplace, parsed.Replace); diff != "" {
			t.Errorf("Replace map mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("returns error when target dir path is an existing file", func(t *testing.T) {
		blockingFile := filepath.Join(t.TempDir(), "not-a-dir")
		if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}

		if _, err := WriteOverlayJSON(blockingFile, "overlay.json", map[string]string{}); err == nil {
			t.Errorf("WriteOverlayJSON(file path) = nil, want error")
		}
	})
}
