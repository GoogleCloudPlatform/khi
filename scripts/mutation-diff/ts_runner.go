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
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

const tsBrowserRuntimeFileName = "zzz_khi_mutant_runtime.js"

// TSRunnerOptions configures TypeScript AST mutation planning and single-build Karma evaluation.
type TSRunnerOptions struct {
	RepoRoot               string
	BaseLabel              string
	ChangedLinesByFile     map[string]map[int]struct{}
	ChangedTestLinesByFile map[string]map[int]struct{}
	TSCoverageByFile       map[string]gotestutil.IstanbulFileCoverage
	HasTSCoverage          bool
	CheckRedundantTests    bool
	MaxMutants             int
	Timeout                time.Duration
	LogWriter              io.Writer
	Progress               *ProgressReporter
}

type rawTSFileExtraction struct {
	Comments  []RawComment     `json:"comments"`
	Sites     []rawTSExtracted `json:"sites"`
	SpecCases []rawTSSpecCase  `json:"specCases"`
}

type rawTSCandidate struct {
	OrigExpr           string `json:"origExpr"`
	MutExpr            string `json:"mutExpr"`
	ReplacementPreview string `json:"replacementPreview"`
}

type rawTSExtracted struct {
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
	Candidates   []rawTSCandidate `json:"candidates"`
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
	Mutants      []*Mutant
}

type tsFilePlan struct {
	plan    *FilePlan
	tsSites []TSMutationSite
}

type karmaPluginConfig struct {
	CandidateIDs          []int             `json:"candidateIds"`
	FileByMutantID        map[string]string `json:"fileByMutantId"`
	MaxMutants            int               `json:"maxMutants"`
	MinPerMutantTimeoutMs int               `json:"minPerMutantTimeoutMs"`
	OverallTimeoutMs      int               `json:"overallTimeoutMs"`
	ResultsPath           string            `json:"resultsPath"`
}

type karmaPluginResults struct {
	BaselineFailed bool                       `json:"baselineFailed"`
	ErrorMessage   string                     `json:"errorMessage"`
	Outcomes       map[string]string          `json:"outcomes"`
	SpecOrder      []TSSpecOrderEntry         `json:"specOrder"`
	SpecPolarities map[string][]string        `json:"specPolarities"`
	MutantRuns     map[string]TSMutantSpecRun `json:"mutantRuns"`
}

