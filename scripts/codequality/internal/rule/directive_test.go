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
	"go/parser"
	"go/token"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseDirectivesAndMatching(t *testing.T) {
	src := `package sample

// CODEQUALITY_ALLOW(func_cognitive, dispatch): opcode table switch
/* CODEQUALITY_ALLOW(func_code_lines, protocol): generated protocol frame layout */
func Dispatch(x int) int {
	// CODEQUALITY_ALLOW_NOTHING is ignored because marker has suffix.
	a := x + 1 // CODEQUALITY_ALLOW(func_surviving_mutants, equivalent, AOR): offset checked by integration golden
	// CODEQUALITY_ALLOW(func_uncovered_lines, external): real tty branch
	if a < 0 {
		return 0
	}
	return a
}
// CODEQUALITY_ALLOW(file_code_lines, dispatch): trailing comment at EOF`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	directives, errs := ParseDirectives("sample.go", fset, file, []byte(src))
	if len(errs) != 0 {
		t.Fatalf("unexpected directive errors: %+v", errs)
	}

	want := []Directive{
		{
			CommentLine:  3,
			TargetLine:   5,
			Rule:         RuleFuncCognitive,
			Category:     CategoryDispatch,
			AllOperators: true,
			Reason:       "opcode table switch",
		},
		{
			CommentLine:  4,
			TargetLine:   5,
			Rule:         RuleFuncCodeLines,
			Category:     CategoryProtocol,
			AllOperators: true,
			Reason:       "generated protocol frame layout",
		},
		{
			CommentLine:  7,
			TargetLine:   7,
			Rule:         RuleFuncSurvivingMutants,
			Category:     CategoryEquivalent,
			AllOperators: false,
			Operators:    map[MutantOperator]struct{}{OperatorAOR: {}},
			Reason:       "offset checked by integration golden",
		},
		{
			CommentLine:  8,
			TargetLine:   9,
			Rule:         RuleFuncUncoveredLines,
			Category:     CategoryExternal,
			AllOperators: true,
			Reason:       "real tty branch",
		},
		{
			CommentLine:  14,
			TargetLine:   14,
			Rule:         RuleFileCodeLines,
			Category:     CategoryDispatch,
			AllOperators: true,
			Reason:       "trailing comment at EOF",
		},
	}
	if diff := cmp.Diff(want, directives); diff != "" {
		t.Errorf("ParseDirectives() mismatch (-want +got):\n%s", diff)
	}

	// Verify ParseDirectivesFromComments handles out-of-range, blank/comment gaps, inline, and final-line block comments.
	tsSrc := `  /* CODEQUALITY_ALLOW(func_surviving_mutants, perf): memoized cache check */

  // plain comment
  if (cached) {
    return v; // CODEQUALITY_ALLOW(func_surviving_mutants, equivalent, BLR): fallback
  }
  /* CODEQUALITY_ALLOW(file_code_lines, dispatch): trailing block directive */
  /* trailing non-directive block comment */`
	tsDirs, tsErrs := ParseDirectivesFromComments("sample.ts", []RawComment{
		{Line: 0, Text: "// synthetic header comment"},
		{Line: 1, Text: "/* CODEQUALITY_ALLOW(func_surviving_mutants, perf): memoized cache check */"},
		{Line: 5, Text: "// CODEQUALITY_ALLOW(func_surviving_mutants, equivalent, BLR): fallback"},
		{Line: 7, Text: "/* CODEQUALITY_ALLOW(file_code_lines, dispatch): trailing block directive */"},
		{Line: 8, Text: "/* trailing non-directive block comment */"},
	}, []byte(tsSrc))
	if len(tsErrs) != 0 || len(tsDirs) != 3 || tsDirs[0].TargetLine != 4 || tsDirs[1].TargetLine != 5 || tsDirs[2].TargetLine != 7 {
		t.Errorf("ParseDirectivesFromComments() = (%+v, %+v), want targets [4, 5, 7]", tsDirs, tsErrs)
	}
	if _, bareErrs := ParseDirectivesFromComments("bare.ts", []RawComment{{Line: 1, Text: "// CODEQUALITY_ALLOW"}}, []byte("// CODEQUALITY_ALLOW\nexport const x = 1;")); len(bareErrs) != 1 {
		t.Errorf("ParseDirectivesFromComments(bare) errs = %+v, want 1 error", bareErrs)
	}

	if got := MatchDeclDirective(directives, RuleFuncCognitive, 5); got == nil || got.Category != CategoryDispatch {
		t.Errorf("MatchDeclDirective(func_cognitive) = %+v, want dispatch match", got)
	}
	if got := MatchDeclDirective(directives, RuleFuncParams, 5); got != nil {
		t.Errorf("MatchDeclDirective(func_params) = %+v, want nil", got)
	}
	if got := MatchLineDirective(directives, RuleFuncSurvivingMutants, 99, OperatorAOR); got != nil {
		t.Errorf("MatchLineDirective(wrong line) = %+v, want nil", got)
	}
	if got := MatchLineDirective(directives, RuleFuncSurvivingMutants, 5, OperatorAOR); got != nil {
		t.Errorf("MatchLineDirective(wrong rule on line 5) = %+v, want nil", got)
	}
	if got := MatchLineDirective(directives, RuleFuncSurvivingMutants, 7, OperatorROR); got != nil {
		t.Errorf("MatchLineDirective(ROR on AOR-only) = %+v, want nil", got)
	}
	if got := MatchLineDirective(directives, RuleFuncSurvivingMutants, 7, OperatorAOR); got == nil || !got.Used {
		t.Errorf("MatchLineDirective(AOR) = %+v, want used match", got)
	}
	if got := MatchLineDirective(tsDirs, RuleFuncSurvivingMutants, 4, OperatorROR); got == nil || !got.Used {
		t.Errorf("MatchLineDirective(AllOperators by TargetLine) = %+v, want used match", got)
	}
	if got := MatchLineDirective(tsDirs, RuleFuncSurvivingMutants, 1, OperatorROR); got == nil {
		t.Errorf("MatchLineDirective(AllOperators by CommentLine) = %+v, want match", got)
	}

	// Fast mode skips dynamic unused errors while reporting unused AST directives in scope.
	fastUnused := CollectUnusedDirectives(directives, UnusedScope{
		File:           "sample.go",
		IncludeDynamic: false,
		EvaluatedLines: map[int]struct{}{5: {}, 9: {}},
	})
	wantFastUnused := []DirectiveError{
		{
			File:    "sample.go",
			Line:    4,
			Message: "unused CODEQUALITY_ALLOW(func_code_lines, protocol): target has no suppressible HARD_LIMIT violation",
		},
	}
	if diff := cmp.Diff(wantFastUnused, fastUnused); diff != "" {
		t.Errorf("CollectUnusedDirectives(fast) mismatch (-want +got):\n%s", diff)
	}

	// Full mode includes dynamic directives in scope while ignoring lines outside EvaluatedLines (line 14).
	fullUnused := CollectUnusedDirectives(directives, UnusedScope{
		File:           "sample.go",
		IncludeDynamic: true,
		EvaluatedLines: map[int]struct{}{5: {}, 9: {}},
	})
	if len(fullUnused) != 2 {
		t.Errorf("len(CollectUnusedDirectives(full)) = %d, want 2: %+v", len(fullUnused), fullUnused)
	}
}

