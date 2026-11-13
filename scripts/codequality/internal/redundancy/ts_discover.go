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
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/astcheck"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// RawTSSpecCase represents a Jasmine spec declaration extracted by mutate_ts.js from a *.spec.ts file.
type RawTSSpecCase struct {
	// FullName is the full Jasmine spec title path.
	FullName string `json:"fullName"`
	// Line is the 1-based line number of the spec declaration.
	Line int `json:"line"`
	// StartLine is the 1-based starting line of the spec block.
	StartLine int `json:"startLine"`
	// EndLine is the 1-based ending line of the spec block.
	EndLine int `json:"endLine"`
	// ItStartLine is the 1-based starting line of the it(...) call.
	ItStartLine int `json:"itStartLine"`
	// ItEndLine is the 1-based ending line of the it(...) call.
	ItEndLine int `json:"itEndLine"`
	// SuiteHeaderLines lists 1-based header lines of enclosing describe(...) suites.
	SuiteHeaderLines []int `json:"suiteHeaderLines"`
	// WeakAssertion reports whether the spec only uses weak assertions.
	WeakAssertion bool `json:"weakAssertion"`
}

// ChangedTSSpec holds source location and diff membership for a Jasmine spec in a modified *.spec.ts file.
type ChangedTSSpec struct {
	// File is the slash-separated relative path of the spec file.
	File string
	// FullName is the full Jasmine spec title path.
	FullName string
	// Line is the 1-based line number of the spec declaration.
	Line int
	// InDiff reports whether the spec overlaps the modified line set.
	InDiff bool
	// WeakAssertion reports whether the spec only uses weak assertions.
	WeakAssertion bool
}

// TSSpecOrderEntry identifies a single executed Jasmine spec in Pass 0 execution order.
type TSSpecOrderEntry struct {
	// ID is the Jasmine spec identifier.
	ID string `json:"id"`
	// FullName is the full Jasmine spec title path.
	FullName string `json:"fullName"`
}

// TSMutantSpecRun records which Jasmine spec IDs ran and failed during a single mutant evaluation pass.
type TSMutantSpecRun struct {
	// RanSpecIDs lists Jasmine spec IDs executed during the mutant pass.
	RanSpecIDs []string `json:"ranSpecIds"`
	// FailedSpecIDs lists Jasmine spec IDs that failed during the mutant pass.
	FailedSpecIDs []string `json:"failedSpecIds"`
}

// TSAnalyzeInput bundles Jasmine spec execution order, polarity traces, and per-mutant failure sets.
type TSAnalyzeInput struct {
	// ChangedSpecs lists Jasmine specs extracted from modified spec files.
	ChangedSpecs []ChangedTSSpec
	// SpecOrder lists executed Jasmine specs in Pass 0 order.
	SpecOrder []TSSpecOrderEntry
	// SpecPolarities maps Jasmine spec IDs to observed polarity tokens.
	SpecPolarities map[string][]string
	// MutantRuns maps mutant IDs to spec execution outcomes.
	MutantRuns map[string]TSMutantSpecRun
	// MutantFileByID maps mutant and probe IDs to production TypeScript file paths.
	MutantFileByID map[int]string
	// BudgetSkippedMutants holds mutant IDs skipped due to budget or timeout limits.
	BudgetSkippedMutants map[int]struct{}
}

