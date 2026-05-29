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

package khifilev6

import pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"

// TimelineAccumulator acts as a facade, internally orchestrating a TimelineRegistry
// and TimelinePathPool to properly manage aliasing and deep hierarchical paths.
type TimelineAccumulator struct {
	pathPool *TimelinePathPool
	registry *TimelineRegistry
}

// NewTimelineAccumulator creates a new TimelineAccumulator facade.
func NewTimelineAccumulator(idGen *IDGenerator, internPool *InternPool, logAcc *LogAccumulator) *TimelineAccumulator {
	pathPool := NewTimelinePathPool(idGen, internPool)
	registry := NewTimelineRegistry(idGen, internPool, logAcc)
	return &TimelineAccumulator{
		pathPool: pathPool,
		registry: registry,
	}
}

// GetPath resolves or creates a TimelinePath in the underlying pool.
func (a *TimelineAccumulator) GetPath(parent *TimelinePath, segments ...PathSegment) *TimelinePath {
	return a.pathPool.Get(parent, segments...)
}

// GetBuilder retrieves a thread-safe TimelineBuilder for the given path.
func (a *TimelineAccumulator) GetBuilder(path *TimelinePath) *TimelineBuilder {
	return a.registry.GetBuilder(path)
}

// SetAlias configures aliasPath to be an alias of targetPath.
// Returns an error if aliasPath already has a builder attached (i.e., GetBuilder was already called on it).
func (a *TimelineAccumulator) SetAlias(aliasPath, targetPath *TimelinePath) error {
	return a.registry.SetAlias(aliasPath, targetPath)
}

// Accumulate extracts the arrays of Timeline and TimelineItems protobuf messages.
func (a *TimelineAccumulator) Accumulate() ([]*pb.Timeline, []*pb.TimelineItems) {
	return ExtractTimelinesAndItemsChunkSource(a.pathPool, a.registry)
}
