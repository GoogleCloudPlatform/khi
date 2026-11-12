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

package tscheck

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/redundancy"
	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

// TSMutant describes a single candidate logic change extracted from a TypeScript file.
type TSMutant struct {
	ID                  int
	File                string
	Line                int
	Operator            rule.MutantOperator
	OriginalExpr        string
	MutatedExpr         string
	OriginalLine        string
	MutatedLine         string
	SuppressionCategory rule.AllowCategory
	Status              rule.MutantStatus
}

// ToDetail converts m into a report-ready rule.MutantDetail.
func (m *TSMutant) ToDetail() rule.MutantDetail {
	return rule.MutantDetail{
		Operator:     m.Operator,
		Line:         m.Line,
		OriginalExpr: m.OriginalExpr,
		MutatedExpr:  m.MutatedExpr,
		OriginalLine: m.OriginalLine,
		MutatedLine:  m.MutatedLine,
	}
}

// RawTSCandidate represents one expression replacement variant returned by mutate_ts.js.
type RawTSCandidate struct {
	OrigExpr           string `json:"origExpr"`
	MutExpr            string `json:"mutExpr"`
	ReplacementPreview string `json:"replacementPreview"`
}

// RawTSExtracted represents one AST mutation site returned by mutate_ts.js.
type RawTSExtracted struct {
	Kind         string           `json:"kind"`
	Op           string           `json:"op"`
	AddKeepRight bool             `json:"addKeepRight"`
	BoolVal      bool             `json:"boolVal"`
	Operator     string           `json:"operator"`
	Line         int              `json:"line"`
	StartOffset  int              `json:"startOffset"`
	EndOffset    int              `json:"endOffset"`
	LeftStart    int              `json:"leftStart"`
	LeftEnd      int              `json:"leftEnd"`
	RightStart   int              `json:"rightStart"`
	RightEnd     int              `json:"rightEnd"`
	Candidates   []RawTSCandidate `json:"candidates"`
}

// RawTSFileExtraction holds JSON output per TypeScript or *.spec.ts file from mutate_ts.js.
type RawTSFileExtraction struct {
	Comments  []rule.RawComment          `json:"comments"`
	Sites     []RawTSExtracted           `json:"sites"`
	SpecCases []redundancy.RawTSSpecCase `json:"specCases"`
}

// TSMutationSite represents an active AST span in a TypeScript file to render into Super-Mutant code.
type TSMutationSite struct {
	Kind         string
	Op           string
	AddKeepRight bool
	BoolVal      bool
	StartOffset  int
	EndOffset    int
	LeftStart    int
	LeftEnd      int
	RightStart   int
	RightEnd     int
	Mutants      []*TSMutant
}

// TSFilePlan holds extracted mutants, suppressions, and directive errors for one TypeScript file.
type TSFilePlan struct {
	File            string
	Source          []byte
	Mutants         []*TSMutant
	Suppressed      []rule.SuppressedItem
	DirectiveErrors []rule.DirectiveError
}

// PlanTSFileInput bundles parameters for planning TypeScript mutations on a single file.
type PlanTSFileInput struct {
	FilePath      string
	Source        []byte
	Extraction    RawTSFileExtraction
	ChangedLines  map[int]struct{}
	CoveredLines  map[int]struct{}
	HasTSCoverage bool
	NextID        *int
	NextProbeID   *int
}

// InvokeTSMutatorScript runs mutate_ts.js via Node to extract TypeScript AST sites, comments, and Jasmine specs.
func InvokeTSMutatorScript(ctx context.Context, repoRoot string, relFiles []string) (map[string]RawTSFileExtraction, error) {
	scriptPath, err := resolveTSHelperScriptPath(repoRoot, "mutate_ts.js")
	if err != nil {
		return nil, err
	}
	args := make([]string, 0, 2+len(relFiles))
	args = append(args, scriptPath, repoRoot)
	args = append(args, relFiles...)

	outBytes, err := gotestutil.RunCommandOutput(ctx, repoRoot, "node", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to extract TypeScript AST mutation sites: %w", err)
	}
	var parsed map[string]RawTSFileExtraction
	if err := json.Unmarshal(outBytes, &parsed); err != nil {
		return nil, fmt.Errorf("failed to decode mutate_ts.js output: %w", err)
	}
	return parsed, nil
}