// DiscoverChangedTSSpecs maps AST-extracted Jasmine spec declarations against VCS changed lines for filePath.
func DiscoverChangedTSSpecs(filePath string, specCases []RawTSSpecCase, changedLines map[int]struct{}) []ChangedTSSpec {
	out := make([]ChangedTSSpec, 0, len(specCases))
	hasAnyInDiff := false
	for _, sc := range specCases {
		if sc.FullName == "" {
			continue
		}
		inDiff := isTSSpecInDiff(sc, changedLines)
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

// AnalyzeTS detects subsumed Jasmine specs per modified *.spec.ts file.
func AnalyzeTS(in TSAnalyzeInput) []FuncRedundancy {
	specsByName := groupTSSpecsByName(in.ChangedSpecs)
	ranBySpecID, killsBySpecID := indexTSMutantRuns(in.MutantRuns)
	leavesByFile := buildTSLeavesByFile(in, specsByName, ranBySpecID, killsBySpecID)

	specFiles := make([]string, 0, len(leavesByFile))
	for f := range leavesByFile {
		specFiles = append(specFiles, f)
	}
	slices.Sort(specFiles)

	var out []FuncRedundancy
	for _, specFile := range specFiles {
		details := evaluateTSFileLeaves(specFile, leavesByFile[specFile], in)
		if len(details) > 0 {
			out = append(out, FuncRedundancy{
				File:     specFile,
				FuncName: "<spec>",
				Line:     details[0].Line,
				EndLine:  details[len(details)-1].Line,
				Details:  details,
			})
		}
	}
	return out
}

func isTSSpecInDiff(sc RawTSSpecCase, changedLines map[int]struct{}) bool {
	return astcheck.OverlapsLines(sc.StartLine, sc.EndLine, changedLines) ||
		astcheck.OverlapsLines(sc.ItStartLine, sc.ItEndLine, changedLines) ||
		anyLineInSet(sc.SuiteHeaderLines, changedLines)
}

func anyLineInSet(lines []int, set map[int]struct{}) bool {
	for _, line := range lines {
		if _, ok := set[line]; ok {
			return true
		}
	}
	return false
}

func groupTSSpecsByName(specs []ChangedTSSpec) map[string][]ChangedTSSpec {
	byName := make(map[string][]ChangedTSSpec)
	for _, sp := range specs {
		byName[sp.FullName] = append(byName[sp.FullName], sp)
	}
	return byName
}

func indexTSMutantRuns(mutantRuns map[string]TSMutantSpecRun) (map[string]map[int]struct{}, map[string]map[int]struct{}) {
	ranBySpecID := make(map[string]map[int]struct{})
	killsBySpecID := make(map[string]map[int]struct{})
	for mutIDStr, runRes := range mutantRuns {
		mutID, err := strconv.Atoi(mutIDStr)
		if err != nil || mutID <= 0 {
			continue
		}
		recordSpecIDHit(ranBySpecID, runRes.RanSpecIDs, mutID)
		recordSpecIDHit(killsBySpecID, runRes.FailedSpecIDs, mutID)
	}
	return ranBySpecID, killsBySpecID
}

func recordSpecIDHit(dst map[string]map[int]struct{}, specIDs []string, mutID int) {
	for _, specID := range specIDs {
		m := dst[specID]
		if m == nil {
			m = make(map[int]struct{})
			dst[specID] = m
		}
		m[mutID] = struct{}{}
	}
}

func buildTSLeavesByFile(
	in TSAnalyzeInput,
	specsByName map[string][]ChangedTSSpec,
	ranBySpecID map[string]map[int]struct{},
	killsBySpecID map[string]map[int]struct{},
) map[string][]leafDescriptor {
	cursorByName := make(map[string]int, len(specsByName))
	leavesByFile := make(map[string][]leafDescriptor)
	for idx, entry := range in.SpecOrder {
		meta, ok := matchNextOrderedTSSpec(entry, specsByName, cursorByName)
		if !ok {
			continue
		}
		pol := filterTSPolarityTokensForSpec(meta.File, in.SpecPolarities[entry.ID], in.MutantFileByID)
		if len(pol) == 0 {
			continue
		}
		leavesByFile[meta.File] = append(leavesByFile[meta.File], leafDescriptor{
			name:        entry.FullName,
			file:        meta.File,
			line:        meta.Line,
			isCandidate: meta.InDiff,
			onlyErrNil:  meta.WeakAssertion,
			orderIdx:    idx,
			polarity:    pol,
			ranMutants:  filterMutantIDsForSpec(meta.File, ranBySpecID[entry.ID], in.MutantFileByID),
			kills:       filterMutantIDsForSpec(meta.File, killsBySpecID[entry.ID], in.MutantFileByID),
		})
	}
	return leavesByFile
}

func matchNextOrderedTSSpec(
	entry TSSpecOrderEntry,
	specsByName map[string][]ChangedTSSpec,
	cursorByName map[string]int,
) (ChangedTSSpec, bool) {
	if entry.ID == "" {
		return ChangedTSSpec{}, false
	}
	list := specsByName[entry.FullName]
	cur := cursorByName[entry.FullName]
	if cur >= len(list) {
		return ChangedTSSpec{}, false
	}
	cursorByName[entry.FullName] = cur + 1
	return list[cur], true
}

func evaluateTSFileLeaves(specFile string, leaves []leafDescriptor, in TSAnalyzeInput) []rule.RedundantDetail {
	skippedSubject := filterMutantIDsForSpec(specFile, in.BudgetSkippedMutants, in.MutantFileByID)
	var details []rule.RedundantDetail
	for i := range leaves {
		if d, ok := evaluateLeafRedundancy(&leaves[i], leaves, skippedSubject); ok {
			details = append(details, d)
		}
	}
	sortDetails(details)
	return details
}

func filterMutantIDsForSpec(specFile string, ids map[int]struct{}, mutantFileByID map[int]string) map[int]struct{} {
	out := make(map[int]struct{}, len(ids))
	for id := range ids {
		if isSubjectMutantID(specFile, id, mutantFileByID) {
			out[id] = struct{}{}
		}
	}
	return out
}

func filterTSPolarityTokensForSpec(specFile string, rawTokens []string, mutantFileByID map[int]string) map[string]struct{} {
	pol := make(map[string]struct{}, len(rawTokens))
	for _, tok := range rawTokens {
		if isSubjectPolarityToken(specFile, tok, mutantFileByID) {
			pol[tok] = struct{}{}
		}
	}
	return pol
}

func isSubjectPolarityToken(specFile string, tok string, mutantFileByID map[int]string) bool {
	idPart, _, _ := strings.Cut(tok, ":")
	if after, isProbe := strings.CutPrefix(idPart, "P"); isProbe {
		pID, err := strconv.Atoi(after)
		return err == nil && isSubjectMutantID(specFile, -pID, mutantFileByID)
	}
	mID, err := strconv.Atoi(idPart)
	return err == nil && isSubjectMutantID(specFile, mID, mutantFileByID)
}

func isSubjectMutantID(specFile string, id int, mutantFileByID map[int]string) bool {
	if len(mutantFileByID) == 0 {
		return true
	}
	prodFile, ok := mutantFileByID[id]
	return ok && IsSubjectMutantForTSSpec(specFile, prodFile)
}
