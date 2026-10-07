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

// unifiedDiffContextLines is the number of unchanged lines shown around each change, matching `diff -u`.
const unifiedDiffContextLines = 3

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

type diffOpKind int

const (
	diffOpEqual diffOpKind = iota
	diffOpDelete
	diffOpInsert
)

// diffOp is a single line of an edit script.
// oldPos and newPos are the numbers of old and new lines consumed before this op.
type diffOp struct {
	kind   diffOpKind
	line   string
	oldPos int
	newPos int
}

// computeUnifiedDiff returns the line-based unified diff that transforms oldText into newText.
func computeUnifiedDiff(oldText, newText, oldLabel, newLabel string) UnifiedDiff {
	ops := diffLines(splitLines(oldText), splitLines(newText))
	changes := countOps(ops)
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
	return countOps(diffLines(splitLines(oldText), splitLines(newText)))
}

func countOps(ops []diffOp) LineChangeCount {
	var changes LineChangeCount
	for _, op := range ops {
		switch op.kind {
		case diffOpInsert:
			changes.Added++
		case diffOpDelete:
			changes.Deleted++
		}
	}
	return changes
}

// splitLines splits text into lines without their trailing newline characters.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

// diffLines returns the edit script from a to b.
// It strips the common prefix and suffix first because manifest revisions usually differ only in a few places.
func diffLines(a, b []string) []diffOp {
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix && a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}

	ops := make([]diffOp, 0, len(a)+len(b)-prefix-suffix)
	for i := 0; i < prefix; i++ {
		ops = append(ops, diffOp{kind: diffOpEqual, line: a[i], oldPos: i, newPos: i})
	}

	midA := a[prefix : len(a)-suffix]
	midB := b[prefix : len(b)-suffix]
	midOps, ok := myersDiff(midA, midB, maxMyersEditDistance)
	if !ok {
		midOps = replaceAll(midA, midB)
	}
	for _, op := range midOps {
		op.oldPos += prefix
		op.newPos += prefix
		ops = append(ops, op)
	}

	for i := 0; i < suffix; i++ {
		oldIdx := len(a) - suffix + i
		newIdx := len(b) - suffix + i
		ops = append(ops, diffOp{kind: diffOpEqual, line: a[oldIdx], oldPos: oldIdx, newPos: newIdx})
	}
	return ops
}

// replaceAll returns an edit script that deletes every line of a and then inserts every line of b.
func replaceAll(a, b []string) []diffOp {
	ops := make([]diffOp, 0, len(a)+len(b))
	for i, line := range a {
		ops = append(ops, diffOp{kind: diffOpDelete, line: line, oldPos: i, newPos: 0})
	}
	for i, line := range b {
		ops = append(ops, diffOp{kind: diffOpInsert, line: line, oldPos: len(a), newPos: i})
	}
	return ops
}

// myersDiff computes a shortest edit script from a to b with the Myers O(ND) algorithm.
// It returns false when the edit distance exceeds maxD.
func myersDiff(a, b []string, maxD int) ([]diffOp, bool) {
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
func backtrackMyers(a, b []string, trace [][]int) []diffOp {
	x, y := len(a), len(b)
	var reversed []diffOp
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
			reversed = append(reversed, diffOp{kind: diffOpEqual, line: a[x], oldPos: x, newPos: y})
		}
		if d > 0 {
			if x == prevX {
				reversed = append(reversed, diffOp{kind: diffOpInsert, line: b[prevY], oldPos: prevX, newPos: prevY})
			} else {
				reversed = append(reversed, diffOp{kind: diffOpDelete, line: a[prevX], oldPos: prevX, newPos: prevY})
			}
		}
		x, y = prevX, prevY
	}

	ops := make([]diffOp, len(reversed))
	for i, op := range reversed {
		ops[len(reversed)-1-i] = op
	}
	return ops
}

// writeUnifiedHunks writes `@@` hunks for ops, merging changes whose unchanged gap fits in the shared context.
func writeUnifiedHunks(sb *strings.Builder, ops []diffOp) {
	i := 0
	for i < len(ops) {
		if ops[i].kind == diffOpEqual {
			i++
			continue
		}
		start := max(0, i-unifiedDiffContextLines)
		lastChange := i
		for j := i + 1; j < len(ops) && j-lastChange-1 <= 2*unifiedDiffContextLines; j++ {
			if ops[j].kind != diffOpEqual {
				lastChange = j
			}
		}
		end := min(len(ops), lastChange+unifiedDiffContextLines+1)
		writeHunk(sb, ops[start:end])
		i = end
	}
}

func writeHunk(sb *strings.Builder, hunk []diffOp) {
	var oldCount, newCount int
	for _, op := range hunk {
		if op.kind != diffOpInsert {
			oldCount++
		}
		if op.kind != diffOpDelete {
			newCount++
		}
	}
	fmt.Fprintf(sb, "@@ -%s +%s @@\n",
		formatHunkRange(hunk[0].oldPos, oldCount),
		formatHunkRange(hunk[0].newPos, newCount),
	)
	for _, op := range hunk {
		switch op.kind {
		case diffOpEqual:
			sb.WriteString(" ")
		case diffOpDelete:
			sb.WriteString("-")
		case diffOpInsert:
			sb.WriteString("+")
		}
		sb.WriteString(op.line)
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
