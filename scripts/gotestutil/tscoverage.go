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
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// IstanbulPosition represents a 1-based line and 0-based nullable column in an Istanbul location.
type IstanbulPosition struct {
	// Line is the 1-based source line number.
	Line int `json:"line"`
	// Column is the 0-based source column, or nil when Istanbul remaps end-of-line to Infinity.
	Column *int `json:"column"`
}

// IstanbulRange represents a start/end source interval in Istanbul coverage metadata.
type IstanbulRange struct {
	// Start holds the inclusive starting position of the range.
	Start IstanbulPosition `json:"start"`
	// End holds the ending position of the range.
	End IstanbulPosition `json:"end"`
}

// IstanbulSpan holds a normalized source interval and its execution hit count.
type IstanbulSpan struct {
	// StartLine is the 1-based starting line number.
	StartLine int
	// StartCol is the 0-based starting column number.
	StartCol int
	// EndLine is the 1-based ending line number.
	EndLine int
	// EndCol is the 0-based ending column number, or -1 for end-of-line.
	EndCol int
	// Count is the total execution hit count recorded across runs.
	Count int
	// Skip reports whether Istanbul marked this span as ignored.
	Skip bool
}

// IstanbulFnSpan holds normalized declaration and body intervals for a function or method.
type IstanbulFnSpan struct {
	// Decl is the normalized span of the function header or signature.
	Decl IstanbulSpan
	// Body is the normalized span of the function body.
	Body IstanbulSpan
	// Count is the total invocation count of the function.
	Count int
	// Skip reports whether Istanbul marked this function as ignored.
	Skip bool
}

// IstanbulBranchSpan holds a normalized interval for a single branch arm.
type IstanbulBranchSpan struct {
	// BranchType identifies the Istanbul branch construct (e.g., "if", "cond-expr", "binary-expr").
	BranchType string
	// Arm is the normalized source span of this branch arm.
	Arm IstanbulSpan
	// Count is the total execution count of this branch arm.
	Count int
	// Skip reports whether Istanbul marked this branch arm as ignored.
	Skip bool
}

// IstanbulFileCoverage holds merged statement, function, and branch coverage for a TypeScript file.
type IstanbulFileCoverage struct {
	// File is the normalized repository-relative slash path of the TypeScript source file.
	File string
	// Statements contains sorted statement spans for the file.
	Statements []IstanbulSpan
	// Functions contains sorted function spans for the file.
	Functions []IstanbulFnSpan
	// Branches contains sorted branch arm spans for the file.
	Branches []IstanbulBranchSpan
}

type rawIstanbulRange struct {
	Start IstanbulPosition `json:"start"`
	End   IstanbulPosition `json:"end"`
	Skip  bool             `json:"skip"`
}

type rawIstanbulFnMeta struct {
	Name string           `json:"name"`
	Decl rawIstanbulRange `json:"decl"`
	Loc  rawIstanbulRange `json:"loc"`
	Skip bool             `json:"skip"`
}

type rawIstanbulBranchMeta struct {
	Type      string             `json:"type"`
	Loc       rawIstanbulRange   `json:"loc"`
	Locations []rawIstanbulRange `json:"locations"`
	Skip      bool               `json:"skip"`
}

type rawIstanbulFileCoverage struct {
	Path         string                           `json:"path"`
	StatementMap map[string]rawIstanbulRange      `json:"statementMap"`
	FnMap        map[string]rawIstanbulFnMeta     `json:"fnMap"`
	BranchMap    map[string]rawIstanbulBranchMeta `json:"branchMap"`
	S            map[string]int                   `json:"s"`
	F            map[string]int                   `json:"f"`
	B            map[string][]int                 `json:"b"`
}

type tsSpanKey struct {
	startLine int
	startCol  int
	endLine   int
	endCol    int
	skip      bool
}

type tsFnKey struct {
	decl tsSpanKey
	body tsSpanKey
	skip bool
}

type tsBranchKey struct {
	branchType string
	arm        tsSpanKey
	skip       bool
}

type tsFileAccumulator struct {
	statements map[tsSpanKey]int
	functions  map[tsFnKey]int
	branches   map[tsBranchKey]int
}

