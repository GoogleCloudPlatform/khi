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

package workbench

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func numberedLines(prefix string, count int, replace map[int]string) string {
	var sb strings.Builder
	for i := 1; i <= count; i++ {
		if line, ok := replace[i]; ok {
			sb.WriteString(line)
		} else {
			sb.WriteString(prefix)
			sb.WriteString(strings.Repeat("x", i))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// fullyReplacedMiddle returns texts whose n middle lines all differ between a shared first and last line,
// together with the hunk expected when the middle is reported as a full replacement.
func fullyReplacedMiddle(n int) (oldText, newText, wantHunk string) {
	var oldLines, newLines, deletes, inserts strings.Builder
	for i := range n {
		fmt.Fprintf(&oldLines, "old%04d\n", i)
		fmt.Fprintf(&newLines, "new%04d\n", i)
		fmt.Fprintf(&deletes, "-old%04d\n", i)
		fmt.Fprintf(&inserts, "+new%04d\n", i)
	}
	oldText = "head\n" + oldLines.String() + "tail\n"
	newText = "head\n" + newLines.String() + "tail\n"
	wantHunk = fmt.Sprintf("@@ -1,%d +1,%d @@\n head\n", n+2, n+2) + deletes.String() + inserts.String() + " tail\n"
	return oldText, newText, wantHunk
}

func TestComputeUnifiedDiff(t *testing.T) {
	// Replacing this many lines needs an edit distance of twice the count, beyond maxMyersEditDistance.
	replacedLineCount := maxMyersEditDistance/2 + 100
	replacedOld, replacedNew, replacedHunk := fullyReplacedMiddle(replacedLineCount)
	testCases := []struct {
		name        string
		oldText     string
		newText     string
		oldLabel    string
		newLabel    string
		wantText    string
		wantChanges LineChangeCount
	}{
		{
			name:     "identical inputs produce an empty diff",
			oldText:  "a\nb\n",
			newText:  "a\nb\n",
			oldLabel: "revision 0",
			newLabel: "revision 1",
		},
		{
			name:        "empty old text reports every line as added",
			oldText:     "",
			newText:     "a\nb\nc\n",
			oldLabel:    "none",
			newLabel:    "revision 0",
			wantText:    "--- none\n+++ revision 0\n@@ -0,0 +1,3 @@\n+a\n+b\n+c\n",
			wantChanges: LineChangeCount{Added: 3},
		},
		{
			name:        "empty new text reports every line as deleted",
			oldText:     "a\nb\n",
			newText:     "",
			oldLabel:    "revision 0",
			newLabel:    "revision 1",
			wantText:    "--- revision 0\n+++ revision 1\n@@ -1,2 +0,0 @@\n-a\n-b\n",
			wantChanges: LineChangeCount{Deleted: 2},
		},
		{
			name:        "single line ranges omit the count",
			oldText:     "a\n",
			newText:     "b\n",
			oldLabel:    "revision 0",
			newLabel:    "revision 1",
			wantText:    "--- revision 0\n+++ revision 1\n@@ -1 +1 @@\n-a\n+b\n",
			wantChanges: LineChangeCount{Added: 1, Deleted: 1},
		},
		{
			name:        "missing trailing newline is treated like a terminated last line",
			oldText:     "a\nb",
			newText:     "a\nb\nc\n",
			oldLabel:    "revision 0",
			newLabel:    "revision 1",
			wantText:    "--- revision 0\n+++ revision 1\n@@ -1,2 +1,3 @@\n a\n b\n+c\n",
			wantChanges: LineChangeCount{Added: 1},
		},
		{
			name:     "change in the middle shows three context lines on each side",
			oldText:  numberedLines("l", 10, nil),
			newText:  numberedLines("l", 10, map[int]string{5: "changed"}),
			oldLabel: "revision 0",
			newLabel: "revision 1",
			wantText: "--- revision 0\n+++ revision 1\n@@ -2,7 +2,7 @@\n" +
				" lxx\n lxxx\n lxxxx\n-lxxxxx\n+changed\n lxxxxxx\n lxxxxxxx\n lxxxxxxxx\n",
			wantChanges: LineChangeCount{Added: 1, Deleted: 1},
		},
		{
			name:     "changes separated by more than six unchanged lines produce separate hunks",
			oldText:  numberedLines("l", 20, nil),
			newText:  numberedLines("l", 20, map[int]string{2: "A", 18: "B"}),
			oldLabel: "revision 0",
			newLabel: "revision 1",
			wantText: "--- revision 0\n+++ revision 1\n" +
				"@@ -1,5 +1,5 @@\n lx\n-lxx\n+A\n lxxx\n lxxxx\n lxxxxx\n" +
				"@@ -15,6 +15,6 @@\n" +
				" l" + strings.Repeat("x", 15) + "\n" +
				" l" + strings.Repeat("x", 16) + "\n" +
				" l" + strings.Repeat("x", 17) + "\n" +
				"-l" + strings.Repeat("x", 18) + "\n" +
				"+B\n" +
				" l" + strings.Repeat("x", 19) + "\n" +
				" l" + strings.Repeat("x", 20) + "\n",
			wantChanges: LineChangeCount{Added: 2, Deleted: 2},
		},
		{
			name:     "changes separated by exactly six unchanged lines share one hunk",
			oldText:  numberedLines("l", 9, nil),
			newText:  numberedLines("l", 9, map[int]string{1: "A", 8: "B"}),
			oldLabel: "revision 0",
			newLabel: "revision 1",
			wantText: "--- revision 0\n+++ revision 1\n@@ -1,9 +1,9 @@\n" +
				"-lx\n+A\n lxx\n lxxx\n lxxxx\n lxxxxx\n lxxxxxx\n lxxxxxxx\n-lxxxxxxxx\n+B\n lxxxxxxxxx\n",
			wantChanges: LineChangeCount{Added: 2, Deleted: 2},
		},
		{
			name:     "changes separated by exactly seven unchanged lines produce separate hunks",
			oldText:  numberedLines("l", 12, nil),
			newText:  numberedLines("l", 12, map[int]string{1: "A", 9: "B"}),
			oldLabel: "revision 0",
			newLabel: "revision 1",
			wantText: "--- revision 0\n+++ revision 1\n" +
				"@@ -1,4 +1,4 @@\n-lx\n+A\n lxx\n lxxx\n lxxxx\n" +
				"@@ -6,7 +6,7 @@\n" +
				" l" + strings.Repeat("x", 6) + "\n" +
				" l" + strings.Repeat("x", 7) + "\n" +
				" l" + strings.Repeat("x", 8) + "\n" +
				"-l" + strings.Repeat("x", 9) + "\n" +
				"+B\n" +
				" l" + strings.Repeat("x", 10) + "\n" +
				" l" + strings.Repeat("x", 11) + "\n" +
				" l" + strings.Repeat("x", 12) + "\n",
			wantChanges: LineChangeCount{Added: 2, Deleted: 2},
		},
		{
			name:        "edit distance beyond maxMyersEditDistance reports the differing middle as a full replacement",
			oldText:     replacedOld,
			newText:     replacedNew,
			oldLabel:    "revision 0",
			newLabel:    "revision 1",
			wantText:    "--- revision 0\n+++ revision 1\n" + replacedHunk,
			wantChanges: LineChangeCount{Added: replacedLineCount, Deleted: replacedLineCount},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeUnifiedDiff(tc.oldText, tc.newText, tc.oldLabel, tc.newLabel)
			if diff := cmp.Diff(tc.wantText, got.Text); diff != "" {
				t.Errorf("computeUnifiedDiff() text mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantChanges, got.Changes); diff != "" {
				t.Errorf("computeUnifiedDiff() changes mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantChanges, countLineChanges(tc.oldText, tc.newText)); diff != "" {
				t.Errorf("countLineChanges() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDiffLines(t *testing.T) {
	testCases := []struct {
		name        string
		a           []string
		b           []string
		wantChanges LineChangeCount
	}{
		{
			name:        "interleaved edits find the shortest edit script",
			a:           []string{"a", "b", "c", "a", "b", "b", "a"},
			b:           []string{"c", "b", "a", "b", "a", "c"},
			wantChanges: LineChangeCount{Added: 2, Deleted: 3},
		},
		{
			name:        "completely different inputs replace every line",
			a:           []string{"a", "b"},
			b:           []string{"c", "d", "e"},
			wantChanges: LineChangeCount{Added: 3, Deleted: 2},
		},
		{
			name:        "common prefix and suffix are kept as unchanged lines",
			a:           []string{"p", "x", "s"},
			b:           []string{"p", "y", "z", "s"},
			wantChanges: LineChangeCount{Added: 2, Deleted: 1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ops := diffLines(tc.a, tc.b)
			var gotOld, gotNew []string
			for _, op := range ops {
				if op.kind != diffOpInsert {
					gotOld = append(gotOld, op.line)
				}
				if op.kind != diffOpDelete {
					gotNew = append(gotNew, op.line)
				}
			}
			if diff := cmp.Diff(tc.a, gotOld); diff != "" {
				t.Errorf("diffLines() old side mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.b, gotNew); diff != "" {
				t.Errorf("diffLines() new side mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantChanges, countOps(ops)); diff != "" {
				t.Errorf("diffLines() changes mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMyersDiff_ExceedsMaxEditDistance(t *testing.T) {
	testCases := []struct {
		name   string
		a      []string
		b      []string
		maxD   int
		wantOK bool
	}{
		{
			name:   "edit distance within the bound succeeds",
			a:      []string{"a", "b"},
			b:      []string{"a", "c"},
			maxD:   2,
			wantOK: true,
		},
		{
			name:   "edit distance beyond the bound gives up",
			a:      []string{"a", "b"},
			b:      []string{"c", "d"},
			maxD:   3,
			wantOK: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, gotOK := myersDiff(tc.a, tc.b, tc.maxD)
			if gotOK != tc.wantOK {
				t.Errorf("myersDiff() ok = %v, want %v", gotOK, tc.wantOK)
			}
		})
	}
}
