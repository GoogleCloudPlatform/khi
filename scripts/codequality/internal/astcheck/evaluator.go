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
	"cmp"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

// TargetReader loads raw Go source bytes from the current working tree.
type TargetReader func(relPath string) ([]byte, error)

// Options configures Stage-1 AST structural evaluation across changed Go files.
type Options struct {
	Thresholds             rule.Thresholds
	ChangedProdLinesByFile map[string]map[int]struct{}
	ChangedTestLinesByFile map[string]map[int]struct{}
	DeletedGoFiles         []string
	CloneCorpusFiles       []string
	ReadTargetFile         TargetReader
	ReadBaseFile           BaseReader
}

// Result holds Stage-1 AST findings alongside parsed target/base metadata shared with Stage-2 evaluators.
type Result struct {
	Findings        []rule.Finding
	Suppressed      []rule.SuppressedItem
	DirectiveErrors []rule.DirectiveError
	Files           map[string]*FileMetrics
	TouchedFuncs    map[string][]FuncMetrics
	EvaluatedLines  map[string]map[int]struct{}
	FuncsEvaluated  int
	Base            *BaseSnapshot
}

type evalSession struct {
	opts         Options
	base         *BaseSnapshot
	targetClones *CloneIndex
	res          *Result
}

// Evaluate runs all Stage-1 AST quality rules on changed Go production and test files.
func Evaluate(opts Options) (*Result, error) {
	prodFiles := sortedNonEmptyKeys(opts.ChangedProdLinesByFile)
	testFiles := sortedNonEmptyKeys(opts.ChangedTestLinesByFile)

	baseSnap, err := LoadBaseSnapshot(opts.ReadBaseFile, opts.Thresholds, prodFiles, testFiles, opts.DeletedGoFiles)
	if err != nil {
		return nil, err
	}

	sess := &evalSession{
		opts:         opts,
		base:         baseSnap,
		targetClones: buildTargetCloneIndex(opts, prodFiles),
		res: &Result{
			Files:          make(map[string]*FileMetrics),
			TouchedFuncs:   make(map[string][]FuncMetrics),
			EvaluatedLines: make(map[string]map[int]struct{}),
			Base:           baseSnap,
		},
	}

	for _, relPath := range prodFiles {
		if err := sess.evaluateProdFile(relPath); err != nil {
			return nil, err
		}
	}
	for _, relPath := range testFiles {
		if err := sess.evaluateTestFile(relPath); err != nil {
			return nil, err
		}
	}
	sortFindings(sess.res.Findings)
	return sess.res, nil
}

func buildTargetCloneIndex(opts Options, changedProd []string) *CloneIndex {
	allFiles := slices.Concat(opts.CloneCorpusFiles, changedProd)
	slices.Sort(allFiles)
	allFiles = slices.Compact(allFiles)

	var blocks []FuncCloneBlock
	for _, f := range allFiles {
		src, err := opts.ReadTargetFile(f)
		if err != nil {
			continue
		}
		blocks = append(blocks, ExtractCloneBlocks(f, src)...)
	}
	minWindow := opts.Thresholds.Limit(rule.RuleCodeClone) + 1
	return NewCloneIndex(minWindow, blocks)
}

func (s *evalSession) evaluateProdFile(relPath string) error {
	src, fm, err := s.loadTargetMetrics(relPath)
	if err != nil || fm == nil {
		return err
	}
	changed := s.opts.ChangedProdLinesByFile[relPath]
	s.markEvaluatedLines(relPath, fm.PkgLine, changed)
	s.checkCatchAllFilename(relPath)
	s.checkFileSize(fm, rule.RuleFileCodeLines)

	cloneByFunc := indexCloneBlocksByKey(ExtractCloneBlocks(relPath, src))
	for _, fn := range fm.Funcs {
		if !OverlapsLines(fn.StartLine, fn.EndLine, changed) {
			continue
		}
		s.res.FuncsEvaluated++
		s.res.TouchedFuncs[relPath] = append(s.res.TouchedFuncs[relPath], fn)
		s.res.EvaluatedLines[relPath][fn.StartLine] = struct{}{}
		s.evaluateProdFunc(fm, fn, cloneByFunc[fn.Key])
	}
	return nil
}

