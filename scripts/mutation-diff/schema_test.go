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

package main

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGenerateRuntimeSourceParsesCleanly(t *testing.T) {
	src := GenerateRuntimeSource("samplepkg", 42)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, runtimeHelperFileName, src, 0)
	if err != nil {
		t.Fatalf("GenerateRuntimeSource() produced invalid Go syntax: %v", err)
	}
	if file.Name.Name != "samplepkg" {
		t.Errorf("package name = %q, want %q", file.Name.Name, "samplepkg")
	}
	if !strings.Contains(string(src), "__khiMutMu.Lock()") {
		t.Errorf("GenerateRuntimeSource() missing mutex lock around trace file write")
	}
}

func TestWritePackageOverlay(t *testing.T) {
	repoRoot := t.TempDir()
	workDir := t.TempDir()

	src := []byte(`package mathutil

func add(a, b int) int {
	return a + b
}
`)
	plan, err := PlanFileMutations("pkg/mathutil/add.go", src, &MutateFileOptions{
		ChangedLines: map[int]struct{}{4: {}},
	})
	if err != nil {
		t.Fatalf("PlanFileMutations() error: %v", err)
	}

	overlayPath, err := WritePackageOverlay(repoRoot, "pkg/mathutil", "mathutil", []OverlayFileSpec{
		{
			RelFile: plan.File,
			Source:  plan.Source,
			Sites:   plan.Sites,
		},
	}, 1, workDir)
	if err != nil {
		t.Fatalf("WritePackageOverlay() error: %v", err)
	}

	raw, err := os.ReadFile(overlayPath)
	if err != nil {
		t.Fatalf("failed to read overlay.json: %v", err)
	}
	var parsed goOverlayJSON
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("failed to unmarshal overlay.json: %v", err)
	}

	wantAddOrig := filepath.Join(repoRoot, "pkg", "mathutil", "add.go")
	wantRuntimeOrig := filepath.Join(repoRoot, "pkg", "mathutil", runtimeHelperFileName)
	if _, ok := parsed.Replace[wantAddOrig]; !ok {
		t.Errorf("overlay.json missing entry for %s", wantAddOrig)
	}
	if _, ok := parsed.Replace[wantRuntimeOrig]; !ok {
		t.Errorf("overlay.json missing entry for %s", wantRuntimeOrig)
	}
}

func TestParseTraceFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("missing file returns empty set", func(t *testing.T) {
		got, err := ParseTraceFile(filepath.Join(tmpDir, "nonexistent.trace"))
		if err != nil {
			t.Fatalf("ParseTraceFile() unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("len(got) = %d, want 0", len(got))
		}
	})

	t.Run("parses deduplicated positive mutant IDs and polarity tokens", func(t *testing.T) {
		tracePath := filepath.Join(tmpDir, "reached.trace")
		content := `0
1:T
4:S
1:F
-2
7
`
		if err := os.WriteFile(tracePath, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to write trace fixture: %v", err)
		}
		tokens, got, err := ParsePolarityTraceFile(tracePath)
		if err != nil {
			t.Fatalf("ParsePolarityTraceFile() unexpected error: %v", err)
		}
		want := map[int]struct{}{
			1: {},
			4: {},
			7: {},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("ParsePolarityTraceFile() reached mismatch (-want +got):\n%s", diff)
		}
		wantTokens := map[string]struct{}{
			"1:T": {},
			"1:F": {},
			"4:S": {},
			"7:S": {},
		}
		if diff := cmp.Diff(wantTokens, tokens); diff != "" {
			t.Errorf("ParsePolarityTraceFile() tokens mismatch (-want +got):\n%s", diff)
		}
	})
}
