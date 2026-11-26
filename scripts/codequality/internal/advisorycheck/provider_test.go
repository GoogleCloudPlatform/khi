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

package advisorycheck

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// TestProviderMetadataAndDeterministicOutput verifies provider metadata, context cancellation propagation, and deterministic item sorting/IDs.
func TestProviderMetadataAndDeterministicOutput(t *testing.T) {
	p := NewProvider()
	if p.Name() != "advisorycheck" {
		t.Errorf("Name() = %q, want advisorycheck", p.Name())
	}
	if p.Speed() != rule.SpeedFast {
		t.Errorf("Speed() = %q, want %q", p.Speed(), rule.SpeedFast)
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := p.Evaluate(canceledCtx, rule.AdvisoryRequest{}); !errors.Is(err, context.Canceled) {
		t.Errorf("Evaluate(canceled) error = %v, want context.Canceled", err)
	}

	repoRoot := t.TempDir()
	writeRepoFiles(t, repoRoot, map[string]string{
		"pkg/a/a.go": "package a\n",
		"pkg/a/a_test.go": `package a

import "testing"

func TestKeep(t *testing.T) {}
`,
		"pkg/b/b.go": `package b

// nolint:errcheck // broad suppression
func B() {}
`,
	})
	baseFiles := map[string]string{
		"pkg/a/a_test.go": `package a

import "testing"

func TestRemoved(t *testing.T) {}
func TestKeep(t *testing.T) {}
`,
		"pkg/b/b.go": `package b

func B() {}
`,
	}

	req := rule.AdvisoryRequest{
		RepoRoot:     repoRoot,
		ChangedFiles: []string{"pkg/b/b.go", "pkg/a/a_test.go", "pkg/b/b.go"},
		ChangedLines: map[string]map[int]struct{}{
			"pkg/b/b.go": {3: {}},
		},
		ReadBaseFile: func(relPath string) ([]byte, bool, error) {
			s, ok := baseFiles[relPath]
			return []byte(s), ok, nil
		},
	}

	first, skipFirst, errFirst := p.Evaluate(context.Background(), req)
	second, skipSecond, errSecond := p.Evaluate(context.Background(), req)
	if errFirst != nil || errSecond != nil {
		t.Fatalf("Evaluate() unexpected error: first=%v second=%v", errFirst, errSecond)
	}
	if skipFirst != nil || skipSecond != nil {
		t.Fatalf("Evaluate() unexpected skip: first=%+v second=%+v", skipFirst, skipSecond)
	}
	if diff := cmp.Diff(first, second); diff != "" {
		t.Errorf("Evaluate() non-deterministic output (-first +second):\n%s", diff)
	}
	if len(first) != 2 || first[0].Check != "test_removed" || first[1].Check != "suppression_added" {
		t.Errorf("Evaluate() items = %+v, want [test_removed, suppression_added]", first)
	}
}

// TestCrossFileGoTestMoveDiffSignals verifies cross-file Test* move suppression of test_removed alongside anti-Goodhart pairing for test_cases_reduced and test_assertion_weakened.
func TestCrossFileGoTestMoveDiffSignals(t *testing.T) {
	testCases := []struct {
		name           string
		targetFiles    map[string]string
		baseFiles      map[string]string
		changedFiles   []string
		deletedFiles   []string
		want           []advisoryCheckSnapshot
		wantFirstCands []string
	}{
		// Moving TestLive verbatim from modified svc_test.go into a brand-new svc_integration_test.go emits zero findings.
		{
			name: "verbatim move to new integration file emits zero findings",
			targetFiles: map[string]string{
				"pkg/svc/svc.go": "package svc\n",
				"pkg/svc/svc_test.go": `package svc

import "testing"

func TestUnit(t *testing.T) {}
`,
				"pkg/svc/svc_integration_test.go": `package svc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLive(t *testing.T) {
	if d := cmp.Diff("ok", run()); d != "" {
		t.Errorf("mismatch: %s", d)
	}
}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/svc_test.go": `package svc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestUnit(t *testing.T) {}

func TestLive(t *testing.T) {
	if d := cmp.Diff("ok", run()); d != "" {
		t.Errorf("mismatch: %s", d)
	}
}
`,
			},
			changedFiles: []string{"pkg/svc/svc_test.go", "pkg/svc/svc_integration_test.go"},
			want:         nil,
		},
		// Moving all tests out of deleted old_test.go into existing paired_test.go emits zero test_removed findings.
		{
			name: "moving all tests from deleted file emits zero test_removed",
			targetFiles: map[string]string{
				"pkg/svc/paired.go": "package svc\n",
				"pkg/svc/paired_test.go": `package svc

import "testing"

func TestKeep(t *testing.T) {}
func TestMoved(t *testing.T) {}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/paired_test.go": `package svc

import "testing"

func TestKeep(t *testing.T) {}
`,
				"pkg/svc/old_test.go": `package svc

import "testing"

func TestMoved(t *testing.T) {}
`,
			},
			changedFiles: []string{"pkg/svc/paired_test.go"},
			deletedFiles: []string{"pkg/svc/old_test.go"},
			want:         nil,
		},
		// Moving TestMoved out of deleted old_test.go while genuinely dropping TestDropped emits test_removed only for TestDropped.
		{
			name: "partial move from deleted file still flags genuinely deleted test",
			targetFiles: map[string]string{
				"pkg/svc/svc.go": "package svc\n",
				"pkg/svc/svc_integration_test.go": `package svc

import "testing"

func TestMoved(t *testing.T) {}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/old_test.go": `package svc

import "testing"

func TestMoved(t *testing.T) {}
func TestDropped(t *testing.T) {}
`,
			},
			changedFiles: []string{"pkg/svc/svc_integration_test.go"},
			deletedFiles: []string{"pkg/svc/old_test.go"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestRemoved, File: "pkg/svc/old_test.go", Symbol: "TestDropped"},
			},
		},
		// Moving TestOld across files while renaming it to TestNew still emits test_removed for TestOld.
		{
			name: "cross-file rename still emits test_removed",
			targetFiles: map[string]string{
				"pkg/svc/svc.go":      "package svc\n",
				"pkg/svc/svc_test.go": "package svc\n",
				"pkg/svc/svc_integration_test.go": `package svc

import "testing"

func TestNew(t *testing.T) {}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/svc_test.go": `package svc

import "testing"

func TestOld(t *testing.T) {}
`,
			},
			changedFiles: []string{"pkg/svc/svc_test.go", "pkg/svc/svc_integration_test.go"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestRemoved, File: "pkg/svc/svc_test.go", Symbol: "TestOld"},
			},
		},
		// Moving TestFoo across different packages still emits test_removed on the origin package.
		{
			name: "cross-package move still emits test_removed",
			targetFiles: map[string]string{
				"pkg/a/a.go":      "package a\n",
				"pkg/a/a_test.go": "package a\n",
				"pkg/b/b.go":      "package b\n",
				"pkg/b/b_test.go": `package b

import "testing"

func TestFoo(t *testing.T) {}
`,
			},
			baseFiles: map[string]string{
				"pkg/a/a_test.go": `package a

import "testing"

func TestFoo(t *testing.T) {}
`,
				"pkg/b/b_test.go": "package b\n",
			},
			changedFiles: []string{"pkg/a/a_test.go", "pkg/b/b_test.go"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestRemoved, File: "pkg/a/a_test.go", Symbol: "TestFoo"},
			},
		},
		// Moving TestCases to a new sibling file while dropping table rows still triggers test_cases_reduced.
		{
			name: "cross-file move reducing table cases triggers test_cases_reduced",
			targetFiles: map[string]string{
				"pkg/svc/svc.go":      "package svc\n",
				"pkg/svc/svc_test.go": "package svc\n",
				"pkg/svc/svc_integration_test.go": `package svc

import "testing"

func TestCases(t *testing.T) {
	cases := []struct{ name string }{{name: "a"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {})
	}
}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/svc_test.go": `package svc

import "testing"

func TestCases(t *testing.T) {
	cases := []struct{ name string }{{name: "a"}, {name: "b"}, {name: "c"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {})
	}
}
`,
			},
			changedFiles: []string{"pkg/svc/svc_test.go", "pkg/svc/svc_integration_test.go"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestCasesReduced, File: "pkg/svc/svc_integration_test.go", Symbol: "TestCases"},
			},
		},
		// Moving TestAssert from deleted file into a new file while dropping cmp.Diff still triggers test_assertion_weakened.
		{
			name: "cross-file move from deleted file dropping cmp.Diff triggers test_assertion_weakened",
			targetFiles: map[string]string{
				"pkg/svc/svc.go": "package svc\n",
				"pkg/svc/svc_integration_test.go": `package svc

import "testing"

func TestAssert(t *testing.T) {
	run()
}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/old_test.go": `package svc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAssert(t *testing.T) {
	if d := cmp.Diff("ok", run()); d != "" {
		t.Errorf("mismatch: %s", d)
	}
}
`,
			},
			changedFiles: []string{"pkg/svc/svc_integration_test.go"},
			deletedFiles: []string{"pkg/svc/old_test.go"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_integration_test.go", Symbol: "TestAssert"},
			},
		},
		// Collapsing two same-named TestFoo definitions from two files into one target file leaves both ambiguous removals flagged.
		{
			name: "ambiguous 2-to-1 same-name collapse flags test_removed on both origins",
			targetFiles: map[string]string{
				"pkg/svc/svc.go":    "package svc\n",
				"pkg/svc/a_test.go": "package svc_test\n",
				"pkg/svc/z_test.go": "package svc\n",
				"pkg/svc/c_integration_test.go": `package svc_test

import "testing"

func TestFoo(t *testing.T) {}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/a_test.go": `package svc_test

import "testing"

func TestFoo(t *testing.T) {}
`,
				"pkg/svc/z_test.go": `package svc

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFoo(t *testing.T) {
	cases := []struct{ n int }{{n: 1}, {n: 2}, {n: 3}}
	for _, tc := range cases {
		t.Run("c", func(t *testing.T) {
			if d := cmp.Diff(tc.n, run()); d != "" {
				t.Errorf("mismatch: %s", d)
			}
		})
	}
}
`,
			},
			changedFiles: []string{"pkg/svc/a_test.go", "pkg/svc/z_test.go", "pkg/svc/c_integration_test.go"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestRemoved, File: "pkg/svc/a_test.go", Symbol: "TestFoo"},
				{Check: checkTestRemoved, File: "pkg/svc/z_test.go", Symbol: "TestFoo"},
			},
		},
		// Fanning out one TestFoo into two destination files rejects the ambiguous move and flags test_removed plus destination skips and sleeps.
		{
			name: "ambiguous 1-to-2 fan-out flags test_removed and clones zero skip/sleep credit",
			targetFiles: map[string]string{
				"pkg/svc/svc.go":    "package svc\n",
				"pkg/svc/a_test.go": "package svc\n",
				"pkg/svc/b.go":      "package svc\n",
				"pkg/svc/c.go":      "package svc\n",
				"pkg/svc/b_test.go": `package svc

import (
	"testing"
	"time"
)

func TestFoo(t *testing.T) {
	t.Skip("b")
	time.Sleep(time.Millisecond)
}
`,
				"pkg/svc/c_test.go": `package svc

import (
	"testing"
	"time"
)

func TestFoo(t *testing.T) {
	t.Skip("c")
	time.Sleep(time.Millisecond)
}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/a_test.go": `package svc

import (
	"testing"
	"time"
)

func TestFoo(t *testing.T) {
	t.Skip("orig")
	time.Sleep(time.Millisecond)
}
`,
			},
			changedFiles: []string{"pkg/svc/a_test.go", "pkg/svc/b_test.go", "pkg/svc/c_test.go"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestRemoved, File: "pkg/svc/a_test.go", Symbol: "TestFoo"},
				{Check: checkTestSkipAdded, File: "pkg/svc/b_test.go", Symbol: "TestFoo"},
				{Check: checkTestSleepAdded, File: "pkg/svc/b_test.go", Symbol: "TestFoo"},
				{Check: checkTestSkipAdded, File: "pkg/svc/c_test.go", Symbol: "TestFoo"},
				{Check: checkTestSleepAdded, File: "pkg/svc/c_test.go", Symbol: "TestFoo"},
			},
			wantFirstCands: []string{"TestFoo"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			writeRepoFiles(t, repoRoot, tc.targetFiles)
			req := rule.AdvisoryRequest{
				RepoRoot:     repoRoot,
				ChangedFiles: tc.changedFiles,
				DeletedFiles: tc.deletedFiles,
				ReadBaseFile: func(relPath string) ([]byte, bool, error) {
					s, ok := tc.baseFiles[relPath]
					return []byte(s), ok, nil
				},
			}

			items, _, err := NewProvider().Evaluate(context.Background(), req)
			if err != nil {
				t.Fatalf("Evaluate() unexpected error: %v", err)
			}
			assertAdvisorySnapshots(t, items, tc.want)
			if tc.wantFirstCands != nil {
				gotCands, _ := items[0].Evidence["renamed_candidates"].([]string)
				if diff := cmp.Diff(tc.wantFirstCands, gotCands); diff != "" {
					t.Errorf("renamed_candidates mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func writeRepoFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("MkdirAll() error: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile() error: %v", err)
		}
	}
}
