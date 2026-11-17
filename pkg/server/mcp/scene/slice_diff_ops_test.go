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

package scene

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

func TestBuildSelectedLineBitmap(t *testing.T) {
	testCases := []struct {
		name           string
		totalLines     int
		sels           []yamlLineSelection
		wantBitmap     []bool
		wantHighlights map[int][]byteSpan
	}{
		{
			name:           "empty selections returns nil",
			totalLines:     3,
			wantHighlights: map[int][]byteSpan{},
		},
		{
			name:       "single line and zero highlightLine skips highlight map",
			totalLines: 3,
			sels: []yamlLineSelection{
				{startLine: 2, endLine: 2, highlightLine: 0},
			},
			wantBitmap:     []bool{false, true, false},
			wantHighlights: map[int][]byteSpan{},
		},
		{
			name:       "records 0-based line range and highlight span",
			totalLines: 4,
			sels: []yamlLineSelection{
				{startLine: 2, endLine: 3, highlightLine: 2, highlightSpan: byteSpan{start: 2, end: 7}},
			},
			wantBitmap: []bool{false, true, true, false},
			wantHighlights: map[int][]byteSpan{
				1: {{start: 2, end: 7}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			highlights := make(map[int][]byteSpan)
			got := buildSelectedLineBitmap(tc.totalLines, tc.sels, highlights)
			if diff := cmp.Diff(tc.wantBitmap, got); diff != "" {
				t.Errorf("bitmap mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantHighlights, highlights, cmp.AllowUnexported(byteSpan{})); diff != "" {
				t.Errorf("highlights mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCollectSelectedRuns(t *testing.T) {
	testCases := []struct {
		name     string
		selected []bool
		lo       int
		hi       int
		want     []lineInterval
	}{
		{
			name:     "run starting at zero and spanning to end",
			selected: []bool{true, true, true},
			lo:       0,
			hi:       3,
			want:     []lineInterval{{start: 0, end: 3}},
		},
		{
			name:     "run starting at zero and closing before end",
			selected: []bool{true, false, true},
			lo:       0,
			hi:       3,
			want:     []lineInterval{{start: 0, end: 1}, {start: 2, end: 3}},
		},
		{
			name:     "no selected elements returns nil",
			selected: []bool{false, false},
			lo:       0,
			hi:       2,
			want:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := collectSelectedRuns(tc.selected, tc.lo, tc.hi)
			if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(lineInterval{})); diff != "" {
				t.Errorf("collectSelectedRuns() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGroupSelectedReplacementPairs(t *testing.T) {
	del := func(pos int, line string) workbench.DiffOp {
		return workbench.DiffOp{Kind: workbench.DiffOpDelete, OldPos: pos, Line: line}
	}
	ins := func(pos int, line string) workbench.DiffOp {
		return workbench.DiffOp{Kind: workbench.DiffOpInsert, NewPos: pos, Line: line}
	}

	type groupedResult struct {
		Ops []workbench.DiffOp
		Sel []bool
	}

	testCases := []struct {
		name     string
		ops      []workbench.DiffOp
		selected []bool
		want     groupedResult
	}{
		{
			name:     "leaves unselected replacement block in original order",
			ops:      []workbench.DiffOp{del(0, "a"), ins(0, "b")},
			selected: []bool{false, false},
			want: groupedResult{
				Ops: []workbench.DiffOp{del(0, "a"), ins(0, "b")},
				Sel: []bool{false, false},
			},
		},
		{
			name:     "leaves mismatched run counts in original order",
			ops:      []workbench.DiffOp{del(0, "a"), del(1, "b"), ins(0, "x"), ins(1, "y")},
			selected: []bool{true, false, false, false},
			want: groupedResult{
				Ops: []workbench.DiffOp{del(0, "a"), del(1, "b"), ins(0, "x"), ins(1, "y")},
				Sel: []bool{true, false, false, false},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotOps, gotSel := groupSelectedReplacementPairs(tc.ops, tc.selected)
			got := groupedResult{Ops: gotOps, Sel: gotSel}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("groupSelectedReplacementPairs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