// PlanTSFileMutations converts raw AST sites extracted by mutate_ts.js into TSMutant entries and active Super-Mutant sites.
func PlanTSFileMutations(in PlanTSFileInput) (*TSFilePlan, []TSMutationSite) {
	directives, dirErrs := rule.ParseDirectivesFromComments(in.FilePath, in.Extraction.Comments, in.Source)
	lineStarts := computeTSLineOffsets(in.Source)
	fp := &TSFilePlan{
		File:            in.FilePath,
		Source:          in.Source,
		DirectiveErrors: dirErrs,
	}

	p := &tsSitePlanner{
		in:         in,
		directives: directives,
		lineStarts: lineStarts,
		seenSpans:  make(map[[2]int]struct{}),
		fp:         fp,
	}
	for _, raw := range in.Extraction.Sites {
		p.processRawSite(raw)
	}
	unused := rule.CollectUnusedDirectives(p.directives, rule.UnusedScope{
		File:           in.FilePath,
		IncludeDynamic: true,
		EvaluatedLines: in.ChangedLines,
	})
	fp.DirectiveErrors = append(fp.DirectiveErrors, unused...)
	return fp, p.activeSites
}

type tsSitePlanner struct {
	in          PlanTSFileInput
	directives  []rule.Directive
	lineStarts  []int
	seenSpans   map[[2]int]struct{}
	fp          *TSFilePlan
	activeSites []TSMutationSite
}

func (p *tsSitePlanner) processRawSite(raw RawTSExtracted) {
	if raw.StartOffset < 0 || raw.EndOffset > len(p.in.Source) || raw.StartOffset >= raw.EndOffset {
		return
	}
	if _, changed := p.in.ChangedLines[raw.Line]; !changed {
		p.maybeAddUnchangedProbe(raw)
		return
	}
	if !p.claimUniqueSpan(raw.StartOffset, raw.EndOffset) {
		return
	}
	op := rule.MutantOperator(raw.Operator)
	if !op.IsValid() || len(raw.Candidates) == 0 {
		return
	}
	if d := rule.MatchLineDirective(p.directives, rule.RuleFuncSurvivingMutants, raw.Line, op); d != nil {
		p.appendStaticMutants(raw, op, rule.MutantSuppressed, d.Category)
		p.fp.Suppressed = append(p.fp.Suppressed, rule.SuppressedItem{
			Rule:     rule.RuleFuncSurvivingMutants,
			File:     p.in.FilePath,
			Line:     raw.Line,
			Subject:  "<line>",
			Category: d.Category,
			Reason:   d.Reason,
			After:    len(raw.Candidates),
			Limit:    0,
		})
		return
	}
	if p.in.HasTSCoverage {
		if _, cov := p.in.CoveredLines[raw.Line]; !cov {
			p.appendStaticMutants(raw, op, rule.MutantUncovered, "")
			return
		}
	}
	p.appendActiveSite(raw, op)
}

func (p *tsSitePlanner) maybeAddUnchangedProbe(raw RawTSExtracted) {
	if p.in.NextProbeID == nil || !isTSBranchProbeKind(raw.Kind) {
		return
	}
	if !p.claimUniqueSpan(raw.StartOffset, raw.EndOffset) {
		return
	}
	probeID := *p.in.NextProbeID
	*p.in.NextProbeID++
	probeMut := &TSMutant{ID: -probeID}
	slots := []*TSMutant{probeMut}
	if raw.Kind == "ordered_cmp" || raw.Kind == "nullish" || raw.Kind == "cond" {
		slots = []*TSMutant{probeMut, probeMut}
	}
	p.activeSites = append(p.activeSites, toTSMutationSite(raw, slots))
}

