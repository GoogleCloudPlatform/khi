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
	"context"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var _ AdvisoryProvider = (*stubAdvisoryProvider)(nil)

type speedOutcome struct {
	Speed        Speed
	IncludesFast bool
	IncludesSlow bool
	HasErr       bool
}

type filterOutcome struct {
	Filter           Filter
	IncludesMust     bool
	IncludesAdvisory bool
	HasErr           bool
}

type advisoryEvalOutcome struct {
	Items  []QAItem
	Skip   *SkippedSource
	HasErr bool
}

type stubAdvisoryProvider struct {
	name  string
	speed Speed
	items []QAItem
	skip  *SkippedSource
	err   error
	seen  []AdvisoryRequest
}

func (p *stubAdvisoryProvider) Name() string {
	return p.name
}

func (p *stubAdvisoryProvider) Speed() Speed {
	return p.speed
}

func (p *stubAdvisoryProvider) Evaluate(_ context.Context, req AdvisoryRequest) ([]QAItem, *SkippedSource, error) {
	p.seen = append(p.seen, req)
	return p.items, p.skip, p.err
}

func TestSpeedConstantsAndParsing(t *testing.T) {
	testCases := []struct {
		name string
		raw  string
		want speedOutcome
	}{
		// Parses fast speed and includes only fast checks.
		{
			name: "fast speed",
			raw:  "fast",
			want: speedOutcome{
				Speed:        SpeedFast,
				IncludesFast: true,
				IncludesSlow: false,
				HasErr:       false,
			},
		},
		// Parses slow speed and includes only slow checks.
		{
			name: "slow speed",
			raw:  "slow",
			want: speedOutcome{
				Speed:        SpeedSlow,
				IncludesFast: false,
				IncludesSlow: true,
				HasErr:       false,
			},
		},
		// Parses all speed and includes both fast and slow checks.
		{
			name: "all speed",
			raw:  "all",
			want: speedOutcome{
				Speed:        SpeedAll,
				IncludesFast: true,
				IncludesSlow: true,
				HasErr:       false,
			},
		},
		// Rejects an unknown speed value and reports neither fast nor slow.
		{
			name: "invalid speed",
			raw:  "medium",
			want: speedOutcome{
				Speed:        "",
				IncludesFast: false,
				IncludesSlow: false,
				HasErr:       true,
			},
		},
	}

	if SpeedFast != "fast" || SpeedSlow != "slow" || SpeedAll != "all" {
		t.Fatalf("Speed constants = (%q, %q, %q), want (fast, slow, all)", SpeedFast, SpeedSlow, SpeedAll)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsedSpeed, err := ParseSpeed(tc.raw)

			got := speedOutcome{
				Speed:        parsedSpeed,
				IncludesFast: parsedSpeed.IncludesFast(),
				IncludesSlow: parsedSpeed.IncludesSlow(),
				HasErr:       err != nil,
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParseSpeed(%q) mismatch (-want +got):\n%s", tc.raw, diff)
			}
		})
	}
}

func TestFilterConstantsAndParsing(t *testing.T) {
	testCases := []struct {
		name string
		raw  string
		want filterOutcome
	}{
		// Parses all filter and includes both must and advisory items.
		{
			name: "all filter",
			raw:  "all",
			want: filterOutcome{
				Filter:           FilterAll,
				IncludesMust:     true,
				IncludesAdvisory: true,
				HasErr:           false,
			},
		},
		// Parses must filter and includes only must items.
		{
			name: "must filter",
			raw:  "must",
			want: filterOutcome{
				Filter:           FilterMust,
				IncludesMust:     true,
				IncludesAdvisory: false,
				HasErr:           false,
			},
		},
		// Parses advisory filter and includes only advisory items.
		{
			name: "advisory filter",
			raw:  "advisory",
			want: filterOutcome{
				Filter:           FilterAdvisory,
				IncludesMust:     false,
				IncludesAdvisory: true,
				HasErr:           false,
			},
		},
		// Rejects an unknown filter value and includes neither category.
		{
			name: "invalid filter",
			raw:  "warnings",
			want: filterOutcome{
				Filter:           "",
				IncludesMust:     false,
				IncludesAdvisory: false,
				HasErr:           true,
			},
		},
	}

	if FilterAll != "all" || FilterMust != "must" || FilterAdvisory != "advisory" {
		t.Fatalf("Filter constants = (%q, %q, %q), want (all, must, advisory)", FilterAll, FilterMust, FilterAdvisory)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsedFilter, err := ParseFilter(tc.raw)

			got := filterOutcome{
				Filter:           parsedFilter,
				IncludesMust:     parsedFilter.IncludesMust(),
				IncludesAdvisory: parsedFilter.IncludesAdvisory(),
				HasErr:           err != nil,
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParseFilter(%q) mismatch (-want +got):\n%s", tc.raw, diff)
			}
		})
	}
}

