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

package rule

import "testing"

func TestCoverageAndMutationSummaries(t *testing.T) {
	testCases := []struct {
		name        string
		cov         CoverageSummary
		mut         MutationSummary
		rep         Report
		wantPatch   float64
		wantOverall float64
		wantScore   float64
		wantBlock   bool
		wantLang    string
		wantFence   string
	}{
		{
			name:        "zero denominators default cleanly for Go",
			cov:         CoverageSummary{},
			mut:         MutationSummary{},
			rep:         Report{},
			wantPatch:   100.0,
			wantOverall: 0.0,
			wantScore:   100.0,
			wantBlock:   false,
			wantLang:    "Go",
			wantFence:   "go",
		},
		{
			name:        "partial coverage and findings block when non-advisory",
			cov:         CoverageSummary{PatchCovered: 3, PatchExecutable: 4, OverallCovered: 1, OverallStatements: 2},
			mut:         MutationSummary{Killed: 2, TimedOut: 1, Survived: 1},
			rep:         Report{Findings: []Finding{{Rule: RuleFuncCognitive}}},
			wantPatch:   75.0,
			wantOverall: 50.0,
			wantScore:   75.0,
			wantBlock:   true,
			wantLang:    "Go",
			wantFence:   "go",
		},
		{
			name:        "advisory TypeScript report has issues without blocking",
			cov:         CoverageSummary{PatchCovered: 2, PatchExecutable: 2, OverallCovered: 4, OverallStatements: 4},
			mut:         MutationSummary{Killed: 4},
			rep:         Report{Language: "TypeScript", Advisory: true, DirectiveErrors: []DirectiveError{{File: "a.ts", Line: 1}}},
			wantPatch:   100.0,
			wantOverall: 100.0,
			wantScore:   100.0,
			wantBlock:   false,
			wantLang:    "TypeScript",
			wantFence:   "ts",
		},
		// Marks HasBlockingIssues true when a non-advisory Decisions report contains at least one blocking finding.
		{
			name:        "blocking decision finding triggers HasBlockingIssues on non-advisory report",
			cov:         CoverageSummary{},
			mut:         MutationSummary{},
			rep:         Report{Language: "Decisions", Decisions: DecisionSummary{Findings: []DecisionItemFinding{{RuleID: "r1", Blocking: true}}}},
			wantPatch:   100.0,
			wantOverall: 0.0,
			wantScore:   100.0,
			wantBlock:   true,
			wantLang:    "Decisions",
			wantFence:   "go",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cov.PatchPercent(); got != tc.wantPatch {
				t.Errorf("PatchPercent() = %v, want %v", got, tc.wantPatch)
			}
			if got := tc.cov.OverallPercent(); got != tc.wantOverall {
				t.Errorf("OverallPercent() = %v, want %v", got, tc.wantOverall)
			}
			if got := tc.mut.Score(); got != tc.wantScore {
				t.Errorf("Score() = %v, want %v", got, tc.wantScore)
			}
			if got := tc.rep.HasBlockingIssues(); got != tc.wantBlock {
				t.Errorf("HasBlockingIssues() = %v, want %v", got, tc.wantBlock)
			}
			if got := tc.rep.LanguageTitle(); got != tc.wantLang {
				t.Errorf("LanguageTitle() = %q, want %q", got, tc.wantLang)
			}
			if got := tc.rep.CodeFence(); got != tc.wantFence {
				t.Errorf("CodeFence() = %q, want %q", got, tc.wantFence)
			}
		})
	}
}

// TestFindingIsAdvisoryAndBlockingCounts verifies Finding.IsAdvisory, Report.BlockingFindingCount, Report.AdvisoryFindingCount, and Report.HasBlockingIssues.
func TestFindingIsAdvisoryAndBlockingCounts(t *testing.T) {
	testCases := []struct {
		name         string
		rep          Report
		wantBlocking int
		wantAdvisory int
		wantHasBlock bool
		wantHasIssue bool
	}{
		// Report with only ViolationAdvisory findings has issues without blocking.
		{
			name: "advisory findings only do not block",
			rep: Report{
				Findings: []Finding{
					{Rule: RuleFuncUncoveredLines, Kind: ViolationAdvisory},
					{Rule: RuleFuncSurvivingMutants, Kind: ViolationAdvisory},
				},
			},
			wantBlocking: 0,
			wantAdvisory: 2,
			wantHasBlock: false,
			wantHasIssue: true,
		},
		// Mixed blocking and advisory findings count separately and block.
		{
			name: "mixed blocking and advisory findings",
			rep: Report{
				Findings: []Finding{
					{Rule: RuleFuncCognitive, Kind: ViolationHardLimit},
					{Rule: RuleFuncUncoveredLines, Kind: ViolationAdvisory},
					{Rule: RuleFuncNesting, Kind: ViolationRatchetRegression},
				},
			},
			wantBlocking: 2,
			wantAdvisory: 1,
			wantHasBlock: true,
			wantHasIssue: true,
		},
		// Report marked Advisory never blocks even with blocking finding kinds or directive errors.
		{
			name: "report marked advisory suppresses blocking",
			rep: Report{
				Advisory: true,
				Findings: []Finding{{Rule: RuleFuncCognitive, Kind: ViolationHardLimit}},
			},
			wantBlocking: 1,
			wantAdvisory: 0,
			wantHasBlock: false,
			wantHasIssue: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.rep.BlockingFindingCount(); got != tc.wantBlocking {
				t.Errorf("BlockingFindingCount() = %d, want %d", got, tc.wantBlocking)
			}
			if got := tc.rep.AdvisoryFindingCount(); got != tc.wantAdvisory {
				t.Errorf("AdvisoryFindingCount() = %d, want %d", got, tc.wantAdvisory)
			}
			if got := tc.rep.HasBlockingIssues(); got != tc.wantHasBlock {
				t.Errorf("HasBlockingIssues() = %v, want %v", got, tc.wantHasBlock)
			}
			if got := tc.rep.HasIssues(); got != tc.wantHasIssue {
				t.Errorf("HasIssues() = %v, want %v", got, tc.wantHasIssue)
			}
		})
	}
}
