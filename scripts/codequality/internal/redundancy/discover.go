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

package redundancy

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

var outcomeDurationRe = regexp.MustCompile(`\s+\([^)]*\)$`)

// SubtestMeta holds source location, diff membership, and assertion strength for a table row or t.Run subtest.
type SubtestMeta struct {
	// NormalizedName is the normalized Go subtest path segment.
	NormalizedName string
	// Line is the 1-based source line number of the subtest or table row.
	Line int
	// InDiff reports whether the subtest overlaps the modified line set.
	InDiff bool
	// OnlyErrorNil reports whether the subtest body contains only err != nil checks.
	OnlyErrorNil bool
}

// ChangedTestFunc describes a top-level Test* function in a *_test.go file overlapping the VCS diff.
type ChangedTestFunc struct {
	// File is the slash-separated relative path of the test file.
	File string
	// FuncName is the top-level Test* function name.
	FuncName string
	// Line is the 1-based starting line of the test function.
	Line int
	// EndLine is the 1-based ending line of the test function.
	EndLine int
	// EntireFuncChanged reports whether the function declaration line itself is in the diff.
	EntireFuncChanged bool
	// OnlyErrorNil reports whether the test body contains only err != nil checks.
	OnlyErrorNil bool
	// Subtests maps normalized subtest names to extracted SubtestMeta.
	Subtests map[string]SubtestMeta
}

// VerboseTestResult holds parsed per-leaf execution and failure state from a `go test -v` run.
type VerboseTestResult struct {
	// LeafOrder lists executed leaf subtest names in execution order.
	LeafOrder []string
	// RanLeafSet contains all executed non-skipped leaf test names.
	RanLeafSet map[string]struct{}
	// FailedLeafs contains all leaf test names that failed.
	FailedLeafs map[string]struct{}
}

type subtestExtractor struct {
	fset           *token.FileSet
	changedLines   map[int]struct{}
	loopOnlyErrNil bool
	result         map[string]SubtestMeta
}

type errNilOnlyWalker struct {
	hasNonErrAssertion bool
}

type verboseScanState struct {
	seenOrder  []string
	seenSet    map[string]struct{}
	skippedSet map[string]struct{}
	failedAll  map[string]struct{}
	activeLeaf string
}

// DiscoverChangedTestFuncs parses src and extracts Test* functions overlapping changedLines.
func DiscoverChangedTestFuncs(filePath string, src []byte, changedLines map[int]struct{}) ([]ChangedTestFunc, error) {
	if len(changedLines) == 0 {
		return nil, nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, src, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse test file %s: %w", filePath, err)
	}

	var out []ChangedTestFunc
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || !isUnitTestFuncDecl(fn) {
			continue
		}
		startLine := fset.Position(fn.Pos()).Line
		endLine := fset.Position(fn.End()).Line
		if !astcheck.OverlapsLines(startLine, endLine, changedLines) {
			continue
		}
		_, headerChanged := changedLines[startLine]
		funcOnlyErrNil := bodyHasOnlyErrorNilChecks(fn.Body)
		out = append(out, ChangedTestFunc{
			File:              filePath,
			FuncName:          fn.Name.Name,
			Line:              startLine,
			EndLine:           endLine,
			EntireFuncChanged: headerChanged,
			OnlyErrorNil:      funcOnlyErrNil,
			Subtests:          extractSubtestMetadata(fset, fn.Body, changedLines, funcOnlyErrNil),
		})
	}
	return out, nil
}

// ParseVerboseTestOutput extracts executed leaf tests and failed leaf tests from `go test -v` output.
func ParseVerboseTestOutput(out []byte, exitedNonZero bool) VerboseTestResult {
	st := &verboseScanState{
		seenSet:    make(map[string]struct{}),
		skippedSet: make(map[string]struct{}),
		failedAll:  make(map[string]struct{}),
	}
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		st.step(strings.TrimSpace(sc.Text()))
	}
	return st.finalize(exitedNonZero)
}

// BuildTestFuncsRunRegex builds a -test.run regex matching any of the given top-level Test* functions.
func BuildTestFuncsRunRegex(funcs []ChangedTestFunc) string {
	if len(funcs) == 0 {
		return ""
	}
	quoted := make([]string, 0, len(funcs))
	for _, fn := range funcs {
		quoted = append(quoted, regexp.QuoteMeta(fn.FuncName))
	}
	return "^(?:" + strings.Join(quoted, "|") + ")$"
}

