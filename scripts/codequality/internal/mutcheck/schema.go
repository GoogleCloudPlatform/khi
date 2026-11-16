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
	"bufio"
	"cmp"
	"errors"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/overlay"
)

const (
	runtimeHelperFileName = "zzz_khi_mutant_runtime.go"
	runtimeTemplate       = `package %s

import (
	"cmp"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
)

type __khiMutNumeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

type __khiMutAddable interface {
	__khiMutNumeric | ~string
}

var __khiMutSelected = sync.OnceValue(func() int {
	v, err := strconv.Atoi(os.Getenv("KHI_MUTANT_ID"))
	if err != nil {
		return 0
	}
	return v
})

var __khiMutTracePath = sync.OnceValue(func() string {
	return os.Getenv("KHI_MUTANT_TRACE_FILE")
})

var __khiMutMu sync.Mutex
var __khiMutSeen [%d][3]atomic.Bool

func __khiMutRecordSlot(id int, slot int, suffix string) {
	if id <= 0 || id >= len(__khiMutSeen) {
		return
	}
	if __khiMutSeen[id][slot].Load() || __khiMutSeen[id][slot].Swap(true) {
		return
	}
	p := __khiMutTracePath()
	if p == "" {
		return
	}
	__khiMutMu.Lock()
	defer __khiMutMu.Unlock()
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	_, _ = f.WriteString(strconv.Itoa(id) + suffix)
	_ = f.Close()
}

func __khiMutRecordBool(id int, v bool) {
	if v {
		__khiMutRecordSlot(id, 2, ":T\n")
	} else {
		__khiMutRecordSlot(id, 1, ":F\n")
	}
}

func __khiMutActive(id int) bool {
	sel := __khiMutSelected()
	if sel == 0 {
		__khiMutRecordSlot(id, 0, ":S\n")
		return false
	}
	return sel == id
}

func __khiMutFlip(id int, v bool) bool {
	sel := __khiMutSelected()
	if sel == 0 {
		__khiMutRecordBool(id, v)
		return v
	}
	if sel == id {
		return !v
	}
	return v
}

func __khiMutCond(idTrue, idFalse int, cond bool) bool {
	sel := __khiMutSelected()
	if sel == 0 {
		__khiMutRecordBool(idTrue, cond)
		__khiMutRecordBool(idFalse, cond)
		return cond
	}
	if sel == idTrue {
		return true
	}
	if sel == idFalse {
		return false
	}
	return cond
}

func __khiMutAnd(id int, lhs, rhs func() bool) bool {
	sel := __khiMutSelected()
	if sel == 0 {
		res := lhs() && rhs()
		__khiMutRecordBool(id, res)
		return res
	}
	if sel == id {
		return lhs() || rhs()
	}
	return lhs() && rhs()
}

func __khiMutOr(id int, lhs, rhs func() bool) bool {
	sel := __khiMutSelected()
	if sel == 0 {
		res := lhs() || rhs()
		__khiMutRecordBool(id, res)
		return res
	}
	if sel == id {
		return lhs() && rhs()
	}
	return lhs() || rhs()
}

func __khiMutLT[T cmp.Ordered](id1, id2 int, a, b T) bool {
	sel := __khiMutSelected()
	if sel == 0 {
		res := a < b
		__khiMutRecordBool(id1, res)
		__khiMutRecordBool(id2, res)
		return res
	}
	if sel == id1 {
		return a <= b
	}
	if sel == id2 {
		return a >= b
	}
	return a < b
}

func __khiMutLE[T cmp.Ordered](id1, id2 int, a, b T) bool {
	sel := __khiMutSelected()
	if sel == 0 {
		res := a <= b
		__khiMutRecordBool(id1, res)
		__khiMutRecordBool(id2, res)
		return res
	}
	if sel == id1 {
		return a < b
	}
	if sel == id2 {
		return a > b
	}
	return a <= b
}

func __khiMutGT[T cmp.Ordered](id1, id2 int, a, b T) bool {
	sel := __khiMutSelected()
	if sel == 0 {
		res := a > b
		__khiMutRecordBool(id1, res)
		__khiMutRecordBool(id2, res)
		return res
	}
	if sel == id1 {
		return a >= b
	}
	if sel == id2 {
		return a <= b
	}
	return a > b
}

func __khiMutGE[T cmp.Ordered](id1, id2 int, a, b T) bool {
	sel := __khiMutSelected()
	if sel == 0 {
		res := a >= b
		__khiMutRecordBool(id1, res)
		__khiMutRecordBool(id2, res)
		return res
	}
	if sel == id1 {
		return a > b
	}
	if sel == id2 {
		return a < b
	}
	return a >= b
}

func __khiMutAddL[T __khiMutAddable](id int, a, b T) T {
	if __khiMutActive(id) {
		return a
	}
	return a + b
}

func __khiMutAddR[T __khiMutAddable](id int, a, b T) T {
	if __khiMutActive(id) {
		return b
	}
	return a + b
}

func __khiMutSub[T __khiMutNumeric](id int, a, b T) T {
	if __khiMutActive(id) {
		return a + b
	}
	return a - b
}

func __khiMutMul[T __khiMutNumeric](id int, a, b T) T {
	if __khiMutActive(id) {
		return a / b
	}
	return a * b
}

func __khiMutDiv[T __khiMutNumeric](id int, a, b T) T {
	if __khiMutActive(id) {
		return a * b
	}
	return a / b
}
`
)

