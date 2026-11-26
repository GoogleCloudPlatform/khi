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
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// AnalyzeInput bundles per-subtest polarity traces and per-mutant failure sets for one Go package.
type AnalyzeInput struct {
	// ChangedFuncs lists Test* functions modified in the diff.
	ChangedFuncs []ChangedTestFunc
	// LeafOrder lists executed leaf subtest names in execution order.
	LeafOrder []string
	// PolarityByLeaf maps leaf test names to executed polarity token sets.
	PolarityByLeaf map[string]map[string]struct{}
	// MutantRuns maps mutant IDs to verbose test execution results.
	MutantRuns map[int]VerboseTestResult
	// BudgetSkippedMutants holds mutant IDs skipped due to budget or timeout limits.
	BudgetSkippedMutants map[int]struct{}
}

// FuncRedundancy groups subsumed test/subtest details under their enclosing top-level Test* function.
type FuncRedundancy struct {
	// File is the slash-separated relative path of the test file.
	File string
	// FuncName is the enclosing top-level Test* function name.
	FuncName string
	// Line is the 1-based starting line of the test function.
	Line int
	// EndLine is the 1-based ending line of the test function.
	EndLine int
	// Details holds redundant subtest details within this test function.
	Details []rule.RedundantDetail
}

type leafDescriptor struct {
	topFunc     string
	name        string
	file        string
	line        int
	isCandidate bool
	onlyErrNil  bool
	polarity    map[string]struct{}
	ranMutants  map[int]struct{}
	kills       map[int]struct{}
}

// Analyze detects redundant candidate tests or subtests and groups them by enclosing Test* function.
func Analyze(in AnalyzeInput) []FuncRedundancy {
	if len(in.ChangedFuncs) == 0 || len(in.LeafOrder) == 0 {
		return nil
	}
	funcByName := indexFuncsByName(in.ChangedFuncs)
	leaves := buildLeafDescriptors(in, funcByName)

	byFunc := make(map[string]*FuncRedundancy)
	var funcOrder []string
	for i := range leaves {
		detail, ok := evaluateLeafRedundancy(&leaves[i], leaves, in.BudgetSkippedMutants)
		if !ok {
			continue
		}
		fnMeta := funcByName[leaves[i].topFunc]
		grp, exists := byFunc[fnMeta.FuncName]
		if !exists {
			grp = &FuncRedundancy{
				File:     fnMeta.File,
				FuncName: fnMeta.FuncName,
				Line:     fnMeta.Line,
				EndLine:  fnMeta.EndLine,
			}
			byFunc[fnMeta.FuncName] = grp
			funcOrder = append(funcOrder, fnMeta.FuncName)
		}
		grp.Details = append(grp.Details, detail)
	}

	out := make([]FuncRedundancy, 0, len(funcOrder))
	for _, name := range funcOrder {
		grp := byFunc[name]
		sortDetails(grp.Details)
		out = append(out, *grp)
	}
	slices.SortFunc(out, func(a, b FuncRedundancy) int {
		return cmp.Or(cmp.Compare(a.File, b.File), cmp.Compare(a.Line, b.Line))
	})
	return out
}

// BuildFindings converts grouped FuncRedundancy items into unified test_redundancy rule Findings.
func BuildFindings(groups []FuncRedundancy, limit int) []rule.Finding {
	var findings []rule.Finding
	for _, grp := range groups {
		after := len(grp.Details)
		if after <= limit {
			continue
		}
		findings = append(findings, rule.Finding{
			Rule:           rule.RuleTestRedundancy,
			Kind:           rule.ViolationAdvisory,
			File:           grp.File,
			Line:           grp.Details[0].Line,
			EndLine:        grp.EndLine,
			Subject:        grp.FuncName,
			After:          after,
			Limit:          limit,
			Detail:         fmt.Sprintf("%d redundant subtest(s) subsumed by stronger tests", after),
			RedundantTests: grp.Details,
		})
	}
	return findings
}

func indexFuncsByName(funcs []ChangedTestFunc) map[string]ChangedTestFunc {
	out := make(map[string]ChangedTestFunc, len(funcs))
	for _, fn := range funcs {
		out[fn.FuncName] = fn
	}
	return out
}

func buildLeafDescriptors(in AnalyzeInput, funcByName map[string]ChangedTestFunc) []leafDescriptor {
	var leaves []leafDescriptor
	for _, leaf := range in.LeafOrder {
		topFn, _ := SplitTopAndSubtest(leaf)
		fnMeta, tracked := funcByName[topFn]
		pol := in.PolarityByLeaf[leaf]
		if !tracked || len(pol) == 0 {
			continue
		}
		desc := newLeafDescriptor(leaf, fnMeta, pol)
		populateMutantKills(&desc, in.MutantRuns)
		leaves = append(leaves, desc)
	}
	return leaves
}

