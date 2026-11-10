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
)

// UnifiedDiffContextLines is the number of unchanged lines shown around each change, matching `diff -u`.
const UnifiedDiffContextLines = 3

// maxMyersEditDistance bounds the Myers search so that memory stays O(maxMyersEditDistance^2).
// Beyond this bound, the differing middle section is reported as a full replacement, which is still a valid diff.
const maxMyersEditDistance = 1000

// LineChangeCount holds the numbers of added and deleted lines in a diff.
type LineChangeCount struct {
	Added   int
	Deleted int
}

// UnifiedDiff is a unified diff text together with the line change counts over the whole diff.
type UnifiedDiff struct {
	// Text is the unified diff including the `---` and `+++` headers, or empty when both inputs are identical.
	Text    string
	Changes LineChangeCount
}

// DiffOpKind classifies a single line operation in a line-based edit script.
type DiffOpKind int

const (
	// DiffOpEqual represents an unchanged line present in both inputs.
	DiffOpEqual DiffOpKind = iota
	// DiffOpDelete represents a line removed from the old input.
	DiffOpDelete
	// DiffOpInsert represents a line added in the new input.
	DiffOpInsert
)

// DiffOp represents a single line of an edit script.
//
// OldPos and NewPos are the 0-based numbers of old and new lines consumed before this operation.
type DiffOp struct {
	Kind   DiffOpKind
	Line   string
	OldPos int
	NewPos int
}

// computeUnifiedDiff returns the line-based unified diff that transforms oldText into newText.
func computeUnifiedDiff(oldText, newText, oldLabel, newLabel string) UnifiedDiff {
	ops := DiffLines(SplitLines(oldText), SplitLines(newText))
	changes := CountDiffOps(ops)
	if changes.Added == 0 && changes.Deleted == 0 {
		return UnifiedDiff{}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s\n+++ %s\n", oldLabel, newLabel)
	writeUnifiedHunks(&sb, ops)
	return UnifiedDiff{
		Text:    sb.String(),
		Changes: changes,
	}
}

// countLineChanges returns the same change counts as computeUnifiedDiff without rendering the diff text.
func countLineChanges(oldText, newText string) LineChangeCount {
	return CountDiffOps(DiffLines(SplitLines(oldText), SplitLines(newText)))
}

// CountDiffOps totals the added and deleted line counts across ops.
func CountDiffOps(ops []DiffOp) LineChangeCount {
	var changes LineChangeCount
	for _, op := range ops {
		switch op.Kind {
		case DiffOpInsert:
			changes.Added++
		case DiffOpDelete:
			changes.Deleted++
		}
	}
	return changes
}

// SplitLines splits text into lines without their trailing newline characters.
func SplitLines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

// DiffLines returns the shortest line-based edit script from a to b.
//
// It strips the common prefix and suffix first because manifest revisions usually differ only in a few places.
func DiffLines(a, b []string) []DiffOp {
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix && a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}

	ops := make([]DiffOp, 0, len(a)+len(b)-prefix-suffix)
	for i := 0; i < prefix; i++ {
		ops = append(ops, DiffOp{Kind: DiffOpEqual, Line: a[i], OldPos: i, NewPos: i})
	}

	midA := a[prefix : len(a)-suffix]
	midB := b[prefix : len(b)-suffix]
	midOps, ok := myersDiff(midA, midB, maxMyersEditDistance)
	if !ok {
		midOps = fullReplacementOps(midA, midB)
	}
	for _, op := range midOps {
		op.OldPos += prefix
		op.NewPos += prefix
		ops = append(ops, op)
	}

	for i := 0; i < suffix; i++ {
		oldIdx := len(a) - suffix + i
		newIdx := len(b) - suffix + i
		ops = append(ops, DiffOp{Kind: DiffOpEqual, Line: a[oldIdx], OldPos: oldIdx, NewPos: newIdx})
	}
	return ops
}

// fullReplacementOps returns an edit script that deletes every line of a and then inserts every line of b.
func fullReplacementOps(a, b []string) []DiffOp {
	ops := make([]DiffOp, 0, len(a)+len(b))
	for i, line := range a {
		ops = append(ops, DiffOp{Kind: DiffOpDelete, Line: line, OldPos: i, NewPos: 0})
	}
	for i, line := range b {
		ops = append(ops, DiffOp{Kind: DiffOpInsert, Line: line, OldPos: len(a), NewPos: i})
	}
	return ops
}