// OverlayFileSpec holds a single mutated Go file to materialize into a temporary overlay directory.
type OverlayFileSpec struct {
	// RelFile is the slash-separated relative path of the Go file.
	RelFile string
	// Source holds the original Go source bytes.
	Source []byte
	// Sites lists mutation sites to render into the overlay source.
	Sites []MutationSite
}

// OverlayRequest configures Super-Mutant source staging and overlay.json creation for one package.
type OverlayRequest struct {
	// RepoRoot is the repository root directory path.
	RepoRoot string
	// PkgRelDir is the package directory relative to RepoRoot.
	PkgRelDir string
	// PkgName is the declared Go package name.
	PkgName string
	// Files lists mutated file specifications to materialize.
	Files []OverlayFileSpec
	// MaxMutantID is the highest mutant ID allocated for sizing the runtime bitmap.
	MaxMutantID int
	// WorkDir is the temporary directory where the overlay files are written.
	WorkDir string
}

type spanBounds struct {
	start      int
	end        int
	excludeIdx int
}

// RenderSuperMutantSource splices activeSites into src from innermost to outermost AST spans.
func RenderSuperMutantSource(src []byte, activeSites []MutationSite) []byte {
	if len(activeSites) == 0 {
		return src
	}
	sorted := make([]MutationSite, len(activeSites))
	copy(sorted, activeSites)
	slices.SortFunc(sorted, func(a, b MutationSite) int {
		return cmp.Or(
			cmp.Compare(a.StartOffset, b.StartOffset),
			cmp.Compare(b.EndOffset, a.EndOffset),
		)
	})
	return []byte(renderSpan(src, sorted, spanBounds{start: 0, end: len(src), excludeIdx: -1}))
}

// WritePackageOverlay materializes Super-Mutant Go sources, runtime helper, and overlay.json inside req.WorkDir.
func WritePackageOverlay(req OverlayRequest) (string, error) {
	absRoot, err := filepath.Abs(req.RepoRoot)
	if err != nil {
		return "", fmt.Errorf("failed to resolve repo root: %w", err)
	}
	absWork, err := filepath.Abs(req.WorkDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve overlay workdir: %w", err)
	}

	replace := make(map[string]string, len(req.Files)+1)
	srcStaging := filepath.Join(absWork, "src")

	for _, f := range req.Files {
		if len(f.Sites) == 0 {
			continue
		}
		rendered := RenderSuperMutantSource(f.Source, f.Sites)
		dstPath := filepath.Join(srcStaging, filepath.FromSlash(f.RelFile))
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return "", fmt.Errorf("failed to create overlay dir: %w", err)
		}
		if err := os.WriteFile(dstPath, rendered, 0o600); err != nil {
			return "", fmt.Errorf("failed to write mutated file %s: %w", f.RelFile, err)
		}
		replace[filepath.Join(absRoot, filepath.FromSlash(f.RelFile))] = dstPath
	}

	runtimeBytes := GenerateRuntimeSource(req.PkgName, req.MaxMutantID)
	runtimeDst := filepath.Join(srcStaging, filepath.FromSlash(req.PkgRelDir), runtimeHelperFileName)
	if err := os.MkdirAll(filepath.Dir(runtimeDst), 0o755); err != nil {
		return "", fmt.Errorf("failed to create runtime helper dir: %w", err)
	}
	if err := os.WriteFile(runtimeDst, runtimeBytes, 0o600); err != nil {
		return "", fmt.Errorf("failed to write runtime helper: %w", err)
	}
	replace[filepath.Join(absRoot, filepath.FromSlash(req.PkgRelDir), runtimeHelperFileName)] = runtimeDst

	return overlay.WriteOverlayJSON(absWork, "overlay.json", replace)
}

