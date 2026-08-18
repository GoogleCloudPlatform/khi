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
	"sort"

	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
)

// FilterContext holds the mutable sets of matching timeline and log IDs across pipeline filter stages.
type FilterContext struct {
	TimelineIDs map[uint32]struct{}
	LogIDs      map[uint32]struct{}
}

// NewFilterContext initializes an empty FilterContext.
func NewFilterContext() *FilterContext {
	return &FilterContext{
		TimelineIDs: make(map[uint32]struct{}),
		LogIDs:      make(map[uint32]struct{}),
	}
}

// ProgressReporter is a callback invoked during filter execution to stream progress updates.
type ProgressReporter func(stageName string, current uint32, total uint32) error

// TimelineFilter represents a single, modular filter stage in the backend timeline and log search pipeline.
type TimelineFilter interface {
	// Name returns the human-readable display name of the filter stage.
	Name() string
	// Process executes the filtering logic on the FilterContext against the SearchIndex.
	Process(ctx context.Context, filterCtx *FilterContext, index *SearchIndex, report ProgressReporter) error
}

// Pipeline executes a sequential chain of TimelineFilter stages.
type Pipeline struct {
	filters []TimelineFilter
}

// NewPipeline creates a new Pipeline with the specified sequence of TimelineFilter stages.
func NewPipeline(filters ...TimelineFilter) *Pipeline {
	return &Pipeline{
		filters: filters,
	}
}

// NewDefaultPipeline creates a standard 6-stage timeline and log search pipeline matching frontend filter semantics.
func NewDefaultPipeline(params FilterPipelineParams) *Pipeline {
	return NewPipeline(
		NewTimelineCELFilter(params.TimelineQuery),
		NewIncludeDescendantsFilter(),
		NewTimelineCELExclusionFilter(params.TimelineExclusionQuery),
		NewLogCELFilter(params.LogQuery),
		NewIncludeAncestorsFilter(),
		NewExcludeNoLogsFilter(params.ExcludeNoLogs),
	)
}

// Execute runs all registered filter stages sequentially and constructs the final FilterResult.
func (p *Pipeline) Execute(
	ctx context.Context,
	index *SearchIndex,
	report ProgressReporter,
) (*apiv1.FilterResult, error) {
	filterCtx := NewFilterContext()

	for _, filter := range p.filters {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if err := filter.Process(ctx, filterCtx, index, report); err != nil {
			return nil, err
		}
	}

	// Convert maps to sorted slices for deterministic output
	resultTimelineIDs := make([]uint32, 0, len(filterCtx.TimelineIDs))
	for id := range filterCtx.TimelineIDs {
		resultTimelineIDs = append(resultTimelineIDs, id)
	}
	sort.Slice(resultTimelineIDs, func(i, j int) bool { return resultTimelineIDs[i] < resultTimelineIDs[j] })

	resultLogIDs := make([]uint32, 0, len(filterCtx.LogIDs))
	for id := range filterCtx.LogIDs {
		resultLogIDs = append(resultLogIDs, id)
	}
	sort.Slice(resultLogIDs, func(i, j int) bool { return resultLogIDs[i] < resultLogIDs[j] })

	return &apiv1.FilterResult{
		TimelineIds: resultTimelineIDs,
		LogIds:      resultLogIDs,
	}, nil
}
