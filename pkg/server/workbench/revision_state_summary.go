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
	"slices"
	"sort"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
)

const (
	// maxStateCycleLen caps the cycle period considered during tandem repeat folding.
	maxStateCycleLen = 8
	// maxDisplayedStateSlots is the maximum number of state labels shown across compressed steps before middle elision.
	maxDisplayedStateSlots = 14
	// truncatedPrefixStateSlots is the state label budget reserved for leading steps when middle elision is applied.
	truncatedPrefixStateSlots = 8
	// truncatedSuffixStateSlots is the state label budget reserved for trailing steps when middle elision is applied.
	truncatedSuffixStateSlots = 4
	// maxOmittedDistinctStates caps distinct unseen middle state labels listed in a truncated summary.
	maxOmittedDistinctStates = 5
)

// RevisionStateStep represents either a single state or a consecutively repeated cycle of states.
type RevisionStateStep struct {
	// States holds one state label when Count == 1, or two or more state labels when Count >= 2.
	States []string
	// Count is the number of consecutive occurrences of States.
	Count int
}

// RevisionStateSummary summarizes the effective revision state sequence of a timeline over a time window.
type RevisionStateSummary struct {
	// Prefix holds all compressed steps when Truncated is false, or the leading steps before the ellipsis when Truncated is true.
	Prefix []RevisionStateStep
	// Suffix holds the trailing steps after the ellipsis when Truncated is true.
	Suffix []RevisionStateStep
	// OmittedStates lists distinct state labels that appeared only in the elided middle steps, in first-seen order.
	OmittedStates []string
	// ExtraOmittedStateCount is the number of additional distinct omitted state labels beyond OmittedStates.
	ExtraOmittedStateCount int
	// TotalTransitions is the total number of consecutive state changes in the window before cycle folding or truncation.
	TotalTransitions int
	// Truncated reports whether middle steps were omitted to stay within the display budget.
	Truncated bool
}

// IsEmpty reports whether the summary contains no revision states.
func (s RevisionStateSummary) IsEmpty() bool {
	return len(s.Prefix) == 0
}

type revisionRangeBounds struct {
	hasStart bool
	startNs  int64
	hasEnd   bool
	endNs    int64
}

func newRevisionRangeBounds(startTime, endTime time.Time) revisionRangeBounds {
	var b revisionRangeBounds
	if !startTime.IsZero() {
		b.hasStart = true
		b.startNs = startTime.UnixNano()
	}
	if !endTime.IsZero() {
		b.hasEnd = true
		b.endNs = endTime.UnixNano()
	}
	return b
}

// SummarizeRevisionStates summarizes the effective revision state transitions of revs within [startTime, endTime].
//
// Consecutive identical states are deduplicated, tandem repeat cycles are folded, and zero startTime or endTime leaves that bound open.
func SummarizeRevisionStates(revs []cel.RevisionInfo, startTime, endTime time.Time) RevisionStateSummary {
	return summarizeRevisionStatesWithBounds(revs, newRevisionRangeBounds(startTime, endTime))
}

func summarizeRevisionStatesWithBounds(revs []cel.RevisionInfo, bounds revisionRangeBounds) RevisionStateSummary {
	effective := effectiveRevisionsInRange(revs, bounds)
	if len(effective) == 0 {
		return RevisionStateSummary{}
	}

	states := make([]string, 0, len(effective))
	for _, rev := range effective {
		if rev.State == "" {
			continue
		}
		if len(states) == 0 || states[len(states)-1] != rev.State {
			states = append(states, rev.State)
		}
	}
	if len(states) == 0 {
		return RevisionStateSummary{}
	}

	steps := foldRevisionStateCycles(states)
	return truncateRevisionStateSteps(steps, len(states)-1)
}

