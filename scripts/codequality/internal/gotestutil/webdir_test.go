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

package gotestutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWebDir(t *testing.T) {
	rawRepoRoot := t.TempDir()
	repoRoot, err := filepath.EvalSymlinks(rawRepoRoot)
	if err != nil {
		t.Fatalf("failed resolving temp repo root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/test\n"), 0o600); err != nil {
		t.Fatalf("failed writing go.mod: %v", err)
	}
	nestedWorkDir := filepath.Join(repoRoot, "scripts", "sub")
	if err := os.MkdirAll(nestedWorkDir, 0o755); err != nil {
		t.Fatalf("failed creating nested working directory: %v", err)
	}
	// Place package.json in cwd so an empty webDir must skip the explicit directory check instead of matching cwd/package.json.
	if err := os.WriteFile(filepath.Join(nestedWorkDir, "package.json"), []byte(`{"name":"cwd"}`), 0o600); err != nil {
		t.Fatalf("failed writing cwd package.json: %v", err)
	}
	t.Chdir(nestedWorkDir)

	customWebDir := filepath.Join(t.TempDir(), "custom-web")
	if err := os.MkdirAll(customWebDir, 0o755); err != nil {
		t.Fatalf("failed creating custom web directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(customWebDir, "package.json"), []byte(`{"name":"custom"}`), 0o600); err != nil {
		t.Fatalf("failed writing custom package.json: %v", err)
	}

	missingPkgDir := filepath.Join(t.TempDir(), "missing-pkg")
	if err := os.MkdirAll(missingPkgDir, 0o755); err != nil {
		t.Fatalf("failed creating directory without package.json: %v", err)
	}

	wantDefaultWebDir := filepath.Join(repoRoot, "web")
	testCases := []struct {
		name   string
		webDir string
		want   string
	}{
		// Returns the explicit webDir when it is non-empty and contains package.json.
		{
			name:   "explicit webDir with package.json",
			webDir: customWebDir,
			want:   customWebDir,
		},
		// Falls back to repoRoot/web when webDir is non-empty but lacks package.json.
		{
			name:   "non-empty webDir without package.json falls back to repoRoot/web",
			webDir: missingPkgDir,
			want:   wantDefaultWebDir,
		},
		// Ignores package.json in the working directory and resolves repoRoot/web when webDir is empty.
		{
			name:   "empty webDir ignores cwd package.json and falls back to repoRoot/web",
			webDir: "",
			want:   wantDefaultWebDir,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveWebDir(tc.webDir)

			if err != nil {
				t.Fatalf("ResolveWebDir(%q) unexpected error: %v", tc.webDir, err)
			}
			if got != tc.want {
				t.Errorf("ResolveWebDir(%q) = %q, want %q", tc.webDir, got, tc.want)
			}
		})
	}
}
