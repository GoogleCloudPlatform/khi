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
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDefaultExclusionRules(t *testing.T) {
	src := `package sample

import (
	"errors"
	"fmt"
	"log/slog"
)

const maxSize = 1024 * 1024

type customSlice [4 + 4]int

func process(items []int, flag bool) ([]int, bool, error) {
	defer func() {
		if flag {
			panic("unexpected")
		}
	}()
	const localLimit = 2 * 3
	chunk := 4 * 1024 * 1024
	out := make([]int, len(items)+1, len(items)*2)
	exact := make([]int, len(items)+2)
	seen := make(map[int]bool, len(items)+4)
	var sb stringsBuilder
	sb.Grow(len(items) * 10)
	slog.Info("processing", "empty", len(items) == 0)
	fmt.Printf("count=%d\n", len(items)+1)
	if err := validate(items); err != nil {
		return nil, false, fmt.Errorf("validate failed: %w", err)
	}
	if err := validate(exact); nil != err {
		return nil, false, errors.Join(err, errors.New("joined"))
	} else if len(exact) > 1 {
		exact[0] = 1
	}
	if checkErr != nil && len(items) > 2 {
		return nil, false, checkErr
	}
	if len(items) == 0 {
		return nil, false, errors.New("no items")
	}
	if chunk > localLimit {
		out[0] = items[0] + 1
		seen[items[0]] = true
	}
	return out, true, nil
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse sample source: %v", err)
	}

	rules := DefaultExclusionRules()
	var unexcludedSnippets []string

	var ancestors []ast.Node
	var walk func(n ast.Node)
	walk = func(n ast.Node) {
		if n == nil {
			return
		}
		ctx := &ExclusionContext{
			FilePath:  "sample.go",
			Fset:      fset,
			File:      file,
			Source:    []byte(src),
			Ancestors: ancestors,
		}
		scope := EvaluateExclusionRules(rules, ctx, n)
		if scope == ExclusionSubtree {
			return
		}
		if scope == ExclusionNone {
			if bin, ok := n.(*ast.BinaryExpr); ok {
				start := fset.Position(bin.Pos()).Offset
				end := fset.Position(bin.End()).Offset
				unexcludedSnippets = append(unexcludedSnippets, src[start:end])
			}
		}
		ancestors = append(ancestors, n)
		ast.Inspect(n, func(child ast.Node) bool {
			if child == n {
				return true
			}
			walk(child)
			return false
		})
		ancestors = ancestors[:len(ancestors)-1]
	}
	walk(file)

	// Only real business logic expressions should survive noise exclusion:
	// - len(items)+1 in make slice length (2nd arg is length, 3rd arg len(items)*2 is excluded capacity)
	// - len(items) == 0 in business guard (slog.Info's len(items) == 0 is excluded)
	// - chunk > localLimit (8 * 8 literal math is excluded via ExclusionNodeOnly)
	// - items[0] + 1
	want := []string{
		"len(items)+1",
		"len(items)+2",
		"len(exact) > 1",
		"checkErr != nil && len(items) > 2",
		"len(items) > 2",
		"len(items) == 0",
		"chunk > localLimit",
		"items[0] + 1",
	}
	if diff := cmp.Diff(want, unexcludedSnippets); diff != "" {
		t.Errorf("unexcluded binary expressions mismatch (-want +got):\n%s", diff)
	}
}

type customPrefixCallRule struct {
	prefix string
}

var _ ExclusionRule = (*customPrefixCallRule)(nil)

func (r *customPrefixCallRule) Name() string {
	return "custom_prefix_call"
}

func (r *customPrefixCallRule) Evaluate(_ *ExclusionContext, node ast.Node) ExclusionScope {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return ExclusionNone
	}
	ident, ok := call.Fun.(*ast.Ident)
	if ok && strings.HasPrefix(ident.Name, r.prefix) {
		return ExclusionSubtree
	}
	return ExclusionNone
}

func TestCustomExclusionRuleExtensibility(t *testing.T) {
	src := `package sample

func run(x int) int {
	traceMetric(x + 1)
	return x + 2
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	rules := append(DefaultExclusionRules(), &customPrefixCallRule{prefix: "trace"})
	var got []string
	var ancestors []ast.Node
	var walk func(n ast.Node)
	walk = func(n ast.Node) {
		if n == nil {
			return
		}
		ctx := &ExclusionContext{
			FilePath:  "sample.go",
			Fset:      fset,
			File:      file,
			Source:    []byte(src),
			Ancestors: ancestors,
		}
		if EvaluateExclusionRules(rules, ctx, n) == ExclusionSubtree {
			return
		}
		if bin, ok := n.(*ast.BinaryExpr); ok {
			got = append(got, src[fset.Position(bin.Pos()).Offset:fset.Position(bin.End()).Offset])
		}
		ancestors = append(ancestors, n)
		ast.Inspect(n, func(child ast.Node) bool {
			if child == n {
				return true
			}
			walk(child)
			return false
		})
		ancestors = ancestors[:len(ancestors)-1]
	}
	walk(file)

	want := []string{"x + 2"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("custom rule filtering mismatch (-want +got):\n%s", diff)
	}
}

func TestParseSuppressions(t *testing.T) {
	src := `package sample

func calc(a int) int {
	// NOMUTANTS(perf): fast-path cache check does not alter observable return value
	if a == 0 {
		return 0
	}
	b := a + 1 // NOMUTANTS(equivalent, AOR): pre-offset adjustment verified via integration golden
	c := b + 1
	// NOMUTANTS: missing category is rejected
	if c > 10 {
		return c
	}
	// NOMUTANTS(equivalent):
	if c > 20 {
		return c
	}
	// NOMUTANTS(invariant): caller guarantees c > 0
	if c > 30 {
		return c
	}
	// NOMUTANTS(equivalent, TYPO): invalid operator code
	if c > 40 {
		return c
	}
	/* NOMUTANTS(external): OS-specific line ending fallback */
	return c - 1
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	idx := ParseSuppressions("sample.go", fset, file, []byte(src))

	type queryOutcome struct {
		Suppressed bool
		Category   SuppressionCategory
	}
	testCases := []struct {
		name     string
		line     int
		operator MutantOperator
		want     queryOutcome
	}{
		{
			name:     "standalone preceding comment suppresses ROR with perf category",
			line:     5,
			operator: OperatorROR,
			want:     queryOutcome{Suppressed: true, Category: SuppressionPerf},
		},
		{
			name:     "inline operator-scoped comment suppresses matching AOR with equivalent category",
			line:     8,
			operator: OperatorAOR,
			want:     queryOutcome{Suppressed: true, Category: SuppressionEquivalent},
		},
		{
			name:     "inline operator-scoped comment does not suppress SBR on same line",
			line:     8,
			operator: OperatorSBR,
			want:     queryOutcome{Suppressed: false},
		},
		{
			name:     "inline comment does not leak onto following line",
			line:     9,
			operator: OperatorAOR,
			want:     queryOutcome{Suppressed: false},
		},
		{
			name:     "missing category does not suppress",
			line:     11,
			operator: OperatorROR,
			want:     queryOutcome{Suppressed: false},
		},
		{
			name:     "empty reason after colon does not suppress",
			line:     15,
			operator: OperatorROR,
			want:     queryOutcome{Suppressed: false},
		},
		{
			name:     "forbidden invariant category emits dedicated warning and does not suppress",
			line:     19,
			operator: OperatorROR,
			want:     queryOutcome{Suppressed: false},
		},
		{
			name:     "unknown operator code emits warning and does not suppress",
			line:     23,
			operator: OperatorROR,
			want:     queryOutcome{Suppressed: false},
		},
		{
			name:     "standalone block comment suppresses next code line with external category",
			line:     27,
			operator: OperatorAOR,
			want:     queryOutcome{Suppressed: true, Category: SuppressionExternal},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cat, ok := idx.LookupSuppression(tc.line, tc.operator)
			got := queryOutcome{Suppressed: ok, Category: cat}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("LookupSuppression(line=%d, op=%q) mismatch (-want +got):\n%s", tc.line, tc.operator, diff)
			}
		})
	}

	wantWarnings := []DirectiveWarning{
		{
			File:    "sample.go",
			Line:    10,
			Message: "// NOMUTANTS requires a category and reason: // NOMUTANTS(<category>): <reason> (allowed categories: equivalent, perf, observability, external)",
		},
		{
			File:    "sample.go",
			Line:    14,
			Message: "// NOMUTANTS(<category>) requires a non-empty reason after ':'",
		},
		{
			File:    "sample.go",
			Line:    18,
			Message: "invariant or unreachable conditions cannot be suppressed with NOMUTANTS; simplify or delete the redundant branch instead (allowed categories: equivalent, perf, observability, external)",
		},
		{
			File:    "sample.go",
			Line:    22,
			Message: `unknown operator "TYPO" in // NOMUTANTS(...)`,
		},
	}
	if diff := cmp.Diff(wantWarnings, idx.Warnings); diff != "" {
		t.Errorf("Warnings mismatch (-want +got):\n%s", diff)
	}
}
