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
	"go/token"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// IsGoUnitTestFunc reports whether fn is a top-level Go Test* function declaration (with a non-lowercase rune after Test and a single parameter list entry), excluding TestMain.
func IsGoUnitTestFunc(fn *ast.FuncDecl) bool {
	if fn == nil || fn.Recv != nil || fn.Name == nil || fn.Type == nil || fn.Type.Params == nil {
		return false
	}
	name := fn.Name.Name
	if name == "TestMain" || !strings.HasPrefix(name, "Test") {
		return false
	}
	r, _ := utf8.DecodeRuneInString(name[len("Test"):])
	if unicode.IsLower(r) {
		return false
	}
	return len(fn.Type.Params.List) == 1
}

// StringLiteralValue unquotes a Go string literal expression after unwrapping parentheses, reporting whether expr was a valid string literal.
func StringLiteralValue(expr ast.Expr) (string, bool) {
	lit, ok := ast.Unparen(expr).(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	unquoted, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return unquoted, true
}
