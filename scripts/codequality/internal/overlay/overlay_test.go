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
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBuildReconstructsBaseForGoTest(t *testing.T) {
	testCases := []struct {
		name        string
		missingRepo bool
		readErr     error
		wantError   bool
	}{
		{
			name:      "reconstructs modified deleted embedded and skipped directories hermetically",
			wantError: false,
		},
		{
			name:        "tolerates nonexistent repo root directory cleanly",
			missingRepo: true,
			wantError:   false,
		},
		{
			name:      "propagates base reader error",
			readErr:   errors.New("vcs read failure"),
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repoDir := t.TempDir()
			if tc.missingRepo {
				repoDir = filepath.Join(repoDir, "does-not-exist")
			} else {
				writeTempModule(t, repoDir)
			}
			workDir := t.TempDir()

			snap, err := Build(Config{
				RepoRoot:       repoDir,
				WorkDir:        workDir,
				ProdFiles:      []string{"./calc/calc.go", "calc/new_only.go", ""},
				TestFiles:      []string{"calc/calc_test.go"},
				DeletedGoFiles: []string{"calc/deleted_helper.go"},
				ReadBaseFile: func(relPath string) ([]byte, bool, error) {
					if tc.readErr != nil {
						return nil, false, tc.readErr
					}
					return fakeBaseFiles(relPath)
				},
			})
			if (err != nil) != tc.wantError {
				t.Fatalf("Build() error = %v, wantError %v", err, tc.wantError)
			}
			if tc.wantError || tc.missingRepo {
				return
			}
			verifyBuiltBaseSnapshot(t, snap, workDir)
		})
	}
}

func verifyBuiltBaseSnapshot(t *testing.T, snap *Snapshot, workDir string) {
	t.Helper()
	if gotSrc, ok := snap.BaseSource("./calc/calc.go"); !ok || len(gotSrc) == 0 {
		t.Errorf("BaseSource(calc/calc.go) = (%d bytes, %v), want non-empty true", len(gotSrc), ok)
	}
	if _, ok := snap.BaseSource("calc/new_only.go"); ok {
		t.Errorf("BaseSource(calc/new_only.go) = true, want false")
	}
	assertMirrorSkipsAndPreserves(t, snap.RepoRoot())

	covPath := filepath.Join(workDir, "ov.cov")
	cmd := exec.CommandContext(context.Background(), "go", "test", "-coverprofile="+covPath, "./calc", "./peer/web")
	cmd.Dir = snap.RepoRoot()
	if out, runErr := cmd.CombinedOutput(); runErr != nil {
		t.Fatalf("go test in base RepoRoot failed: %v\n%s", runErr, string(out))
	}
	covBytes, err := os.ReadFile(covPath)
	if err != nil || len(covBytes) == 0 {
		t.Fatalf("expected non-empty coverprofile, got err=%v", err)
	}

	jsonPath, jsonErr := WriteOverlayJSON(workDir, "extra.json", map[string]string{"/a": "/b"})
	if jsonErr != nil {
		t.Fatalf("WriteOverlayJSON() error = %v", jsonErr)
	}
	jsonBytes, readErr := os.ReadFile(jsonPath)
	if readErr != nil || string(jsonBytes) == "" {
		t.Fatalf("expected non-empty overlay json, got err=%v", readErr)
	}
}

func assertMirrorSkipsAndPreserves(t *testing.T, baseRoot string) {
	t.Helper()
	for _, skipped := range []string{".git", ".jj", "node_modules", "web", "calc/new_only.go", "calc/dangling_link"} {
		if _, err := os.Lstat(filepath.Join(baseRoot, filepath.FromSlash(skipped))); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("expected %s to be absent in baseRoot, got err=%v", skipped, err)
		}
	}
	for _, kept := range []string{"peer/web/embed.go", "peer/web/assets/hello.txt"} {
		info, err := os.Lstat(filepath.Join(baseRoot, filepath.FromSlash(kept)))
		if err != nil || !info.Mode().IsRegular() {
			t.Errorf("expected regular file %s in baseRoot, got info=%v err=%v", kept, info, err)
		}
	}
}

func writeTempModule(t *testing.T, repoDir string) {
	t.Helper()
	for _, sub := range []string{"calc", "peer/web/assets", ".git", ".jj", "node_modules", "web"} {
		if err := os.MkdirAll(filepath.Join(repoDir, filepath.FromSlash(sub)), 0o755); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
	}
	files := map[string]string{
		".git/HEAD":                 "ref: refs/heads/main\n",
		".jj/repo":                  "jj\n",
		"node_modules/pkg.js":       "module.exports = {};\n",
		"web/app.ts":                "export const x = 1;\n",
		"peer/web/assets/hello.txt": "embedded hello\n",
		"peer/web/embed.go": `package web

import "embed"

//go:embed assets/hello.txt
var Assets embed.FS

func ReadHello() string {
	b, _ := Assets.ReadFile("assets/hello.txt")
	return string(b)
}
`,
		"peer/web/embed_test.go": `package web

import "testing"

func TestReadHello(t *testing.T) {
	if ReadHello() != "embedded hello\n" {
		t.Fatal("embed mismatch")
	}
}
`,
		"go.mod": "module example.com/overlaytest\n\ngo 1.24\n",
		"calc/calc.go": `package calc

func Add(a, b int) int { return NewHelper(a, b) }
`,
		// Defines legacySum with conflicting behavior so base compilation fails if new_only.go is not excluded!
		"calc/new_only.go": `package calc

func NewHelper(a, b int) int { return a + b + 100 }
func legacySum(a, b int) int { return -999 }
`,
		"calc/calc_test.go": `package calc

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 103 {
		t.Fatal("bad")
	}
}
`,
	}
	for rel, body := range files {
		if err := os.WriteFile(filepath.Join(repoDir, filepath.FromSlash(rel)), []byte(body), 0o600); err != nil {
			t.Fatalf("WriteFile(%s) failed: %v", rel, err)
		}
	}
	if err := os.Symlink("nonexistent_target", filepath.Join(repoDir, "calc", "dangling_link")); err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}
}

func fakeBaseFiles(relPath string) ([]byte, bool, error) {
	switch relPath {
	case "calc/calc.go":
		return []byte("package calc\n\nfunc Add(a, b int) int { return legacySum(a, b) }\n"), true, nil
	case "calc/deleted_helper.go":
		return []byte("package calc\n\nfunc legacySum(a, b int) int { return a + b }\n"), true, nil
	case "calc/calc_test.go":
		return []byte("package calc\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(1, 2) != 3 {\n\t\tt.Fatal(\"bad base\")\n\t}\n}\n"), true, nil
	default:
		return nil, false, nil
	}
}
