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

package main

import (
	"bufio"
	"bytes"
	"cmp"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// RedundantTestKind classifies why a changed unit test or subtest was flagged as redundant.
type RedundantTestKind string

const (
	// RedundantDuplicate indicates a test exercises a subset/identical branch polarity and mutant kill set as another test.
	RedundantDuplicate RedundantTestKind = "DUPLICATE"
	// RedundantWeakAssertion indicates an error-nil-only test executes changed code with 0 kills already covered by another test.
	RedundantWeakAssertion RedundantTestKind = "WEAK_TEST"
)

// RedundantTestFinding describes a changed test or subtest whose fault-detection footprint is subsumed by another test.
type RedundantTestFinding struct {
	GlobalIndex     int
	Kind            RedundantTestKind
	File            string
	Line            int
	TestName        string
	DominatedBy     string
	DominatedByFile string
	DominatedByLine int
	CandidateKills  int
	DominatorKills  int
}

// SubtestMeta holds source location and diff metadata for a table case or t.Run subtest.
type SubtestMeta struct {
	NormalizedName string
	Line           int
	InDiff         bool
	OnlyErrorNil   bool
}

// ChangedTestFunc describes a top-level Test* function in a *_test.go file that overlaps the VCS diff.
type ChangedTestFunc struct {
	File              string
	FuncName          string
	Line              int
	EntireFuncChanged bool
	OnlyErrorNil      bool
	Subtests          map[string]SubtestMeta
}

// VerboseTestResult holds parsed per-leaf execution and failure state from a `go test -v` run.
type VerboseTestResult struct {
	LeafOrder   []string
	RanLeafSet  map[string]struct{}
	FailedLeafs map[string]struct{}
}

// DiscoverChangedTestFuncs parses a modified *_test.go file and extracts Test* functions overlapping changedLines.
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
		if !spanHasChangedLine(startLine, endLine, changedLines) {
			continue
		}

		_, headerChanged := changedLines[startLine]
		funcOnlyErrNil := bodyHasOnlyErrorNilChecks(fn.Body)
		subtests := extractSubtestMetadata(fset, fn.Body, changedLines, funcOnlyErrNil)

		out = append(out, ChangedTestFunc{
			File:              filePath,
			FuncName:          fn.Name.Name,
			Line:              startLine,
			EntireFuncChanged: headerChanged,
			OnlyErrorNil:      funcOnlyErrNil,
			Subtests:          subtests,
		})
	}
	return out, nil
}

func extractSubtestMetadata(fset *token.FileSet, body *ast.BlockStmt, changedLines map[int]struct{}, defaultOnlyErrNil bool) map[string]SubtestMeta {
	result := make(map[string]SubtestMeta)

	// Detect loop-body assertion strength when t.Run(tc.name, func(t *testing.T) { ... }) is used.
	loopRunOnlyErrNil := defaultOnlyErrNil
	ast.Inspect(body, func(n ast.Node) bool {
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
		if nameLit, ok := stringLiteralValue(call.Args[0]); ok {
			norm := normalizeGoSubtestName(nameLit)
			sLine := fset.Position(call.Pos()).Line
			eLine := fset.Position(call.End()).Line
			result[norm] = SubtestMeta{
				NormalizedName: norm,
				Line:           sLine,
				InDiff:         spanHasChangedLine(sLine, eLine, changedLines),
				OnlyErrorNil:   subOnlyErr,
			}
		} else {
			loopRunOnlyErrNil = subOnlyErr
		}
		return true
	})

	// Detect table-driven slice composite literals: []struct{...}{ {name: "foo", ...}, ... }.
	ast.Inspect(body, func(n ast.Node) bool {
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		if _, isSlice := comp.Type.(*ast.ArrayType); !isSlice {
			return true
		}
		for _, elt := range comp.Elts {
			rowLit, ok := elt.(*ast.CompositeLit)
			if !ok {
				continue
			}
			caseName, found := extractTableRowName(rowLit)
			if !found {
				continue
			}
			norm := normalizeGoSubtestName(caseName)
			if _, exists := result[norm]; exists {
				continue
			}
			sLine := fset.Position(rowLit.Pos()).Line
			eLine := fset.Position(rowLit.End()).Line
			result[norm] = SubtestMeta{
				NormalizedName: norm,
				Line:           sLine,
				InDiff:         spanHasChangedLine(sLine, eLine, changedLines),
				OnlyErrorNil:   loopRunOnlyErrNil,
			}
		}
		return true
	})

	return result
}

