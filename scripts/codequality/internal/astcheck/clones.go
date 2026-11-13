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
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

// StmtFingerprint pairs a normalized structural token with its 1-based source line.
type StmtFingerprint struct {
	// Token is the normalized structural statement fingerprint.
	Token string
	// Line is the 1-based source line number of the statement.
	Line int
}

// FuncCloneBlock holds the normalized statement sequence for one production Go function.
type FuncCloneBlock struct {
	// File is the slash-separated relative path of the Go file.
	File string
	// FuncKey is the canonical function or method identifier.
	FuncKey string
	// FuncName is the human-readable function or method display name.
	FuncName string
	// StartLine is the 1-based starting line of the function declaration.
	StartLine int
	// EndLine is the 1-based ending line of the function declaration.
	EndLine int
	// Stmts holds the flattened sequence of statement fingerprints in the function body.
	Stmts []StmtFingerprint
}

// CloneMatch describes the longest contiguous duplicated statement span found against another function.
type CloneMatch struct {
	// Statements is the count of contiguous matching normalized statements.
	Statements int
	// LocalStart is the starting line number in the target function.
	LocalStart int
	// LocalEnd is the ending line number in the target function.
	LocalEnd int
	// PeerFile is the relative file path containing the matching peer function.
	PeerFile string
	// PeerFunc is the display name of the matching peer function.
	PeerFunc string
	// PeerStart is the starting line number of the matching span in the peer function.
	PeerStart int
	// PeerEnd is the ending line number of the matching span in the peer function.
	PeerEnd int
}

// CloneIndex indexes normalized statement windows across production Go functions for Type-2 clone lookup.
type CloneIndex struct {
	minWindow int
	blocks    []FuncCloneBlock
	windows   map[string][]windowAnchor
}

type windowAnchor struct {
	blockIdx int
	stmtIdx  int
}

// NewCloneIndex builds a CloneIndex for minWindow consecutive statements from blocks.
func NewCloneIndex(minWindow int, blocks []FuncCloneBlock) *CloneIndex {
	idx := &CloneIndex{
		minWindow: minWindow,
		blocks:    blocks,
		windows:   make(map[string][]windowAnchor),
	}
	if minWindow < 1 {
		return idx
	}
	for bIdx, blk := range blocks {
		idx.indexBlock(bIdx, blk)
	}
	return idx
}

// FindMaxClone returns the longest contiguous duplicated statement span (>= minWindow) for target against peers.
func (idx *CloneIndex) FindMaxClone(target FuncCloneBlock) (CloneMatch, bool) {
	if len(idx.windows) == 0 || len(target.Stmts) < idx.minWindow {
		return CloneMatch{}, false
	}
	var best CloneMatch
	for i := 0; i+idx.minWindow <= len(target.Stmts); i++ {
		sig := joinWindowTokens(target.Stmts[i : i+idx.minWindow])
		for _, anchor := range idx.windows[sig] {
			cand := idx.measureMatch(target, i, anchor)
			if cand.Statements > best.Statements {
				best = cand
			}
		}
	}
	return best, best.Statements >= idx.minWindow
}

// ExtractCloneBlocks parses src and returns normalized statement sequences for each production function in relPath.
func ExtractCloneBlocks(relPath string, src []byte) []FuncCloneBlock {
	cleanPath := strings.TrimPrefix(filepath.ToSlash(relPath), "./")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, cleanPath, src, parser.ParseComments)
	if err != nil || isGeneratedAST(file) {
		return nil
	}

	var blocks []FuncCloneBlock
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		stmts := flattenBodyFingerprints(fset, fd.Body)
		if len(stmts) == 0 {
			continue
		}
		key, display := FormatFuncIdentity(fd)
		blocks = append(blocks, FuncCloneBlock{
			File:      cleanPath,
			FuncKey:   key,
			FuncName:  display,
			StartLine: fset.Position(fd.Pos()).Line,
			EndLine:   fset.Position(fd.End()).Line,
			Stmts:     stmts,
		})
	}
	return blocks
}