// LoadTSCoverageIfPresent opens and parses an Istanbul coverage-final.json file, returning an empty map when absent.
func LoadTSCoverageIfPresent(coveragePath string, repoRoot string) (map[string]IstanbulFileCoverage, error) {
	f, err := os.Open(coveragePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(map[string]IstanbulFileCoverage), nil
		}
		return nil, fmt.Errorf("failed to open TypeScript coverage file %s: %w", coveragePath, err)
	}
	defer f.Close()

	return ParseIstanbulCoverageJSON(f, repoRoot)
}

// ParseIstanbulCoverageJSON parses an Istanbul coverage-final.json stream into repository-relative production TypeScript coverage entries.
func ParseIstanbulCoverageJSON(r io.Reader, repoRoot string) (map[string]IstanbulFileCoverage, error) {
	var rawMap map[string]rawIstanbulFileCoverage
	dec := json.NewDecoder(r)
	if err := dec.Decode(&rawMap); err != nil {
		return nil, fmt.Errorf("failed to decode Istanbul coverage JSON: %w", err)
	}

	accByFile := make(map[string]*tsFileAccumulator)
	for keyPath, entry := range rawMap {
		candidatePath := entry.Path
		if candidatePath == "" {
			candidatePath = keyPath
		}
		relFile := normalizeTSCoveragePath(candidatePath, repoRoot)
		if !IsTargetProductionTSFile(relFile) {
			continue
		}

		acc, ok := accByFile[relFile]
		if !ok {
			acc = &tsFileAccumulator{
				statements: make(map[tsSpanKey]int),
				functions:  make(map[tsFnKey]int),
				branches:   make(map[tsBranchKey]int),
			}
			accByFile[relFile] = acc
		}

		for id, loc := range entry.StatementMap {
			sp, valid := toSpanKey(loc)
			if !valid {
				continue
			}
			acc.statements[sp] += entry.S[id]
		}

		for id, fn := range entry.FnMap {
			declKey, declValid := toSpanKey(fn.Decl)
			bodyKey, bodyValid := toSpanKey(fn.Loc)
			if !declValid && !bodyValid {
				continue
			}
			if !declValid {
				declKey = bodyKey
			}
			if !bodyValid {
				bodyKey = declKey
			}
			fk := tsFnKey{
				decl: declKey,
				body: bodyKey,
				skip: fn.Skip || fn.Decl.Skip || fn.Loc.Skip,
			}
			acc.functions[fk] += entry.F[id]
		}

		for id, br := range entry.BranchMap {
			counts := entry.B[id]
			for idx, armLoc := range br.Locations {
				armKey, valid := toSpanKey(armLoc)
				if !valid {
					continue
				}
				hit := 0
				if idx < len(counts) {
					hit = counts[idx]
				}
				bk := tsBranchKey{
					branchType: br.Type,
					arm:        armKey,
					skip:       br.Skip || armLoc.Skip,
				}
				acc.branches[bk] += hit
			}
		}
	}

	out := make(map[string]IstanbulFileCoverage, len(accByFile))
	for relFile, acc := range accByFile {
		fc := IstanbulFileCoverage{
			File: relFile,
		}
		for k, count := range acc.statements {
			fc.Statements = append(fc.Statements, spanFromKey(k, count))
		}
		slices.SortFunc(fc.Statements, CompareIstanbulSpan)

		for k, count := range acc.functions {
			fc.Functions = append(fc.Functions, IstanbulFnSpan{
				Decl:  spanFromKey(k.decl, count),
				Body:  spanFromKey(k.body, count),
				Count: count,
				Skip:  k.skip,
			})
		}
		slices.SortFunc(fc.Functions, func(a, b IstanbulFnSpan) int {
			return cmp.Or(
				CompareIstanbulSpan(a.Decl, b.Decl),
				CompareIstanbulSpan(a.Body, b.Body),
			)
		})

		for k, count := range acc.branches {
			fc.Branches = append(fc.Branches, IstanbulBranchSpan{
				BranchType: k.branchType,
				Arm:        spanFromKey(k.arm, count),
				Count:      count,
				Skip:       k.skip,
			})
		}
		slices.SortFunc(fc.Branches, func(a, b IstanbulBranchSpan) int {
			return cmp.Or(
				CompareIstanbulSpan(a.Arm, b.Arm),
				cmp.Compare(a.BranchType, b.BranchType),
			)
		})

		out[relFile] = fc
	}

	return out, nil
}

// CompareIstanbulSpan orders two IstanbulSpan values by start line, start column, end line, and end column.
func CompareIstanbulSpan(a, b IstanbulSpan) int {
	return cmp.Or(
		cmp.Compare(a.StartLine, b.StartLine),
		cmp.Compare(a.StartCol, b.StartCol),
		cmp.Compare(a.EndLine, b.EndLine),
		cmp.Compare(a.EndCol, b.EndCol),
	)
}