// effectiveRevisionsInRange returns the slice of sorted revisions active during bounds.
//
// When no revision changes at startNs itself, the latest revision before startNs is included to preserve the window's initial state.
func effectiveRevisionsInRange(revs []cel.RevisionInfo, b revisionRangeBounds) []cel.RevisionInfo {
	if len(revs) == 0 {
		return nil
	}
	if b.hasStart && b.hasEnd && b.startNs > b.endNs {
		return nil
	}

	endIdx := len(revs)
	if b.hasEnd {
		endIdx = sort.Search(len(revs), func(i int) bool {
			return revs[i].ChangedTime > b.endNs
		})
		if endIdx == 0 {
			return nil
		}
	}

	startIdx := 0
	if b.hasStart {
		firstInWindow := sort.Search(endIdx, func(i int) bool {
			return revs[i].ChangedTime >= b.startNs
		})
		startIdx = firstInWindow
		if firstInWindow > 0 && (firstInWindow == endIdx || revs[firstInWindow].ChangedTime > b.startNs) {
			startIdx = firstInWindow - 1
		}
	}
	return revs[startIdx:endIdx]
}

// foldRevisionStateCycles compresses consecutive tandem repeats of length 2..maxStateCycleLen into cycle steps.
func foldRevisionStateCycles(states []string) []RevisionStateStep {
	steps := make([]RevisionStateStep, 0, len(states))
	for i := 0; i < len(states); {
		maxLen := min(maxStateCycleLen, (len(states)-i)/2)
		bestLen := 0
		bestCount := 0
		bestSaved := 0

		for cycleLen := 2; cycleLen <= maxLen; cycleLen++ {
			pattern := states[i : i+cycleLen]
			count := 1
			for i+(count+1)*cycleLen <= len(states) &&
				slices.Equal(pattern, states[i+count*cycleLen:i+(count+1)*cycleLen]) {
				count++
			}
			if count >= 2 {
				saved := cycleLen * (count - 1)
				if saved > bestSaved {
					bestLen = cycleLen
					bestCount = count
					bestSaved = saved
				}
			}
		}

		if bestCount >= 2 {
			steps = append(steps, RevisionStateStep{
				States: slices.Clone(states[i : i+bestLen]),
				Count:  bestCount,
			})
			i += bestLen * bestCount
			continue
		}

		steps = append(steps, RevisionStateStep{
			States: []string{states[i]},
			Count:  1,
		})
		i++
	}
	return steps
}

func truncateRevisionStateSteps(steps []RevisionStateStep, totalTransitions int) RevisionStateSummary {
	totalSlots := 0
	for _, s := range steps {
		totalSlots += len(s.States)
	}
	if totalSlots <= maxDisplayedStateSlots {
		return RevisionStateSummary{
			Prefix:           steps,
			TotalTransitions: totalTransitions,
		}
	}

	prefixEnd := 1
	prefixSlots := len(steps[0].States)
	for prefixEnd < len(steps)-1 && prefixSlots+len(steps[prefixEnd].States) <= truncatedPrefixStateSlots {
		prefixSlots += len(steps[prefixEnd].States)
		prefixEnd++
	}

	suffixStart := len(steps) - 1
	suffixSlots := len(steps[suffixStart].States)
	for suffixStart-1 > prefixEnd && suffixSlots+len(steps[suffixStart-1].States) <= truncatedSuffixStateSlots {
		suffixStart--
		suffixSlots += len(steps[suffixStart].States)
	}

	if prefixEnd >= suffixStart {
		return RevisionStateSummary{
			Prefix:           steps,
			TotalTransitions: totalTransitions,
		}
	}

	prefix := slices.Clone(steps[:prefixEnd])
	suffix := slices.Clone(steps[suffixStart:])

	seen := make(map[string]bool, prefixSlots+suffixSlots)
	for _, step := range prefix {
		for _, st := range step.States {
			seen[st] = true
		}
	}
	for _, step := range suffix {
		for _, st := range step.States {
			seen[st] = true
		}
	}

	var omittedStates []string
	extraOmitted := 0
	for _, step := range steps[prefixEnd:suffixStart] {
		for _, st := range step.States {
			if seen[st] {
				continue
			}
			seen[st] = true
			if len(omittedStates) < maxOmittedDistinctStates {
				omittedStates = append(omittedStates, st)
			} else {
				extraOmitted++
			}
		}
	}

	return RevisionStateSummary{
		Prefix:                 prefix,
		Suffix:                 suffix,
		OmittedStates:          omittedStates,
		ExtraOmittedStateCount: extraOmitted,
		TotalTransitions:       totalTransitions,
		Truncated:              true,
	}
}