// DiscoverProdGoFiles walks repoRoot and returns slash-separated relative paths for all target production Go files.
func DiscoverProdGoFiles(repoRoot string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(repoRoot, func(fullPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(repoRoot, fullPath)
		if err != nil {
			return nil
		}
		slashRel := filepath.ToSlash(rel)
		if d.IsDir() {
			if shouldSkipWalkDir(slashRel) {
				return filepath.SkipDir
			}
			return nil
		}
		if gotestutil.IsTargetGoFile(slashRel) {
			files = append(files, slashRel)
		}
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return files, nil
}

func (idx *CloneIndex) indexBlock(bIdx int, blk FuncCloneBlock) {
	for i := 0; i+idx.minWindow <= len(blk.Stmts); i++ {
		sig := joinWindowTokens(blk.Stmts[i : i+idx.minWindow])
		idx.windows[sig] = append(idx.windows[sig], windowAnchor{blockIdx: bIdx, stmtIdx: i})
	}
}

func (idx *CloneIndex) measureMatch(target FuncCloneBlock, tStart int, anchor windowAnchor) CloneMatch {
	peer := idx.blocks[anchor.blockIdx]
	if peer.File == target.File && peer.FuncKey == target.FuncKey {
		return CloneMatch{}
	}
	length := idx.minWindow
	for tStart+length < len(target.Stmts) &&
		anchor.stmtIdx+length < len(peer.Stmts) &&
		target.Stmts[tStart+length].Token == peer.Stmts[anchor.stmtIdx+length].Token {
		length++
	}
	return CloneMatch{
		Statements: length,
		LocalStart: target.Stmts[tStart].Line,
		LocalEnd:   target.Stmts[tStart+length-1].Line,
		PeerFile:   peer.File,
		PeerFunc:   peer.FuncName,
		PeerStart:  peer.Stmts[anchor.stmtIdx].Line,
		PeerEnd:    peer.Stmts[anchor.stmtIdx+length-1].Line,
	}
}

func joinWindowTokens(stmts []StmtFingerprint) string {
	var b strings.Builder
	for _, s := range stmts {
		b.WriteString(s.Token)
		b.WriteByte('\n')
	}
	return b.String()
}

func flattenBodyFingerprints(fset *token.FileSet, body *ast.BlockStmt) []StmtFingerprint {
	var out []StmtFingerprint
	ast.Inspect(body, func(n ast.Node) bool {
		stmt, ok := n.(ast.Stmt)
		if !ok {
			return true
		}
		switch stmt.(type) {
		case *ast.BlockStmt, *ast.LabeledStmt, *ast.EmptyStmt:
			return true
		}
		out = append(out, StmtFingerprint{
			Token: fingerprintStmt(stmt),
			Line:  fset.Position(stmt.Pos()).Line,
		})
		return true
	})
	return out
}

func fingerprintStmt(stmt ast.Stmt) string {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		return fmt.Sprintf("ASSIGN(%s,%d,%s)", s.Tok, len(s.Lhs), fingerprintExprList(s.Rhs))
	case *ast.ReturnStmt:
		return "RET(" + fingerprintExprList(s.Results) + ")"
	case *ast.IfStmt:
		return "IF(" + fingerprintExpr(s.Cond) + ")"
	case *ast.ForStmt:
		return "FOR(" + fingerprintExpr(s.Cond) + ")"
	case *ast.RangeStmt:
		return fmt.Sprintf("RANGE(%s,%s)", s.Tok, fingerprintExpr(s.X))
	case *ast.SwitchStmt:
		return "SWITCH(" + fingerprintExpr(s.Tag) + ")"
	case *ast.CaseClause:
		return "CASE(" + fingerprintCaseList(s.List) + ")"
	case *ast.ExprStmt:
		return "EXPR(" + fingerprintExpr(s.X) + ")"
	case *ast.DeclStmt:
		return "DECL"
	case *ast.BranchStmt:
		return "BRANCH(" + s.Tok.String() + ")"
	case *ast.DeferStmt:
		return "DEFER(" + fingerprintExpr(s.Call) + ")"
	case *ast.SendStmt:
		return "SEND(" + fingerprintExpr(s.Value) + ")"
	case *ast.IncDecStmt:
		return "INCDEC(" + s.Tok.String() + ")"
	default:
		return fmt.Sprintf("STMT(%T)", stmt)
	}
}

func fingerprintCaseList(exprs []ast.Expr) string {
	if len(exprs) == 0 {
		return "default"
	}
	parts := make([]string, len(exprs))
	for i, e := range exprs {
		switch x := e.(type) {
		case *ast.Ident:
			parts[i] = x.Name
		case *ast.SelectorExpr:
			parts[i] = fingerprintCallee(x)
		default:
			parts[i] = fingerprintExpr(e)
		}
	}
	return strings.Join(parts, ",")
}

func fingerprintExprList(exprs []ast.Expr) string {
	parts := make([]string, len(exprs))
	for i, e := range exprs {
		parts[i] = fingerprintExpr(e)
	}
	return strings.Join(parts, ",")
}

func fingerprintExpr(expr ast.Expr) string {
	if expr == nil {
		return "_"
	}
	switch e := expr.(type) {
	case *ast.Ident:
		return "$id"
	case *ast.BasicLit:
		return "$lit:" + e.Kind.String()
	case *ast.BinaryExpr:
		return fmt.Sprintf("BIN(%s,%s,%s)", e.Op, fingerprintExpr(e.X), fingerprintExpr(e.Y))
	case *ast.UnaryExpr:
		return fmt.Sprintf("UN(%s,%s)", e.Op, fingerprintExpr(e.X))
	case *ast.CallExpr:
		return fmt.Sprintf("CALL(%s,%d)", fingerprintCallee(e.Fun), len(e.Args))
	case *ast.SelectorExpr:
		return "SEL(." + e.Sel.Name + ")"
	case *ast.IndexExpr:
		return "IDX(" + fingerprintExpr(e.X) + ")"
	case *ast.CompositeLit:
		return fmt.Sprintf("COMP(%d)", len(e.Elts))
	case *ast.StarExpr:
		return "PTR(" + fingerprintExpr(e.X) + ")"
	default:
		return fmt.Sprintf("EX(%T)", expr)
	}
}

func fingerprintCallee(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return "." + f.Sel.Name
	default:
		return "fn"
	}
}

func shouldSkipWalkDir(slashRel string) bool {
	if slashRel == "." {
		return false
	}
	base := filepath.Base(slashRel)
	if strings.HasPrefix(base, ".") || base == "node_modules" || base == "vendor" || base == "testdata" {
		return true
	}
	return slashRel == "web" || strings.HasPrefix(slashRel, "pkg/generated") || strings.HasPrefix(slashRel, "pkg/testutil")
}
