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

func TestDecisionSummaryCounts(t *testing.T) {
	testCases := []struct {
		name         string
		summary      DecisionSummary
		wantBlocking int
		wantAdvisory int
		wantHasBlock bool
	}{
		// Reports zero blocking and zero advisory findings for an empty summary.
		{
			name:         "empty summary has zero blocking and advisory counts",
			summary:      DecisionSummary{},
			wantBlocking: 0,
			wantAdvisory: 0,
			wantHasBlock: false,
		},
		// Counts non-blocking findings as advisory without setting HasBlockingFindings.
		{
			name: "advisory only findings do not mark blocking",
			summary: DecisionSummary{
				Findings: []DecisionItemFinding{
					{RuleID: "pkg_doc_alignment", Blocking: false},
					{RuleID: "pkg_doc_alignment", Blocking: false},
				},
			},
			wantBlocking: 0,
			wantAdvisory: 2,
			wantHasBlock: false,
		},
		// Separates blocking and advisory counts and reports HasBlockingFindings=true when blocking items exist.
		{
			name: "mixed advisory and blocking findings partition counts accurately",
			summary: DecisionSummary{
				Findings: []DecisionItemFinding{
					{RuleID: "pkg_doc_alignment", Blocking: false},
					{RuleID: "strict_rule", Blocking: true},
					{RuleID: "strict_rule_2", Blocking: true},
				},
			},
			wantBlocking: 2,
			wantAdvisory: 1,
			wantHasBlock: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.summary.BlockingCount(); got != tc.wantBlocking {
				t.Errorf("BlockingCount() = %d, want %d", got, tc.wantBlocking)
			}
			if got := tc.summary.AdvisoryCount(); got != tc.wantAdvisory {
				t.Errorf("AdvisoryCount() = %d, want %d", got, tc.wantAdvisory)
			}
			if got := tc.summary.HasBlockingFindings(); got != tc.wantHasBlock {
				t.Errorf("HasBlockingFindings() = %v, want %v", got, tc.wantHasBlock)
			}
		})
	}
}