// ParsePolarityTraceFile reads directed polarity tokens (e.g. "12:T", "12:F", "12:S") and reached IDs from tracePath.
func ParsePolarityTraceFile(tracePath string) (map[string]struct{}, map[int]struct{}, error) {
	tokens := make(map[string]struct{})
	reached := make(map[int]struct{})
	f, err := os.Open(tracePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return tokens, reached, nil
		}
		return nil, nil, fmt.Errorf("failed to open trace file: %w", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if err := parseTraceLine(strings.TrimSpace(sc.Text()), tokens, reached); err != nil {
			return nil, nil, err
		}
	}
	if err := sc.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed reading trace file: %w", err)
	}
	return tokens, reached, nil
}

// GenerateRuntimeSource returns the Go source for zzz_khi_mutant_runtime.go for package pkgName.
func GenerateRuntimeSource(pkgName string, maxMutantID int) []byte {
	seenCap := max(maxMutantID+1, 2)
	content := fmt.Sprintf(runtimeTemplate, pkgName, seenCap)
	return []byte(content)
}

func renderSpan(src []byte, sites []MutationSite, bounds spanBounds) string {
	var sb strings.Builder
	cursor := bounds.start
	for i := range sites {
		if shouldSkipNestedSite(sites, i, cursor, bounds) {
			continue
		}
		s := &sites[i]
		sb.Write(src[cursor:s.StartOffset])
		sb.WriteString(renderSingleSite(src, sites, i))
		cursor = s.EndOffset
	}
	sb.Write(src[cursor:bounds.end])
	return sb.String()
}

func shouldSkipNestedSite(sites []MutationSite, idx int, cursor int, bounds spanBounds) bool {
	if idx == bounds.excludeIdx {
		return true
	}
	s := &sites[idx]
	if s.StartOffset < cursor || s.EndOffset > bounds.end {
		return true
	}
	if bounds.excludeIdx >= 0 {
		parent := &sites[bounds.excludeIdx]
		return s.StartOffset == parent.StartOffset && s.EndOffset == parent.EndOffset
	}
	return false
}

func renderSingleSite(src []byte, sites []MutationSite, idx int) string {
	s := &sites[idx]
	switch s.Kind {
	case siteOrderedCmp:
		return renderOrderedCmpSite(src, sites, idx)
	case siteEqCmp:
		return renderEqCmpSite(src, sites, idx)
	case siteLogical:
		return renderLogicalSite(src, sites, idx)
	case siteCond:
		inner := renderSpan(src, sites, spanBounds{start: s.StartOffset, end: s.EndOffset, excludeIdx: idx})
		return fmt.Sprintf("__khiMutCond(%d, %d, %s)", s.Mutants[0].ID, s.Mutants[1].ID, inner)
	case siteArith:
		return renderArithSite(src, sites, idx)
	case siteUnaryNot:
		op := renderSpan(src, sites, spanBounds{start: s.LeftStart, end: s.LeftEnd, excludeIdx: idx})
		return fmt.Sprintf("__khiMutFlip(%d, !(%s))", s.Mutants[0].ID, op)
	case siteBoolLit:
		return renderBoolLitSite(s)
	case siteStmtSkip:
		st := renderSpan(src, sites, spanBounds{start: s.StartOffset, end: s.EndOffset, excludeIdx: idx})
		return fmt.Sprintf("if !__khiMutActive(%d) { %s }", s.Mutants[0].ID, st)
	default:
		return string(src[s.StartOffset:s.EndOffset])
	}
}