// ExecuteTSMutationRun plans, bundles, and evaluates differential mutants across modified TypeScript files.
func ExecuteTSMutationRun(ctx context.Context, opts *TSRunnerOptions) (*MutationReport, error) {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultOverallTimeout
	}
	// Reserve a 35-second outer grace buffer beyond the Karma plugin's internal budget timer
	// so esbuild startup and graceful SKIPPED_BUDGET JSON serialization complete before SIGKILL.
	hardCtx, cancel := context.WithTimeout(ctx, timeout+35*time.Second)
	defer cancel()

	changedFiles := sortedTSChangedFiles(opts.ChangedLinesByFile)
	if len(changedFiles) == 0 {
		return BuildMutationReportForLang(LangTS, opts.BaseLabel, nil, nil, nil), nil
	}

	var changedSpecFiles []string
	if opts.CheckRedundantTests {
		changedSpecFiles = sortedTSTestChangedFiles(opts.ChangedTestLinesByFile)
	}
	extractTargets := make([]string, 0, len(changedFiles)+len(changedSpecFiles))
	extractTargets = append(extractTargets, changedFiles...)
	extractTargets = append(extractTargets, changedSpecFiles...)

	opts.Progress.Phase(3, 4, fmt.Sprintf("Extracting AST mutation sites from %d TypeScript file(s)...", len(changedFiles)))
	extractions, err := invokeTSMutatorScript(hardCtx, opts.RepoRoot, extractTargets)
	if err != nil {
		return nil, err
	}

	var changedSpecs []ChangedTSSpec
	for _, specFile := range changedSpecFiles {
		if ext, ok := extractions[specFile]; ok {
			changedSpecs = append(changedSpecs, DiscoverChangedTSSpecs(specFile, ext.SpecCases, opts.ChangedTestLinesByFile[specFile])...)
		}
	}

	var tsPlans []tsFilePlan
	var allPlans []*FilePlan
	var allWarnings []DirectiveWarning
	var pendingMutants []*Mutant
	var filesWithPending []string
	mutantFileByID := make(map[int]string)
	nextMutantID := 1
	nextProbeID := 1
	globalIdx := 1

	for _, relFile := range changedFiles {
		ext, ok := extractions[relFile]
		if !ok {
			continue
		}
		absFile := filepath.Join(opts.RepoRoot, filepath.FromSlash(relFile))
		src, err := os.ReadFile(absFile)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("failed to read %s: %w", relFile, err)
		}

		var coveredLines map[int]struct{}
		if opts.HasTSCoverage {
			if fc, hasCov := opts.TSCoverageByFile[relFile]; hasCov {
				coveredLines = gotestutil.CoveredTSLineSet(fc)
			} else {
				coveredLines = make(map[int]struct{})
			}
		}

		var probeCounter *int
		if len(changedSpecs) > 0 && hasMatchingChangedTSSpec(relFile, changedSpecFiles) {
			probeCounter = &nextProbeID
		}

		fp, sites := PlanTSFileMutations(relFile, src, ext, opts.ChangedLinesByFile[relFile], coveredLines, opts.HasTSCoverage, &nextMutantID, probeCounter)
		fileHasPending := false
		for _, m := range fp.Mutants {
			m.GlobalIndex = globalIdx
			globalIdx++
			if m.Status == MutantPending {
				pendingMutants = append(pendingMutants, m)
				mutantFileByID[m.ID] = relFile
				fileHasPending = true
			}
		}
		for _, s := range sites {
			for _, sm := range s.Mutants {
				if sm.ID < 0 {
					mutantFileByID[sm.ID] = relFile
				}
			}
		}
		if fileHasPending {
			filesWithPending = append(filesWithPending, relFile)
			tsPlans = append(tsPlans, tsFilePlan{plan: fp, tsSites: sites})
		}
		allPlans = append(allPlans, fp)
		allWarnings = append(allWarnings, fp.Warnings...)
	}

	if len(pendingMutants) == 0 {
		return BuildMutationReportForLang(LangTS, opts.BaseLabel, allPlans, allWarnings, nil), nil
	}

	specPlan, err := gotestutil.PlanTSTestExecution(opts.RepoRoot, filesWithPending, false)
	if err != nil {
		return nil, err
	}
	if specPlan.SkipKarma {
		markPendingMutants(allPlans, MutantUncovered)
		return BuildMutationReportForLang(LangTS, opts.BaseLabel, allPlans, allWarnings, nil), nil
	}

	maxMutantID := nextMutantID - 1
	maxProbeID := nextProbeID - 1
	maxBudget := opts.MaxMutants
	if maxBudget <= 0 {
		maxBudget = defaultMaxMutants
	}

	opts.Progress.Phase(4, 4, fmt.Sprintf("Evaluating up to %d TypeScript mutant(s) across %d spec file(s)...",
		min(len(pendingMutants), maxBudget), len(specPlan.IncludeSpecs)))

	workDir, err := os.MkdirTemp("", "khi-ts-mutation-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create TypeScript mutation workspace: %w", err)
	}
	defer os.RemoveAll(workDir)

	resultsPath, cfgPath, err := prepareTSMutationWorkspace(opts.RepoRoot, workDir, tsPlans, pendingMutants, maxMutantID, maxProbeID, maxBudget, timeout)
	if err != nil {
		return nil, err
	}

	if err := runKarmaMutationSession(hardCtx, opts.RepoRoot, workDir, cfgPath, specPlan.IncludeSpecs, opts.LogWriter); err != nil {
		return nil, err
	}

	resBytes, err := os.ReadFile(resultsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read TypeScript mutation results %s: %w", resultsPath, err)
	}
	var pluginRes karmaPluginResults
	if err := json.Unmarshal(resBytes, &pluginRes); err != nil {
		return nil, fmt.Errorf("failed to decode TypeScript mutation results: %w", err)
	}
	if pluginRes.BaselineFailed {
		return nil, fmt.Errorf("baseline TypeScript Karma tests failed: %s", pluginRes.ErrorMessage)
	}

	for _, m := range pendingMutants {
		switch pluginRes.Outcomes[strconv.Itoa(m.ID)] {
		case string(MutantKilled):
			m.Status = MutantKilled
		case string(MutantTimedOut):
			m.Status = MutantTimedOut
		case string(MutantSurvived):
			m.Status = MutantSurvived
		case string(MutantUncovered):
			m.Status = MutantUncovered
		default:
			m.Status = MutantSkippedBudget
		}
	}

	var redundantFindings []RedundantTestFinding
	if opts.CheckRedundantTests && len(changedSpecs) > 0 {
		skippedIDs := collectBudgetSkippedMutantIDs(allPlans)
		redundantFindings = AnalyzeTSTestRedundancy(changedSpecs, pluginRes.SpecOrder, pluginRes.SpecPolarities, pluginRes.MutantRuns, mutantFileByID, skippedIDs)
		for i := range redundantFindings {
			redundantFindings[i].GlobalIndex = i + 1
		}
	}

	return BuildMutationReportForLang(LangTS, opts.BaseLabel, allPlans, allWarnings, redundantFindings), nil
}

