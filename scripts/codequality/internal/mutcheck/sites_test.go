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
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSiteHelpers(t *testing.T) {
	src := "package p\r\n" +
		"func F(s string, ch chan bool, ok bool) (int, bool) {\r\n" +
		"\tfn := func() interface{} { return false }\r\n" +
		"\tvoidFn := func() { return true }\r\n" +
		"\tshortFn := func() int { return 1, true }\r\n" +
		"\tif !ok {}\r\n" +
		"\tif <-ch {}\r\n" +
		"\tif true {}\r\n" +
		"\tswitch s { case \"a\": s += \"x\" }\r\n" +
		"\tselect { case <-ch: s = \"y\" }\r\n" +
		"\t_ = \"\" + s\r\n" +
		"\t_ = 'a' == 'b'\r\n" +
		"\t_, _ = fn, voidFn\r\n" +
		"\t_ = shortFn\r\n" +
		"\treturn 0, true\r\n" +
		"}\r\n" +
		"func NonBoolRet() int { return true }\r\n" +
		"func PtrRet() *bool { return true }\r\n"

	plan, err := PlanFileMutations("p.go", []byte(src), PlanOptions{
		TargetLines: lineRangeSet(1, 30),
	})
	if err != nil {
		t.Fatalf("PlanFileMutations() error: %v", err)
	}
	if len(plan.Mutants) == 0 {
		t.Fatalf("expected non-empty mutants")
	}

	gotAssign := map[token.Token]bool{
		token.ASSIGN:     isMutatableAssignTok(token.ASSIGN),
		token.ADD_ASSIGN: isMutatableAssignTok(token.ADD_ASSIGN),
		token.SUB_ASSIGN: isMutatableAssignTok(token.SUB_ASSIGN),
		token.MUL_ASSIGN: isMutatableAssignTok(token.MUL_ASSIGN),
	}
	wantAssign := map[token.Token]bool{
		token.ASSIGN:     true,
		token.ADD_ASSIGN: true,
		token.SUB_ASSIGN: true,
		token.MUL_ASSIGN: false,
	}
	if diff := cmp.Diff(wantAssign, gotAssign); diff != "" {
		t.Errorf("isMutatableAssignTok mismatch (-want +got):\n%s", diff)
	}

	fset := token.NewFileSet()
	expr, err := parser.ParseExprFrom(fset, "x.go", "0.0", 0)
	if err != nil {
		t.Fatalf("ParseExprFrom failed: %v", err)
	}
	if !isZeroOrEmptyLiteral(expr) {
		t.Errorf("isZeroOrEmptyLiteral(0.0) = false, want true")
	}
	if isZeroOrEmptyLiteral(&ast.BasicLit{Kind: token.CHAR, Value: "'a'"}) {
		t.Errorf("isZeroOrEmptyLiteral('a') = true, want false")
	}
	if isAllBlankLHS([]ast.Expr{&ast.Ident{Name: "x"}}) {
		t.Errorf("isAllBlankLHS([x]) = true, want false")
	}
	if isStandardBoolReturnSlot(nil, &ast.ReturnStmt{}, &ast.Ident{Name: "true"}) {
		t.Errorf("isStandardBoolReturnSlot(missing slot) = true, want false")
	}
	retNode := &ast.Ident{Name: "true"}
	if isStandardBoolReturnSlot(nil, &ast.ReturnStmt{Results: []ast.Expr{retNode}}, retNode) {
		t.Errorf("isStandardBoolReturnSlot(no enclosing func) = true, want false")
	}
}
