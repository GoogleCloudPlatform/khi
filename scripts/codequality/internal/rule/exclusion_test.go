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

package rule

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type nodeOnlyStubRule struct{}

var _ ExclusionRule = (*nodeOnlyStubRule)(nil)

func (r *nodeOnlyStubRule) Name() string { return "node_only_stub" }
func (r *nodeOnlyStubRule) Evaluate(_ *ExclusionContext, _ ast.Node) ExclusionScope {
	return ExclusionNodeOnly
}

func TestEvaluateExclusionRules(t *testing.T) {
	src := `package sample

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

const SingleConst = 42 // @const_decl
type MyID string // @type_decl
type customSlice []int
var GlobalCounter = initCount() // @var_decl

func Work(
	ctx interface {
		Err() error
		Done() <-chan struct{}
	},
	n int,
	ptr *int,
	okFlag bool,
	lastErr error,
	otherErr error,
	errs []error,
) error {
	defer func() {}() // @defer_stmt
	pure :=
		10 + -2 + +3 // @pure_lit
	negVar :=
		1 + -n // @unary_var_lit
	bitLit :=
		1 | ^2 // @unary_xor_lit
	_ = pure + negVar + bitLit

	var holder struct{ LastErr error } // @struct_type
	var argCtx interface{ Err(int) error }
	_ =
		holder.LastErr == nil // @sel_err_nil
	_ =
		nil == lastErr // @nil_eq_err
	_ =
		ctx.Err() != nil // @ctx_err_nil
	_ =
		ptr == nil // @ptr_eq_nil
	_ =
		nil == ptr // @nil_eq_ptr
	_ =
		lastErr == otherErr // @err_eq_err
	_ =
		n == 0 // @int_eq_lit
	_ =
		ctx.Done() == nil // @ctx_done_nil
	_ =
		argCtx.Err(1) == nil // @ctx_err_with_arg
	_ =
		plainErr() == nil // @plain_func_eq_nil
	_ =
		errs[0] == nil // @index_err_nil

	var sb strings.Builder
	sb.Grow(n + 8) // @builder_grow
	s := make(
		[]int,
		n+1, // @slice_len_3arg
		n*2, // @slice_cap_3arg
	)
	arrBad := make(
		[4]int,
		0,
		n*2, // @array_cap_3arg
	)
	namedSlice := make(
		customSlice,
		0,
		n*2, // @named_slice_cap
	)
	m := make(
		map[string]int,
		n+1, // @map_hint_2arg
	)
	mapBad := make(
		map[string]int,
		n+1, // @map_bad_3arg_first
		n*2,
	)
	sPlain := make(
		[]int,
		n+1, // @slice_len_2arg
	)
	notMake := otherAlloc(
		1,
		n+1, // @other_alloc_arg2
		n*2, // @other_alloc_arg3
	)
	chained := getFactory().Build(
		n + 1, // @sel_call_arg
	)
	_ = len(s) + len(arrBad) + len(namedSlice) + len(m) + len(mapBad) + len(sPlain) + notMake + chained

	_ = errors.New("standalone") // @errors_new
	_ = errors.Join(lastErr, otherErr) // @errors_join
	_ = fmt.Errorf("wrap: %w", lastErr) // @fmt_errorf
	slog.Info("starting", "n", n) // @slog_info
	_ = errors.Is(lastErr, otherErr) // @errors_is
	_ = fmt.Sprintf("%d", n) // @fmt_sprintf
	_ = plainErr() // @plain_call
	_ = strings.TrimSpace("x") // @strings_call

	if ( // @guard_valid_if
		nil != lastErr) {
		n++ // @guard_skipped_stmt
		return lastErr
	}
	if ( // @guard_empty_if
		lastErr != nil) {
	}
	if ( // @guard_nonreturn_if
		lastErr != nil) {
		n++
	}
	if ( // @guard_eql_if
		lastErr == nil) {
		return nil
	}
	if ( // @guard_ident_if
		okFlag) {
		return nil
	}
	if ( // @guard_ptr_if
		ptr != nil) {
		return nil
	}
	if ( // @guard_else_if
		lastErr != nil) {
		return lastErr
	} else {
		return nil
	}
	if n < 0 { // @domain_if
		panic("negative n") // @panic_call
	}
	return nil
}

func initCount() int { return 1 }
func plainErr() error { return nil }
func otherAlloc(a, b, c int) int { return a + b + c }
func getFactory() struct{ Build func(int) int } {
	return struct{ Build func(int) int }{Build: func(v int) int { return v }}
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, 0)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	rules := DefaultExclusionRules()
	var gotNames []string
	for _, r := range rules {
		gotNames = append(gotNames, r.Name())
	}
	wantNames := []string{
		"const_and_type",
		"error_guard",
		"logging_and_diag_call",
		"allocation_hint",
		"defer_and_panic",
	}
	if diff := cmp.Diff(wantNames, gotNames); diff != "" {
		t.Errorf("rule names mismatch (-want +got):\n%s", diff)
	}

	gotScopes := collectStmtAndDeclScopes(fset, file, []byte(src), rules)
	gotTags := evaluateTaggedLines(src, gotScopes)
	wantTags := map[string]string{
		"const_decl":         "Subtree",
		"type_decl":          "Subtree",
		"var_decl":           "None",
		"defer_stmt":         "Subtree",
		"pure_lit":           "Subtree",
		"unary_var_lit":      "None",
		"unary_xor_lit":      "None",
		"struct_type":        "Subtree",
		"sel_err_nil":        "Subtree",
		"nil_eq_err":         "Subtree",
		"ctx_err_nil":        "Subtree",
		"ptr_eq_nil":         "None",
		"nil_eq_ptr":         "None",
		"err_eq_err":         "None",
		"int_eq_lit":         "None",
		"ctx_done_nil":       "None",
		"ctx_err_with_arg":   "None",
		"plain_func_eq_nil":  "None",
		"index_err_nil":      "None",
		"builder_grow":       "Subtree",
		"slice_len_3arg":     "None",
		"slice_cap_3arg":     "Subtree",
		"array_cap_3arg":     "None",
		"named_slice_cap":    "None",
		"map_hint_2arg":      "Subtree",
		"map_bad_3arg_first": "None",
		"slice_len_2arg":     "None",
		"other_alloc_arg2":   "None",
		"other_alloc_arg3":   "None",
		"sel_call_arg":       "None",
		"errors_new":         "Subtree",
		"errors_join":        "Subtree",
		"fmt_errorf":         "Subtree",
		"slog_info":          "Subtree",
		"errors_is":          "None",
		"fmt_sprintf":        "None",
		"plain_call":         "None",
		"strings_call":       "None",
		"guard_valid_if":     "Subtree",
		"guard_skipped_stmt": "Skipped",
		"guard_empty_if":     "None",
		"guard_nonreturn_if": "None",
		"guard_eql_if":       "None",
		"guard_ident_if":     "None",
		"guard_ptr_if":       "None",
		"guard_else_if":      "None",
		"domain_if":          "None",
		"panic_call":         "Subtree",
	}
	if diff := cmp.Diff(wantTags, gotTags); diff != "" {
		t.Errorf("tagged scopes mismatch (-want +got):\n%s", diff)
	}

	emptyCtx := &ExclusionContext{}
	if emptyCtx.Parent() != nil {
		t.Errorf("empty Parent() = %v, want nil", emptyCtx.Parent())
	}
	if got := EvaluateExclusionRules([]ExclusionRule{&nodeOnlyStubRule{}}, emptyCtx, file); got != ExclusionNodeOnly {
		t.Errorf("EvaluateExclusionRules(nodeOnlyStub) = %v, want %v", got, ExclusionNodeOnly)
	}
}

func collectStmtAndDeclScopes(fset *token.FileSet, file *ast.File, src []byte, rules []ExclusionRule) map[int]ExclusionScope {
	out := make(map[int]ExclusionScope)
	var walk func(n ast.Node, ancestors []ast.Node)
	walk = func(n ast.Node, ancestors []ast.Node) {
		if n == nil {
			return
		}
		ctx := &ExclusionContext{FilePath: "sample.go", Fset: fset, File: file, Source: src, Ancestors: ancestors}
		scope := EvaluateExclusionRules(rules, ctx, n)
		line := fset.Position(n.Pos()).Line
		if _, exists := out[line]; !exists || scope == ExclusionSubtree {
			out[line] = scope
		}
		if scope == ExclusionSubtree {
			return
		}
		ancestors = append(ancestors, n)
		ast.Inspect(n, func(child ast.Node) bool {
			if child == n {
				return true
			}
			if child != nil {
				walk(child, ancestors)
			}
			return false
		})
	}
	walk(file, nil)
	return out
}

func evaluateTaggedLines(src string, scopes map[int]ExclusionScope) map[string]string {
	out := make(map[string]string)
	for i, line := range strings.Split(src, "\n") {
		_, tag, ok := strings.Cut(line, "// @")
		if !ok {
			continue
		}
		scope, visited := scopes[i+1]
		switch {
		case !visited:
			out[strings.TrimSpace(tag)] = "Skipped"
		case scope == ExclusionSubtree:
			out[strings.TrimSpace(tag)] = "Subtree"
		default:
			out[strings.TrimSpace(tag)] = "None"
		}
	}
	return out
}

func TestBoilerplateLineSet(t *testing.T) {
	testCases := []struct {
		name string
		src  string
		want map[int]struct{}
	}{
		{
			name: "nil file returns empty set",
			src:  "",
			want: map[int]struct{}{},
		},
		{
			name: "excludes main entrypoint error guards defer panic and logs while keeping domain statements",
			src: `package main

import (
	"errors"
	"fmt"
	"log/slog"
)

type runner struct{}

func declOnly()

func (r runner) main() {
	_ = r
}

func main() {
	_ = helper(1, nil, nil)
}

func helper(v int, inErr error, ch <-chan int) error {
	defer func() {}()
	<-ch
	step(v)
	slog.Info("check")
	fmt.Println("trace")
	errors.New("standalone error expr is not a log call")
	fmt.Errorf("standalone errorf is not a log call")
	if err := step(v); err != nil {
		return err
	}
	if inErr != nil {
		return inErr
	}
	if v < 0 {
		panic("negative")
	}
	if v == 0 {
		return fmt.Errorf("zero is domain error")
	}
	return nil
}

func step(v int) error { return nil }
`,
			want: map[int]struct{}{
				17: {},
				18: {},
				19: {},
				22: {},
				25: {},
				26: {},
				30: {},
				32: {},
				33: {},
				34: {},
				36: {},
			},
		},
		{
			name: "preserves func main in non-main package",
			src: `package worker

func main() {
	v := 1
	_ = v
}
`,
			want: map[int]struct{}{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			var file *ast.File
			if tc.src != "" {
				var err error
				file, err = parser.ParseFile(fset, "main.go", tc.src, 0)
				if err != nil {
					t.Fatalf("ParseFile failed: %v", err)
				}
			}
			got := BoilerplateLineSet(fset, file)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("BoilerplateLineSet() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