// PlanTSFileMutations converts raw AST sites extracted by mutate_ts.js into Mutant entries and active Super-Mutant sites.
func PlanTSFileMutations(
	filePath string,
	src []byte,
	ext rawTSFileExtraction,
	changedLines map[int]struct{},
	coveredLines map[int]struct{},
	hasTSCoverage bool,
	nextID *int,
	nextProbeID *int,
) (*FilePlan, []TSMutationSite) {
	suppressions := ParseSuppressionsFromComments(filePath, ext.Comments, src)
	lineStarts := computeLineOffsets(src)

	fp := &FilePlan{
		File:     filePath,
		Source:   src,
		Warnings: suppressions.Warnings,
	}

	type spanPair struct {
		start int
		end   int
	}
	seenSpans := make(map[spanPair]struct{})
	var activeSites []TSMutationSite

	for _, raw := range ext.Sites {
		if raw.StartOffset < 0 || raw.EndOffset > len(src) || raw.StartOffset >= raw.EndOffset {
			continue
		}
		if !hasLine(changedLines, raw.Line) {
			if nextProbeID == nil || !isBranchProbeKind(raw.Kind) {
				continue
			}
			sp := spanPair{start: raw.StartOffset, end: raw.EndOffset}
			if _, dup := seenSpans[sp]; dup {
				continue
			}
			seenSpans[sp] = struct{}{}

			probeID := *nextProbeID
			*nextProbeID++
			probeMut := &Mutant{ID: -probeID}
			var probeSlots []*Mutant
			if raw.Kind == "ordered_cmp" || raw.Kind == "nullish" || raw.Kind == "cond" {
				probeSlots = []*Mutant{probeMut, probeMut}
			} else {
				probeSlots = []*Mutant{probeMut}
			}
			activeSites = append(activeSites, TSMutationSite{
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
				Mutants:      probeSlots,
			})
			continue
		}
		sp := spanPair{start: raw.StartOffset, end: raw.EndOffset}
		if _, dup := seenSpans[sp]; dup {
			continue
		}
		seenSpans[sp] = struct{}{}

		op := MutantOperator(raw.Operator)
		if !op.IsValid() || len(raw.Candidates) == 0 {
			continue
		}

		if supCat, suppressed := suppressions.LookupSuppression(raw.Line, op); suppressed {
			for _, cand := range raw.Candidates {
				origLine, mutLine := buildDiffLines(src, lineStarts, raw.Line, raw.StartOffset, raw.EndOffset, cand.ReplacementPreview)
				fp.Mutants = append(fp.Mutants, &Mutant{
					File:                filePath,
					Line:                raw.Line,
					Operator:            op,
					OriginalExpr:        cand.OrigExpr,
					MutatedExpr:         cand.MutExpr,
					OriginalLine:        origLine,
					MutatedLine:         mutLine,
					SuppressionCategory: supCat,
					Status:              MutantSuppressed,
				})
			}
			continue
		}

		if hasTSCoverage && !hasLine(coveredLines, raw.Line) {
			for _, cand := range raw.Candidates {
				origLine, mutLine := buildDiffLines(src, lineStarts, raw.Line, raw.StartOffset, raw.EndOffset, cand.ReplacementPreview)
				fp.Mutants = append(fp.Mutants, &Mutant{
					File:         filePath,
					Line:         raw.Line,
					Operator:     op,
					OriginalExpr: cand.OrigExpr,
					MutatedExpr:  cand.MutExpr,
					OriginalLine: origLine,
					MutatedLine:  mutLine,
					Status:       MutantUncovered,
				})
			}
			continue
		}

		siteMutants := make([]*Mutant, 0, len(raw.Candidates))
		for _, cand := range raw.Candidates {
			id := *nextID
			*nextID++
			origLine, mutLine := buildDiffLines(src, lineStarts, raw.Line, raw.StartOffset, raw.EndOffset, cand.ReplacementPreview)
			m := &Mutant{
				ID:           id,
				File:         filePath,
				Line:         raw.Line,
				Operator:     op,
				OriginalExpr: cand.OrigExpr,
				MutatedExpr:  cand.MutExpr,
				OriginalLine: origLine,
				MutatedLine:  mutLine,
				Status:       MutantPending,
			}
			fp.Mutants = append(fp.Mutants, m)
			siteMutants = append(siteMutants, m)
		}

		activeSites = append(activeSites, TSMutationSite{
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
			Mutants:      siteMutants,
		})
	}

	return fp, activeSites
}

// RenderTSSuperMutantSource splices activeSites into TypeScript source bytes from innermost to outermost spans.
func RenderTSSuperMutantSource(src []byte, activeSites []TSMutationSite) []byte {
	if len(activeSites) == 0 {
		return src
	}

	sorted := make([]TSMutationSite, len(activeSites))
	copy(sorted, activeSites)
	slices.SortFunc(sorted, func(a, b TSMutationSite) int {
		return cmp.Or(
			cmp.Compare(a.StartOffset, b.StartOffset),
			cmp.Compare(b.EndOffset, a.EndOffset),
		)
	})

	rendered := renderTSSpan(src, sorted, 0, len(src), -1)
	return []byte("// @ts-nocheck\n" + rendered)
}

