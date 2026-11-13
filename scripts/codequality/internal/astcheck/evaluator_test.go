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
	"maps"
	"os"
	"slices"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/google/go-cmp/cmp"
)

func TestEvaluateRulesRatchetAndSuppression(t *testing.T) {
	th := rule.DefaultThresholds()
	th.Limits[rule.RuleFileCodeLines] = 4
	th.Limits[rule.RuleFuncCognitive] = 2
	th.Limits[rule.RuleTRunCognitive] = 1
	th.Limits[rule.RuleTestHelperCognitive] = 1
	th.Limits[rule.RuleCodeClone] = 4

	targetFiles := map[string]string{
		"pkg/peer/peer.go": `package peer

func PeerClone(xs []int) int {
	n := 0
	k := 1
	for _, x := range xs {
		n += x * k
	}
	return n
}
`,
		"pkg/svc/empty_diff.go": "package svc\n",
		"pkg/svc/util.go": `package svc

func Untouched() {}

func ExistingHelper(x int) int {
	if x > 0 {
		if x > 1 {
			return x
		}
	}
	return 0
}
`,
		"pkg/svc/new_util.go": `package svc

// CODEQUALITY_ALLOW(broken_directive)

// CODEQUALITY_ALLOW(func_cognitive, dispatch): intentional opcode dispatch
func AllowedDispatch(x int) int {
	if x > 0 {
		if x > 1 {
			return x
		}
	}
	return 0
}

func RegressedLegacy(x int, a int, flag bool) int {
	if flag {
		return a
	}
	if x > 0 {
		if x > 1 {
			if x > 2 {
				return x
			}
		}
	}
	return 0
}

func ClonedInNew(ys []int) int {
	m := 0
	j := 1
	for _, y := range ys {
		m += y * j
	}
	return m
}

func EmptyBody() {
}
`,
		"pkg/svc/gen_test.go": "// Code generated. DO NOT EDIT.\npackage svc\n",
		"pkg/svc/svc_test.go": `package svc

import "testing"

func complexHelper(a, b bool) int {
	if a {
		if b {
			return 1
		}
	}
	return 0
}

func TestSvc(t *testing.T) {
	t.Run("untouched_sub", func(t *testing.T) {})
	t.Run("branchy", func(t *testing.T) {
		if true {
			if true {
			}
		}
	})
}
`,
	}

	baseFiles := map[string]string{
		"pkg/svc/util.go": targetFiles["pkg/svc/util.go"],
		"pkg/svc/legacy.go": `package svc

func RegressedLegacy(x int) int {
	if x > 0 {
		if x > 1 {
			return x
		}
	}
	return 0
}
`,
	}

	peerReads := 0
	res, err := Evaluate(Options{
		Thresholds: th,
		ChangedProdLinesByFile: map[string]map[int]struct{}{
			"pkg/svc/empty_diff.go": {},
			"pkg/svc/util.go":       {7: {}},
			"pkg/svc/new_util.go":   {7: {}, 16: {}, 30: {}, 39: {}},
			"pkg/svc/missing.go":    {1: {}},
		},
		ChangedTestLinesByFile: map[string]map[int]struct{}{
			"pkg/svc/gen_test.go": {1: {}},
			"pkg/svc/svc_test.go": {6: {}, 17: {}},
		},
		DeletedGoFiles:   []string{"pkg/svc/legacy.go"},
		CloneCorpusFiles: []string{"pkg/peer/peer.go", "pkg/peer/peer.go", "pkg/svc/new_util.go", "pkg/peer/missing_corpus.go"},
		ReadTargetFile: func(relPath string) ([]byte, error) {
			if relPath == "pkg/peer/peer.go" {
				peerReads++
			}
			s, ok := targetFiles[relPath]
			if !ok {
				return nil, os.ErrNotExist
			}
			return []byte(s), nil
		},
		ReadBaseFile: func(relPath string) ([]byte, bool, error) {
			s, ok := baseFiles[relPath]
			return []byte(s), ok, nil
		},
	})
	if err != nil {
		t.Fatalf("Evaluate() unexpected error: %v", err)
	}
	if peerReads != 1 {
		t.Errorf("peerReads = %d, want 1", peerReads)
	}
	if res.FuncsEvaluated != 7 {
		t.Errorf("FuncsEvaluated = %d, want 7", res.FuncsEvaluated)
	}
	if len(res.DirectiveErrors) != 1 {
		t.Errorf("len(DirectiveErrors) = %d, want 1", len(res.DirectiveErrors))
	}

	gotFileKeys := slices.Sorted(maps.Keys(res.Files))
	wantFileKeys := []string{"pkg/svc/new_util.go", "pkg/svc/svc_test.go", "pkg/svc/util.go"}
	if diff := cmp.Diff(wantFileKeys, gotFileKeys); diff != "" {
		t.Errorf("Files keys mismatch (-want +got):\n%s", diff)
	}

	var gotTouchedNames []string
	for _, fn := range res.TouchedFuncs["pkg/svc/new_util.go"] {
		gotTouchedNames = append(gotTouchedNames, fn.DisplayName)
	}
	wantTouchedNames := []string{"AllowedDispatch", "RegressedLegacy", "ClonedInNew", "EmptyBody"}
	if diff := cmp.Diff(wantTouchedNames, gotTouchedNames); diff != "" {
		t.Errorf("TouchedFuncs mismatch (-want +got):\n%s", diff)
	}

	wantEvaluatedLines := map[string]map[int]struct{}{
		"pkg/svc/util.go":     {1: {}, 5: {}, 7: {}},
		"pkg/svc/new_util.go": {1: {}, 6: {}, 7: {}, 15: {}, 16: {}, 29: {}, 30: {}, 38: {}, 39: {}},
		"pkg/svc/svc_test.go": {1: {}, 5: {}, 6: {}, 16: {}, 17: {}},
	}
	if diff := cmp.Diff(wantEvaluatedLines, res.EvaluatedLines); diff != "" {
		t.Errorf("EvaluatedLines mismatch (-want +got):\n%s", diff)
	}

	wantSuppressed := []rule.SuppressedItem{
		{
			Rule:     rule.RuleFuncCognitive,
			File:     "pkg/svc/new_util.go",
			Line:     6,
			Subject:  "AllowedDispatch",
			Category: "dispatch",
			Reason:   "intentional opcode dispatch",
			After:    3,
			Limit:    2,
		},
	}
	if diff := cmp.Diff(wantSuppressed, res.Suppressed); diff != "" {
		t.Errorf("Suppressed mismatch (-want +got):\n%s", diff)
	}

	wantFindings := []rule.Finding{
		{
			Rule:    rule.RuleCatchAllFilename,
			Kind:    rule.ViolationHardLimit,
			File:    "pkg/svc/new_util.go",
			Line:    1,
			EndLine: 1,
			Subject: "<filename>",
			After:   1,
			Limit:   0,
			Detail:  `file "new_util.go" uses catch-all suffix "_util.go"; group code into domain-named files`,
		},
		{
			Rule:    rule.RuleFileCodeLines,
			Kind:    rule.ViolationHardLimit,
			File:    "pkg/svc/new_util.go",
			Line:    1,
			EndLine: 1,
			Subject: "<file>",
			After:   31,
			Limit:   4,
		},
		{
			Rule:      rule.RuleControlFlagParam,
			Kind:      rule.ViolationHardLimit,
			File:      "pkg/svc/new_util.go",
			Line:      15,
			EndLine:   27,
			Subject:   "RegressedLegacy",
			HasBefore: true,
			After:     1,
			Limit:     0,
			Detail:    "boolean control-flow parameter(s): flag",
		},
		{
			Rule:      rule.RuleFuncCognitive,
			Kind:      rule.ViolationRatchetRegression,
			File:      "pkg/svc/new_util.go",
			Line:      15,
			EndLine:   27,
			Subject:   "RegressedLegacy",
			Before:    3,
			HasBefore: true,
			After:     7,
			Limit:     2,
			Hotspots: []rule.Hotspot{
				{Line: 16, Delta: 1, Summary: "+1 (if)"},
				{Line: 19, Delta: 1, Summary: "+1 (if)"},
				{Line: 20, Delta: 2, Summary: "+2 (nested if, depth=1)"},
				{Line: 21, Delta: 3, Summary: "+3 (nested if, depth=2)"},
			},
		},
		{
			Rule:    rule.RuleCodeClone,
			Kind:    rule.ViolationHardLimit,
			File:    "pkg/svc/new_util.go",
			Line:    30,
			EndLine: 35,
			Subject: "ClonedInNew",
			After:   5,
			Limit:   4,
			Detail:  "lines 30-35 duplicate 5 normalized statements from pkg/peer/peer.go:4-9 (PeerClone)",
		},
		{
			Rule:    rule.RuleTestHelperCognitive,
			Kind:    rule.ViolationHardLimit,
			File:    "pkg/svc/svc_test.go",
			Line:    5,
			EndLine: 12,
			Subject: "complexHelper",
			After:   3,
			Limit:   1,
			Hotspots: []rule.Hotspot{
				{Line: 6, Delta: 1, Summary: "+1 (if)"},
				{Line: 7, Delta: 2, Summary: "+2 (nested if, depth=1)"},
			},
		},
		{
			Rule:    rule.RuleTRunCognitive,
			Kind:    rule.ViolationHardLimit,
			File:    "pkg/svc/svc_test.go",
			Line:    16,
			EndLine: 21,
			Subject: "TestSvc [t.Run(branchy)]",
			After:   3,
			Limit:   1,
			Hotspots: []rule.Hotspot{
				{Line: 17, Delta: 1, Summary: "+1 (if)"},
				{Line: 18, Delta: 2, Summary: "+2 (nested if, depth=1)"},
			},
		},
	}
	if diff := cmp.Diff(wantFindings, res.Findings); diff != "" {
		t.Errorf("Findings mismatch (-want +got):\n%s", diff)
	}
}

func TestEvaluateZeroCloneThreshold(t *testing.T) {
	th := rule.DefaultThresholds()
	th.Limits[rule.RuleCodeClone] = 0

	files := map[string]string{
		"pkg/a/a.go": "package a\nfunc One() int { return 1 }\n",
		"pkg/b/b.go": "package b\nfunc Two() int { return 2 }\n",
	}

	res, err := Evaluate(Options{
		Thresholds:             th,
		ChangedProdLinesByFile: map[string]map[int]struct{}{"pkg/a/a.go": {2: {}}},
		CloneCorpusFiles:       []string{"pkg/b/b.go"},
		ReadTargetFile: func(relPath string) ([]byte, error) {
			return []byte(files[relPath]), nil
		},
	})
	if err != nil {
		t.Fatalf("Evaluate() unexpected error: %v", err)
	}
	if len(res.Findings) != 1 || res.Findings[0].Rule != rule.RuleCodeClone {
		t.Fatalf("Findings = %+v, want single RuleCodeClone finding", res.Findings)
	}
}
