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
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/google/go-cmp/cmp"
)

func parseFuncBodyForTest(t *testing.T, src string) (*token.FileSet, *ast.BlockStmt) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	for _, d := range file.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			return fset, fd.Body
		}
	}
	t.Fatal("no FuncDecl found")
	return nil, nil
}

func runComplexityAnalysis(fset *token.FileSet, body *ast.BlockStmt, subtestMode bool) ComplexitySummary {
	if subtestMode {
		return AnalyzeSubtestBlockComplexity(fset, body)
	}
	return AnalyzeBlockComplexity(fset, body)
}

func TestAnalyzeBlockComplexity(t *testing.T) {
	subtestFixture := `package p
import "testing"
func F(t *testing.T, a, b bool, check func(bool) bool) {
	if check(a && b) {
	}
	t.Run("nested", func(t *testing.T) {
		if a {
		}
	})
}`

	testCases := []struct {
		name        string
		src         string
		subtestMode bool
		want        ComplexitySummary
	}{
		{
			name: "nil body returns zero",
			src:  "package p\nfunc F()",
			want: ComplexitySummary{},
		},
		{
			name: "if else-if else nested bodies mixed bool ops and binary expr closure",
			src: `package p
func F(a, b, c bool, check func(bool) bool) int {
	if (a && b) || check(a || b) {
		return 1
	} else if a && b && c {
		if a {
			return 2
		}
	} else {
		if b {
			return 3
		}
	}
	return 1 + func() int {
		if c {
			return 4
		}
		return 0
	}()
}`,
			want: ComplexitySummary{
				Cognitive:  13,
				MaxNesting: 2,
				Hotspots: []rule.Hotspot{
					{Line: 3, Delta: 2, Summary: "+2 (mixed && / || sequence)"},
					{Line: 3, Delta: 1, Summary: "+1 (boolean operator sequence)"},
					{Line: 3, Delta: 1, Summary: "+1 (if)"},
					{Line: 5, Delta: 1, Summary: "+1 (boolean operator sequence)"},
					{Line: 5, Delta: 1, Summary: "+1 (else if)"},
					{Line: 6, Delta: 2, Summary: "+2 (nested if, depth=1)"},
					{Line: 9, Delta: 1, Summary: "+1 (else)"},
					{Line: 10, Delta: 2, Summary: "+2 (nested if, depth=1)"},
					{Line: 15, Delta: 2, Summary: "+2 (nested if, depth=1)"},
				},
			},
		},
		{
			name: "type-switch wrapping select wrapping for wrapping leaf if and unlabeled break",
			src: `package p
func F(x any, ch chan int, ok bool) {
	switch x.(type) {
	case int:
		select {
		case <-ch:
			for i := 0; i < 3; i++ {
				if ok {
					break
				}
			}
		}
	}
}`,
			want: ComplexitySummary{
				Cognitive:  10,
				MaxNesting: 4,
				Hotspots: []rule.Hotspot{
					{Line: 3, Delta: 1, Summary: "+1 (type switch)"},
					{Line: 5, Delta: 2, Summary: "+2 (nested select, depth=1)"},
					{Line: 7, Delta: 3, Summary: "+3 (nested for, depth=2)"},
					{Line: 8, Delta: 4, Summary: "+4 (nested if, depth=3)"},
				},
			},
		},
		{
			name: "leaf switch records nesting depth one",
			src: `package p
func F(v int) {
	switch v {
	case 1:
	}
}`,
			want: ComplexitySummary{
				Cognitive:  1,
				MaxNesting: 1,
				Hotspots: []rule.Hotspot{
					{Line: 3, Delta: 1, Summary: "+1 (switch)"},
				},
			},
		},
		{
			name: "leaf for records nesting depth one",
			src: `package p
func F() {
	for i := 0; i < 3; i++ {
	}
}`,
			want: ComplexitySummary{
				Cognitive:  1,
				MaxNesting: 1,
				Hotspots: []rule.Hotspot{
					{Line: 3, Delta: 1, Summary: "+1 (for)"},
				},
			},
		},
		{
			name: "range wrapping switch wrapping leaf type-switch and labeled continue",
			src: `package p
func F(xs []int, x any) {
outer:
	for _, v := range xs {
		switch v {
		case 1:
			switch x.(type) {
			case int:
				continue outer
			}
		}
	}
}`,
			want: ComplexitySummary{
				Cognitive:  7,
				MaxNesting: 3,
				Hotspots: []rule.Hotspot{
					{Line: 4, Delta: 1, Summary: "+1 (range)"},
					{Line: 5, Delta: 2, Summary: "+2 (nested switch, depth=1)"},
					{Line: 7, Delta: 3, Summary: "+3 (nested type switch, depth=2)"},
					{Line: 9, Delta: 1, Summary: "+1 (labeled continue)"},
				},
			},
		},
		{
			name: "leaf select and labeled goto",
			src: `package p
func F(ch chan int) {
	select {
	case <-ch:
		goto done
	default:
	}
done:
	return
}`,
			want: ComplexitySummary{
				Cognitive:  2,
				MaxNesting: 1,
				Hotspots: []rule.Hotspot{
					{Line: 3, Delta: 1, Summary: "+1 (select)"},
					{Line: 5, Delta: 1, Summary: "+1 (labeled goto)"},
				},
			},
		},
		{
			name: "leaf range loop",
			src: `package p
func F(xs []int) {
	for range xs {
	}
}`,
			want: ComplexitySummary{
				Cognitive:  1,
				MaxNesting: 1,
				Hotspots: []rule.Hotspot{
					{Line: 3, Delta: 1, Summary: "+1 (range)"},
				},
			},
		},
		{
			name: "leaf closure with unary wrapped three-way logical sequence",
			src: `package p
func F(a, b, c bool) {
	{
		fn := func() {
			ok := !(a && b && c)
			_ = ok
		}
		fn()
	}
}`,
			want: ComplexitySummary{
				Cognitive:  1,
				MaxNesting: 1,
				Hotspots: []rule.Hotspot{
					{Line: 5, Delta: 1, Summary: "+1 (boolean operator sequence)"},
				},
			},
		},
		{
			name:        "subtest mode skips nested t.Run but inspects non-subtest calls",
			subtestMode: true,
			src:         subtestFixture,
			want: ComplexitySummary{
				Cognitive:  2,
				MaxNesting: 1,
				Hotspots: []rule.Hotspot{
					{Line: 4, Delta: 1, Summary: "+1 (boolean operator sequence)"},
					{Line: 4, Delta: 1, Summary: "+1 (if)"},
				},
			},
		},
		{
			name:        "normal mode walks nested t.Run closure body",
			subtestMode: false,
			src:         subtestFixture,
			want: ComplexitySummary{
				Cognitive:  4,
				MaxNesting: 2,
				Hotspots: []rule.Hotspot{
					{Line: 4, Delta: 1, Summary: "+1 (boolean operator sequence)"},
					{Line: 4, Delta: 1, Summary: "+1 (if)"},
					{Line: 7, Delta: 2, Summary: "+2 (nested if, depth=1)"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset, body := parseFuncBodyForTest(t, tc.src)
			got := runComplexityAnalysis(fset, body, tc.subtestMode)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ComplexitySummary mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