func renderTSSpan(src []byte, sites []TSMutationSite, start int, end int, excludeIdx int) string {
	var sb strings.Builder
	cursor := start

	for i := range sites {
		if i == excludeIdx {
			continue
		}
		s := &sites[i]
		if s.StartOffset < cursor || s.EndOffset > end {
			continue
		}
		if excludeIdx >= 0 {
			parent := &sites[excludeIdx]
			if s.StartOffset == parent.StartOffset && s.EndOffset == parent.EndOffset {
				continue
			}
		}

		sb.Write(src[cursor:s.StartOffset])
		sb.WriteString(renderSingleTSSite(src, sites, i))
		cursor = s.EndOffset
	}

	sb.Write(src[cursor:end])
	return sb.String()
}

func renderSingleTSSite(src []byte, sites []TSMutationSite, idx int) string {
	s := &sites[idx]
	switch s.Kind {
	case "ordered_cmp":
		lhs := renderTSSpan(src, sites, s.LeftStart, s.LeftEnd, idx)
		rhs := renderTSSpan(src, sites, s.RightStart, s.RightEnd, idx)
		fn := "lt"
		switch s.Op {
		case "<=":
			fn = "le"
		case ">":
			fn = "gt"
		case ">=":
			fn = "ge"
		}
		return fmt.Sprintf("(globalThis.__khiMut.%s(%d, %d, (%s), (%s)))", fn, s.Mutants[0].ID, s.Mutants[1].ID, lhs, rhs)

	case "eq_cmp":
		lhs := renderTSSpan(src, sites, s.LeftStart, s.LeftEnd, idx)
		rhs := renderTSSpan(src, sites, s.RightStart, s.RightEnd, idx)
		return fmt.Sprintf("(globalThis.__khiMut.flip(%d, (%s) %s (%s)))", s.Mutants[0].ID, lhs, s.Op, rhs)

	case "logical":
		lhs := renderTSSpan(src, sites, s.LeftStart, s.LeftEnd, idx)
		rhs := renderTSSpan(src, sites, s.RightStart, s.RightEnd, idx)
		fn := "and"
		if s.Op == "||" {
			fn = "or"
		}
		return fmt.Sprintf("(globalThis.__khiMut.%s(%d, () => (%s), () => (%s)))", fn, s.Mutants[0].ID, lhs, rhs)

	case "nullish":
		lhs := renderTSSpan(src, sites, s.LeftStart, s.LeftEnd, idx)
		rhs := renderTSSpan(src, sites, s.RightStart, s.RightEnd, idx)
		return fmt.Sprintf("(globalThis.__khiMut.coal(%d, %d, () => (%s), () => (%s)))", s.Mutants[0].ID, s.Mutants[1].ID, lhs, rhs)

	case "cond":
		inner := renderTSSpan(src, sites, s.StartOffset, s.EndOffset, idx)
		return fmt.Sprintf("(globalThis.__khiMut.cond(%d, %d, Boolean(%s)))", s.Mutants[0].ID, s.Mutants[1].ID, inner)

	case "arith":
		lhs := renderTSSpan(src, sites, s.LeftStart, s.LeftEnd, idx)
		rhs := renderTSSpan(src, sites, s.RightStart, s.RightEnd, idx)
		fn := "addL"
		switch s.Op {
		case "+":
			if s.AddKeepRight {
				fn = "addR"
			}
		case "-":
			fn = "sub"
		case "*":
			fn = "mul"
		case "/":
			fn = "div"
		}
		return fmt.Sprintf("(globalThis.__khiMut.%s(%d, (%s), (%s)))", fn, s.Mutants[0].ID, lhs, rhs)

	case "unary_not":
		operand := renderTSSpan(src, sites, s.LeftStart, s.LeftEnd, idx)
		return fmt.Sprintf("(globalThis.__khiMut.flip(%d, !(%s)))", s.Mutants[0].ID, operand)

	case "bool_lit":
		lit := "false"
		if s.BoolVal {
			lit = "true"
		}
		return fmt.Sprintf("(globalThis.__khiMut.flip(%d, %s))", s.Mutants[0].ID, lit)

	case "stmt_skip":
		stmtText := renderTSSpan(src, sites, s.StartOffset, s.EndOffset, idx)
		return fmt.Sprintf("if (!globalThis.__khiMut.active(%d)) { %s }", s.Mutants[0].ID, stmtText)

	default:
		return string(src[s.StartOffset:s.EndOffset])
	}
}

