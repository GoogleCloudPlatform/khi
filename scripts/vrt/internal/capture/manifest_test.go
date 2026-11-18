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

package capture

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSanitizeFingerprint(t *testing.T) {
	testCases := []struct {
		name           string
		browserVersion string
		osCodename     string
		want           string
	}{
		{
			name:           "normalizes Chromium version and Debian codename",
			browserVersion: "/HeadlessChrome/136.0.7103.92",
			osCodename:     "trixie!",
			want:           "headlesschrome-136.0.7103.92-trixie",
		},
		{
			name:           "omits empty codename",
			browserVersion: "Chrome/135.0.0.0",
			osCodename:     "",
			want:           "chrome-135.0.0.0",
		},
		{
			name:           "returns unknown for empty inputs",
			browserVersion: "   ",
			osCodename:     "",
			want:           "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeFingerprint(tc.browserVersion, tc.osCodename)
			if got != tc.want {
				t.Errorf("SanitizeFingerprint(%q, %q) = %q, want %q", tc.browserVersion, tc.osCodename, got, tc.want)
			}
		})
	}
}

func TestWriteAndReadManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", ManifestFileName)

	want := Manifest{
		Version:            ManifestVersion,
		CommitSHA:          "abc123def456",
		BuilderFingerprint: "headlesschrome-136.0.7103.92-trixie",
		CapturedAt:         "2026-11-17T02:00:00Z",
		ViewportWidth:      1280,
		ViewportHeight:     720,
		Entries: map[string]SnapshotEntry{
			StorySnapshotID("header--default"): {
				ID:           StorySnapshotID("header--default"),
				StoryID:      "header--default",
				Title:        "Header",
				Name:         "Default",
				ImportPath:   "./src/app/header/components/header.component.stories.ts",
				ImageRelPath: ImageRelPathForID(StorySnapshotID("header--default")),
				SHA256:       "deadbeef",
				Width:        1280,
				Height:       720,
				DurationMs:   142,
			},
		},
	}

	if err := WriteManifest(path, want); err != nil {
		t.Fatalf("WriteManifest() unexpected error = %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) unexpected error = %v", path, err)
	}
	if len(raw) == 0 || raw[len(raw)-1] != '\n' {
		t.Errorf("WriteManifest() output missing trailing newline: %q", string(raw))
	}

	got, err := ReadManifest(path)
	if err != nil {
		t.Fatalf("ReadManifest() unexpected error = %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ReadManifest() mismatch (-want +got):\n%s", diff)
	}
}

func TestReadManifest_ErrorsAndNilEntries(t *testing.T) {
	dir := t.TempDir()
	emptyEntriesPath := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(emptyEntriesPath, []byte(`{"version":1}`), 0o600); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	got, err := ReadManifest(emptyEntriesPath)
	if err != nil {
		t.Fatalf("ReadManifest() unexpected error = %v", err)
	}
	if got.Entries == nil {
		t.Fatal("ReadManifest() Entries is nil, want initialized empty map")
	}

	invalidPath := filepath.Join(dir, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("{bad"), 0o600); err != nil {
		t.Fatalf("failed to write invalid manifest: %v", err)
	}
	if _, err := ReadManifest(invalidPath); err == nil {
		t.Error("ReadManifest(invalid) expected error, got nil")
	}
	if _, err := ReadManifest(filepath.Join(dir, "missing.json")); err == nil {
		t.Error("ReadManifest(missing) expected error, got nil")
	}
}