// BuildSingleLeafRunRegex builds a slash-segmented -test.run pattern matching a single leaf test or subtest.
func BuildSingleLeafRunRegex(leafName string) string {
	segments := strings.Split(leafName, "/")
	parts := make([]string, len(segments))
	for i, seg := range segments {
		parts[i] = "^" + regexp.QuoteMeta(seg) + "$"
	}
	return strings.Join(parts, "/")
}

// SplitTopAndSubtest splits a Go test name ("TestFoo/case_a") into ("TestFoo", "case_a").
func SplitTopAndSubtest(leaf string) (string, string) {
	top, sub, _ := strings.Cut(leaf, "/")
	return top, sub
}

func extractSubtestMetadata(fset *token.FileSet, body *ast.BlockStmt, changed map[int]struct{}, defaultOnlyErrNil bool) map[string]SubtestMeta {
	ex := &subtestExtractor{
		fset:           fset,
		changedLines:   changed,
		loopOnlyErrNil: defaultOnlyErrNil,
		result:         make(map[string]SubtestMeta),
	}
	ast.Inspect(body, ex.visitRunCalls)
	ast.Inspect(body, ex.visitTableSlices)
	return ex.result
}

func (e *subtestExtractor) visitRunCalls(n ast.Node) bool {
	call, ok := n.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 {
		return true
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Run" {
		return true
	}
	fnLit, ok := call.Args[1].(*ast.FuncLit)
	if !ok {
		return true
	}
	subOnlyErr := bodyHasOnlyErrorNilChecks(fnLit.Body)
	nameLit, isLit := stringLiteralValue(call.Args[0])
	if !isLit {
		e.loopOnlyErrNil = subOnlyErr
		return true
	}
	norm := normalizeGoSubtestName(nameLit)
	sLine := e.fset.Position(call.Pos()).Line
	eLine := e.fset.Position(call.End()).Line
	e.result[norm] = SubtestMeta{
		NormalizedName: norm,
		Line:           sLine,
		InDiff:         astcheck.OverlapsLines(sLine, eLine, e.changedLines),
		OnlyErrorNil:   subOnlyErr,
	}
	return true
}

func (e *subtestExtractor) visitTableSlices(n ast.Node) bool {
	comp, ok := n.(*ast.CompositeLit)
	if !ok {
		return true
	}
	if _, isSlice := comp.Type.(*ast.ArrayType); !isSlice {
		return true
	}
	for _, elt := range comp.Elts {
		e.recordTableRow(elt)
	}
	return true
}

func (e *subtestExtractor) recordTableRow(elt ast.Expr) {
	rowLit, ok := elt.(*ast.CompositeLit)
	if !ok {
		return
	}
	caseName, found := extractTableRowName(rowLit)
	if !found {
		return
	}
	norm := normalizeGoSubtestName(caseName)
	if _, exists := e.result[norm]; exists {
		return
	}
	sLine := e.fset.Position(rowLit.Pos()).Line
	eLine := e.fset.Position(rowLit.End()).Line
	e.result[norm] = SubtestMeta{
		NormalizedName: norm,
		Line:           sLine,
		InDiff:         astcheck.OverlapsLines(sLine, eLine, e.changedLines),
		OnlyErrorNil:   e.loopOnlyErrNil,
	}
}

func extractTableRowName(row *ast.CompositeLit) (string, bool) {
	for _, elt := range row.Elts {
		if name, ok := extractKeyedRowName(elt); ok {
			return name, true
		}
	}
	if len(row.Elts) > 0 {
		if _, isKV := row.Elts[0].(*ast.KeyValueExpr); !isKV {
			if val, _ := stringLiteralValue(row.Elts[0]); val != "" {
				return val, true
			}
		}
	}
	return "", false
}

func extractKeyedRowName(elt ast.Expr) (string, bool) {
	kv, ok := elt.(*ast.KeyValueExpr)
	if !ok {
		return "", false
	}
	keyIdent, ok := kv.Key.(*ast.Ident)
	if !ok {
		return "", false
	}
	switch strings.ToLower(keyIdent.Name) {
	case "name", "desc", "testname":
		val, _ := stringLiteralValue(kv.Value)
		return val, val != ""
	default:
		return "", false
	}
}