// CoveredTSLineSet returns the set of 1-based lines in fc whose narrowest covering statement or non-if branch arm was executed.
func CoveredTSLineSet(fc IstanbulFileCoverage) map[int]struct{} {
	maxLine := 0
	var spans []IstanbulSpan

	for _, st := range fc.Statements {
		if st.Skip {
			continue
		}
		spans = append(spans, st)
		if st.EndLine > maxLine {
			maxLine = st.EndLine
		}
	}
	for _, fn := range fc.Functions {
		if fn.Skip || fn.Body.Skip {
			continue
		}
		spans = append(spans, fn.Body)
		if fn.Body.EndLine > maxLine {
			maxLine = fn.Body.EndLine
		}
	}
	for _, br := range fc.Branches {
		if br.Skip || br.Arm.Skip || br.BranchType == "if" || br.BranchType == "binary-expr" || br.BranchType == "default-arg" {
			continue
		}
		spans = append(spans, br.Arm)
		if br.Arm.EndLine > maxLine {
			maxLine = br.Arm.EndLine
		}
	}

	covered := make(map[int]struct{})
	for line := 1; line <= maxLine; line++ {
		var best *IstanbulSpan
		hasSingleLineUncovered := false
		for i := range spans {
			sp := &spans[i]
			if line < sp.StartLine || line > sp.EndLine {
				continue
			}
			if sp.Count == 0 && sp.StartLine == line && sp.EndLine == line {
				hasSingleLineUncovered = true
			}
			if best == nil || isNarrowerIstanbulSpan(*sp, *best) {
				best = sp
			}
		}
		if best != nil && best.Count > 0 && !hasSingleLineUncovered {
			covered[line] = struct{}{}
		}
	}
	return covered
}

func isNarrowerIstanbulSpan(a, b IstanbulSpan) bool {
	lineSpanA := a.EndLine - a.StartLine
	lineSpanB := b.EndLine - b.StartLine
	if lineSpanA != lineSpanB {
		return lineSpanA < lineSpanB
	}
	if lineSpanA == 0 && a.EndCol >= 0 && b.EndCol >= 0 {
		colSpanA := a.EndCol - a.StartCol
		colSpanB := b.EndCol - b.StartCol
		if colSpanA != colSpanB {
			return colSpanA < colSpanB
		}
	}
	return a.Count == 0 && b.Count > 0
}

func normalizeTSCoveragePath(rawPath string, repoRoot string) string {
	slashPath := filepath.ToSlash(strings.TrimSpace(rawPath))
	if slashPath == "" || strings.Contains(slashPath, "/node_modules/") {
		return ""
	}
	if repoRoot != "" {
		rootPrefix := strings.TrimSuffix(filepath.ToSlash(repoRoot), "/") + "/"
		if strings.HasPrefix(slashPath, rootPrefix) {
			return strings.TrimPrefix(slashPath, rootPrefix)
		}
	}
	slashPath = strings.TrimPrefix(slashPath, "./")
	if strings.HasPrefix(slashPath, "web/src/") {
		return slashPath
	}
	if strings.HasPrefix(slashPath, "src/") {
		return "web/" + slashPath
	}
	if idx := strings.Index(slashPath, "/web/src/"); idx >= 0 {
		return slashPath[idx+1:]
	}
	return slashPath
}

func toSpanKey(r rawIstanbulRange) (tsSpanKey, bool) {
	if r.Start.Line <= 0 || r.End.Line < r.Start.Line {
		return tsSpanKey{}, false
	}
	sCol := 0
	if r.Start.Column != nil {
		sCol = *r.Start.Column
	}
	eCol := -1
	if r.End.Column != nil {
		eCol = *r.End.Column
	}
	return tsSpanKey{
		startLine: r.Start.Line,
		startCol:  sCol,
		endLine:   r.End.Line,
		endCol:    eCol,
		skip:      r.Skip,
	}, true
}

func spanFromKey(k tsSpanKey, count int) IstanbulSpan {
	return IstanbulSpan{
		StartLine: k.startLine,
		StartCol:  k.startCol,
		EndLine:   k.endLine,
		EndCol:    k.endCol,
		Count:     count,
		Skip:      k.skip,
	}
}