func newLeafDescriptor(
	leaf string,
	fnMeta ChangedTestFunc,
	pol map[string]struct{},
) leafDescriptor {
	topFn, subName := SplitTopAndSubtest(leaf)
	desc := leafDescriptor{
		topFunc:     topFn,
		name:        leaf,
		file:        fnMeta.File,
		line:        fnMeta.Line,
		isCandidate: true,
		onlyErrNil:  fnMeta.OnlyErrorNil,
		polarity:    pol,
		ranMutants:  make(map[int]struct{}),
		kills:       make(map[int]struct{}),
	}
	if subName != "" {
		subMeta, found := lookupSubtestMeta(fnMeta.Subtests, subName)
		if found {
			desc.line = subMeta.Line
			desc.onlyErrNil = subMeta.OnlyErrorNil
		}
		desc.isCandidate = fnMeta.EntireFuncChanged || subMeta.InDiff
	}
	return desc
}

func populateMutantKills(desc *leafDescriptor, mutantRuns map[int]VerboseTestResult) {
	for mutID, runRes := range mutantRuns {
		if runRes.Aborted {
			continue
		}
		if _, ran := runRes.RanLeafSet[desc.name]; ran {
			desc.ranMutants[mutID] = struct{}{}
		}
		if _, failed := runRes.FailedLeafs[desc.name]; failed {
			desc.kills[mutID] = struct{}{}
		}
	}
}

func evaluateLeafRedundancy(
	cand *leafDescriptor,
	all []leafDescriptor,
	skipped map[int]struct{},
) (rule.RedundantDetail, bool) {
	if !cand.isCandidate || len(cand.ranMutants) == 0 || touchedSkippedMutant(cand.polarity, skipped) {
		return rule.RedundantDetail{}, false
	}
	if len(cand.kills) > 0 {
		dom := findDominatingDuplicate(cand, all)
		if dom == nil {
			return rule.RedundantDetail{}, false
		}
		return toRedundantDetail(rule.RedundantSamePath, cand, dom), true
	}
	if cand.onlyErrNil {
		if dom := findDominatingCoveringTest(cand, all); dom != nil {
			return toRedundantDetail(rule.RedundantNoOutputCheck, cand, dom), true
		}
	}
	return rule.RedundantDetail{}, false
}

func toRedundantDetail(kind rule.RedundantKind, cand *leafDescriptor, dom *leafDescriptor) rule.RedundantDetail {
	return rule.RedundantDetail{
		Kind:            kind,
		Line:            cand.line,
		TestName:        cand.name,
		DominatedBy:     dom.name,
		DominatedByFile: dom.file,
		DominatedByLine: dom.line,
		CandidateKills:  len(cand.kills),
		DominatorKills:  len(dom.kills),
	}
}

func findDominatingDuplicate(cand *leafDescriptor, all []leafDescriptor) *leafDescriptor {
	var best *leafDescriptor
	for j := range all {
		dom := &all[j]
		if dom.topFunc != cand.topFunc {
			continue
		}
		if len(dom.kills) <= len(cand.kills) || !maps.Equal(cand.polarity, dom.polarity) || !killsSubsumedOnSharedRuns(cand, dom) {
			continue
		}
		if best == nil || beatsAsDominator(dom, best) {
			best = dom
		}
	}
	return best
}

func findDominatingCoveringTest(cand *leafDescriptor, all []leafDescriptor) *leafDescriptor {
	var best *leafDescriptor
	for j := range all {
		dom := &all[j]
		if len(dom.kills) > 0 && isStringSubset(cand.polarity, dom.polarity) {
			if best == nil || beatsAsDominator(dom, best) {
				best = dom
			}
		}
	}
	return best
}

func killsSubsumedOnSharedRuns(cand *leafDescriptor, dom *leafDescriptor) bool {
	for mutID := range cand.kills {
		if _, killed := dom.kills[mutID]; !killed {
			return false
		}
	}
	return true
}

func beatsAsDominator(dom *leafDescriptor, best *leafDescriptor) bool {
	if len(dom.kills) != len(best.kills) {
		return len(dom.kills) > len(best.kills)
	}
	if dom.isCandidate != best.isCandidate {
		return !dom.isCandidate
	}
	return cmp.Or(
		cmp.Compare(dom.file, best.file),
		cmp.Compare(dom.line, best.line),
		cmp.Compare(dom.name, best.name),
	) < 0
}

func isStringSubset(sub map[string]struct{}, sup map[string]struct{}) bool {
	for k := range sub {
		if _, ok := sup[k]; !ok {
			return false
		}
	}
	return true
}

func touchedSkippedMutant(polarity map[string]struct{}, skipped map[int]struct{}) bool {
	for tok := range polarity {
		idStr, _, _ := strings.Cut(tok, ":")
		if id, err := strconv.Atoi(idStr); err == nil {
			if _, ok := skipped[id]; ok {
				return true
			}
		}
	}
	return false
}

func lookupSubtestMeta(subtests map[string]SubtestMeta, subName string) (SubtestMeta, bool) {
	if m, ok := subtests[subName]; ok {
		return m, true
	}
	if hashIdx := strings.LastIndexByte(subName, '#'); hashIdx > 0 {
		if m, ok := subtests[subName[:hashIdx]]; ok {
			return m, true
		}
	}
	return SubtestMeta{}, false
}

func sortDetails(details []rule.RedundantDetail) {
	slices.SortFunc(details, func(a, b rule.RedundantDetail) int {
		return cmp.Or(cmp.Compare(a.Line, b.Line), cmp.Compare(a.TestName, b.TestName))
	})
}