func (s *evalSession) evaluateTestFile(relPath string) error {
	_, fm, err := s.loadTargetMetrics(relPath)
	if err != nil || fm == nil {
		return err
	}
	changed := s.opts.ChangedTestLinesByFile[relPath]
	s.markEvaluatedLines(relPath, fm.PkgLine, changed)
	s.checkCatchAllFilename(relPath)
	s.checkFileSize(fm, rule.RuleTestFileCodeLines)

	for _, fn := range fm.Funcs {
		if fn.IsTestEntry || !OverlapsLines(fn.StartLine, fn.EndLine, changed) {
			continue
		}
		s.res.FuncsEvaluated++
		s.res.EvaluatedLines[relPath][fn.StartLine] = struct{}{}
		baseFn, hasBase := s.base.LookupFunc(relPath, fn.Key)
		s.recordCheck(fm, checkInput{
			ruleID:   rule.RuleTestHelperCognitive,
			declLine: fn.StartLine,
			endLine:  fn.EndLine,
			subject:  fn.DisplayName,
			sample:   rule.Sample{After: fn.Cognitive, Limit: s.opts.Thresholds.Limit(rule.RuleTestHelperCognitive), Before: baseFn.Cognitive, HasBefore: hasBase},
			hotspots: fn.Hotspots,
		})
	}
	for _, tr := range fm.TRuns {
		if !OverlapsLines(tr.StartLine, tr.EndLine, changed) {
			continue
		}
		s.res.FuncsEvaluated++
		s.res.EvaluatedLines[relPath][tr.StartLine] = struct{}{}
		baseTR, hasBase := s.base.LookupTRun(relPath, tr.ParentFunc, tr.SubtestLabel)
		s.recordCheck(fm, checkInput{
			ruleID:   rule.RuleTRunCognitive,
			declLine: tr.StartLine,
			endLine:  tr.EndLine,
			subject:  fmt.Sprintf("%s [t.Run(%s)]", tr.ParentFunc, tr.SubtestLabel),
			sample:   rule.Sample{After: tr.Cognitive, Limit: s.opts.Thresholds.Limit(rule.RuleTRunCognitive), Before: baseTR.Cognitive, HasBefore: hasBase},
			hotspots: tr.Hotspots,
		})
	}
	return nil
}

func (s *evalSession) loadTargetMetrics(relPath string) ([]byte, *FileMetrics, error) {
	src, err := s.opts.ReadTargetFile(relPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to read %s: %w", relPath, err)
	}
	fm, err := AnalyzeFile(relPath, src, s.opts.Thresholds.MinParamsForFlagCheck)
	if err != nil || fm == nil {
		return src, nil, err
	}
	s.res.Files[relPath] = fm
	s.res.DirectiveErrors = append(s.res.DirectiveErrors, fm.DirectiveErrors...)
	return src, fm, nil
}

func (s *evalSession) markEvaluatedLines(relPath string, pkgLine int, changed map[int]struct{}) {
	lines := make(map[int]struct{}, len(changed)+1)
	lines[pkgLine] = struct{}{}
	for ln := range changed {
		lines[ln] = struct{}{}
	}
	s.res.EvaluatedLines[relPath] = lines
}

func (s *evalSession) checkCatchAllFilename(relPath string) {
	detail, bad := CheckCatchAllFilename(relPath)
	if !bad {
		return
	}
	existed := s.base.HasFile(relPath)
	before := 0
	if existed {
		before = 1
	}
	sample := rule.Sample{
		After:     1,
		Limit:     s.opts.Thresholds.Limit(rule.RuleCatchAllFilename),
		Before:    before,
		HasBefore: existed,
	}
	kind := rule.ClassifyRatchet(sample)
	if kind == rule.ViolationNone {
		return
	}
	s.res.Findings = append(s.res.Findings, rule.Finding{
		Rule:      rule.RuleCatchAllFilename,
		Kind:      kind,
		File:      relPath,
		Line:      1,
		EndLine:   1,
		Subject:   "<filename>",
		Before:    sample.Before,
		HasBefore: sample.HasBefore,
		After:     sample.After,
		Limit:     sample.Limit,
		Detail:    detail,
	})
}

func (s *evalSession) checkFileSize(fm *FileMetrics, ruleID rule.RuleID) {
	baseFM, hasBase := s.base.FileMetrics(fm.File)
	before := 0
	if hasBase {
		before = baseFM.CodeLines
	}
	s.recordCheck(fm, checkInput{
		ruleID:   ruleID,
		declLine: fm.PkgLine,
		endLine:  fm.PkgLine,
		subject:  "<file>",
		sample: rule.Sample{
			After:     fm.CodeLines,
			Limit:     s.opts.Thresholds.Limit(ruleID),
			Before:    before,
			HasBefore: hasBase,
		},
	})
}

func (s *evalSession) evaluateProdFunc(fm *FileMetrics, fn FuncMetrics, cloneBlk FuncCloneBlock) {
	baseFn, hasBase := s.base.LookupFunc(fm.File, fn.Key)
	for _, c := range s.buildProdFuncChecks(fn, baseFn, hasBase) {
		s.recordCheck(fm, c)
	}
	s.evaluateFuncClone(fm, fn, cloneBlk)
}

