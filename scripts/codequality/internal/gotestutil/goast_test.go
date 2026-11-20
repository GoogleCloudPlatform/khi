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

package gotestutil

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestIsGoUnitTestFunc(t *testing.T) {
	testCases := []struct {
		name  string
		build func(t *testing.T) *ast.FuncDecl
		want  bool
	}{
		// Rejects a nil function declaration.
		{
			name:  "nil declaration",
			build: func(t *testing.T) *ast.FuncDecl { return nil },
			want:  false,
		},
		// Rejects methods with a receiver even when named TestMethod.
		{
			name: "method with receiver",
			build: func(t *testing.T) *ast.FuncDecl {
				return mustParseFuncDecl(t, "package p; type s struct{}; func (s) TestMethod(t *testing.T) {}")
			},
			want: false,
		},
		// Rejects synthetic function declarations missing a Name identifier.
		{
			name: "missing name identifier",
			build: func(t *testing.T) *ast.FuncDecl {
				return &ast.FuncDecl{Type: &ast.FuncType{Params: &ast.FieldList{}}}
			},
			want: false,
		},
		// Rejects synthetic function declarations missing a FuncType.
		{
			name: "missing func type",
			build: func(t *testing.T) *ast.FuncDecl {
				return &ast.FuncDecl{Name: ast.NewIdent("TestMissingType")}
			},
			want: false,
		},
		// Rejects synthetic function declarations missing a parameter list.
		{
			name: "missing parameter list",
			build: func(t *testing.T) *ast.FuncDecl {
				return &ast.FuncDecl{Name: ast.NewIdent("TestMissingParams"), Type: &ast.FuncType{}}
			},
			want: false,
		},
		// Rejects the special TestMain entry point.
		{
			name: "TestMain declaration",
			build: func(t *testing.T) *ast.FuncDecl {
				return mustParseFuncDecl(t, "package p; func TestMain(m *testing.M) {}")
			},
			want: false,
		},
		// Rejects top-level functions without the Test prefix.
		{
			name: "non-Test prefix",
			build: func(t *testing.T) *ast.FuncDecl {
				return mustParseFuncDecl(t, "package p; func Helper(t *testing.T) {}")
			},
			want: false,
		},
		// Rejects Test-prefixed names whose immediately following rune is lowercase.
		{
			name: "lowercase suffix after Test",
			build: func(t *testing.T) *ast.FuncDecl {
				return mustParseFuncDecl(t, "package p; func Testhelper(t *testing.T) {}")
			},
			want: false,
		},
		// Rejects Test-prefixed functions that do not take a single parameter entry.
		{
			name: "zero parameter entries",
			build: func(t *testing.T) *ast.FuncDecl {
				return mustParseFuncDecl(t, "package p; func TestNoParams() {}")
			},
			want: false,
		},
		// Accepts top-level Test* functions with a non-lowercase suffix rune and one parameter entry.
		{
			name: "valid unit test function",
			build: func(t *testing.T) *ast.FuncDecl {
				return mustParseFuncDecl(t, "package p; func TestExample(t *testing.T) {}")
			},
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			decl := tc.build(t)

			got := IsGoUnitTestFunc(decl)

			if got != tc.want {
				t.Errorf("IsGoUnitTestFunc() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStringLiteralValue(t *testing.T) {
	type stringLiteralOutcome struct {
		Value string
		OK    bool
	}

	testCases := []struct {
		name string
		expr ast.Expr
		want stringLiteralOutcome
	}{
		// Rejects non-BasicLit expressions such as identifiers.
		{
			name: "non-literal identifier",
			expr: ast.NewIdent("tcName"),
			want: stringLiteralOutcome{Value: "", OK: false},
		},
		// Rejects BasicLit nodes whose token kind is not STRING.
		{
			name: "integer basic literal",
			expr: &ast.BasicLit{Kind: token.INT, Value: "42"},
			want: stringLiteralOutcome{Value: "", OK: false},
		},
		// Rejects malformed string literals that strconv.Unquote cannot parse.
		{
			name: "invalid quoted string literal",
			expr: &ast.BasicLit{Kind: token.STRING, Value: `"unterminated`},
			want: stringLiteralOutcome{Value: "", OK: false},
		},
		// Unwraps enclosing parentheses and unquotes valid string literals.
		{
			name: "parenthesized string literal",
			expr: &ast.ParenExpr{
				X: &ast.BasicLit{Kind: token.STRING, Value: `"hello \"world\""`},
			},
			want: stringLiteralOutcome{Value: `hello "world"`, OK: true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			val, ok := StringLiteralValue(tc.expr)
			got := stringLiteralOutcome{Value: val, OK: ok}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("StringLiteralValue() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func mustParseFuncDecl(t *testing.T, src string) *ast.FuncDecl {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample_test.go", src, 0)
	if err != nil {
		t.Fatalf("parser.ParseFile() unexpected error: %v", err)
	}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			return fn
		}
	}
	t.Fatal("no *ast.FuncDecl found in fixture")
	return nil
}
