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

package langgo

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

var goPredeclared = map[string]bool{
	"": true, "_": true, "true": true, "false": true, "iota": true, "nil": true,
	"any": true, "bool": true, "byte": true, "comparable": true,
	"complex64": true, "complex128": true, "error": true,
	"float32": true, "float64": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"rune": true, "string": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "uintptr": true,
	"append": true, "cap": true, "clear": true, "close": true, "complex": true,
	"copy": true, "delete": true, "imag": true, "len": true, "make": true,
	"max": true, "min": true, "new": true, "panic": true, "print": true,
	"println": true, "real": true, "recover": true,
}

type localRenamer struct {
	mapping   map[string]string
	protected map[*ast.Ident]bool
	nextID    int
}

// NormalizeDeclBytes strips comments and normalizes Go function, receiver, parameter, and local variable identifiers.
func NormalizeDeclBytes(declBytes []byte) string {
	wrapped := append([]byte("package p\n\n"), declBytes...)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "snippet.go", wrapped, 0)
	if err != nil || len(file.Decls) == 0 {
		return string(bytes.TrimSpace(declBytes))
	}

	normalizeDeclAST(file.Decls[0])
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file.Decls[0]); err != nil {
		return string(bytes.TrimSpace(declBytes))
	}
	return strings.TrimSpace(buf.String())
}

func newLocalRenamer() *localRenamer {
	return &localRenamer{
		mapping:   make(map[string]string),
		protected: make(map[*ast.Ident]bool),
		nextID:    1,
	}
}

func (r *localRenamer) register(name string) {
	if goPredeclared[name] {
		return
	}
	if _, exists := r.mapping[name]; !exists {
		r.mapping[name] = fmt.Sprintf("v%d", r.nextID)
		r.nextID++
	}
}

func normalizeDeclAST(decl ast.Decl) {
	renamer := newLocalRenamer()
	switch d := decl.(type) {
	case *ast.FuncDecl:
		prepareFuncDecl(d, renamer)
	case *ast.GenDecl:
		prepareVarDecl(d, renamer)
	}
	collectNestedClosuresAndProtected(decl, renamer)
	applyIdentifierRenames(decl, renamer)
}

func prepareFuncDecl(d *ast.FuncDecl, renamer *localRenamer) {
	if d.Name != nil {
		d.Name.Name = "targetFunc"
		renamer.protected[d.Name] = true
	}
	if d.Recv != nil {
		for _, field := range d.Recv.List {
			registerFieldNames(field, renamer)
			normalizeReceiverTypeIdent(field.Type, renamer)
		}
	}
	collectFuncTypeLocals(d.Type, renamer)
	if d.Body != nil {
		collectBlockLocals(d.Body, renamer)
	}
}

func prepareVarDecl(d *ast.GenDecl, renamer *localRenamer) {
	for _, spec := range d.Specs {
		vs := spec.(*ast.ValueSpec)
		renameTopVarIdents(vs.Names, renamer)
	}
}

func renameTopVarIdents(names []*ast.Ident, renamer *localRenamer) {
	for i, nameIdent := range names {
		if i == 0 {
			nameIdent.Name = "targetTask"
		} else {
			nameIdent.Name = fmt.Sprintf("targetTask%d", i+1)
		}
		renamer.protected[nameIdent] = true
	}
}

func normalizeReceiverTypeIdent(expr ast.Expr, renamer *localRenamer) {
	switch t := expr.(type) {
	case *ast.Ident:
		t.Name = "RecvType"
		renamer.protected[t] = true
	case *ast.StarExpr:
		normalizeReceiverTypeIdent(t.X, renamer)
	case *ast.IndexExpr:
		normalizeReceiverTypeIdent(t.X, renamer)
		renamer.register(t.Index.(*ast.Ident).Name)
	case *ast.IndexListExpr:
		normalizeReceiverTypeIdent(t.X, renamer)
		for _, idx := range t.Indices {
			renamer.register(idx.(*ast.Ident).Name)
		}
	}
}

func collectFuncTypeLocals(ft *ast.FuncType, renamer *localRenamer) {
	registerFieldList(ft.TypeParams, renamer)
	registerFieldList(ft.Params, renamer)
	registerFieldList(ft.Results, renamer)
}

func registerFieldList(fl *ast.FieldList, renamer *localRenamer) {
	if fl == nil {
		return
	}
	for _, field := range fl.List {
		registerFieldNames(field, renamer)
	}
}

func registerFieldNames(field *ast.Field, renamer *localRenamer) {
	for _, ident := range field.Names {
		renamer.register(ident.Name)
	}
}

func collectBlockLocals(block *ast.BlockStmt, renamer *localRenamer) {
	ast.Inspect(block, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.AssignStmt:
			registerDefineAssign(stmt, renamer)
		case *ast.RangeStmt:
			registerDefineRange(stmt, renamer)
		case *ast.DeclStmt:
			registerLocalDecl(stmt, renamer)
		}
		return true
	})
}

func registerDefineAssign(stmt *ast.AssignStmt, renamer *localRenamer) {
	if stmt.Tok != token.DEFINE {
		return
	}
	for _, lhs := range stmt.Lhs {
		renamer.register(lhs.(*ast.Ident).Name)
	}
}

func registerDefineRange(stmt *ast.RangeStmt, renamer *localRenamer) {
	if stmt.Tok != token.DEFINE {
		return
	}
	renamer.register(stmt.Key.(*ast.Ident).Name)
	if stmt.Value != nil {
		renamer.register(stmt.Value.(*ast.Ident).Name)
	}
}

func registerLocalDecl(stmt *ast.DeclStmt, renamer *localRenamer) {
	gd := stmt.Decl.(*ast.GenDecl)
	if gd.Tok != token.VAR && gd.Tok != token.CONST {
		return
	}
	for _, spec := range gd.Specs {
		for _, id := range spec.(*ast.ValueSpec).Names {
			renamer.register(id.Name)
		}
	}
}

func collectNestedClosuresAndProtected(decl ast.Decl, renamer *localRenamer) {
	ast.Inspect(decl, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.SelectorExpr:
			renamer.protected[node.Sel] = true
		case *ast.KeyValueExpr:
			keyIdent, _ := node.Key.(*ast.Ident)
			renamer.protected[keyIdent] = true
		case *ast.FuncLit:
			collectFuncTypeLocals(node.Type, renamer)
			if node.Body != nil {
				collectBlockLocals(node.Body, renamer)
			}
		}
		return true
	})
}

func applyIdentifierRenames(decl ast.Decl, renamer *localRenamer) {
	ast.Inspect(decl, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok && !renamer.protected[ident] {
			if mapped, exists := renamer.mapping[ident.Name]; exists {
				ident.Name = mapped
			}
		}
		return true
	})
}
