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

	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"google.golang.org/protobuf/proto"
)

// FilterPipelineParams contains the query parameters for running the timeline and log filter pipeline.
type FilterPipelineParams struct {
	TimelineQuery          string
	TimelineExclusionQuery string
	LogQuery               string
	ExcludeNoLogs          bool
}

// FilterTimeline executes the complete 6-stage filtering pipeline and streams progress updates.
func (w *Workbench) FilterTimeline(
	ctx context.Context,
	params FilterPipelineParams,
	sendProgress func(*apiv1.FilterProgress) error,
) (*apiv1.FilterResult, error) {
	w.mu.RLock()
	if w.closed {
		w.mu.RUnlock()
		return nil, ErrWorkbenchClosed
	}
	idx := w.searchIndex
	w.mu.RUnlock()

	if idx == nil {
		return nil, fmt.Errorf("search index is not initialized")
	}

	report := func(stageName string, current uint32, total uint32) error {
		if sendProgress == nil {
			return nil
		}
		return sendProgress(&apiv1.FilterProgress{
			StageName: proto.String(stageName),
			Current:   proto.Uint32(current),
			Total:     proto.Uint32(total),
		})
	}

	pipeline := NewDefaultPipeline(params)
	return pipeline.Execute(ctx, idx, report)
}