func (s *evalSession) buildProdFuncChecks(fn FuncMetrics, baseFn FuncMetrics, hasBase bool) []checkInput {
	th := s.opts.Thresholds
	mk := func(id rule.RuleID, after int, before int) checkInput {
		return checkInput{
			ruleID:   id,
			declLine: fn.StartLine,
			endLine:  fn.EndLine,
			subject:  fn.DisplayName,
			sample:   rule.Sample{After: after, Limit: th.Limit(id), Before: before, HasBefore: hasBase},
		}
	}
	cog := mk(rule.RuleFuncCognitive, fn.Cognitive, baseFn.Cognitive)
	cog.hotspots = fn.Hotspots
	flagChk := mk(rule.RuleControlFlagParam, len(fn.ControlFlagParams), len(baseFn.ControlFlagParams))
	flagChk.detail = formatFlagParamDetail(fn.ControlFlagParams)
	return []checkInput{
		cog,
		mk(rule.RuleFuncNesting, fn.MaxNesting, baseFn.MaxNesting),
		mk(rule.RuleFuncCodeLines, fn.CodeLines, baseFn.CodeLines),
		mk(rule.RuleFuncStatements, fn.Statements, baseFn.Statements),
		mk(rule.RuleFuncParams, fn.Params, baseFn.Params),
		mk(rule.RuleFuncResults, fn.Results, baseFn.Results),
		flagChk,
	}
}

func (s *evalSession) evaluateFuncClone(fm *FileMetrics, fn FuncMetrics, cloneBlk FuncCloneBlock) {
	if len(cloneBlk.Stmts) == 0 {
		return
	}
	match, found := s.targetClones.FindMaxClone(cloneBlk)
	if !found {
		return
	}
	baseOverlap, hasBase := s.base.MeasureBaseCloneOverlap(fm.File, fn.Key, match.PeerFile)
	detail := fmt.Sprintf(
		"lines %d-%d duplicate %d normalized statements from %s:%d-%d (%s)",
		match.LocalStart, match.LocalEnd, match.Statements,
		match.PeerFile, match.PeerStart, match.PeerEnd, match.PeerFunc,
	)
	s.recordCheck(fm, checkInput{
		ruleID:     rule.RuleCodeClone,
		declLine:   fn.StartLine,
		reportLine: match.LocalStart,
		endLine:    match.LocalEnd,
		subject:    fn.DisplayName,
		sample: rule.Sample{
			After:     match.Statements,
			Limit:     s.opts.Thresholds.Limit(rule.RuleCodeClone),
			Before:    baseOverlap,
			HasBefore: hasBase,
		},
		detail: detail,
	})
}

type checkInput struct {
	ruleID     rule.RuleID
	declLine   int
	reportLine int
	endLine    int
	subject    string
	sample     rule.Sample
	detail     string
	hotspots   []rule.Hotspot
}

func (s *evalSession) recordCheck(fm *FileMetrics, in checkInput) {
	kind := rule.ClassifyRatchet(in.sample)
	if kind == rule.ViolationNone {
		return
	}
	line := in.reportLine
	if line == 0 {
		line = in.declLine
	}
	if kind == rule.ViolationHardLimit {
		if d := rule.MatchDeclDirective(fm.Directives, in.ruleID, in.declLine); d != nil {
			s.res.Suppressed = append(s.res.Suppressed, rule.SuppressedItem{
				Rule:     in.ruleID,
				File:     fm.File,
				Line:     line,
				Subject:  in.subject,
				Category: d.Category,
				Reason:   d.Reason,
				After:    in.sample.After,
				Limit:    in.sample.Limit,
			})
			return
		}
	}
	s.res.Findings = append(s.res.Findings, rule.Finding{
		Rule:      in.ruleID,
		Kind:      kind,
		File:      fm.File,
		Line:      line,
		EndLine:   in.endLine,
		Subject:   in.subject,
		Before:    in.sample.Before,
		HasBefore: in.sample.HasBefore,
		After:     in.sample.After,
		Limit:     in.sample.Limit,
		Detail:    in.detail,
		Hotspots:  in.hotspots,
	})
}

func formatFlagParamDetail(names []string) string {
	return "boolean control-flow parameter(s): " + strings.Join(names, ", ")
}

func indexCloneBlocksByKey(blocks []FuncCloneBlock) map[string]FuncCloneBlock {
	out := make(map[string]FuncCloneBlock, len(blocks))
	for _, blk := range blocks {
		out[blk.FuncKey] = blk
	}
	return out
}

func sortedNonEmptyKeys(m map[string]map[int]struct{}) []string {
	var keys []string
	for k, lines := range m {
		if len(lines) > 0 {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	return keys
}

func sortFindings(findings []rule.Finding) {
	slices.SortFunc(findings, func(a, b rule.Finding) int {
		return cmp.Or(
			cmp.Compare(a.File, b.File),
			cmp.Compare(a.Line, b.Line),
			cmp.Compare(string(a.Rule), string(b.Rule)),
		)
	})
}
