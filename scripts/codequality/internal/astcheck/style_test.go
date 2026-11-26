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

package astcheck

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// TestAnalyzeFileStyleIssues verifies that multiline_string_escape detects escaped multiline strings while respecting format calls and special characters.
func TestAnalyzeFileStyleIssues(t *testing.T) {
	testCases := []struct {
		name    string
		relPath string
		src     string
		want    []StyleIssue
	}{
		// Compliant production file with raw backquote multiline string, direct and qualified format calls, and special-char strings reports no issues.
		{
			name:    "compliant production file",
			relPath: "pkg/demo/demo.go",
			src: `package demo

import "fmt"

const rawMulti = ` + "`line1\nline2`" + `

func Sprintf(format string) string { return format }

func Format(val string) string {
	_ = "single trailing\n"
	_ = "has ` + "`backquote`" + `\nline2"
	_ = "ansi\x1b[31m\nline2"
	_ = "crlf\r\nline2"
	_ = func() string { return "ok" }()
	_ = Sprintf("direct\nformat")
	return fmt.Sprintf("first\nsecond: %s\n", val)
}
`,
			want: nil,
		},
		// Production file with non-format escaped multiline double-quoted strings (const, non-format call, and lambda call) reports multiline_string_escape.
		{
			name:    "production escaped multiline string",
			relPath: "pkg/demo/bad.go",
			src: `package demo

import "fmt"

const fixture = "line1\nline2\n"

func badCalls() {
	_, _ = fmt.Print("nonfmt\ncall")
	_ = func(s string) string { return s }("lambda\ncall")
}
`,
			want: []StyleIssue{
				{
					Rule:    rule.RuleMultilineStringEscape,
					Line:    5,
					Subject: "<file>",
					Reason:  `double-quoted string embeds middle "\n"; use a raw backquote literal`,
				},
				{
					Rule:    rule.RuleMultilineStringEscape,
					Line:    8,
					Subject: "<file>",
					Reason:  `double-quoted string embeds middle "\n"; use a raw backquote literal`,
				},
				{
					Rule:    rule.RuleMultilineStringEscape,
					Line:    9,
					Subject: "<file>",
					Reason:  `double-quoted string embeds middle "\n"; use a raw backquote literal`,
				},
			},
		},
		// Test file reports multiline_string_escape on non-format literals while permitting t.Errorf format strings.
		{
			name:    "test file format call exempt and non-format flagged",
			relPath: "pkg/demo/bad_test.go",
			src: `package demo

import "testing"

func TestMultiline(t *testing.T) {
	_ = "a\nb"
	t.Errorf("format\nwith\nnewlines: %s", "ok")
}
`,
			want: []StyleIssue{
				{
					Rule:    rule.RuleMultilineStringEscape,
					Line:    6,
					Subject: "<file>",
					Reason:  `double-quoted string embeds middle "\n"; use a raw backquote literal`,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fm, err := AnalyzeFile(tc.relPath, []byte(tc.src))
			if err != nil {
				t.Fatalf("AnalyzeFile() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, fm.StyleIssues); diff != "" {
				t.Errorf("StyleIssues mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestAnalyzeDeferUnlockBeforeLock verifies that defer_unlock_before_lock detects defer Unlock/RUnlock before Lock/RLock/TryLock while permitting inherited locks.
func TestAnalyzeDeferUnlockBeforeLock(t *testing.T) {
	testCases := []struct {
		name    string
		relPath string
		src     string
		want    []StyleIssue
	}{
		// Lock before defer Unlock and inherited lock inside if-branch produce zero violations.
		{
			name:    "lock before defer unlock passes",
			relPath: "pkg/demo/lock.go",
			src: `package demo

import "sync"

type store struct {
	mu sync.RWMutex
}

func (s *store) good(cond bool) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if cond {
		s.mu.RUnlock()
		s.mu.Lock()
		defer s.mu.Unlock()
		return 1
	}
	return 0
}
`,
			want: nil,
		},
		// Defer RUnlock before RLock and defer Unlock before TryLock/Lock both trigger defer_unlock_before_lock.
		{
			name:    "defer unlock before lock and trylock trigger violations",
			relPath: "pkg/demo/lock.go",
			src: `package demo

import "sync"

type store struct {
	mu sync.RWMutex
}

func (s *store) badRead() {
	defer s.mu.RUnlock()
	s.mu.RLock()
}

func (s *store) badTryLock() {
	defer s.mu.Unlock()
	if s.mu.TryLock() {
		return
	}
	s.mu.Lock()
}
`,
			want: []StyleIssue{
				{
					Rule:    rule.RuleDeferUnlockBeforeLock,
					Line:    10,
					Subject: "badRead",
					Reason:  "defer Unlock/RUnlock placed before Lock/RLock acquisition in the same block",
				},
				{
					Rule:    rule.RuleDeferUnlockBeforeLock,
					Line:    15,
					Subject: "badTryLock",
					Reason:  "defer Unlock/RUnlock placed before Lock/RLock acquisition in the same block",
				},
			},
		},
		// Test file with defer Unlock before Lock also triggers defer_unlock_before_lock.
		{
			name:    "test file defer unlock before lock triggers violation",
			relPath: "pkg/demo/lock_test.go",
			src: `package demo

import (
	"sync"
	"testing"
)

func TestLockOrder(t *testing.T) {
	var mu sync.Mutex
	defer mu.Unlock()
	mu.Lock()
}
`,
			want: []StyleIssue{
				{
					Rule:    rule.RuleDeferUnlockBeforeLock,
					Line:    10,
					Subject: "TestLockOrder",
					Reason:  "defer Unlock/RUnlock placed before Lock/RLock acquisition in the same block",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fm, err := AnalyzeFile(tc.relPath, []byte(tc.src))
			if err != nil {
				t.Fatalf("AnalyzeFile() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, fm.StyleIssues); diff != "" {
				t.Errorf("StyleIssues mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestEvaluateStyleRulesRatchet verifies that Evaluate reports full Finding details and hotspots for new or worsened style/lock violations while ratcheting unchanged legacy violations.
func TestEvaluateStyleRulesRatchet(t *testing.T) {
	baseTest := `package demo

import (
	"sync"
	"testing"
)

func TestLegacy(t *testing.T) {
	var mu sync.Mutex
	defer mu.Unlock()
	mu.Lock()
	_ = "a\nb"
}
`
	worsenedTest := `package demo

import (
	"sync"
	"testing"
)

func TestLegacy(t *testing.T) {
	var mu sync.Mutex
	defer mu.Unlock()
	mu.Lock()
	_ = "a\nb"
	_ = "c\nd"
}

func TestAdded(t *testing.T) {
	var mu sync.Mutex
	defer mu.Unlock()
	mu.Lock()
}
`
	multilineReason := `double-quoted string embeds middle "\n"; use a raw backquote literal`
	lockReason := "defer Unlock/RUnlock placed before Lock/RLock acquisition in the same block"
	testCases := []struct {
		name         string
		baseSrc      string
		hasBase      bool
		targetSrc    string
		wantFindings []rule.Finding
	}{
		// Unchanged legacy test file keeps violation counts equal so ratchet emits no findings.
		{
			name:         "legacy unchanged ratcheted",
			baseSrc:      baseTest,
			hasBase:      true,
			targetSrc:    baseTest,
			wantFindings: nil,
		},
		// Adding second occurrences of multiline_string_escape and defer_unlock_before_lock triggers RATCHET_REGRESSION with hotspots.
		{
			name:      "legacy worsened triggers ratchet",
			baseSrc:   baseTest,
			hasBase:   true,
			targetSrc: worsenedTest,
			wantFindings: []rule.Finding{
				{
					Rule:      rule.RuleDeferUnlockBeforeLock,
					Kind:      rule.ViolationRatchetRegression,
					File:      "pkg/demo/demo_test.go",
					Line:      10,
					EndLine:   10,
					Subject:   "TestLegacy",
					Before:    1,
					HasBefore: true,
					After:     2,
					Limit:     0,
					Detail:    lockReason + " (+1 more occurrence(s) in file)",
					Hotspots: []rule.Hotspot{
						{Line: 10, Delta: 1, Summary: lockReason},
						{Line: 18, Delta: 1, Summary: lockReason},
					},
				},
				{
					Rule:      rule.RuleMultilineStringEscape,
					Kind:      rule.ViolationRatchetRegression,
					File:      "pkg/demo/demo_test.go",
					Line:      12,
					EndLine:   12,
					Subject:   "<file>",
					Before:    1,
					HasBefore: true,
					After:     2,
					Limit:     0,
					Detail:    multilineReason + " (+1 more occurrence(s) in file)",
					Hotspots: []rule.Hotspot{
						{Line: 12, Delta: 1, Summary: multilineReason},
						{Line: 13, Delta: 1, Summary: multilineReason},
					},
				},
			},
		},
		// Brand-new test file with single violation per rule reports HARD_LIMIT without plural suffix.
		{
			name:      "new file triggers hard limit",
			hasBase:   false,
			targetSrc: baseTest,
			wantFindings: []rule.Finding{
				{
					Rule:    rule.RuleDeferUnlockBeforeLock,
					Kind:    rule.ViolationHardLimit,
					File:    "pkg/demo/demo_test.go",
					Line:    10,
					EndLine: 10,
					Subject: "TestLegacy",
					After:   1,
					Limit:   0,
					Detail:  lockReason,
					Hotspots: []rule.Hotspot{
						{Line: 10, Delta: 1, Summary: lockReason},
					},
				},
				{
					Rule:    rule.RuleMultilineStringEscape,
					Kind:    rule.ViolationHardLimit,
					File:    "pkg/demo/demo_test.go",
					Line:    12,
					EndLine: 12,
					Subject: "<file>",
					After:   1,
					Limit:   0,
					Detail:  multilineReason,
					Hotspots: []rule.Hotspot{
						{Line: 12, Delta: 1, Summary: multilineReason},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			relPath := "pkg/demo/demo_test.go"
			opts := Options{
				Thresholds:             rule.DefaultThresholds(),
				ChangedTestLinesByFile: map[string]map[int]struct{}{relPath: {12: {}}},
				ReadTargetFile: func(string) ([]byte, error) {
					return []byte(tc.targetSrc), nil
				},
				ReadBaseFile: func(string) ([]byte, bool, error) {
					return []byte(tc.baseSrc), tc.hasBase, nil
				},
			}

			res, err := Evaluate(opts)
			if err != nil {
				t.Fatalf("Evaluate() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.wantFindings, res.Findings); diff != "" {
				t.Errorf("Findings mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