func extractTableRowName(row *ast.CompositeLit) (string, bool) {
	for _, elt := range row.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		keyIdent, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		lowerKey := strings.ToLower(keyIdent.Name)
		if lowerKey == "name" || lowerKey == "desc" || lowerKey == "testname" {
			if val, ok := stringLiteralValue(kv.Value); ok && val != "" {
				return val, true
			}
		}
	}
	if len(row.Elts) > 0 {
		if _, isKV := row.Elts[0].(*ast.KeyValueExpr); !isKV {
			if val, ok := stringLiteralValue(row.Elts[0]); ok && val != "" {
				return val, true
			}
		}
	}
	return "", false
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

// bodyHasOnlyErrorNilChecks reports whether a test block contains no failure calls outside `if ... err != nil` guards.
func bodyHasOnlyErrorNilChecks(block *ast.BlockStmt) bool {
	hasNonErrAssertion := false
	var walk func(n ast.Node, insideErrGuard bool)
	walk = func(n ast.Node, insideErrGuard bool) {
		if n == nil || hasNonErrAssertion {
			return
		}
		if ifStmt, ok := n.(*ast.IfStmt); ok && isErrorNotNilGuard(ifStmt.Cond) && ifStmt.Else == nil {
			if ifStmt.Init != nil {
				walk(ifStmt.Init, insideErrGuard)
			}
			walk(ifStmt.Body, true)
			return
		}
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				switch sel.Sel.Name {
				case "Error", "Errorf", "Fatal", "Fatalf", "Fail", "FailNow":
					if !insideErrGuard {
						hasNonErrAssertion = true
						return
					}
				}
			}
		}
		ast.Inspect(n, func(child ast.Node) bool {
			if child == n {
				return true
			}
			if child != nil {
				walk(child, insideErrGuard)
			}
			return false
		})
	}
	walk(block, false)
	return !hasNonErrAssertion
}

// ParseVerboseTestOutput extracts executed leaf tests and failed leaf tests from `go test -v` output.
func ParseVerboseTestOutput(out []byte, exitedNonZero bool) VerboseTestResult {
	var seenOrder []string
	seenSet := make(map[string]struct{})
	skippedSet := make(map[string]struct{})
	failedAll := make(map[string]struct{})
	activeLeaf := ""

	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(line, "=== RUN "):
			name := strings.TrimSpace(strings.TrimPrefix(line, "=== RUN "))
			if name != "" {
				if _, ok := seenSet[name]; !ok {
					seenSet[name] = struct{}{}
					seenOrder = append(seenOrder, name)
				}
				activeLeaf = name
			}
		case strings.HasPrefix(line, "=== CONT "):
			name := strings.TrimSpace(strings.TrimPrefix(line, "=== CONT "))
			if name != "" {
				activeLeaf = name
			}
		case strings.HasPrefix(line, "--- PASS: "):
			name := parseOutcomeTestName(strings.TrimPrefix(line, "--- PASS: "))
			if activeLeaf == name {
				activeLeaf = ""
			}
		case strings.HasPrefix(line, "--- SKIP: "):
			name := parseOutcomeTestName(strings.TrimPrefix(line, "--- SKIP: "))
			if name != "" {
				skippedSet[name] = struct{}{}
			}
			if activeLeaf == name {
				activeLeaf = ""
			}
		case strings.HasPrefix(line, "--- FAIL: "):
			name := parseOutcomeTestName(strings.TrimPrefix(line, "--- FAIL: "))
			if name != "" {
				failedAll[name] = struct{}{}
			}
			if activeLeaf == name {
				activeLeaf = ""
			}
		}
	}

	isLeaf := func(name string) bool {
		prefix := name + "/"
		for other := range seenSet {
			if strings.HasPrefix(other, prefix) {
				return false
			}
		}
		return true
	}

	res := VerboseTestResult{
		RanLeafSet:  make(map[string]struct{}),
		FailedLeafs: make(map[string]struct{}),
	}
	for _, name := range seenOrder {
		if !isLeaf(name) {
			continue
		}
		if _, skipped := skippedSet[name]; skipped {
			continue
		}
		res.LeafOrder = append(res.LeafOrder, name)
		res.RanLeafSet[name] = struct{}{}
		if _, failed := failedAll[name]; failed {
			res.FailedLeafs[name] = struct{}{}
		}
	}

	if exitedNonZero && len(res.FailedLeafs) == 0 && activeLeaf != "" {
		if _, ok := res.RanLeafSet[activeLeaf]; ok {
			res.FailedLeafs[activeLeaf] = struct{}{}
		}
	}
	return res
}