func (p *tsSitePlanner) claimUniqueSpan(start int, end int) bool {
	key := [2]int{start, end}
	if _, dup := p.seenSpans[key]; dup {
		return false
	}
	p.seenSpans[key] = struct{}{}
	return true
}

func (p *tsSitePlanner) appendStaticMutants(
	raw RawTSExtracted,
	op rule.MutantOperator,
	status rule.MutantStatus,
	cat rule.AllowCategory,
) {
	for _, cand := range raw.Candidates {
		origLine, mutLine := buildTSDiffLines(p.in.Source, p.lineStarts, raw, cand.ReplacementPreview)
		p.fp.Mutants = append(p.fp.Mutants, &TSMutant{
			File:                p.in.FilePath,
			Line:                raw.Line,
			Operator:            op,
			OriginalExpr:        cand.OrigExpr,
			MutatedExpr:         cand.MutExpr,
			OriginalLine:        origLine,
			MutatedLine:         mutLine,
			SuppressionCategory: cat,
			Status:              status,
		})
	}
}

func (p *tsSitePlanner) appendActiveSite(raw RawTSExtracted, op rule.MutantOperator) {
	siteMutants := make([]*TSMutant, 0, len(raw.Candidates))
	for _, cand := range raw.Candidates {
		id := *p.in.NextID
		*p.in.NextID++
		origLine, mutLine := buildTSDiffLines(p.in.Source, p.lineStarts, raw, cand.ReplacementPreview)
		m := &TSMutant{
			ID:           id,
			File:         p.in.FilePath,
			Line:         raw.Line,
			Operator:     op,
			OriginalExpr: cand.OrigExpr,
			MutatedExpr:  cand.MutExpr,
			OriginalLine: origLine,
			MutatedLine:  mutLine,
			Status:       rule.MutantPending,
		}
		p.fp.Mutants = append(p.fp.Mutants, m)
		siteMutants = append(siteMutants, m)
	}
	p.activeSites = append(p.activeSites, toTSMutationSite(raw, siteMutants))
}

func toTSMutationSite(raw RawTSExtracted, mutants []*TSMutant) TSMutationSite {
	return TSMutationSite{
		Kind:         raw.Kind,
		Op:           raw.Op,
		AddKeepRight: raw.AddKeepRight,
		BoolVal:      raw.BoolVal,
		StartOffset:  raw.StartOffset,
		EndOffset:    raw.EndOffset,
		LeftStart:    raw.LeftStart,
		LeftEnd:      raw.LeftEnd,
		RightStart:   raw.RightStart,
		RightEnd:     raw.RightEnd,
		Mutants:      mutants,
	}
}

func isTSBranchProbeKind(kind string) bool {
	switch kind {
	case "ordered_cmp", "eq_cmp", "logical", "nullish", "cond", "unary_not", "bool_lit":
		return true
	default:
		return false
	}
}

func computeTSLineOffsets(src []byte) []int {
	starts := []int{0}
	for i, b := range src {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func buildTSDiffLines(src []byte, lineStarts []int, raw RawTSExtracted, preview string) (string, string) {
	if raw.Line < 1 || raw.Line > len(lineStarts) {
		return "", preview
	}
	lineStart := lineStarts[raw.Line-1]
	lineEnd := len(src)
	if raw.Line < len(lineStarts) {
		lineEnd = lineStarts[raw.Line] - 1
	}
	origLine := strings.TrimSpace(string(src[lineStart:lineEnd]))
	if raw.StartOffset >= lineStart && raw.EndOffset <= lineEnd {
		mutLine := strings.TrimSpace(string(src[lineStart:raw.StartOffset]) + preview + string(src[raw.EndOffset:lineEnd]))
		return origLine, mutLine
	}
	return origLine, preview
}