func TestQAReportHasMustFix(t *testing.T) {
	testCases := []struct {
		name   string
		report QAReport
		want   bool
	}{
		// Returns false for an empty report without items.
		{
			name:   "empty report",
			report: QAReport{},
			want:   false,
		},
		// Returns false when only advisory items and skipped sources exist.
		{
			name: "advisory and skipped sources only",
			report: QAReport{
				Advisory:       []QAItem{{ID: "adv-1", Source: "decisions", Summary: "check doc"}},
				SkippedSources: []SkippedSource{{Source: "similarity", Reason: "no project"}},
			},
			want: false,
		},
		// Returns true when at least one MustFix item is present.
		{
			name: "contains mustfix item",
			report: QAReport{
				MustFix: []QAItem{{ID: "must-1", Source: "astcheck", Summary: "cognitive too high"}},
			},
			want: true,
		},
	}

	for _, tc := range testCases {
		got := tc.report.HasMustFix()

		if got != tc.want {
			t.Errorf("%s: QAReport.HasMustFix() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestQAReportNormalize(t *testing.T) {
	testCases := []struct {
		name   string
		report QAReport
		want   QAReport
	}{
		// Replaces nil slices with initialized non-nil empty slices while preserving metadata.
		{
			name: "nil slices become empty slices",
			report: QAReport{
				BaseLabel: "main",
				Speed:     SpeedFast,
				Filter:    FilterMust,
			},
			want: QAReport{
				BaseLabel:      "main",
				Speed:          SpeedFast,
				Filter:         FilterMust,
				MustFix:        []QAItem{},
				Advisory:       []QAItem{},
				SkippedSources: []SkippedSource{},
			},
		},
		// Deterministically orders MustFix items, Advisory items, and SkippedSources.
		{
			name: "sorts items and skipped sources deterministically",
			report: QAReport{
				BaseLabel: "@-",
				Speed:     SpeedAll,
				Filter:    FilterAll,
				MustFix: []QAItem{
					{ID: "m3", Source: "astcheck", Check: "func_cognitive", File: "pkg/b/b.go", Line: 1, Symbol: "B"},
					{ID: "m2", Source: "astcheck", Check: "func_cognitive", File: "pkg/a/a.go", Line: 20, Symbol: "A2"},
					{ID: "m1", Source: "astcheck", Check: "func_cognitive", File: "pkg/a/a.go", Line: 5, Symbol: "A1"},
				},
				Advisory: []QAItem{
					{ID: "a2", Source: "decisions", Check: "doc", File: "pkg/z/z.go", Line: 12, Symbol: "Z"},
					{ID: "a1", Source: "decisions", Check: "doc", File: "pkg/m/m.go", Line: 4, Symbol: "M"},
				},
				SkippedSources: []SkippedSource{
					{Source: "similarity", Reason: "missing project"},
					{Source: "decisions", Reason: "missing credentials"},
				},
			},
			want: QAReport{
				BaseLabel: "@-",
				Speed:     SpeedAll,
				Filter:    FilterAll,
				MustFix: []QAItem{
					{ID: "m1", Source: "astcheck", Check: "func_cognitive", File: "pkg/a/a.go", Line: 5, Symbol: "A1"},
					{ID: "m2", Source: "astcheck", Check: "func_cognitive", File: "pkg/a/a.go", Line: 20, Symbol: "A2"},
					{ID: "m3", Source: "astcheck", Check: "func_cognitive", File: "pkg/b/b.go", Line: 1, Symbol: "B"},
				},
				Advisory: []QAItem{
					{ID: "a1", Source: "decisions", Check: "doc", File: "pkg/m/m.go", Line: 4, Symbol: "M"},
					{ID: "a2", Source: "decisions", Check: "doc", File: "pkg/z/z.go", Line: 12, Symbol: "Z"},
				},
				SkippedSources: []SkippedSource{
					{Source: "decisions", Reason: "missing credentials"},
					{Source: "similarity", Reason: "missing project"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			normalized := tc.report

			normalized.Normalize()

			if diff := cmp.Diff(tc.want, normalized); diff != "" {
				t.Errorf("QAReport.Normalize() mismatch (-want +got):\n%s", diff)
			}
			reversed := reverseQAReportSlices(tc.report)
			reversed.Normalize()
			if diff := cmp.Diff(tc.want, reversed); diff != "" {
				t.Errorf("QAReport.Normalize() permutation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func reverseQAReportSlices(rep QAReport) QAReport {
	out := rep
	if rep.MustFix != nil {
		out.MustFix = slices.Clone(rep.MustFix)
		slices.Reverse(out.MustFix)
	}
	if rep.Advisory != nil {
		out.Advisory = slices.Clone(rep.Advisory)
		slices.Reverse(out.Advisory)
	}
	if rep.SkippedSources != nil {
		out.SkippedSources = slices.Clone(rep.SkippedSources)
		slices.Reverse(out.SkippedSources)
	}
	return out
}

func TestAdvisoryProviderContract(t *testing.T) {
	sampleReq := AdvisoryRequest{
		RepoRoot:     "/repo",
		TrustedRoot:  "/trusted",
		ChangedFiles: []string{"pkg/a/a.go"},
		DeletedFiles: []string{"pkg/a/old.go"},
		ChangedLines: map[string]map[int]struct{}{
			"pkg/a/a.go": {10: {}},
		},
	}
	sampleItem := QAItem{
		ID:           "adv-1",
		Source:       "diff_advisory",
		Check:        "comment_mismatch",
		File:         "pkg/a/a.go",
		Line:         10,
		Symbol:       "Process",
		Summary:      "comment mentions mutex without field",
		Detail:       "Process GoDoc mentions sync.Mutex.",
		Evidence:     map[string]any{"score": 0.9},
		ReviewPrompt: "Verify synchronization comment on Process.",
	}
	testCases := []struct {
		name     string
		provider *stubAdvisoryProvider
		wantName string
		wantSpd  Speed
		wantOut  advisoryEvalOutcome
		wantReq  []AdvisoryRequest
	}{
		// Evaluates AdvisoryRequest with plain Go fields and returns populated QAItem entries.
		{
			name: "returns advisory items",
			provider: &stubAdvisoryProvider{
				name:  "diff_advisory",
				speed: SpeedFast,
				items: []QAItem{sampleItem},
			},
			wantName: "diff_advisory",
			wantSpd:  SpeedFast,
			wantOut: advisoryEvalOutcome{
				Items: []QAItem{sampleItem},
			},
			wantReq: []AdvisoryRequest{sampleReq},
		},
		// Returns a structured SkippedSource when the advisory provider cannot run.
		{
			name: "returns skipped source",
			provider: &stubAdvisoryProvider{
				name:  "remote_advisory",
				speed: SpeedSlow,
				skip:  &SkippedSource{Source: "remote_advisory", Reason: "missing credentials"},
			},
			wantName: "remote_advisory",
			wantSpd:  SpeedSlow,
			wantOut: advisoryEvalOutcome{
				Skip: &SkippedSource{Source: "remote_advisory", Reason: "missing credentials"},
			},
			wantReq: []AdvisoryRequest{sampleReq},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var provider AdvisoryProvider = tc.provider

			gotItems, gotSkip, err := provider.Evaluate(context.Background(), sampleReq)

			if provider.Name() != tc.wantName {
				t.Errorf("AdvisoryProvider.Name() = %q, want %q", provider.Name(), tc.wantName)
			}
			if provider.Speed() != tc.wantSpd {
				t.Errorf("AdvisoryProvider.Speed() = %q, want %q", provider.Speed(), tc.wantSpd)
			}
			gotOut := advisoryEvalOutcome{
				Items:  gotItems,
				Skip:   gotSkip,
				HasErr: err != nil,
			}
			if diff := cmp.Diff(tc.wantOut, gotOut); diff != "" {
				t.Errorf("AdvisoryProvider.Evaluate() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantReq, tc.provider.seen); diff != "" {
				t.Errorf("AdvisoryRequest mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestRulePackageDoesNotImportGotestutil verifies that rule production files have zero dependency on gotestutil.
func TestRulePackageDoesNotImportGotestutil(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("os.ReadDir(.) error: %v", err)
	}

	var gotestutilImports []string
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		parsedFile, parseErr := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if parseErr != nil {
			t.Fatalf("parser.ParseFile(%q) error: %v", name, parseErr)
		}
		for _, imp := range parsedFile.Imports {
			importPath, unquoteErr := strconv.Unquote(imp.Path.Value)
			if unquoteErr != nil {
				t.Fatalf("strconv.Unquote(%q) error: %v", imp.Path.Value, unquoteErr)
			}
			if strings.Contains(importPath, "gotestutil") {
				gotestutilImports = append(gotestutilImports, name+":"+importPath)
			}
		}
	}
	slices.Sort(gotestutilImports)

	if len(gotestutilImports) > 0 {
		t.Errorf("rule package unexpectedly imports gotestutil: %v", gotestutilImports)
	}
}