// GenerateTSBrowserRuntimeSource returns the classic browser script defining globalThis.__khiMut and Jasmine hit/polarity tracing.
func GenerateTSBrowserRuntimeSource(maxMutantID int, maxProbeID int) []byte {
	seenCap := max(maxMutantID+1, 2)
	probeCap := max(maxProbeID+1, 2)
	script := fmt.Sprintf(`(function () {
  'use strict';

  function parseSelectedMutant() {
    var k = globalThis.__karma__;
    if (!k || !k.config || !Array.isArray(k.config.args)) {
      return 0;
    }
    for (var i = 0; i < k.config.args.length; i++) {
      var arg = String(k.config.args[i]);
      if (arg.indexOf('--khi-mutant=') === 0) {
        var parsed = parseInt(arg.slice('--khi-mutant='.length), 10);
        return isNaN(parsed) ? 0 : parsed;
      }
    }
    return 0;
  }

  var selectedId = parseSelectedMutant();
  var currentSpecId = null;
  var currentSpecFullName = null;
  var currentSpecMutantSeen = null;
  var currentSpecSlotSeen = null;
  var currentSpecProbeSlotSeen = null;
  var topLevelSeen = new Uint8Array(%d);
  var topLevelHits = [];
  var specHits = Object.create(null);
  var specOrder = [];
  var specPolarities = Object.create(null);

  function recordSlot(id, slot, suffix) {
    if (id < 0) {
      var pId = -id;
      if (pId >= %d) {
        return;
      }
      if (currentSpecId !== null && currentSpecProbeSlotSeen !== null) {
        var pSlotIdx = pId * 3 + slot;
        if (!currentSpecProbeSlotSeen[pSlotIdx]) {
          currentSpecProbeSlotSeen[pSlotIdx] = 1;
          var pPolList = specPolarities[currentSpecId];
          if (!pPolList) {
            pPolList = [];
            specPolarities[currentSpecId] = pPolList;
          }
          pPolList.push('P' + String(pId) + suffix);
        }
      }
      return;
    }
    if (id === 0 || id >= %d) {
      return;
    }
    if (currentSpecFullName !== null && currentSpecMutantSeen !== null) {
      if (!currentSpecMutantSeen[id]) {
        currentSpecMutantSeen[id] = 1;
        var hitList = specHits[currentSpecFullName];
        if (!hitList) {
          hitList = [];
          specHits[currentSpecFullName] = hitList;
        }
        hitList.push(id);
      }
      if (currentSpecId !== null && currentSpecSlotSeen !== null) {
        var slotIdx = id * 3 + slot;
        if (!currentSpecSlotSeen[slotIdx]) {
          currentSpecSlotSeen[slotIdx] = 1;
          var polList = specPolarities[currentSpecId];
          if (!polList) {
            polList = [];
            specPolarities[currentSpecId] = polList;
          }
          polList.push(String(id) + suffix);
        }
      }
      return;
    }
    if (!topLevelSeen[id]) {
      topLevelSeen[id] = 1;
      topLevelHits.push(id);
    }
  }

  function recordBool(id, v) {
    if (v) {
      recordSlot(id, 2, ':T');
    } else {
      recordSlot(id, 1, ':F');
    }
  }

  function active(id) {
    if (selectedId === 0) {
      recordSlot(id, 0, ':S');
      return false;
    }
    return selectedId === id;
  }

  globalThis.__khiMut = {
    active: active,
    flip: function (id, v) {
      if (selectedId === 0) {
        recordBool(id, Boolean(v));
        return v;
      }
      return selectedId === id ? !v : v;
    },
    cond: function (idTrue, idFalse, cond) {
      if (selectedId === 0) {
        var b = Boolean(cond);
        recordBool(idTrue, b);
        recordBool(idFalse, b);
        return cond;
      }
      if (selectedId === idTrue) return true;
      if (selectedId === idFalse) return false;
      return cond;
    },
    and: function (id, lhsFn, rhsFn) {
      var l = lhsFn();
      if (selectedId === 0) {
        var res = l ? rhsFn() : l;
        recordBool(id, Boolean(res));
        return res;
      }
      if (selectedId === id) {
        return l ? l : rhsFn();
      }
      return l ? rhsFn() : l;
    },
    or: function (id, lhsFn, rhsFn) {
      var l = lhsFn();
      if (selectedId === 0) {
        var res = l ? l : rhsFn();
        recordBool(id, Boolean(res));
        return res;
      }
      if (selectedId === id) {
        return l ? rhsFn() : l;
      }
      return l ? l : rhsFn();
    },
    coal: function (idKeepLeft, idKeepRight, lhsFn, rhsFn) {
      if (selectedId === 0) {
        var l = lhsFn();
        var nonNullish = l !== null && l !== undefined;
        recordBool(idKeepLeft, nonNullish);
        recordBool(idKeepRight, nonNullish);
        return nonNullish ? l : rhsFn();
      }
      if (selectedId === idKeepLeft) {
        return lhsFn();
      }
      if (selectedId === idKeepRight) {
        return rhsFn();
      }
      var orig = lhsFn();
      return orig !== null && orig !== undefined ? orig : rhsFn();
    },
    lt: function (id1, id2, a, b) {
      if (selectedId === 0) {
        var res = a < b;
        recordBool(id1, res);
        recordBool(id2, res);
        return res;
      }
      if (selectedId === id1) return a <= b;
      if (selectedId === id2) return a >= b;
      return a < b;
    },
    le: function (id1, id2, a, b) {
      if (selectedId === 0) {
        var res = a <= b;
        recordBool(id1, res);
        recordBool(id2, res);
        return res;
      }
      if (selectedId === id1) return a < b;
      if (selectedId === id2) return a > b;
      return a <= b;
    },
    gt: function (id1, id2, a, b) {
      if (selectedId === 0) {
        var res = a > b;
        recordBool(id1, res);
        recordBool(id2, res);
        return res;
      }
      if (selectedId === id1) return a >= b;
      if (selectedId === id2) return a <= b;
      return a > b;
    },
    ge: function (id1, id2, a, b) {
      if (selectedId === 0) {
        var res = a >= b;
        recordBool(id1, res);
        recordBool(id2, res);
        return res;
      }
      if (selectedId === id1) return a > b;
      if (selectedId === id2) return a < b;
      return a >= b;
    },
    addL: function (id, a, b) {
      return active(id) ? a : a + b;
    },
    addR: function (id, a, b) {
      return active(id) ? b : a + b;
    },
    sub: function (id, a, b) {
      return active(id) ? a + b : a - b;
    },
    mul: function (id, a, b) {
      return active(id) ? a / b : a * b;
    },
    div: function (id, a, b) {
      return active(id) ? a * b : a / b;
    }
  };

  if (selectedId === 0 && globalThis.__karma__) {
    var rawStart = globalThis.__karma__.start;
    var wrapStart = function (fn) {
      return function () {
        selectedId = parseSelectedMutant();
        if (selectedId === 0 && globalThis.jasmine && typeof globalThis.jasmine.getEnv === 'function') {
          globalThis.jasmine.getEnv().addReporter({
            specStarted: function (result) {
              currentSpecId = result && result.id ? String(result.id) : null;
              currentSpecFullName = result ? result.fullName : null;
              currentSpecMutantSeen = new Uint8Array(%d);
              currentSpecSlotSeen = new Uint8Array(%d);
              currentSpecProbeSlotSeen = new Uint8Array(%d);
            },
            specDone: function (result) {
              if (
                result &&
                result.id &&
                result.fullName &&
                result.status !== 'pending' &&
                result.status !== 'disabled' &&
                result.status !== 'excluded'
              ) {
                specOrder.push({ id: String(result.id), fullName: String(result.fullName) });
              }
              currentSpecId = null;
              currentSpecFullName = null;
              currentSpecMutantSeen = null;
              currentSpecSlotSeen = null;
              currentSpecProbeSlotSeen = null;
            },
            jasmineDone: function () {
              if (globalThis.__karma__ && typeof globalThis.__karma__.info === 'function') {
                globalThis.__karma__.info({
                  khiMutantReport: {
                    specHits: specHits,
                    specOrder: specOrder,
                    specPolarities: specPolarities,
                    topLevelHits: topLevelHits
                  }
                });
              }
            }
          });
        }
        return fn.apply(this, arguments);
      };
    };
    var currentStart = typeof rawStart === 'function' ? wrapStart(rawStart) : undefined;
    Object.defineProperty(globalThis.__karma__, 'start', {
      configurable: true,
      enumerable: true,
      get: function () {
        return currentStart;
      },
      set: function (fn) {
        currentStart = typeof fn === 'function' ? wrapStart(fn) : fn;
      }
    });
  }
})();
`, seenCap, probeCap, seenCap, seenCap, seenCap*3, probeCap*3)
	return []byte(script)
}

