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

package simcheck

import (
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
)

func TestNormalizeL2AndCosine(t *testing.T) {
	testCases := []struct {
		name string
		a    []float32
		b    []float32
		want float64
	}{
		{
			name: "identical direction with different magnitudes",
			a:    []float32{3, 4},
			b:    []float32{6, 8},
			want: 1.0,
		},
		{
			name: "orthogonal vectors",
			a:    []float32{1, 0},
			b:    []float32{0, 5},
			want: 0.0,
		},
		{
			name: "zero vector returns zero similarity",
			a:    []float32{0, 0},
			b:    []float32{1, 1},
			want: 0.0,
		},
		{
			name: "empty vector returns zero similarity",
			a:    nil,
			b:    nil,
			want: 0.0,
		},
		{
			name: "mismatched lengths returns zero similarity",
			a:    []float32{1, 0},
			b:    []float32{1, 0, 0},
			want: 0.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := CosineNormalized(NormalizeL2(tc.a), NormalizeL2(tc.b))
			if math.Abs(got-tc.want) > 1e-5 {
				t.Errorf("CosineNormalized() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCosineClamping(t *testing.T) {
	if got := CosineNormalized([]float32{2}, []float32{2}); got != 1.0 {
		t.Errorf("CosineNormalized([2],[2]) = %v, want 1.0", got)
	}
	if got := CosineNormalized([]float32{2}, []float32{-2}); got != -1.0 {
		t.Errorf("CosineNormalized([2],[-2]) = %v, want -1.0", got)
	}
}

func TestUnitContainsAnyLine(t *testing.T) {
	unit := Unit{StartLine: 10, EndLine: 20}
	testCases := []struct {
		name  string
		lines map[int]struct{}
		want  bool
	}{
		{
			name:  "line just before start is outside",
			lines: map[int]struct{}{9: {}},
			want:  false,
		},
		{
			name:  "exact start line is inside",
			lines: map[int]struct{}{10: {}},
			want:  true,
		},
		{
			name:  "exact end line is inside",
			lines: map[int]struct{}{20: {}},
			want:  true,
		},
		{
			name:  "line just after end is outside",
			lines: map[int]struct{}{21: {}},
			want:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := unit.ContainsAnyLine(tc.lines)
			if got != tc.want {
				t.Errorf("ContainsAnyLine(%v) = %v, want %v", tc.lines, got, tc.want)
			}
		})
	}
}

func TestBuildPreviewLinesAndToSummary(t *testing.T) {
	decl := "l1\n\tl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\nl10\n"
	gotPreview := BuildPreviewLines(decl, 10)
	wantPreview := []rule.SourceLine{
		{Number: 10, Text: "l1"},
		{Number: 11, Text: "    l2"},
		{Number: 12, Text: "l3"},
		{Number: 13, Text: "l4"},
		{Number: 14, Text: "l5"},
		{Number: 15, Text: "l6"},
		{Number: 16, Text: "l7"},
		{Number: 17, Text: "l8"},
	}
	if diff := cmp.Diff(wantPreview, gotPreview); diff != "" {
		t.Errorf("BuildPreviewLines() mismatch (-want +got):\n%s", diff)
	}

	var nilRes *Result
	if diff := cmp.Diff(rule.SimilaritySummary{}, nilRes.ToSummary()); diff != "" {
		t.Errorf("nil ToSummary() mismatch (-want +got):\n%s", diff)
	}

	emptyRes := &Result{NewUnitsEvaluated: 0}
	if diff := cmp.Diff(rule.SimilaritySummary{}, emptyRes.ToSummary()); diff != "" {
		t.Errorf("zero-unit ToSummary() mismatch (-want +got):\n%s", diff)
	}

	res := &Result{
		NewUnitsEvaluated: 1,
		CorpusUnitsCount:  20,
		Advisories: []Advisory{{
			Index:   1,
			NewUnit: Unit{FilePath: "pkg/a/a.go", StartLine: 5, EndLine: 15, Name: "NewFn"},
			Candidates: []CandidateMatch{{
				Unit:         Unit{FilePath: "pkg/common/c.go", StartLine: 10, EndLine: 17, Name: "SharedFn", Preview: gotPreview[:2]},
				Score:        0.955,
				SharedModule: true,
				Guidance:     "reuse SharedFn",
			}},
		}},
	}
	wantSummary := rule.SimilaritySummary{
		Enabled:           true,
		NewUnitsEvaluated: 1,
		CorpusUnitsCount:  20,
		Advisories: []rule.SimilarityAdvisory{{
			Index:   1,
			NewUnit: rule.SimilarityUnitRef{FilePath: "pkg/a/a.go", StartLine: 5, EndLine: 15, Name: "NewFn"},
			Candidates: []rule.SimilarityCandidate{{
				Unit:         rule.SimilarityUnitRef{FilePath: "pkg/common/c.go", StartLine: 10, EndLine: 17, Name: "SharedFn", Preview: wantPreview[:2]},
				Score:        0.955,
				SharedModule: true,
				Guidance:     "reuse SharedFn",
			}},
		}},
	}
	if diff := cmp.Diff(wantSummary, res.ToSummary()); diff != "" {
		t.Errorf("ToSummary() mismatch (-want +got):\n%s", diff)
	}
}