func TestParseDirectivesValidationErrors(t *testing.T) {
	testCases := []struct {
		name    string
		comment string
		wantMsg string
	}{
		{
			name:    "missing opening paren",
			comment: "// CODEQUALITY_ALLOW: missing parens",
			wantMsg: "CODEQUALITY_ALLOW requires '(<rule_id>, <category>): <reason>'",
		},
		{
			name:    "missing closing paren",
			comment: "// CODEQUALITY_ALLOW(func_cognitive, dispatch: no close paren",
			wantMsg: "CODEQUALITY_ALLOW is missing closing ')' and ': <reason>'",
		},
		{
			name:    "missing colon after paren",
			comment: "// CODEQUALITY_ALLOW(func_cognitive, dispatch) reason without colon",
			wantMsg: "CODEQUALITY_ALLOW requires ': <reason>' after ')'",
		},
		{
			name:    "reason shorter than 8 chars",
			comment: "// CODEQUALITY_ALLOW(func_cognitive, dispatch): short",
			wantMsg: "CODEQUALITY_ALLOW requires a descriptive reason (>= 8 chars) after ':'",
		},
		{
			name:    "missing category",
			comment: "// CODEQUALITY_ALLOW(func_cognitive): long enough reason",
			wantMsg: "CODEQUALITY_ALLOW requires (<rule_id>, <category>): <reason>",
		},
		{
			name:    "unknown rule id",
			comment: "// CODEQUALITY_ALLOW(no_such_rule, dispatch): long enough reason",
			wantMsg: `unknown CODEQUALITY_ALLOW rule "no_such_rule"`,
		},
		{
			name:    "unsuppressible rule",
			comment: "// CODEQUALITY_ALLOW(test_redundancy, protocol): long enough reason",
			wantMsg: `rule "test_redundancy" does not allow CODEQUALITY_ALLOW suppression`,
		},
		{
			name:    "forbidden invariant category",
			comment: "// CODEQUALITY_ALLOW(func_surviving_mutants, invariant): caller guarantees positive",
			wantMsg: "invariant or unreachable branches cannot be suppressed; simplify or delete the redundant branch instead",
		},
		{
			name:    "disallowed category for rule",
			comment: "// CODEQUALITY_ALLOW(func_params, dispatch): wrong category for params",
			wantMsg: `category "dispatch" is not allowed for rule "func_params"`,
		},
		{
			name:    "operator filter on non-mutant rule",
			comment: "// CODEQUALITY_ALLOW(func_cognitive, dispatch, ROR): operators invalid here",
			wantMsg: `operator filters are only supported for "func_surviving_mutants"`,
		},
		{
			name:    "unknown mutant operator",
			comment: "// CODEQUALITY_ALLOW(func_surviving_mutants, equivalent, BADOP): bad operator code",
			wantMsg: `unknown mutant operator "BADOP" in CODEQUALITY_ALLOW`,
		},
		{
			name:    "empty mutant operator list",
			comment: "// CODEQUALITY_ALLOW(func_surviving_mutants, equivalent, , ): empty ops",
			wantMsg: "CODEQUALITY_ALLOW operator filter list cannot be empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			src := "package p\n" + tc.comment + "\nfunc F() {}\n"
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments)
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}
			_, errs := ParseDirectives("p.go", fset, file, []byte(src))
			if len(errs) != 1 {
				t.Fatalf("len(errs) = %d, want 1: %+v", len(errs), errs)
			}
			if errs[0].Message != tc.wantMsg {
				t.Errorf("Message = %q, want %q", errs[0].Message, tc.wantMsg)
			}
		})
	}
}