// myersDiff computes a shortest edit script from a to b with the Myers O(ND) algorithm.
// It returns false when the edit distance exceeds maxD.
func myersDiff(a, b []string, maxD int) ([]DiffOp, bool) {
	n, m := len(a), len(b)
	if n == 0 && m == 0 {
		return nil, true
	}
	limit := min(n+m, maxD)
	offset := limit + 1
	// v[offset+k] holds the furthest x reached on diagonal k (k = x - y).
	v := make([]int, 2*limit+3)
	// trace[d] keeps a copy of v before step d for backtracking.
	trace := make([][]int, 0, limit+1)

	for d := 0; d <= limit; d++ {
		trace = append(trace, append([]int(nil), v[offset-d-1:offset+d+2]...))
		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[offset+k-1] < v[offset+k+1]) {
				x = v[offset+k+1]
			} else {
				x = v[offset+k-1] + 1
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[offset+k] = x
			if x >= n && y >= m {
				return backtrackMyers(a, b, trace), true
			}
		}
	}
	return nil, false
}

// backtrackMyers reconstructs the edit script from the snapshots recorded by myersDiff.
func backtrackMyers(a, b []string, trace [][]int) []DiffOp {
	x, y := len(a), len(b)
	var reversed []DiffOp
	for d := len(trace) - 1; d >= 0; d-- {
		// snapshot[k+d+1] holds v[k] for k in [-d-1, d+1].
		snapshot := trace[d]
		k := x - y
		var prevK int
		if k == -d || (k != d && snapshot[k-1+d+1] < snapshot[k+1+d+1]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}
		prevX := snapshot[prevK+d+1]
		prevY := prevX - prevK
		for x > prevX && y > prevY {
			x--
			y--
			reversed = append(reversed, DiffOp{Kind: DiffOpEqual, Line: a[x], OldPos: x, NewPos: y})
		}
		if d > 0 {
			if x == prevX {
				reversed = append(reversed, DiffOp{Kind: DiffOpInsert, Line: b[prevY], OldPos: prevX, NewPos: prevY})
			} else {
				reversed = append(reversed, DiffOp{Kind: DiffOpDelete, Line: a[prevX], OldPos: prevX, NewPos: prevY})
			}
		}
		x, y = prevX, prevY
	}

	ops := make([]DiffOp, len(reversed))
	for i, op := range reversed {
		ops[len(reversed)-1-i] = op
	}
	return ops
}

// writeUnifiedHunks writes `@@` hunks for ops, merging changes whose unchanged gap fits in the shared context.
func writeUnifiedHunks(sb *strings.Builder, ops []DiffOp) {
	i := 0
	for i < len(ops) {
		if ops[i].Kind == DiffOpEqual {
			i++
			continue
		}
		start := max(0, i-UnifiedDiffContextLines)
		lastChange := i
		for j := i + 1; j < len(ops) && j-lastChange-1 <= 2*UnifiedDiffContextLines; j++ {
			if ops[j].Kind != DiffOpEqual {
				lastChange = j
			}
		}
		end := min(len(ops), lastChange+UnifiedDiffContextLines+1)
		writeHunk(sb, ops[start:end])
		i = end
	}
}

func writeHunk(sb *strings.Builder, hunk []DiffOp) {
	var oldCount, newCount int
	for _, op := range hunk {
		if op.Kind != DiffOpInsert {
			oldCount++
		}
		if op.Kind != DiffOpDelete {
			newCount++
		}
	}
	fmt.Fprintf(sb, "@@ -%s +%s @@\n",
		formatHunkRange(hunk[0].OldPos, oldCount),
		formatHunkRange(hunk[0].NewPos, newCount),
	)
	for _, op := range hunk {
		switch op.Kind {
		case DiffOpEqual:
			sb.WriteString(" ")
		case DiffOpDelete:
			sb.WriteString("-")
		case DiffOpInsert:
			sb.WriteString("+")
		}
		sb.WriteString(op.Line)
		sb.WriteString("\n")
	}
}

// formatHunkRange formats a hunk range like GNU diff: the 1-based start line, followed by the count unless it is 1.
// An empty range starts at the line before the position, so inserting into an empty file yields "0,0".
func formatHunkRange(pos, count int) string {
	switch count {
	case 0:
		return fmt.Sprintf("%d,0", pos)
	case 1:
		return fmt.Sprintf("%d", pos+1)
	default:
		return fmt.Sprintf("%d,%d", pos+1, count)
	}
}
