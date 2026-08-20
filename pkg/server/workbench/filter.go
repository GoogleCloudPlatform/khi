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
	"log/slog"
	"time"

	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/RoaringBitmap/roaring/v2"
)

// FilterContext holds the mutable sets of matching timeline and log IDs across pipeline filter stages.
type FilterContext struct {
	TimelineIDs *roaring.Bitmap
	LogIDs      *roaring.Bitmap
}

// NewFilterContext initializes an empty FilterContext.
func NewFilterContext() *FilterContext {
	return &FilterContext{
		TimelineIDs: roaring.NewBitmap(),
		LogIDs:      roaring.NewBitmap(),
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
		NewExcludeNoLogsFilter(params.ExcludeNoLogs),
		NewIncludeAncestorsFilter(),
	)
}

// Execute runs all registered filter stages sequentially and constructs the final FilterResult.
func (p *Pipeline) Execute(
	ctx context.Context,
	index *SearchIndex,
	report ProgressReporter,
) (*apiv1.FilterResult, error) {
	filterCtx := NewFilterContext()

	totalStart := time.Now()
	for _, filter := range p.filters {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		stageStart := time.Now()
		if err := filter.Process(ctx, filterCtx, index, report); err != nil {
			return nil, err
		}
		duration := time.Since(stageStart)
		slog.DebugContext(ctx, "filter stage completed",
			"stage", filter.Name(),
			"duration", duration.String(),
			"duration_ms", duration.Milliseconds(),
			"matching_timelines", filterCtx.TimelineIDs.GetCardinality(),
			"matching_logs", filterCtx.LogIDs.GetCardinality(),
		)
	}

	tlMode, tlBitset := EncodeFilterResultBitset(len(index.Timelines), filterCtx.TimelineIDs)
	logMode, logBitset := EncodeFilterResultBitset(len(index.Logs), filterCtx.LogIDs)

	totalDuration := time.Since(totalStart)
	slog.DebugContext(ctx, "filter pipeline completed",
		"total_duration", totalDuration.String(),
		"total_duration_ms", totalDuration.Milliseconds(),
		"result_timelines", filterCtx.TimelineIDs.GetCardinality(),
		"result_logs", filterCtx.LogIDs.GetCardinality(),
	)

	return &apiv1.FilterResult{
		TimelineMode:   tlMode.Enum(),
		TimelineBitset: tlBitset,
		LogMode:        logMode.Enum(),
		LogBitset:      logBitset,
	}, nil
}