func parseOutcomeTestName(rest string) string {
	rest = strings.TrimSpace(rest)
	if idx := strings.LastIndex(rest, " ("); idx > 0 {
		return strings.TrimSpace(rest[:idx])
	}
	return rest
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

type leafDescriptor struct {
	name        string
	file        string
	line        int
	isCandidate bool
	weakAssert  bool
	orderIdx    int
	polarity    map[string]struct{}
	ranMutants  map[int]struct{}
	kills       map[int]struct{}
}

// rawTSSpecCase represents a single Jasmine spec declaration extracted by mutate_ts.js from a *.spec.ts file.
type rawTSSpecCase struct {
	FullName         string `json:"fullName"`
	Line             int    `json:"line"`
	StartLine        int    `json:"startLine"`
	EndLine          int    `json:"endLine"`
	ItStartLine      int    `json:"itStartLine"`
	ItEndLine        int    `json:"itEndLine"`
	SuiteHeaderLines []int  `json:"suiteHeaderLines"`
	WeakAssertion    bool   `json:"weakAssertion"`
}

// ChangedTSSpec holds source location and diff membership for a Jasmine spec in a modified *.spec.ts file.
type ChangedTSSpec struct {
	File          string
	FullName      string
	Line          int
	InDiff        bool
	WeakAssertion bool
}

// TSSpecOrderEntry identifies a single executed Jasmine spec in Pass 0 execution order.
type TSSpecOrderEntry struct {
	ID       string `json:"id"`
	FullName string `json:"fullName"`
}

// TSMutantSpecRun records which Jasmine spec IDs ran and failed during a single mutant evaluation pass.
type TSMutantSpecRun struct {
	RanSpecIDs    []string `json:"ranSpecIds"`
	FailedSpecIDs []string `json:"failedSpecIds"`
}

// DiscoverChangedTSSpecs maps AST-extracted Jasmine spec declarations against VCS changed lines for filePath.
func DiscoverChangedTSSpecs(filePath string, specCases []rawTSSpecCase, changedLines map[int]struct{}) []ChangedTSSpec {
	if len(changedLines) == 0 || len(specCases) == 0 {
		return nil
	}
	out := make([]ChangedTSSpec, 0, len(specCases))
	hasAnyInDiff := false
	for _, sc := range specCases {
		if sc.FullName == "" {
			continue
		}
		inDiff := spanHasChangedLine(sc.StartLine, sc.EndLine, changedLines) ||
			spanHasChangedLine(sc.ItStartLine, sc.ItEndLine, changedLines) ||
			anyLineInSet(sc.SuiteHeaderLines, changedLines)
		if inDiff {
			hasAnyInDiff = true
		}
		out = append(out, ChangedTSSpec{
			File:          filePath,
			FullName:      sc.FullName,
			Line:          sc.Line,
			InDiff:        inDiff,
			WeakAssertion: sc.WeakAssertion,
		})
	}
	if !hasAnyInDiff {
		return nil
	}
	return out
}

// IsSubjectMutantForTSSpec reports whether prodFile belongs to the directory or subtree under test of specFile.
func IsSubjectMutantForTSSpec(specFile string, prodFile string) bool {
	specDir := path.Dir(strings.TrimPrefix(filepath.ToSlash(specFile), "./"))
	prodDir := path.Dir(strings.TrimPrefix(filepath.ToSlash(prodFile), "./"))
	return prodDir == specDir || strings.HasPrefix(prodDir, specDir+"/")
}

// AnalyzeTSTestRedundancy identifies changed Jasmine specs whose polarity and kill sets are subsumed by another spec in the same spec file.
func AnalyzeTSTestRedundancy(
	changedSpecs []ChangedTSSpec,
	specOrder []TSSpecOrderEntry,
	specPolarities map[string][]string,
	mutantRuns map[string]TSMutantSpecRun,
	mutantFileByID map[int]string,
	budgetSkippedMutants map[int]struct{},
) []RedundantTestFinding {
	if len(changedSpecs) == 0 || len(specOrder) == 0 {
		return nil
	}

	specsByName := make(map[string][]ChangedTSSpec)
	specFileSet := make(map[string]struct{})
	hasCandidate := false
	for _, sp := range changedSpecs {
		specsByName[sp.FullName] = append(specsByName[sp.FullName], sp)
		specFileSet[sp.File] = struct{}{}
		if sp.InDiff {
			hasCandidate = true
		}
	}
	if !hasCandidate {
		return nil
	}

	ranBySpecID := make(map[string]map[int]struct{})
	killsBySpecID := make(map[string]map[int]struct{})
	for mutIDStr, runRes := range mutantRuns {
		mutID, err := strconv.Atoi(mutIDStr)
		if err != nil || mutID <= 0 {
			continue
		}
		for _, specID := range runRes.RanSpecIDs {
			m := ranBySpecID[specID]
			if m == nil {
				m = make(map[int]struct{})
				ranBySpecID[specID] = m
			}
			m[mutID] = struct{}{}
		}
		for _, specID := range runRes.FailedSpecIDs {
			m := killsBySpecID[specID]
			if m == nil {
				m = make(map[int]struct{})
				killsBySpecID[specID] = m
			}
			m[mutID] = struct{}{}
		}
	}

	cursorByName := make(map[string]int, len(specsByName))
	leavesByFile := make(map[string][]leafDescriptor, len(specFileSet))

	for idx, entry := range specOrder {
		if entry.ID == "" || entry.FullName == "" {
			continue
		}
		list := specsByName[entry.FullName]
		if len(list) == 0 {
			continue
		}
		cur := cursorByName[entry.FullName]
		if cur >= len(list) {
			continue
		}
		meta := list[cur]
		cursorByName[entry.FullName] = cur + 1

		rawTokens := specPolarities[entry.ID]
		if len(rawTokens) == 0 {
			continue
		}
		pol := filterTSPolarityTokensForSpec(meta.File, rawTokens, mutantFileByID)
		if len(pol) == 0 {
			continue
		}

		ranSubject := filterMutantIDsForSpec(meta.File, ranBySpecID[entry.ID], mutantFileByID)
		killsSubject := filterMutantIDsForSpec(meta.File, killsBySpecID[entry.ID], mutantFileByID)

		leavesByFile[meta.File] = append(leavesByFile[meta.File], leafDescriptor{
			name:        entry.FullName,
			file:        meta.File,
			line:        meta.Line,
			isCandidate: meta.InDiff,
			weakAssert:  meta.WeakAssertion,
			orderIdx:    idx,
			polarity:    pol,
			ranMutants:  ranSubject,
			kills:       killsSubject,
		})
	}

	sortedSpecFiles := make([]string, 0, len(leavesByFile))
	for f := range leavesByFile {
		sortedSpecFiles = append(sortedSpecFiles, f)
	}
	slices.Sort(sortedSpecFiles)

	var findings []RedundantTestFinding
	for _, specFile := range sortedSpecFiles {
		skippedSubject := filterMutantIDsForSpec(specFile, budgetSkippedMutants, mutantFileByID)
		findings = append(findings, evaluateLeafRedundancy(leavesByFile[specFile], skippedSubject)...)
	}
	return findings
}

func filterMutantIDsForSpec(specFile string, ids map[int]struct{}, mutantFileByID map[int]string) map[int]struct{} {
	if len(ids) == 0 {
		return nil
	}
	out := make(map[int]struct{}, len(ids))
	for id := range ids {
		if len(mutantFileByID) == 0 {
			out[id] = struct{}{}
			continue
		}
		if prodFile, ok := mutantFileByID[id]; ok && IsSubjectMutantForTSSpec(specFile, prodFile) {
			out[id] = struct{}{}
		}
	}
	return out
}

func filterTSPolarityTokensForSpec(specFile string, rawTokens []string, mutantFileByID map[int]string) map[string]struct{} {
	pol := make(map[string]struct{}, len(rawTokens))
	for _, tok := range rawTokens {
		if tok == "" {
			continue
		}
		if len(mutantFileByID) == 0 {
			pol[tok] = struct{}{}
			continue
		}
		idPart := tok
		if idx := strings.IndexByte(tok, ':'); idx > 0 {
			idPart = tok[:idx]
		}
		if strings.HasPrefix(idPart, "P") {
			if pID, err := strconv.Atoi(idPart[1:]); err == nil {
				if prodFile, ok := mutantFileByID[-pID]; ok && IsSubjectMutantForTSSpec(specFile, prodFile) {
					pol[tok] = struct{}{}
				}
			}
			continue
		}
		if mID, err := strconv.Atoi(idPart); err == nil {
			if prodFile, ok := mutantFileByID[mID]; ok && IsSubjectMutantForTSSpec(specFile, prodFile) {
				pol[tok] = struct{}{}
			}
		}
	}
	return pol
}

// AnalyzeTestRedundancy identifies changed tests or subtests whose polarity and kill sets are subsumed by another test.
func AnalyzeTestRedundancy(
	changedFuncs []ChangedTestFunc,
	leafOrder []string,
	polarityByLeaf map[string]map[string]struct{},
	mutantRuns map[int]VerboseTestResult,
	budgetSkippedMutants map[int]struct{},
) []RedundantTestFinding {
	if len(changedFuncs) == 0 || len(leafOrder) == 0 {
		return nil
	}

	funcByName := make(map[string]ChangedTestFunc, len(changedFuncs))
	for _, fn := range changedFuncs {
		funcByName[fn.FuncName] = fn
	}

	var leaves []leafDescriptor
	for idx, leaf := range leafOrder {
		topFn, subName := splitTopAndSubtest(leaf)
		fnMeta, tracked := funcByName[topFn]
		if !tracked {
			continue
		}
		pol := polarityByLeaf[leaf]
		if len(pol) == 0 {
			continue
		}

		desc := leafDescriptor{
			name:       leaf,
			file:       fnMeta.File,
			line:       fnMeta.Line,
			weakAssert: fnMeta.OnlyErrorNil,
			orderIdx:   idx,
			polarity:   pol,
			ranMutants: make(map[int]struct{}),
			kills:      make(map[int]struct{}),
		}

		if subName == "" {
			desc.isCandidate = true
		} else {
			subMeta, found := lookupSubtestMeta(fnMeta.Subtests, subName)
			if found {
				desc.line = subMeta.Line
				desc.weakAssert = subMeta.OnlyErrorNil
			}
			desc.isCandidate = fnMeta.EntireFuncChanged || (found && subMeta.InDiff)
		}

		for mutID, runRes := range mutantRuns {
			if _, ran := runRes.RanLeafSet[leaf]; ran {
				desc.ranMutants[mutID] = struct{}{}
			}
			if _, failed := runRes.FailedLeafs[leaf]; failed {
				desc.kills[mutID] = struct{}{}
			}
		}
		leaves = append(leaves, desc)
	}

	return evaluateLeafRedundancy(leaves, budgetSkippedMutants)
}

func evaluateLeafRedundancy(leaves []leafDescriptor, budgetSkippedMutants map[int]struct{}) []RedundantTestFinding {
	var findings []RedundantTestFinding
	for i := range leaves {
		cand := &leaves[i]
		if !cand.isCandidate || len(cand.ranMutants) == 0 {
			continue
		}
		if touchedBudgetSkippedMutant(cand.polarity, budgetSkippedMutants) {
			continue
		}

		if len(cand.kills) > 0 {
			if dom := findDominatingDuplicate(i, leaves); dom != nil {
				findings = append(findings, RedundantTestFinding{
					Kind:            RedundantDuplicate,
					File:            cand.file,
					Line:            cand.line,
					TestName:        cand.name,
					DominatedBy:     dom.name,
					DominatedByFile: dom.file,
					DominatedByLine: dom.line,
					CandidateKills:  len(cand.kills),
					DominatorKills:  len(dom.kills),
				})
			}
			continue
		}

		if cand.weakAssert {
			if dom := findDominatingCoveringTest(i, leaves); dom != nil {
				findings = append(findings, RedundantTestFinding{
					Kind:            RedundantWeakAssertion,
					File:            cand.file,
					Line:            cand.line,
					TestName:        cand.name,
					DominatedBy:     dom.name,
					DominatedByFile: dom.file,
					DominatedByLine: dom.line,
					CandidateKills:  0,
					DominatorKills:  len(dom.kills),
				})
			}
		}
	}

	slices.SortFunc(findings, func(a, b RedundantTestFinding) int {
		return cmp.Or(
			cmp.Compare(a.File, b.File),
			cmp.Compare(a.Line, b.Line),
			cmp.Compare(a.TestName, b.TestName),
		)
	})
	return findings
}

func findDominatingDuplicate(candIdx int, all []leafDescriptor) *leafDescriptor {
	cand := &all[candIdx]
	bestIdx := -1
	for j := range all {
		if j == candIdx {
			continue
		}
		dom := &all[j]
		if len(dom.kills) == 0 {
			continue
		}
		// Require identical branch-polarity equivalence class so multi-call integration tests never swallow focused unit tests.
		if !equalStringSet(cand.polarity, dom.polarity) {
			continue
		}
		if !killsSubsumedOnSharedRuns(cand, dom) {
			continue
		}
		if !dominatesInOrder(dom, cand) {
			continue
		}
		if bestIdx < 0 || dominatesInOrder(dom, &all[bestIdx]) {
			bestIdx = j
		}
	}
	if bestIdx < 0 {
		return nil
	}
	return &all[bestIdx]
}

func findDominatingCoveringTest(candIdx int, all []leafDescriptor) *leafDescriptor {
	cand := &all[candIdx]
	for j := range all {
		if j == candIdx {
			continue
		}
		dom := &all[j]
		if len(dom.kills) == 0 {
			continue
		}
		if isStringSubset(cand.polarity, dom.polarity) {
			return dom
		}
	}
	return nil
}

func killsSubsumedOnSharedRuns(cand *leafDescriptor, dom *leafDescriptor) bool {
	sharedKilled := 0
	for mutID := range cand.kills {
		if _, domRan := dom.ranMutants[mutID]; !domRan {
			return false
		}
		if _, domKilled := dom.kills[mutID]; !domKilled {
			return false
		}
		sharedKilled++
	}
	return sharedKilled > 0 && len(dom.kills) >= len(cand.kills)
}

func dominatesInOrder(dom *leafDescriptor, cand *leafDescriptor) bool {
	if len(dom.kills) != len(cand.kills) {
		return len(dom.kills) > len(cand.kills)
	}
	if len(dom.polarity) != len(cand.polarity) {
		return len(dom.polarity) > len(cand.polarity)
	}
	if dom.isCandidate != cand.isCandidate {
		return !dom.isCandidate
	}
	return dom.orderIdx < cand.orderIdx
}

func equalStringSet(a map[string]struct{}, b map[string]struct{}) bool {
	return len(a) == len(b) && isStringSubset(a, b)
}

func touchedBudgetSkippedMutant(polarity map[string]struct{}, skipped map[int]struct{}) bool {
	if len(skipped) == 0 {
		return false
	}
	for tok := range polarity {
		idStr := tok
		if idx := strings.IndexByte(tok, ':'); idx > 0 {
			idStr = tok[:idx]
		}
		if id, err := strconv.Atoi(idStr); err == nil {
			if _, ok := skipped[id]; ok {
				return true
			}
		}
	}
	return false
}

func isStringSubset(sub map[string]struct{}, sup map[string]struct{}) bool {
	if len(sub) > len(sup) {
		return false
	}
	for k := range sub {
		if _, ok := sup[k]; !ok {
			return false
		}
	}
	return true
}

func splitTopAndSubtest(leaf string) (string, string) {
	if idx := strings.IndexByte(leaf, '/'); idx >= 0 {
		return leaf[:idx], leaf[idx+1:]
	}
	return leaf, ""
}

func lookupSubtestMeta(subtests map[string]SubtestMeta, subName string) (SubtestMeta, bool) {
	if m, ok := subtests[subName]; ok {
		return m, true
	}
	// Handle Go's #01 suffix when duplicate subtest names exist.
	if hashIdx := strings.LastIndexByte(subName, '#'); hashIdx > 0 {
		if m, ok := subtests[subName[:hashIdx]]; ok {
			return m, true
		}
	}
	return SubtestMeta{}, false
}

func spanHasChangedLine(startLine int, endLine int, changed map[int]struct{}) bool {
	for line := startLine; line <= endLine; line++ {
		if _, ok := changed[line]; ok {
			return true
		}
	}
	return false
}

func anyLineInSet(lines []int, set map[int]struct{}) bool {
	for _, line := range lines {
		if _, ok := set[line]; ok {
			return true
		}
	}
	return false
}