func renderOrderedCmpSite(src []byte, sites []MutationSite, idx int) string {
	s := &sites[idx]
	lhs := renderSpan(src, sites, spanBounds{start: s.LeftStart, end: s.LeftEnd, excludeIdx: idx})
	rhs := renderSpan(src, sites, spanBounds{start: s.RightStart, end: s.RightEnd, excludeIdx: idx})
	fn := orderedRuntimeFunc(s.Op)
	return fmt.Sprintf("%s(%d, %d, %s, %s)", fn, s.Mutants[0].ID, s.Mutants[1].ID, lhs, rhs)
}

func orderedRuntimeFunc(op token.Token) string {
	switch op {
	case token.LEQ:
		return "__khiMutLE"
	case token.GTR:
		return "__khiMutGT"
	case token.GEQ:
		return "__khiMutGE"
	default:
		return "__khiMutLT"
	}
}

func renderEqCmpSite(src []byte, sites []MutationSite, idx int) string {
	s := &sites[idx]
	lhs := renderSpan(src, sites, spanBounds{start: s.LeftStart, end: s.LeftEnd, excludeIdx: idx})
	rhs := renderSpan(src, sites, spanBounds{start: s.RightStart, end: s.RightEnd, excludeIdx: idx})
	opStr := "=="
	if s.Op == token.NEQ {
		opStr = "!="
	}
	return fmt.Sprintf("__khiMutFlip(%d, %s %s %s)", s.Mutants[0].ID, lhs, opStr, rhs)
}

func renderLogicalSite(src []byte, sites []MutationSite, idx int) string {
	s := &sites[idx]
	lhs := renderSpan(src, sites, spanBounds{start: s.LeftStart, end: s.LeftEnd, excludeIdx: idx})
	rhs := renderSpan(src, sites, spanBounds{start: s.RightStart, end: s.RightEnd, excludeIdx: idx})
	fn := "__khiMutAnd"
	if s.Op == token.LOR {
		fn = "__khiMutOr"
	}
	return fmt.Sprintf("%s(%d, func() bool { return %s }, func() bool { return %s })", fn, s.Mutants[0].ID, lhs, rhs)
}

func renderArithSite(src []byte, sites []MutationSite, idx int) string {
	s := &sites[idx]
	lhs := renderSpan(src, sites, spanBounds{start: s.LeftStart, end: s.LeftEnd, excludeIdx: idx})
	rhs := renderSpan(src, sites, spanBounds{start: s.RightStart, end: s.RightEnd, excludeIdx: idx})
	fn := arithRuntimeFunc(s)
	return fmt.Sprintf("%s(%d, %s, %s)", fn, s.Mutants[0].ID, lhs, rhs)
}

func arithRuntimeFunc(s *MutationSite) string {
	switch s.Op {
	case token.ADD:
		if s.AddKeepRight {
			return "__khiMutAddR"
		}
		return "__khiMutAddL"
	case token.SUB:
		return "__khiMutSub"
	case token.MUL:
		return "__khiMutMul"
	default:
		return "__khiMutDiv"
	}
}

func renderBoolLitSite(s *MutationSite) string {
	lit := "false"
	if s.BoolVal {
		lit = "true"
	}
	return fmt.Sprintf("__khiMutFlip(%d, %s)", s.Mutants[0].ID, lit)
}

func parseTraceLine(txt string, tokens map[string]struct{}, reached map[int]struct{}) error {
	if txt == "" {
		return nil
	}
	idPart, polPart, hasColon := strings.Cut(txt, ":")
	if !hasColon {
		polPart = "S"
	}
	id, err := strconv.Atoi(idPart)
	if err != nil {
		return fmt.Errorf("invalid mutant id %q in trace file: %w", txt, err)
	}
	if id > 0 {
		reached[id] = struct{}{}
		tokens[strconv.Itoa(id)+":"+polPart] = struct{}{}
	}
	return nil
}
