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
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/architecturegraph"
)

// GetArchitectureGraph builds the Kubernetes architecture graph for the specified request.
func (w *Workbench) GetArchitectureGraph(
	ctx context.Context,
	req *apiv1.GetArchitectureGraphRequest,
) (*apiv1.GetArchitectureGraphResponse, error) {
	w.mu.RLock()
	if w.closed {
		w.mu.RUnlock()
		return nil, ErrWorkbenchClosed
	}
	idx := w.searchIndex
	pool := w.internPool
	w.mu.RUnlock()

	if idx == nil {
		return nil, fmt.Errorf("search index is not initialized")
	}

	builder := architecturegraph.NewBuilder(idx.Timelines, idx.TimelineMap, pool)
	return builder.Build(ctx, req)
}