func prepareTSMutationWorkspace(
	repoRoot string,
	workDir string,
	tsPlans []tsFilePlan,
	pendingMutants []*Mutant,
	maxMutantID int,
	maxProbeID int,
	maxMutants int,
	overallTimeout time.Duration,
) (string, string, error) {
	absRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", "", err
	}
	absWork, err := filepath.Abs(workDir)
	if err != nil {
		return "", "", err
	}

	tmpWebDir := filepath.Join(absWork, "web")
	if err := os.MkdirAll(tmpWebDir, 0o755); err != nil {
		return "", "", err
	}

	// Symlink read-only dependencies and mirror web/src as a symlink tree so original files stay untouched.
	for _, entry := range []string{"node_modules", "package.json", "tsconfig.json"} {
		srcPath := filepath.Join(absRepo, "web", entry)
		dstPath := filepath.Join(tmpWebDir, entry)
		if err := os.Symlink(srcPath, dstPath); err != nil {
			return "", "", fmt.Errorf("failed to symlink %s: %w", entry, err)
		}
	}

	if err := createSymlinkTree(filepath.Join(absRepo, "web", "src"), filepath.Join(tmpWebDir, "src")); err != nil {
		return "", "", fmt.Errorf("failed to stage symlink tree for web/src: %w", err)
	}

	// Materialize Super-Mutant TypeScript sources by replacing only the mutated symlinks with regular files.
	for _, tp := range tsPlans {
		if len(tp.tsSites) == 0 {
			continue
		}
		relUnderWeb := strings.TrimPrefix(tp.plan.File, "web/")
		dstFile := filepath.Join(tmpWebDir, filepath.FromSlash(relUnderWeb))
		if err := os.Remove(dstFile); err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", "", fmt.Errorf("failed to remove symlink before writing mutant %s: %w", dstFile, err)
		}
		rendered := RenderTSSuperMutantSource(tp.plan.Source, tp.tsSites)
		if err := os.WriteFile(dstFile, rendered, 0o600); err != nil {
			return "", "", fmt.Errorf("failed to write Super-Mutant TypeScript file %s: %w", dstFile, err)
		}
	}

	// Write browser runtime helper script.
	runtimePath := filepath.Join(tmpWebDir, tsBrowserRuntimeFileName)
	if err := os.WriteFile(runtimePath, GenerateTSBrowserRuntimeSource(maxMutantID, maxProbeID), 0o600); err != nil {
		return "", "", fmt.Errorf("failed to write %s: %w", runtimePath, err)
	}

	// Keep strict compiler settings identical to tsconfig.spec.json while preserving symlinks;
	// rendered Super-Mutant files carry // @ts-nocheck so injected runtime wrappers do not break narrowing.
	tsconfigMutant := `{
  "extends": "./tsconfig.json",
  "compilerOptions": {
    "outDir": "./out-tsc/spec",
    "preserveSymlinks": true,
    "types": ["jasmine", "@types/gtag.js"]
  },
  "include": ["src/**/*.spec.ts", "src/**/*.d.ts"]
}
`
	if err := os.WriteFile(filepath.Join(tmpWebDir, "tsconfig.mutant.json"), []byte(tsconfigMutant), 0o600); err != nil {
		return "", "", err
	}

	// Write temporary angular.json wired to tsconfig.mutant.json, karma.mutant.conf.js, and zzz_khi_mutant_runtime.js.
	angularJSON := fmt.Sprintf(`{
  "version": 1,
  "newProjectRoot": "projects",
  "projects": {
    "frontend": {
      "projectType": "application",
      "root": "",
      "sourceRoot": "src",
      "prefix": "app",
      "architect": {
        "test": {
          "builder": "@angular/build:karma",
          "options": {
            "polyfills": ["zone.js", "zone.js/testing"],
            "tsConfig": "tsconfig.mutant.json",
            "karmaConfig": "karma.mutant.conf.js",
            "preserveSymlinks": true,
            "sourceMap": false,
            "inlineStyleLanguage": "sass",
            "assets": ["src/favicon.ico", "src/assets"],
            "styles": [
              "./node_modules/@angular/material/prebuilt-themes/indigo-pink.css",
              "src/styles.scss"
            ],
            "scripts": [%q]
          }
        }
      }
    }
  },
  "cli": {
    "analytics": false
  }
}
`, tsBrowserRuntimeFileName)
	if err := os.WriteFile(filepath.Join(tmpWebDir, "angular.json"), []byte(angularJSON), 0o600); err != nil {
		return "", "", err
	}

	pluginScriptPath, err := resolveTSHelperScriptPath(absRepo, "karma_mutation_plugin.js")
	if err != nil {
		return "", "", err
	}

	karmaConf := fmt.Sprintf(`module.exports = function (config) {
  config.set({
    basePath: '',
    frameworks: ['jasmine', 'khi-mutation'],
    plugins: [
      require('karma-jasmine'),
      require('karma-chrome-launcher'),
      require(%q),
    ],
    client: {
      jasmine: {
        random: false,
      },
      clearContext: true,
    },
    reporters: [],
    port: parseInt(process.env.KARMA_PORT, 10) || 9876,
    hostname: '127.0.0.1',
    listenAddress: '127.0.0.1',
    colors: false,
    logLevel: config.LOG_ERROR,
    autoWatch: false,
    singleRun: false,
    restartOnFileChange: false,
    captureTimeout: 180 * 1000,
    browserDisconnectTimeout: 30 * 1000,
    browserDisconnectTolerance: 0,
    browserNoActivityTimeout: 60 * 1000,
    customLaunchers: {
      ChromeHeadlessNoSandbox: {
        base: 'ChromeHeadless',
        flags: [
          '--no-sandbox',
          '--disable-extensions',
          '--enable-unsafe-swiftshader',
          '--enable-webgl',
          '--disable-gpu',
          '--remote-debugging-port=0',
        ],
      },
    },
  });
};
`, pluginScriptPath)
	if err := os.WriteFile(filepath.Join(tmpWebDir, "karma.mutant.conf.js"), []byte(karmaConf), 0o600); err != nil {
		return "", "", err
	}

	resultsPath := filepath.Join(absWork, "mutation-results.json")
	cfgPath := filepath.Join(absWork, "mutation-config.json")

	candIDs := make([]int, 0, len(pendingMutants))
	fileByID := make(map[string]string, len(pendingMutants))
	for _, m := range pendingMutants {
		candIDs = append(candIDs, m.ID)
		fileByID[strconv.Itoa(m.ID)] = m.File
	}

	pluginBudget := max(15*time.Second, overallTimeout-30*time.Second)
	pluginCfg := karmaPluginConfig{
		CandidateIDs:          candIDs,
		FileByMutantID:        fileByID,
		MaxMutants:            maxMutants,
		MinPerMutantTimeoutMs: int(minPerMutantTimeout.Milliseconds()),
		OverallTimeoutMs:      int(pluginBudget.Milliseconds()),
		ResultsPath:           resultsPath,
	}
	cfgBytes, err := json.MarshalIndent(pluginCfg, "", "  ")
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(cfgPath, cfgBytes, 0o600); err != nil {
		return "", "", err
	}

	return resultsPath, cfgPath, nil
}

