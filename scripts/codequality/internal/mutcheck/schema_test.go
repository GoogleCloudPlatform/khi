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

package mutcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderSuperMutantAndWriteOverlay(t *testing.T) {
	src := `package demo

func Calc(a, b int, ok bool) bool {
	if check(a) {
		a++
	}
	if a < b && !ok {
		a = 0 + (a - b*2/1)
		return false
	}
	if a <= b || a > b || a >= b || a == b || a != b {
		return true
	}
	return ok
}

func check(x int) bool { return x > 0 }
`
	plan, err := PlanFileMutations("demo/demo.go", []byte(src), PlanOptions{
		TargetLines: lineRangeSet(1, 25),
	})
	if err != nil {
		t.Fatalf("PlanFileMutations() error: %v", err)
	}

	testCases := []struct {
		name       string
		sites      []MutationSite
		wantSubstr string
	}{
		{
			name:       "renders super mutant runtime calls including __khiMutCond",
			sites:      append(append([]MutationSite(nil), plan.Sites...), MutationSite{StartOffset: 0, EndOffset: 7, Kind: siteKind(99)}),
			wantSubstr: "__khiMutCond",
		},
		{
			name:       "empty sites preserves original source",
			sites:      nil,
			wantSubstr: "func Calc(",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rendered := string(RenderSuperMutantSource(plan.Source, tc.sites))
			if !strings.Contains(rendered, tc.wantSubstr) {
				t.Errorf("rendered missing %q:\n%s", tc.wantSubstr, rendered)
			}
		})
	}

	ovPath, ovErr := WritePackageOverlay(OverlayRequest{
		RepoRoot:  t.TempDir(),
		PkgRelDir: "demo",
		PkgName:   "demo",
		Files: []OverlayFileSpec{
			{RelFile: "demo/clean.go", Source: []byte("package demo\n"), Sites: nil},
			{RelFile: "demo/demo.go", Source: plan.Source, Sites: plan.Sites},
		},
		MaxMutantID: len(plan.Mutants),
		WorkDir:     t.TempDir(),
	})
	if ovErr != nil || ovPath == "" {
		t.Fatalf("WritePackageOverlay() = (%q, %v)", ovPath, ovErr)
	}
}

func TestParsePolarityTraceFile(t *testing.T) {
	testCases := []struct {
		name      string
		traceBody string
		wantErr   bool
		wantCount int
	}{
		{
			name:      "parses polarity and statement tokens",
			traceBody: "1:T\n2:F\n3\n\n",
			wantErr:   false,
			wantCount: 3,
		},
		{
			name:      "rejects non-integer trace token",
			traceBody: "bad:T\n",
			wantErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tracePath := filepath.Join(t.TempDir(), "run.trace")
			if err := os.WriteFile(tracePath, []byte(tc.traceBody), 0o600); err != nil {
				t.Fatalf("WriteFile failed: %v", err)
			}
			tokens, reached, err := ParsePolarityTraceFile(tracePath)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParsePolarityTraceFile() error = %v, wantErr %v", err, tc.wantErr)
			}
			if len(tokens) != tc.wantCount || len(reached) != tc.wantCount {
				t.Errorf("parsed (%d, %d), want %d", len(tokens), len(reached), tc.wantCount)
			}
		})
	}
}
