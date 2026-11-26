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
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// TestAnalyzeGoStressReducedAndSleepAdded verifies test_stress_reduced (>25% reduction outside testing.Short()) and test_sleep_added.
func TestAnalyzeGoStressReducedAndSleepAdded(t *testing.T) {
	testCases := []struct {
		name      string
		baseSrc   string
		targetSrc string
		changed   map[int]struct{}
		want      []advisoryCheckSnapshot
	}{
		// Moving short-mode scaling into `if testing.Short()` while preserving full-strength defaults produces zero findings.
		{
			name: "short mode scaling preserves full strength",
			baseSrc: `package svc

import "testing"

func TestStress(t *testing.T) {
	attempts := 1000
	go func() { _ = attempts }()
}
`,
			targetSrc: `package svc

import "testing"

func TestStress(t *testing.T) {
	attempts := 1000
	if testing.Short() {
		attempts = 10
	}
	go func() { _ = attempts }()
}
`,
			want: nil,
		},
		// Cutting full-strength attempts from 1000 to 10 outside testing.Short() triggers test_stress_reduced.
		{
			name: "cutting full strength iterations triggers test_stress_reduced",
			baseSrc: `package svc

import "testing"

func TestStress(t *testing.T) {
	attempts := 1000
	go func() { _ = attempts }()
}
`,
			targetSrc: `package svc

import "testing"

func TestStress(t *testing.T) {
	attempts := 10
	go func() { _ = attempts }()
}
`,
			changed: map[int]struct{}{6: {}},
			want: []advisoryCheckSnapshot{
				{Check: checkTestStressReduced, File: "pkg/svc/svc_test.go", Symbol: "TestStress"},
			},
		},
		// Adding time.Sleep on a changed test line triggers test_sleep_added.
		{
			name: "new time.Sleep triggers test_sleep_added",
			baseSrc: `package svc

import "testing"

func TestAsync(t *testing.T) {}
`,
			targetSrc: `package svc

import (
	"testing"
	"time"
)

func TestAsync(t *testing.T) {
	time.Sleep(10 * time.Millisecond)
}
`,
			changed: map[int]struct{}{9: {}},
			want: []advisoryCheckSnapshot{
				{Check: checkTestSleepAdded, File: "pkg/svc/svc_test.go", Symbol: "TestAsync"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			writeRepoFiles(t, repoRoot, map[string]string{
				"pkg/svc/svc.go":      "package svc\n",
				"pkg/svc/svc_test.go": tc.targetSrc,
			})
			var changedLines map[string]map[int]struct{}
			if tc.changed != nil {
				changedLines = map[string]map[int]struct{}{"pkg/svc/svc_test.go": tc.changed}
			}
			req := rule.AdvisoryRequest{
				RepoRoot:     repoRoot,
				ChangedFiles: []string{"pkg/svc/svc_test.go"},
				ChangedLines: changedLines,
				ReadBaseFile: func(rel string) ([]byte, bool, error) {
					if rel == "pkg/svc/svc_test.go" {
						return []byte(tc.baseSrc), true, nil
					}
					return nil, false, nil
				},
			}

			items, _, err := NewProvider().Evaluate(context.Background(), req)
			if err != nil {
				t.Fatalf("Evaluate() unexpected error: %v", err)
			}
			assertAdvisorySnapshots(t, items, tc.want)
		})
	}
}

// TestAnalyzeGoSuppressionAddedAndOrphanTestFile verifies real-comment suppression_added (ignoring string literals) and orphan_test_file exclusions.
func TestAnalyzeGoSuppressionAddedAndOrphanTestFile(t *testing.T) {
	testCases := []struct {
		name         string
		targetFiles  map[string]string
		baseFiles    map[string]string
		changedFiles []string
		changedLines map[string]map[int]struct{}
		want         []advisoryCheckSnapshot
	}{
		// String literal mentioning CODEQUALITY_ALLOW or nolint produces zero suppression_added findings.
		{
			name: "string literal does not trigger suppression_added",
			targetFiles: map[string]string{
				"pkg/svc/svc.go": `package svc

const msg = "// CODEQUALITY_ALLOW(func_cognitive, dispatch): literal\n//nolint:all"
`,
			},
			baseFiles:    map[string]string{"pkg/svc/svc.go": "package svc\n"},
			changedFiles: []string{"pkg/svc/svc.go"},
			changedLines: map[string]map[int]struct{}{"pkg/svc/svc.go": {3: {}}},
			want:         nil,
		},
		// Real comment containing CODEQUALITY_ALLOW triggers suppression_added.
		{
			name: "real comment triggers suppression_added",
			targetFiles: map[string]string{
				"pkg/svc/svc.go": `package svc

// CODEQUALITY_ALLOW(func_cognitive, dispatch): flat switch
func F() {}
`,
			},
			baseFiles:    map[string]string{"pkg/svc/svc.go": "package svc\n"},
			changedFiles: []string{"pkg/svc/svc.go"},
			changedLines: map[string]map[int]struct{}{"pkg/svc/svc.go": {3: {}}},
			want: []advisoryCheckSnapshot{
				{Check: checkSuppressionAdded, File: "pkg/svc/svc.go", Symbol: "<comment>"},
			},
		},
		// Ordinary prose comments mentioning CODEQUALITY_ALLOW or nolint produce zero suppression_added findings.
		{
			name: "ordinary prose comments do not trigger suppression_added",
			targetFiles: map[string]string{
				"pkg/svc/svc.go": `package svc

// Verify pipeline ignores CODEQUALITY_ALLOW directives during fast checks.
// CODEQUALITY_ALLOW rules forbid naked //nolint comments in source files.
// nolint is never allowed; avoid nolint:all in prose too.
/* Mentioning CODEQUALITY_ALLOW and nolint in a block comment is safe. */
func F() {}
`,
			},
			baseFiles:    map[string]string{"pkg/svc/svc.go": "package svc\n"},
			changedFiles: []string{"pkg/svc/svc.go"},
			changedLines: map[string]map[int]struct{}{"pkg/svc/svc.go": {3: {}, 4: {}, 5: {}, 6: {}}},
			want:         nil,
		},
		// Valid and naked nolint directives as well as colon-form CODEQUALITY_ALLOW trigger suppression_added.
		{
			name: "valid and naked nolint directives trigger suppression_added",
			targetFiles: map[string]string{
				"pkg/svc/svc.go": `package svc

//nolint
//nolint:errcheck // justified
// CODEQUALITY_ALLOW: colon form
/* nolint:all */
///nolint:errcheck
// nolint  // reason
//nolint naked reason
func F() {}
`,
			},
			baseFiles:    map[string]string{"pkg/svc/svc.go": "package svc\n"},
			changedFiles: []string{"pkg/svc/svc.go"},
			changedLines: map[string]map[int]struct{}{"pkg/svc/svc.go": {3: {}, 4: {}, 5: {}, 6: {}, 7: {}, 8: {}, 9: {}}},
			want: []advisoryCheckSnapshot{
				{Check: checkSuppressionAdded, File: "pkg/svc/svc.go", Symbol: "<comment>"},
				{Check: checkSuppressionAdded, File: "pkg/svc/svc.go", Symbol: "<comment>"},
				{Check: checkSuppressionAdded, File: "pkg/svc/svc.go", Symbol: "<comment>"},
				{Check: checkSuppressionAdded, File: "pkg/svc/svc.go", Symbol: "<comment>"},
				{Check: checkSuppressionAdded, File: "pkg/svc/svc.go", Symbol: "<comment>"},
				{Check: checkSuppressionAdded, File: "pkg/svc/svc.go", Symbol: "<comment>"},
				{Check: checkSuppressionAdded, File: "pkg/svc/svc.go", Symbol: "<comment>"},
			},
		},
		// New orphan *_test.go without sibling *.go triggers orphan_test_file while *_contract_test.go and interface_test.go are excluded.
		{
			name: "orphan test file triggers while contract and interface tests are excluded",
			targetFiles: map[string]string{
				"pkg/svc/extra_test.go":           "package svc\n",
				"pkg/svc/foo_contract_test.go":    "package svc\n",
				"pkg/svc/interface_test.go":       "package svc\n",
				"pkg/svc/paired.go":               "package svc\n",
				"pkg/svc/paired_test.go":          "package svc\n",
				"pkg/svc/foo_integration_test.go": "package svc\n",
			},
			baseFiles: map[string]string{},
			changedFiles: []string{
				"pkg/svc/extra_test.go",
				"pkg/svc/foo_contract_test.go",
				"pkg/svc/interface_test.go",
				"pkg/svc/paired_test.go",
				"pkg/svc/foo_integration_test.go",
			},
			want: []advisoryCheckSnapshot{
				{Check: checkOrphanTestFile, File: "pkg/svc/extra_test.go", Symbol: "extra_test.go"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			writeRepoFiles(t, repoRoot, tc.targetFiles)
			req := rule.AdvisoryRequest{
				RepoRoot:     repoRoot,
				ChangedFiles: tc.changedFiles,
				ChangedLines: tc.changedLines,
				ReadBaseFile: func(rel string) ([]byte, bool, error) {
					s, ok := tc.baseFiles[rel]
					return []byte(s), ok, nil
				},
			}

			items, _, err := NewProvider().Evaluate(context.Background(), req)
			if err != nil {
				t.Fatalf("Evaluate() unexpected error: %v", err)
			}
			assertAdvisorySnapshots(t, items, tc.want)
		})
	}
}

// TestCrossFileGoTestMoveSignals verifies cross-file move baselines for test_skip_added, test_sleep_added, discard checks, and test_stress_reduced.
func TestCrossFileGoTestMoveSignals(t *testing.T) {
	testCases := []struct {
		name         string
		targetFiles  map[string]string
		baseFiles    map[string]string
		changedFiles []string
		changedLines map[string]map[int]struct{}
		deletedFiles []string
		want         []advisoryCheckSnapshot
		wantBefore   map[string]int
	}{
		// Moving a live test gated on RUN_LIVE_ESTIMATION_TESTS verbatim to *_integration_test.go emits zero test_skip_added findings.
		{
			name: "verbatim move of env-gated skip test emits zero test_skip_added",
			targetFiles: map[string]string{
				"pkg/svc/svc.go":      "package svc\n",
				"pkg/svc/svc_test.go": "package svc\n",
				"pkg/svc/svc_integration_test.go": `package svc

import (
	"os"
	"testing"
)

func TestLiveEstimation(t *testing.T) {
	if os.Getenv("RUN_LIVE_ESTIMATION_TESTS") == "" {
		t.Skip("live estimation disabled")
	}
}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/svc_test.go": `package svc

import (
	"os"
	"testing"
)

func TestLiveEstimation(t *testing.T) {
	if os.Getenv("RUN_LIVE_ESTIMATION_TESTS") == "" {
		t.Skip("live estimation disabled")
	}
}
`,
			},
			changedFiles: []string{"pkg/svc/svc_test.go", "pkg/svc/svc_integration_test.go"},
			want:         nil,
		},
		// Moving a test with an existing skip while adding a second skip in the moved function still fires test_skip_added.
		{
			name: "adding new skip inside moved test still triggers test_skip_added",
			targetFiles: map[string]string{
				"pkg/svc/svc.go":      "package svc\n",
				"pkg/svc/svc_test.go": "package svc\n",
				"pkg/svc/svc_integration_test.go": `package svc

import (
	"os"
	"testing"
)

func TestLiveEstimation(t *testing.T) {
	if os.Getenv("RUN_LIVE_ESTIMATION_TESTS") == "" {
		t.Skip("live estimation disabled")
	}
	t.Skip("extra flaky skip")
}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/svc_test.go": `package svc

import (
	"os"
	"testing"
)

func TestLiveEstimation(t *testing.T) {
	if os.Getenv("RUN_LIVE_ESTIMATION_TESTS") == "" {
		t.Skip("live estimation disabled")
	}
}
`,
			},
			changedFiles: []string{"pkg/svc/svc_test.go", "pkg/svc/svc_integration_test.go"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestSkipAdded, File: "pkg/svc/svc_integration_test.go", Symbol: "TestLiveEstimation"},
				{Check: checkTestSkipAdded, File: "pkg/svc/svc_integration_test.go", Symbol: "TestLiveEstimation"},
			},
		},
		// Moving TestLive verbatim alongside a brand-new skipped test flags only the new test.
		{
			name: "verbatim moved skip test alongside new skipped test flags only new test",
			targetFiles: map[string]string{
				"pkg/svc/svc.go":      "package svc\n",
				"pkg/svc/svc_test.go": "package svc\n",
				"pkg/svc/svc_integration_test.go": `package svc

import (
	"os"
	"testing"
)

func TestLiveEstimation(t *testing.T) {
	if os.Getenv("RUN_LIVE_ESTIMATION_TESTS") == "" {
		t.Skip("live estimation disabled")
	}
}

func TestBrandNewSkip(t *testing.T) {
	t.Skip("flaky")
}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/svc_test.go": `package svc

import (
	"os"
	"testing"
)

func TestLiveEstimation(t *testing.T) {
	if os.Getenv("RUN_LIVE_ESTIMATION_TESTS") == "" {
		t.Skip("live estimation disabled")
	}
}
`,
			},
			changedFiles: []string{"pkg/svc/svc_test.go", "pkg/svc/svc_integration_test.go"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestSkipAdded, File: "pkg/svc/svc_integration_test.go", Symbol: "TestBrandNewSkip"},
			},
		},
		// Moving a test with time.Sleep verbatim across files emits zero test_sleep_added findings.
		{
			name: "verbatim move of time.Sleep test emits zero test_sleep_added",
			targetFiles: map[string]string{
				"pkg/svc/svc.go":      "package svc\n",
				"pkg/svc/svc_test.go": "package svc\n",
				"pkg/svc/svc_integration_test.go": `package svc

import (
	"testing"
	"time"
)

func TestTTL(t *testing.T) {
	time.Sleep(10 * time.Millisecond)
}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/svc_test.go": `package svc

import (
	"testing"
	"time"
)

func TestTTL(t *testing.T) {
	time.Sleep(10 * time.Millisecond)
}
`,
			},
			changedFiles: []string{"pkg/svc/svc_test.go", "pkg/svc/svc_integration_test.go"},
			want:         nil,
		},
		// Moving a test with time.Sleep while adding a second time.Sleep still triggers test_sleep_added.
		{
			name: "adding new time.Sleep in moved test triggers test_sleep_added",
			targetFiles: map[string]string{
				"pkg/svc/svc.go":      "package svc\n",
				"pkg/svc/svc_test.go": "package svc\n",
				"pkg/svc/svc_integration_test.go": `package svc

import (
	"testing"
	"time"
)

func TestTTL(t *testing.T) {
	time.Sleep(10 * time.Millisecond)
	time.Sleep(20 * time.Millisecond)
}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/svc_test.go": `package svc

import (
	"testing"
	"time"
)

func TestTTL(t *testing.T) {
	time.Sleep(10 * time.Millisecond)
}
`,
			},
			changedFiles: []string{"pkg/svc/svc_test.go", "pkg/svc/svc_integration_test.go"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestSleepAdded, File: "pkg/svc/svc_integration_test.go", Symbol: "TestTTL"},
				{Check: checkTestSleepAdded, File: "pkg/svc/svc_integration_test.go", Symbol: "TestTTL"},
			},
		},
		// Moving a test with an existing `_ = os.RemoveAll(dir)` discard verbatim preserves one call while adding a second `_ = os.RemoveAll(...)` and `_ = runCritical()` flags both additions.
		{
			name: "verbatim discard preserved and new discard flagged in moved test",
			targetFiles: map[string]string{
				"pkg/svc/svc.go":      "package svc\n",
				"pkg/svc/svc_test.go": "package svc\n",
				"pkg/svc/svc_integration_test.go": `package svc

import (
	"os"
	"testing"
)

func TestCleanup(t *testing.T) {
	_ = os.RemoveAll("/tmp/x")
	_ = os.RemoveAll("/tmp/y")
	_ = runCritical()
}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/svc_test.go": `package svc

import (
	"os"
	"testing"
)

func TestCleanup(t *testing.T) {
	_ = os.RemoveAll("/tmp/x")
}
`,
			},
			changedFiles: []string{"pkg/svc/svc_test.go", "pkg/svc/svc_integration_test.go"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_integration_test.go", Symbol: "TestCleanup"},
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_integration_test.go", Symbol: "TestCleanup"},
			},
		},
		// Moving a stress test across files while reducing iterations from 1000 to 10 outside testing.Short() still triggers test_stress_reduced.
		{
			name: "cross-file move reducing stress iterations triggers test_stress_reduced",
			targetFiles: map[string]string{
				"pkg/svc/svc.go":      "package svc\n",
				"pkg/svc/svc_test.go": "package svc\n",
				"pkg/svc/svc_integration_test.go": `package svc

import "testing"

func TestStressMoved(t *testing.T) {
	workers := 10
	for range workers {
		go func() {}()
	}
}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/svc_test.go": `package svc

import "testing"

func TestStressMoved(t *testing.T) {
	workers := 1000
	for range workers {
		go func() {}()
	}
}
`,
			},
			changedFiles: []string{"pkg/svc/svc_test.go", "pkg/svc/svc_integration_test.go"},
			want: []advisoryCheckSnapshot{
				{Check: checkTestStressReduced, File: "pkg/svc/svc_integration_test.go", Symbol: "TestStressMoved"},
			},
		},
		// Moving TestLive out of svc_test.go debits its skip, sleep, and discard from svc_test.go so adding them to TestUnit fires all three.
		{
			name: "origin debit catches skip sleep and discard added to TestUnit after moving TestLive out",
			targetFiles: map[string]string{
				"pkg/svc/svc.go": "package svc\n",
				"pkg/svc/svc_test.go": `package svc

import (
	"testing"
	"time"
)

func TestUnit(t *testing.T) {
	t.Skip("flaky")
	time.Sleep(5 * time.Second)
	_ = work()
}
`,
				"pkg/svc/svc_integration_test.go": `package svc

import (
	"testing"
	"time"
)

func TestLive(t *testing.T) {
	t.Skip("live")
	time.Sleep(5 * time.Second)
	_ = work()
}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/svc_test.go": `package svc

import (
	"testing"
	"time"
)

func TestUnit(t *testing.T) {}

func TestLive(t *testing.T) {
	t.Skip("live")
	time.Sleep(5 * time.Second)
	_ = work()
}
`,
			},
			changedFiles: []string{"pkg/svc/svc_test.go", "pkg/svc/svc_integration_test.go"},
			changedLines: map[string]map[int]struct{}{
				"pkg/svc/svc_test.go":             {9: {}, 10: {}, 11: {}},
				"pkg/svc/svc_integration_test.go": {8: {}, 9: {}, 10: {}, 11: {}},
			},
			want: []advisoryCheckSnapshot{
				{Check: checkTestSkipAdded, File: "pkg/svc/svc_test.go", Symbol: "TestUnit"},
				{Check: checkTestSleepAdded, File: "pkg/svc/svc_test.go", Symbol: "TestUnit"},
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_test.go", Symbol: "TestUnit"},
			},
			wantBefore: map[string]int{checkTestSkipAdded: 0, checkTestSleepAdded: 0},
		},
		// Stripping skip, sleep, and discard from moved TestLive cannot transfer credit to TestBrandNew in the destination file.
		{
			name: "destination credit scoping catches skip sleep and discard in TestBrandNew when stripped from moved TestLive",
			targetFiles: map[string]string{
				"pkg/svc/svc.go":      "package svc\n",
				"pkg/svc/svc_test.go": "package svc\n",
				"pkg/svc/svc_integration_test.go": `package svc

import (
	"testing"
	"time"
)

func TestLive(t *testing.T) {}

func TestBrandNew(t *testing.T) {
	t.Skip("flaky")
	time.Sleep(5 * time.Second)
	_ = work()
}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/svc_test.go": `package svc

import (
	"testing"
	"time"
)

func TestLive(t *testing.T) {
	t.Skip("live")
	time.Sleep(5 * time.Second)
	_ = work()
}
`,
			},
			changedFiles: []string{"pkg/svc/svc_test.go", "pkg/svc/svc_integration_test.go"},
			changedLines: map[string]map[int]struct{}{
				"pkg/svc/svc_integration_test.go": {11: {}, 12: {}, 13: {}},
			},
			want: []advisoryCheckSnapshot{
				{Check: checkTestSkipAdded, File: "pkg/svc/svc_integration_test.go", Symbol: "TestBrandNew"},
				{Check: checkTestSleepAdded, File: "pkg/svc/svc_integration_test.go", Symbol: "TestBrandNew"},
				{Check: checkTestAssertionWeakened, File: "pkg/svc/svc_integration_test.go", Symbol: "TestBrandNew"},
			},
			wantBefore: map[string]int{checkTestSkipAdded: 0, checkTestSleepAdded: 0},
		},
		// Moving a skip+sleep test verbatim into an existing file with pre-existing skip+sleep alongside a new skip+sleep flags only the new test with before_count=2.
		{
			name: "move skip and sleep into existing file with new sleep and skip preserves before_count and skips untouched and moved",
			targetFiles: map[string]string{
				"pkg/svc/svc.go":      "package svc\n",
				"pkg/svc/svc_test.go": "package svc\n",
				"pkg/svc/svc_integration_test.go": `package svc

import (
	"testing"
	"time"
)

func TestExisting(t *testing.T) {
	t.Skip("existing")
	time.Sleep(time.Millisecond)
}

func TestMoved(t *testing.T) {
	t.Skip("moved")
	time.Sleep(time.Millisecond)
}

func TestAdded(t *testing.T) {
	t.Skip("new")
	time.Sleep(time.Millisecond)
}
`,
			},
			baseFiles: map[string]string{
				"pkg/svc/svc_test.go": `package svc

import (
	"testing"
	"time"
)

func TestMoved(t *testing.T) {
	t.Skip("moved")
	time.Sleep(time.Millisecond)
}
`,
				"pkg/svc/svc_integration_test.go": `package svc

import (
	"testing"
	"time"
)

func TestExisting(t *testing.T) {
	t.Skip("existing")
	time.Sleep(time.Millisecond)
}
`,
			},
			changedFiles: []string{"pkg/svc/svc_test.go", "pkg/svc/svc_integration_test.go"},
			changedLines: map[string]map[int]struct{}{
				"pkg/svc/svc_integration_test.go": {14: {}, 15: {}, 19: {}, 20: {}},
			},
			want: []advisoryCheckSnapshot{
				{Check: checkTestSkipAdded, File: "pkg/svc/svc_integration_test.go", Symbol: "TestAdded"},
				{Check: checkTestSleepAdded, File: "pkg/svc/svc_integration_test.go", Symbol: "TestAdded"},
			},
			wantBefore: map[string]int{checkTestSkipAdded: 2, checkTestSleepAdded: 2},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			writeRepoFiles(t, repoRoot, tc.targetFiles)
			req := rule.AdvisoryRequest{
				RepoRoot:     repoRoot,
				ChangedFiles: tc.changedFiles,
				ChangedLines: tc.changedLines,
				DeletedFiles: tc.deletedFiles,
				ReadBaseFile: func(rel string) ([]byte, bool, error) {
					s, ok := tc.baseFiles[rel]
					return []byte(s), ok, nil
				},
			}

			items, _, err := NewProvider().Evaluate(context.Background(), req)
			if err != nil {
				t.Fatalf("Evaluate() unexpected error: %v", err)
			}
			assertAdvisorySnapshots(t, items, tc.want)
			if tc.wantBefore != nil {
				assertBeforeCounts(t, items, tc.wantBefore)
			}
		})
	}
}

func assertBeforeCounts(t *testing.T, items []rule.QAItem, wantBefore map[string]int) {
	t.Helper()
	got := make(map[string]int)
	for _, item := range items {
		if _, ok := wantBefore[item.Check]; ok {
			got[item.Check], _ = item.Evidence["before_count"].(int)
		}
	}
	if diff := cmp.Diff(wantBefore, got); diff != "" {
		t.Errorf("before_count mismatch (-want +got):\n%s", diff)
	}
}
