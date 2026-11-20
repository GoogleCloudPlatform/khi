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

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestQAReportNormalizeNilReceiver verifies that Normalize and HasMustFix handle a nil QAReport receiver safely.
func TestQAReportNormalizeNilReceiver(t *testing.T) {
	var report *QAReport

	report.Normalize()
	gotHasMustFix := report.HasMustFix()

	if gotHasMustFix {
		t.Errorf("(*QAReport)(nil).HasMustFix() = true, want false")
	}
}

func TestQAReportNormalizeDeduplication(t *testing.T) {
	testCases := []struct {
		name   string
		report QAReport
		want   QAReport
	}{
		// Deduplicates identical MustFix, Advisory, and SkippedSource entries while keeping distinct items.
		{
			name: "deduplicates identical items and skipped sources",
			report: QAReport{
				BaseLabel: "main",
				Speed:     SpeedAll,
				Filter:    FilterAll,
				MustFix: []QAItem{
					{ID: "m1", Source: "astcheck", Check: "func_cognitive", File: "pkg/a/a.go", Line: 5, Symbol: "A", Summary: "too complex"},
					{ID: "m1", Source: "astcheck", Check: "func_cognitive", File: "pkg/a/a.go", Line: 5, Symbol: "A", Summary: "too complex"},
					{ID: "m1", Source: "astcheck", Check: "func_cognitive", File: "pkg/a/a.go", Line: 5, Symbol: "A", Summary: "different summary"},
					{ID: "m2", Source: "astcheck", Check: "func_cognitive", File: "pkg/a/a.go", Line: 5, Symbol: "A", Summary: "too complex"},
					{ID: "m2", Source: "astcheck", Check: "func_cognitive", File: "pkg/a/a.go", Line: 5, Symbol: "B", Summary: "too complex"},
					{ID: "m2", Source: "astcheck", Check: "func_nesting", File: "pkg/a/a.go", Line: 5, Symbol: "B", Summary: "too complex"},
					{ID: "m2", Source: "gofmt", Check: "func_nesting", File: "pkg/a/a.go", Line: 5, Symbol: "B", Summary: "too complex"},
					{ID: "m2", Source: "gofmt", Check: "func_nesting", File: "pkg/a/a.go", Line: 6, Symbol: "B", Summary: "too complex"},
					{ID: "m2", Source: "gofmt", Check: "func_nesting", File: "pkg/b/b.go", Line: 6, Symbol: "B", Summary: "too complex"},
				},
				Advisory: []QAItem{
					{ID: "a1", Source: "decisions", Check: "doc", File: "pkg/b/b.go", Line: 3, Symbol: "C", Summary: "stale doc"},
					{ID: "a1", Source: "decisions", Check: "doc", File: "pkg/b/b.go", Line: 3, Symbol: "C", Summary: "stale doc"},
				},
				SkippedSources: []SkippedSource{
					{Source: "similarity", Reason: "offline"},
					{Source: "similarity", Reason: "offline"},
					{Source: "similarity", Reason: "no project"},
					{Source: "decisions", Reason: "no key"},
				},
			},
			want: QAReport{
				BaseLabel: "main",
				Speed:     SpeedAll,
				Filter:    FilterAll,
				MustFix: []QAItem{
					{ID: "m1", Source: "astcheck", Check: "func_cognitive", File: "pkg/a/a.go", Line: 5, Symbol: "A", Summary: "different summary"},
					{ID: "m1", Source: "astcheck", Check: "func_cognitive", File: "pkg/a/a.go", Line: 5, Symbol: "A", Summary: "too complex"},
					{ID: "m2", Source: "astcheck", Check: "func_cognitive", File: "pkg/a/a.go", Line: 5, Symbol: "A", Summary: "too complex"},
					{ID: "m2", Source: "astcheck", Check: "func_cognitive", File: "pkg/a/a.go", Line: 5, Symbol: "B", Summary: "too complex"},
					{ID: "m2", Source: "astcheck", Check: "func_nesting", File: "pkg/a/a.go", Line: 5, Symbol: "B", Summary: "too complex"},
					{ID: "m2", Source: "gofmt", Check: "func_nesting", File: "pkg/a/a.go", Line: 5, Symbol: "B", Summary: "too complex"},
					{ID: "m2", Source: "gofmt", Check: "func_nesting", File: "pkg/a/a.go", Line: 6, Symbol: "B", Summary: "too complex"},
					{ID: "m2", Source: "gofmt", Check: "func_nesting", File: "pkg/b/b.go", Line: 6, Symbol: "B", Summary: "too complex"},
				},
				Advisory: []QAItem{
					{ID: "a1", Source: "decisions", Check: "doc", File: "pkg/b/b.go", Line: 3, Symbol: "C", Summary: "stale doc"},
				},
				SkippedSources: []SkippedSource{
					{Source: "decisions", Reason: "no key"},
					{Source: "similarity", Reason: "no project"},
					{Source: "similarity", Reason: "offline"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.report

			got.Normalize()

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("QAReport.Normalize() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