func stringLiteralValue(expr ast.Expr) (string, bool) {
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

func normalizeGoSubtestName(raw string) string {
	var sb strings.Builder
	sb.Grow(len(raw))
	for _, r := range raw {
		switch {
		case unicode.IsSpace(r):
			sb.WriteByte('_')
		case !strconv.IsPrint(r):
			quoted := strconv.QuoteRune(r)
			sb.WriteString(quoted[1 : len(quoted)-1])
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func isUnitTestFuncDecl(fn *ast.FuncDecl) bool {
	if fn.Recv != nil {
		return false
	}
	name := fn.Name.Name
	if !strings.HasPrefix(name, "Test") || name == "TestMain" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(name[len("Test"):])
	if unicode.IsLower(r) {
		return false
	}
	return len(fn.Type.Params.List) == 1
}

func bodyHasOnlyErrorNilChecks(block *ast.BlockStmt) bool {
	checker := &errNilOnlyWalker{}
	checker.walk(block, false)
	return !checker.hasNonErrAssertion
}

func (w *errNilOnlyWalker) walk(n ast.Node, insideErrGuard bool) {
	if n == nil {
		return
	}
	if ifStmt, ok := n.(*ast.IfStmt); ok && rule.IsErrorNotNilGuard(ifStmt.Cond) && ifStmt.Else == nil {
		w.walk(ifStmt.Init, insideErrGuard)
		w.walk(ifStmt.Body, true)
		return
	}
	if !insideErrGuard && isTestingFailCall(n) {
		w.hasNonErrAssertion = true
		return
	}
	ast.Inspect(n, func(child ast.Node) bool {
		if child == n {
			return true
		}
		if child != nil {
			w.walk(child, insideErrGuard)
		}
		return false
	})
}

func isTestingFailCall(n ast.Node) bool {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	switch sel.Sel.Name {
	case "Error", "Errorf", "Fatal", "Fatalf", "Fail", "FailNow":
		return true
	default:
		return false
	}
}

func (s *verboseScanState) step(line string) {
	switch {
	case strings.HasPrefix(line, "=== RUN "):
		s.onRun(strings.TrimSpace(strings.TrimPrefix(line, "=== RUN ")))
	case strings.HasPrefix(line, "=== CONT "):
		s.activeLeaf = strings.TrimSpace(strings.TrimPrefix(line, "=== CONT "))
	case strings.HasPrefix(line, "--- PASS: "):
		s.clearActiveIf(parseOutcomeTestName(strings.TrimPrefix(line, "--- PASS: ")))
	case strings.HasPrefix(line, "--- SKIP: "):
		name := parseOutcomeTestName(strings.TrimPrefix(line, "--- SKIP: "))
		s.skippedSet[name] = struct{}{}
		s.clearActiveIf(name)
	case strings.HasPrefix(line, "--- FAIL: "):
		name := parseOutcomeTestName(strings.TrimPrefix(line, "--- FAIL: "))
		s.failedAll[name] = struct{}{}
		s.clearActiveIf(name)
	}
}

func (s *verboseScanState) onRun(name string) {
	if _, ok := s.seenSet[name]; !ok {
		s.seenSet[name] = struct{}{}
		s.seenOrder = append(s.seenOrder, name)
	}
	s.activeLeaf = name
}

func (s *verboseScanState) clearActiveIf(name string) {
	if s.activeLeaf == name {
		s.activeLeaf = ""
	}
}

func (s *verboseScanState) isLeaf(name string) bool {
	prefix := name + "/"
	for other := range s.seenSet {
		if strings.HasPrefix(other, prefix) {
			return false
		}
	}
	return true
}

func (s *verboseScanState) finalize(exitedNonZero bool) VerboseTestResult {
	res := VerboseTestResult{
		RanLeafSet:  make(map[string]struct{}),
		FailedLeafs: make(map[string]struct{}),
	}
	for _, name := range s.seenOrder {
		if _, skipped := s.skippedSet[name]; skipped || !s.isLeaf(name) {
			continue
		}
		res.LeafOrder = append(res.LeafOrder, name)
		res.RanLeafSet[name] = struct{}{}
		if _, failed := s.failedAll[name]; failed {
			res.FailedLeafs[name] = struct{}{}
		}
	}
	if exitedNonZero && len(res.FailedLeafs) == 0 {
		if _, ok := res.RanLeafSet[s.activeLeaf]; ok {
			res.FailedLeafs[s.activeLeaf] = struct{}{}
		}
	}
	return res
}

func parseOutcomeTestName(rest string) string {
	return strings.TrimSpace(outcomeDurationRe.ReplaceAllString(rest, ""))
}
