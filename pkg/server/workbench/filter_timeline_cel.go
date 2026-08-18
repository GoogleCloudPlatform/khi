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
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"golang.org/x/sync/errgroup"
)

// TimelineCELFilter evaluates a CEL query against each timeline in the SearchIndex.
type TimelineCELFilter struct {
	query string
}

var _ TimelineFilter = (*TimelineCELFilter)(nil)

// NewTimelineCELFilter creates a new TimelineCELFilter with the specified CEL expression.
func NewTimelineCELFilter(query string) *TimelineCELFilter {
	return &TimelineCELFilter{query: query}
}

// Name returns the display name of this filter stage.
func (f *TimelineCELFilter) Name() string {
	return "Timeline CEL filter"
}

// Process compiles and evaluates the timeline CEL expression concurrently across workers, populating matching timeline IDs into filterCtx.TimelineIDs.
func (f *TimelineCELFilter) Process(
	ctx context.Context,
	filterCtx *FilterContext,
	index *SearchIndex,
	report ProgressReporter,
) error {
	totalTimelines := uint32(len(index.Timelines))
	if totalTimelines == 0 {
		return nil
	}

	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > int(totalTimelines) {
		numWorkers = int(totalTimelines)
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	g, groupCtx := errgroup.WithContext(ctx)
	results := make([][]uint32, numWorkers)

	chunkSize := (int(totalTimelines) + numWorkers - 1) / numWorkers
	var processedCount uint32
	var reportMu sync.Mutex

	for w := 0; w < numWorkers; w++ {
		workerIdx := w
		start := workerIdx * chunkSize
		end := start + chunkSize
		if end > int(totalTimelines) {
			end = int(totalTimelines)
		}
		if start >= end {
			continue
		}

		g.Go(func() error {
			tlEval, err := cel.NewTimelineEvaluator()
			if err != nil {
				return fmt.Errorf("failed to initialize timeline evaluator: %w", err)
			}
			if err := tlEval.Compile(f.query); err != nil {
				return fmt.Errorf("invalid timeline query: %w", err)
			}

			localMatched := make([]uint32, 0, (end-start)/4)
			for i := start; i < end; i++ {
				select {
				case <-groupCtx.Done():
					return groupCtx.Err()
				default:
				}

				tl := index.Timelines[i]
				matched, err := tlEval.Evaluate(groupCtx, tl.Data)
				if err != nil {
					return fmt.Errorf("error evaluating timeline query: %w", err)
				}
				if matched {
					localMatched = append(localMatched, tl.ID)
				}

				currentProcessed := atomic.AddUint32(&processedCount, 1)
				if currentProcessed%500 == 0 || currentProcessed == totalTimelines {
					if report != nil {
						reportMu.Lock()
						_ = report(f.Name(), currentProcessed, totalTimelines)
						reportMu.Unlock()
					}
				}
			}
			results[workerIdx] = localMatched
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	for _, localMatched := range results {
		for _, id := range localMatched {
			filterCtx.TimelineIDs[id] = struct{}{}
		}
	}

	return nil
}

// TimelineCELExclusionFilter evaluates an exclusion CEL query and removes matching timelines and their descendant subtrees.
type TimelineCELExclusionFilter struct {
	exclusionQuery string
}

var _ TimelineFilter = (*TimelineCELExclusionFilter)(nil)

// NewTimelineCELExclusionFilter creates a new TimelineCELExclusionFilter.
func NewTimelineCELExclusionFilter(exclusionQuery string) *TimelineCELExclusionFilter {
	return &TimelineCELExclusionFilter{exclusionQuery: exclusionQuery}
}

// Name returns the display name of this filter stage.
func (f *TimelineCELExclusionFilter) Name() string {
	return "Timeline CEL exclusion filter"
}

// Process evaluates the exclusion query concurrently on all currently matching timelines and removes excluded subtrees.
func (f *TimelineCELExclusionFilter) Process(
	ctx context.Context,
	filterCtx *FilterContext,
	index *SearchIndex,
	report ProgressReporter,
) error {
	totalTimelines := uint32(len(index.Timelines))
	if f.exclusionQuery == "" {
		if report != nil {
			return report(f.Name(), uint32(len(filterCtx.TimelineIDs)), totalTimelines)
		}
		return nil
	}

	candidateIDs := make([]uint32, 0, len(filterCtx.TimelineIDs))
	for id := range filterCtx.TimelineIDs {
		candidateIDs = append(candidateIDs, id)
	}

	totalCandidates := len(candidateIDs)
	if totalCandidates == 0 {
		if report != nil {
			return report(f.Name(), 0, totalTimelines)
		}
		return nil
	}

	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > totalCandidates {
		numWorkers = totalCandidates
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	g, groupCtx := errgroup.WithContext(ctx)
	results := make([][]uint32, numWorkers)
	chunkSize := (totalCandidates + numWorkers - 1) / numWorkers

	for w := 0; w < numWorkers; w++ {
		workerIdx := w
		start := workerIdx * chunkSize
		end := start + chunkSize
		if end > totalCandidates {
			end = totalCandidates
		}
		if start >= end {
			continue
		}

		g.Go(func() error {
			exclEval, err := cel.NewTimelineEvaluator()
			if err != nil {
				return fmt.Errorf("failed to initialize exclusion evaluator: %w", err)
			}
			if err := exclEval.Compile(f.exclusionQuery); err != nil {
				return fmt.Errorf("invalid timeline exclusion query: %w", err)
			}

			localExcluded := make([]uint32, 0, (end-start)/4)
			for i := start; i < end; i++ {
				select {
				case <-groupCtx.Done():
					return groupCtx.Err()
				default:
				}

				id := candidateIDs[i]
				if tl, ok := index.TimelineMap[id]; ok {
					matched, err := exclEval.Evaluate(groupCtx, tl.Data)
					if err != nil {
						return fmt.Errorf("error evaluating exclusion query: %w", err)
					}
					if matched {
						localExcluded = append(localExcluded, id)
					}
				}
			}
			results[workerIdx] = localExcluded
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	excludedSet := make(map[uint32]struct{})
	var markExcludeDescendants func(id uint32)
	markExcludeDescendants = func(id uint32) {
		if _, exists := excludedSet[id]; exists {
			return
		}
		excludedSet[id] = struct{}{}
		if tl, ok := index.TimelineMap[id]; ok {
			for _, childID := range tl.ChildrenIDs {
				markExcludeDescendants(childID)
			}
		}
	}

	for _, localExcluded := range results {
		for _, id := range localExcluded {
			markExcludeDescendants(id)
		}
	}

	for id := range excludedSet {
		delete(filterCtx.TimelineIDs, id)
	}

	if report != nil {
		if err := report(f.Name(), uint32(len(filterCtx.TimelineIDs)), totalTimelines); err != nil {
			return err
		}
	}

	return nil
}
