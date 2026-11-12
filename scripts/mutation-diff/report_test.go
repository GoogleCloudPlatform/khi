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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAttachLineCoverageAndUncoveredRanges(t *testing.T) {
	plans := []*FilePlan{
		{
			File: "pkg/b/logic.go",
			Mutants: []*Mutant{
				{Line: 10, Status: MutantKilled},
				{Line: 14, Status: MutantUncovered},
			},
		},
	}
	report := BuildMutationReport("trunk()", plans, nil, nil)
	AttachLineCoverage(report, &CoverageReport{
		BaseLabel:            "trunk()",
		PatchExecutableLines: 5,
		PatchCoveredLines:    3,
		Files: []FileCoverageReport{
			{
				File:                   "pkg/a/wiring.go",
				ChangedExecutableLines: 2,
				ChangedCoveredLines:    1,
				UncoveredRanges:        []LineRange{{StartLine: 8, EndLine: 8}},
			},
			{
				File:                   "pkg/b/logic.go",
				ChangedExecutableLines: 3,
				ChangedCoveredLines:    2,
				UncoveredRanges:        []LineRange{{StartLine: 15, EndLine: 16}},
			},
		},
	})

	if len(report.Files) != 2 {
		t.Fatalf("len(report.Files) = %d, want 2", len(report.Files))
	}
	if report.Files[0].File != "pkg/a/wiring.go" {
		t.Errorf("Files[0].File = %q, want pkg/a/wiring.go", report.Files[0].File)
	}
	if got := report.Files[0].UncoveredLineRanges(); got != "8" {
		t.Errorf("Files[0].UncoveredLineRanges() = %q, want %q", got, "8")
	}
	// Line 14 from MutantUncovered merges with lines 15-16 from LineCoverage into "14-16".
	if diff := cmp.Diff("14-16", report.Files[1].UncoveredLineRanges()); diff != "" {
		t.Errorf("Files[1].UncoveredLineRanges() mismatch (-want +got):\n%s", diff)
	}
}