func runKarmaMutationSession(
	ctx context.Context,
	repoRoot string,
	workDir string,
	cfgPath string,
	includeSpecs []string,
	errOut io.Writer,
) error {
	port, err := gotestutil.PickKarmaPort(ctx)
	if err != nil {
		return err
	}

	absRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return err
	}
	absWork, err := filepath.Abs(workDir)
	if err != nil {
		return err
	}
	tmpWebDir := filepath.Join(absWork, "web")

	ngArgs := make([]string, 0, 6+len(includeSpecs))
	ngArgs = append(ngArgs,
		"ng", "test",
		"--browsers=ChromeHeadlessNoSandbox",
		"--watch=false",
		"--progress=false",
	)
	for _, spec := range includeSpecs {
		ngArgs = append(ngArgs, "--include="+spec)
	}

	var cmd *exec.Cmd
	if gotestutil.IsContainerExecutionEnabled() {
		containerCmd, builderImage, err := gotestutil.ResolveContainerCommand()
		if err != nil {
			return err
		}
		args := []string{
			"run", "--rm",
			"--shm-size=1g",
			fmt.Sprintf("-u=%d:%d", os.Getuid(), os.Getgid()),
			fmt.Sprintf("-v=%s:%s:ro", absRepo, absRepo),
			fmt.Sprintf("-v=%s:%s", absWork, absWork),
			"-w=" + tmpWebDir,
			"-e=HOME=/tmp",
			"-e=KARMA_PORT=" + port,
			"-e=KHI_MUTATION_CONFIG_FILE=" + cfgPath,
			builderImage,
			"npx",
		}
		args = append(args, ngArgs...)
		cmd = exec.CommandContext(ctx, containerCmd, args...)
		cmd.Dir = absRepo
	} else {
		cmd = exec.CommandContext(ctx, "npx", ngArgs...)
		cmd.Dir = tmpWebDir
		cmd.Env = append(os.Environ(),
			"KARMA_PORT="+port,
			"KHI_MUTATION_CONFIG_FILE="+cfgPath,
		)
	}

	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined

	if err := cmd.Run(); err != nil {
		if errOut != nil && combined.Len() > 0 {
			_, _ = io.Copy(errOut, &combined)
		}
		return fmt.Errorf("TypeScript Karma mutation run failed (npx %s): %w", strings.Join(ngArgs, " "), err)
	}
	return nil
}

