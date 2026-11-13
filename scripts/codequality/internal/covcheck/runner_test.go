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

package covcheck

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPlanPackagesAndRunCoverProfile(t *testing.T) {
	repoDir := t.TempDir()
	workDir := t.TempDir()
	writeCovModule(t, repoDir)

	testCases := []struct {
		name      string
		full      bool
		prodFiles []string
		wantPkgs  []string
	}{
		{
			name:      "selective target packages sorted",
			full:      false,
			prodFiles: []string{"./sub/sub.go", "root.go", "sub/assets/skip.go", "pkg/generated/skip.go", "missing/none.go"},
			wantPkgs:  []string{".", "./sub"},
		},
		{
			name:      "full mode returns wildcard",
			full:      true,
			prodFiles: nil,
			wantPkgs:  []string{"./..."},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotPkgs := PlanPackages(PlanRequest{
				RepoRoot:  repoDir,
				ProdFiles: tc.prodFiles,
				Full:      tc.full,
			})
			if diff := cmp.Diff(tc.wantPkgs, gotPkgs); diff != "" {
				t.Errorf("PlanPackages() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	var errBuf bytes.Buffer
	blocks, err := RunCoverProfile(context.Background(), RunRequest{
		RepoRoot:    repoDir,
		ModulePath:  "example.com/covmod",
		Packages:    []string{"./sub"},
		WorkDir:     workDir,
		ProfileName: "sub.cov",
		ErrOut:      &errBuf,
	})
	if err != nil {
		t.Fatalf("RunCoverProfile() error: %v", err)
	}
	if len(blocks["sub/sub.go"]) == 0 {
		t.Errorf("expected cover blocks for sub/sub.go, got: %+v", blocks)
	}

	// Empty package list returns empty map without invoking go test.
	emptyBlocks, err := RunCoverProfile(context.Background(), RunRequest{RepoRoot: repoDir})
	if err != nil || len(emptyBlocks) != 0 {
		t.Errorf("RunCoverProfile(empty) = (%v, %v), want empty nil", emptyBlocks, err)
	}
}

func writeCovModule(t *testing.T, repoDir string) {
	t.Helper()
	subDir := filepath.Join(repoDir, "sub", "assets")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	files := map[string]string{
		"sub/assets/readme.txt": "not a go file\n",
		"go.mod": `module example.com/covmod

go 1.24
`,
		"root.go": `package root

func R() int { return 1 }
`,
		"sub/sub.go": `package sub

func S(x int) int { return x + 1 }
`,
		"sub/sub_test.go": `package sub

import "testing"

func TestS(t *testing.T) {
	if S(1) != 2 {
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
}
