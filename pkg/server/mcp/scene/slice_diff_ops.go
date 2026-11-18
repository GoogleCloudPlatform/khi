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

import "github.com/GoogleCloudPlatform/khi/pkg/server/workbench"

func buildSelectedLineBitmap(totalLines int, sels []yamlLineSelection, highlights map[int][]byteSpan) []bool {
	if len(sels) == 0 {
		return nil
	}
	selected := make([]bool, totalLines)
	for _, s := range sels {
		lo := clampInt(s.startLine-1, 0, totalLines-1)
		hi := clampInt(s.endLine-1, lo, totalLines-1)
		for line := lo; line <= hi; line++ {
			selected[line] = true
		}
		if s.highlightLine != 0 {
			hlIdx := s.highlightLine - 1
			highlights[hlIdx] = append(highlights[hlIdx], s.highlightSpan)
		}
	}
	return selected
}

func markMatchingDiffOps(ops []workbench.DiffOp, oldLineSet, newLineSet, selectedOp []bool) {
	for i, op := range ops {
		if isDiffOpSelected(op, oldLineSet, newLineSet) {
			selectedOp[i] = true
		}
	}
}

func isDiffOpSelected(op workbench.DiffOp, oldLineSet, newLineSet []bool) bool {
	switch op.Kind {
	case workbench.DiffOpDelete:
		return len(oldLineSet) > 0 && oldLineSet[op.OldPos]
	case workbench.DiffOpInsert:
		return len(newLineSet) > 0 && newLineSet[op.NewPos]
	default:
		if len(oldLineSet) > 0 && len(newLineSet) > 0 {
			return oldLineSet[op.OldPos] && newLineSet[op.NewPos]
		}
		if len(newLineSet) > 0 {
			return newLineSet[op.NewPos]
		}
		return oldLineSet[op.OldPos]
	}
}

// groupSelectedReplacementPairs keeps paired selected '-' and '+' sub-runs adjacent within multi-line replacement blocks
// so unselected sibling changes in the same block fold into leading/interior/trailing omitted counts rather than splitting '-/+'.
func groupSelectedReplacementPairs(ops []workbench.DiffOp, selected []bool) ([]workbench.DiffOp, []bool) {
	outOps := make([]workbench.DiffOp, 0, len(ops))
	outSel := make([]bool, 0, len(selected))

	i := 0
	for i < len(ops) {
		if ops[i].Kind != workbench.DiffOpDelete {
			outOps = append(outOps, ops[i])
			outSel = append(outSel, selected[i])
			i++
			continue
		}
		delStart := i
		for i < len(ops) && ops[i].Kind == workbench.DiffOpDelete {
			i++
		}
		insStart := i
		for i < len(ops) && ops[i].Kind == workbench.DiffOpInsert {
			i++
		}
		if insStart == i {
			outOps = append(outOps, ops[delStart:i]...)
			outSel = append(outSel, selected[delStart:i]...)
			continue
		}
		groupedOps, groupedSel := reorderReplacementBlock(ops[delStart:i], selected[delStart:i], insStart-delStart)
		outOps = append(outOps, groupedOps...)
		outSel = append(outSel, groupedSel...)
	}

	return outOps, outSel
}

func reorderReplacementBlock(blockOps []workbench.DiffOp, blockSel []bool, delCount int) ([]workbench.DiffOp, []bool) {
	delRuns := collectSelectedRuns(blockSel, 0, delCount)
	insRuns := collectSelectedRuns(blockSel, delCount, len(blockOps))
	if len(delRuns) == 0 || len(delRuns) != len(insRuns) {
		return blockOps, blockSel
	}

	outOps := make([]workbench.DiffOp, 0, len(blockOps))
	outSel := make([]bool, 0, len(blockSel))
	appendSlice := func(lo, hi int) {
		outOps = append(outOps, blockOps[lo:hi]...)
		outSel = append(outSel, blockSel[lo:hi]...)
	}

	// Leading unselected '-' and '+' ops before the first selected pair.
	appendSlice(0, delRuns[0].start)
	appendSlice(delCount, insRuns[0].start)

	for k := range delRuns {
		if k > 0 {
			// Interior unselected '-' and '+' ops between pair k-1 and pair k.
			appendSlice(delRuns[k-1].end, delRuns[k].start)
			appendSlice(insRuns[k-1].end, insRuns[k].start)
		}
		appendSlice(delRuns[k].start, delRuns[k].end)
		appendSlice(insRuns[k].start, insRuns[k].end)
	}

	// Trailing unselected '-' and '+' ops after the last selected pair.
	lastK := len(delRuns) - 1
	appendSlice(delRuns[lastK].end, delCount)
	appendSlice(insRuns[lastK].end, len(blockOps))
	return outOps, outSel
}

func collectSelectedRuns(selected []bool, lo, hi int) []lineInterval {
	var runs []lineInterval
	runStart := -1
	for i := lo; i < hi; i++ {
		if selected[i] {
			if runStart < 0 {
				runStart = i
			}
		} else if runStart >= 0 {
			runs = append(runs, lineInterval{start: runStart, end: i})
			runStart = -1
		}
	}
	if runStart >= 0 {
		runs = append(runs, lineInterval{start: runStart, end: hi})
	}
	return runs
}