func createSymlinkTree(srcRoot string, dstRoot string) error {
	return filepath.WalkDir(srcRoot, func(absPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcRoot, absPath)
		if err != nil {
			return err
		}
		target := filepath.Join(dstRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return os.Symlink(absPath, target)
	})
}

func invokeTSMutatorScript(ctx context.Context, repoRoot string, relFiles []string) (map[string]rawTSFileExtraction, error) {
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

	var parsed map[string]rawTSFileExtraction
	if err := json.Unmarshal(outBytes, &parsed); err != nil {
		return nil, fmt.Errorf("failed to decode mutate_ts.js output: %w", err)
	}
	return parsed, nil
}

func resolveTSHelperScriptPath(repoRoot string, scriptName string) (string, error) {
	repoCandidate := filepath.Join(repoRoot, "scripts", "mutation-diff", scriptName)
	if _, err := os.Stat(repoCandidate); err == nil {
		return repoCandidate, nil
	}
	if _, callerFile, _, ok := runtime.Caller(0); ok {
		callerCandidate := filepath.Join(filepath.Dir(callerFile), scriptName)
		if _, err := os.Stat(callerCandidate); err == nil {
			return callerCandidate, nil
		}
	}
	return "", fmt.Errorf("%s helper script not found under %s", scriptName, repoRoot)
}

func sortedTSChangedFiles(m map[string]map[int]struct{}) []string {
	out := make([]string, 0, len(m))
	for f, lines := range m {
		if len(lines) > 0 && gotestutil.IsTargetProductionTSFile(f) {
			out = append(out, f)
		}
	}
	slices.Sort(out)
	return out
}

func sortedTSTestChangedFiles(m map[string]map[int]struct{}) []string {
	out := make([]string, 0, len(m))
	for f, lines := range m {
		if len(lines) > 0 && gotestutil.IsTargetTSTestFile(f) {
			out = append(out, f)
		}
	}
	slices.Sort(out)
	return out
}

func hasMatchingChangedTSSpec(prodFile string, changedSpecFiles []string) bool {
	for _, specFile := range changedSpecFiles {
		if IsSubjectMutantForTSSpec(specFile, prodFile) {
			return true
		}
	}
	return false
}

func isBranchProbeKind(kind string) bool {
	switch kind {
	case "ordered_cmp", "eq_cmp", "logical", "nullish", "cond", "unary_not", "bool_lit":
		return true
	default:
		return false
	}
}
