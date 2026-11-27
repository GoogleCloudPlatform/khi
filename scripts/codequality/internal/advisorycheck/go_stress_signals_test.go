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

// TestDetectGoStressReductions verifies const/var extraction, underscored integer literals, loop-participating 2-pass matching, >25% threshold, and non-loop false-positive exclusions.
func TestDetectGoStressReductions(t *testing.T) {
	testCases := []struct {
		name      string
		baseSrc   string
		targetSrc string
		changed   map[int]struct{}
		want      []advisoryCheckSnapshot
	}{
		// Replacing `const goroutines = 10000` with `goroutines := 256` triggers test_stress_reduced.
		{
			name: "const reduced to short-var triggers",
			baseSrc: `package svc

import "testing"

func TestConcurrentAccess(t *testing.T) {
	const goroutines = 10000
	for i := 0; i < goroutines; i++ {
		go func() {}()
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestConcurrentAccess(t *testing.T) {
	goroutines := 256
	if testing.Short() {
		goroutines = 32
	}
	for range goroutines {
		go func() {}()
	}
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestStressReduced, File: "pkg/svc/svc_test.go", Symbol: "TestConcurrentAccess"},
			},
		},
		// Underscored integer literal `10_000` is parsed for both stress-test detection and bound comparison.
		{
			name: "underscored literal 10_000 is parsed and triggers",
			baseSrc: `package svc

import "testing"

func TestUnderscoreStress(t *testing.T) {
	const iterations = 10_000
	for i := 0; i < iterations; i++ {
		_ = i
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestUnderscoreStress(t *testing.T) {
	iterations := 500
	for range iterations {
		_ = 1
	}
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestStressReduced, File: "pkg/svc/svc_test.go", Symbol: "TestUnderscoreStress"},
			},
		},
		// Converting two nested `for i < 100` / `for c < 100` loops into `attempts := 20` and `concurrency := 32` triggers twice via Pass 2.
		{
			name: "nested literal for loops converted to smaller loop vars triggers for both",
			baseSrc: `package svc

import "testing"

func TestNestedLoops(t *testing.T) {
	for i := 0; i < 100; i++ {
		for c := 0; c < 100; c++ {
			go func() {}()
		}
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestNestedLoops(t *testing.T) {
	attempts := 20
	concurrency := 32
	if testing.Short() {
		attempts = 5
		concurrency = 12
	}
	for range attempts {
		for range concurrency {
			go func() {}()
		}
	}
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestStressReduced, File: "pkg/svc/svc_test.go", Symbol: "TestNestedLoops"},
				{Check: checkTestStressReduced, File: "pkg/svc/svc_test.go", Symbol: "TestNestedLoops"},
			},
		},
		// Renaming `ThreadCount := 100` to `workerCount := 64` while keeping `ItemCount := 100` -> `keyCount := 100` triggers once via Pass 2.
		{
			name: "renamed loop var reduced from 100 to 64 triggers once",
			baseSrc: `package svc

import "testing"

func TestRenamedWorkerCount(t *testing.T) {
	ThreadCount := 100
	ItemCount := 100
	for tc := 0; tc < ThreadCount; tc++ {
		go func() {
			for ic := 0; ic < ItemCount; ic++ {
				_ = ic
			}
		}()
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestRenamedWorkerCount(t *testing.T) {
	workerCount := 64
	keyCount := 100
	if testing.Short() {
		workerCount = 16
		keyCount = 24
	}
	for range workerCount {
		go func() {
			for range keyCount {
				_ = 1
			}
		}()
	}
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestStressReduced, File: "pkg/svc/svc_test.go", Symbol: "TestRenamedWorkerCount"},
			},
		},
		// Reducing a bound from 100 to 75 (exact 25% boundary) remains silent.
		{
			name: "exact 25 percent reduction boundary 100 to 75 is silent",
			baseSrc: `package svc

import "testing"

func TestBoundary75(t *testing.T) {
	workers := 100
	for range workers {
		go func() {}()
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestBoundary75(t *testing.T) {
	workers := 75
	for range workers {
		go func() {}()
	}
}
`,
			want: nil,
		},
		// Reducing a bound from 100 to 74 (>25% reduction) triggers test_stress_reduced.
		{
			name: "over 25 percent reduction boundary 100 to 74 triggers",
			baseSrc: `package svc

import "testing"

func TestBoundary74(t *testing.T) {
	workers := 100
	for range workers {
		go func() {}()
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestBoundary74(t *testing.T) {
	workers := 74
	for range workers {
		go func() {}()
	}
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestStressReduced, File: "pkg/svc/svc_test.go", Symbol: "TestBoundary74"},
			},
		},
		// Scaling down inside `if testing.Short()` while preserving full-strength defaults remains silent.
		{
			name: "scaling inside testing.Short remains silent",
			baseSrc: `package svc

import "testing"

func TestShortScaling(t *testing.T) {
	const workers = 100
	for i := 0; i < workers; i++ {
		go func() {}()
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestShortScaling(t *testing.T) {
	workers := 100
	if testing.Short() {
		workers = 10
	}
	for range workers {
		go func() {}()
	}
}
`,
			want: nil,
		},
		// Renaming loop variables while preserving `100 -> 100` remains silent.
		{
			name: "rename preserving 100 to 100 remains silent",
			baseSrc: `package svc

import "testing"

func TestRenamePreserved(t *testing.T) {
	ThreadCount := 100
	for tc := 0; tc < ThreadCount; tc++ {
		go func() {}()
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestRenamePreserved(t *testing.T) {
	workerCount := 100
	for range workerCount {
		go func() {}()
	}
}
`,
			want: nil,
		},
		// Non-loop integer constants/vars/fields (delayMs, 5*time.Second, want: 100) do not cross-pair or false-positive.
		{
			name: "non-loop ints do not trigger false positives",
			baseSrc: `package svc

import (
	"testing"
	"time"
)

func TestNonLoopInts(t *testing.T) {
	delayMs := 1000
	_ = delayMs
	_ = 5 * time.Second
	res := struct{ want int }{want: 100}
	_ = res
	for i := 0; i < 100; i++ {
		go func() {}()
	}
}
`,
			targetSrc: `package svc

import (
	"testing"
	"time"
)

func TestNonLoopInts(t *testing.T) {
	timeoutMs := 50
	_ = timeoutMs
	_ = 5 * time.Second
	res := struct{ want int }{want: 100}
	_ = res
	workers := 100
	for range workers {
		go func() {}()
	}
}
`,
			want: nil,
		},
		// Multi-LHS assignment reduction and duplicate range-over-int literal reduction both trigger.
		{
			name: "multi-LHS assign and range literal reduction trigger",
			baseSrc: `package svc

import "testing"

func TestMultiAssignAndRangeLit(t *testing.T) {
	a, b := 100, 200
	_ = a + b
	for range 100 {
		go func() {}()
	}
	for range 100 {
		go func() {}()
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestMultiAssignAndRangeLit(t *testing.T) {
	a, b := 100, 50
	_ = a + b
	for range 100 {
		go func() {}()
	}
	for range 20 {
		go func() {}()
	}
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestStressReduced, File: "pkg/svc/svc_test.go", Symbol: "TestMultiAssignAndRangeLit"},
				{Check: checkTestStressReduced, File: "pkg/svc/svc_test.go", Symbol: "TestMultiAssignAndRangeLit"},
			},
		},
		// Reducing a `<=` or reversed `>` loop bound triggers test_stress_reduced.
		{
			name: "for-loop comparison variants <= and reversed > trigger",
			baseSrc: `package svc

import "testing"

func TestForOpVariants(t *testing.T) {
	for i := 1; i <= 100; i++ {
		go func() {}()
	}
	for j := 0; 100 > j; j++ {
		go func() {}()
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestForOpVariants(t *testing.T) {
	for i := 1; i <= 50; i++ {
		go func() {}()
	}
	for j := 0; 50 > j; j++ {
		go func() {}()
	}
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestStressReduced, File: "pkg/svc/svc_test.go", Symbol: "TestForOpVariants"},
				{Check: checkTestStressReduced, File: "pkg/svc/svc_test.go", Symbol: "TestForOpVariants"},
			},
		},
		// Preserving large 64-bit literal 0x2000000000000000 across a loop-var rename avoids signed multiplication overflow false positives.
		{
			name: "large 64-bit literal unchanged across rename remains silent",
			baseSrc: `package svc

import "testing"

func TestLargeLiteralUnchanged(t *testing.T) {
	workers := 0x2000000000000000
	for range workers {
		go func() {}()
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestLargeLiteralUnchanged(t *testing.T) {
	limit := 0x2000000000000000
	for range limit {
		go func() {}()
	}
}
`,
			want: nil,
		},
		// Reducing a large 64-bit literal 0x3000000000000000 to 10 triggers without signed multiplication overflow false negatives.
		{
			name: "large 64-bit literal reduced to 10 triggers",
			baseSrc: `package svc

import "testing"

func TestLargeLiteralReduced(t *testing.T) {
	workers := 0x3000000000000000
	for range workers {
		go func() {}()
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestLargeLiteralReduced(t *testing.T) {
	workers := 10
	for range workers {
		go func() {}()
	}
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestStressReduced, File: "pkg/svc/svc_test.go", Symbol: "TestLargeLiteralReduced"},
			},
		},
		// Bounds below the minimum threshold of 10 (9 -> 2) remain silent even with >25% reduction.
		{
			name: "bound below 10 floor 9 to 2 remains silent",
			baseSrc: `package svc

import "testing"

func TestBelowFloor(t *testing.T) {
	workers := 9
	for range workers {
		go func() {}()
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestBelowFloor(t *testing.T) {
	workers := 2
	for range workers {
		go func() {}()
	}
}
`,
			want: nil,
		},
		// Bounds at the minimum threshold of 10 (10 -> 5) trigger test_stress_reduced.
		{
			name: "bound at 10 floor 10 to 5 triggers",
			baseSrc: `package svc

import "testing"

func TestAtFloor(t *testing.T) {
	workers := 10
	for range workers {
		go func() {}()
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestAtFloor(t *testing.T) {
	workers := 5
	for range workers {
		go func() {}()
	}
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestStressReduced, File: "pkg/svc/svc_test.go", Symbol: "TestAtFloor"},
			},
		},
		// Inserting a small warmup `for range 4` before an unchanged `for range 100` remains silent.
		{
			name: "inserting small range loop before range 100 remains silent",
			baseSrc: `package svc

import "testing"

func TestWarmupRangeInserted(t *testing.T) {
	for range 100 {
		go func() {}()
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestWarmupRangeInserted(t *testing.T) {
	for range 4 {
		_ = 1
	}
	for range 100 {
		go func() {}()
	}
}
`,
			want: nil,
		},
		// Modernizing `for i < 100` to `workers := 100; for range workers` alongside an unchanged `for i < 5` does not mis-pair `for:i(100)` with `for:i(5)`.
		{
			name: "modernizing one of two for-i loops alongside unchanged smaller for-i remains silent",
			baseSrc: `package svc

import "testing"

func TestModernizeDuplicateForKey(t *testing.T) {
	for i := 0; i < 100; i++ {
		go func() {}()
	}
	for i := 0; i < 5; i++ {
		_ = i
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestModernizeDuplicateForKey(t *testing.T) {
	workers := 100
	for range workers {
		go func() {}()
	}
	for i := 0; i < 5; i++ {
		_ = i
	}
}
`,
			want: nil,
		},
		// Full-strength bound restored in the else branch of `if testing.Short()` remains silent while the short branch `for range 10` is skipped.
		{
			name: "testing.Short else branch restores full strength without false positive",
			baseSrc: `package svc

import "testing"

func TestShortElseBranch(t *testing.T) {
	for range 100 {
		go func() {}()
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestShortElseBranch(t *testing.T) {
	workers := 10
	if testing.Short() {
		for range 10 {
			go func() {}()
		}
	} else {
		workers = 100
	}
	for range workers {
		go func() {}()
	}
}
`,
			want: nil,
		},
		// Non-stress tests without go statements, WaitGroup, or literals >= 50 remain silent on loop reductions.
		{
			name: "non-stress test with small bounds remains silent",
			baseSrc: `package svc

import "testing"

func TestNonStressLoop(t *testing.T) {
	items := 20
	for range items {
		_ = 1
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestNonStressLoop(t *testing.T) {
	items := 10
	for range items {
		_ = 1
	}
}
`,
			want: nil,
		},
		// Exact Pass-1 matched bound (workers=200) is excluded from Pass-2 pairing with a slightly reduced renamed bound (items=100 -> entries=80).
		{
			name: "mixed Pass-1 exact match does not leak into Pass-2 pairing",
			baseSrc: `package svc

import "testing"

func TestMixedPass1AndPass2Silent(t *testing.T) {
	workers := 200
	for range workers {
		go func() {}()
	}
	items := 100
	for range items {
		_ = 1
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestMixedPass1AndPass2Silent(t *testing.T) {
	workers := 200
	for range workers {
		go func() {}()
	}
	entries := 80
	for range entries {
		_ = 1
	}
}
`,
			want: nil,
		},
		// Exact Pass-1 match (workers=200) alongside renamed Pass-2 reduction (items=100 -> entries=40) triggers once.
		{
			name: "mixed Pass-1 exact match and Pass-2 renamed reduction triggers once",
			baseSrc: `package svc

import "testing"

func TestMixedPass1AndPass2Reduced(t *testing.T) {
	workers := 200
	for range workers {
		go func() {}()
	}
	items := 100
	for range items {
		_ = 1
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestMixedPass1AndPass2Reduced(t *testing.T) {
	workers := 200
	for range workers {
		go func() {}()
	}
	entries := 40
	for range entries {
		_ = 1
	}
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestStressReduced, File: "pkg/svc/svc_test.go", Symbol: "TestMixedPass1AndPass2Reduced"},
			},
		},
		// Plain `=` assignment reduction (var workers int; workers = 100 -> 50) triggers test_stress_reduced.
		{
			name: "plain assignment reduction triggers",
			baseSrc: `package svc

import "testing"

func TestPlainAssign(t *testing.T) {
	var workers int
	workers = 100
	for range workers {
		go func() {}()
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestPlainAssign(t *testing.T) {
	var workers int
	workers = 50
	for range workers {
		go func() {}()
	}
}
`,
			want: []advisoryCheckSnapshot{
				{Check: checkTestStressReduced, File: "pkg/svc/svc_test.go", Symbol: "TestPlainAssign"},
			},
		},
		// Pure-deletion stress reduction still triggers even when ChangedLines only contains an added line in another test function.
		{
			name: "pure deletion stress reduction triggers despite unrelated ChangedLines",
			baseSrc: `package svc
import "testing"
func TestStressPureDelete(t *testing.T) {
	workers := 10
	workers = 1000
	for range workers { go func() {}() }
}
func TestOther(t *testing.T) {}
`,
			targetSrc: `package svc
import "testing"
func TestStressPureDelete(t *testing.T) {
	workers := 10
	for range workers { go func() {}() }
}
func TestOther(t *testing.T) {
	// unrelated addition
}
`,
			changed: map[int]struct{}{8: {}},
			want: []advisoryCheckSnapshot{
				{Check: checkTestStressReduced, File: "pkg/svc/svc_test.go", Symbol: "TestStressPureDelete"},
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

// TestDetectGoStressReductionEvidence verifies Summary, Evidence, and ReviewPrompt fields for exact and renamed bound reductions.
func TestDetectGoStressReductionEvidence(t *testing.T) {
	testCases := []struct {
		name         string
		baseSrc      string
		targetSrc    string
		wantSummary  string
		wantEvidence map[string]any
	}{
		// Exact-key reduction formats summary without arrow and omits target_bound_key from Evidence.
		{
			name: "exact bound reduction metadata",
			baseSrc: `package svc

import "testing"

func TestExactMeta(t *testing.T) {
	workers := 100
	for range workers {
		go func() {}()
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestExactMeta(t *testing.T) {
	workers := 50
	for range workers {
		go func() {}()
	}
}
`,
			wantSummary: "stress bound var:workers reduced from 100 to 50 outside testing.Short()",
			wantEvidence: map[string]any{
				"bound_key": "var:workers",
				"before":    100,
				"after":     50,
			},
		},
		// Renamed bound reduction formats summary with arrow and includes target_bound_key in Evidence.
		{
			name: "renamed bound reduction metadata",
			baseSrc: `package svc

import "testing"

func TestRenamedMeta(t *testing.T) {
	workers := 100
	for range workers {
		go func() {}()
	}
}
`,
			targetSrc: `package svc

import "testing"

func TestRenamedMeta(t *testing.T) {
	concurrency := 40
	for range concurrency {
		go func() {}()
	}
}
`,
			wantSummary: "stress bound var:workers -> var:concurrency reduced from 100 to 40 outside testing.Short()",
			wantEvidence: map[string]any{
				"bound_key":        "var:workers",
				"target_bound_key": "var:concurrency",
				"before":           100,
				"after":            40,
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
			req := rule.AdvisoryRequest{
				RepoRoot:     repoRoot,
				ChangedFiles: []string{"pkg/svc/svc_test.go"},
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
			if len(items) != 1 {
				t.Fatalf("Evaluate() got %d items, want 1: %+v", len(items), items)
			}
			if items[0].Summary != tc.wantSummary {
				t.Errorf("Summary = %q, want %q", items[0].Summary, tc.wantSummary)
			}
			if diff := cmp.Diff(tc.wantEvidence, items[0].Evidence); diff != "" {
				t.Errorf("Evidence mismatch (-want +got):\n%s", diff)
			}
			if items[0].ReviewPrompt == "" {
				t.Errorf("ReviewPrompt = %q, want non-empty prompt", items[0].ReviewPrompt)
			}
		})
	}
}
