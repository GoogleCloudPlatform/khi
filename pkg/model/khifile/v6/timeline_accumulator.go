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

import (
	"fmt"
	"slices"
	"sync"
	"sync/atomic"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
)

// DefaultTimelineItemsFlushThreshold is the default number of accumulated items before streaming a TimelineChunk.
const DefaultTimelineItemsFlushThreshold = 10000

// TimelineAccumulator acts as a facade, internally orchestrating a TimelineRegistry
// and TimelinePathPool to properly manage aliasing and deep hierarchical paths,
// while streaming completed TimelineItems chunks directly to the destination Writer.
type TimelineAccumulator struct {
	pathPool       *TimelinePathPool
	registry       *TimelineRegistry
	writer         *Writer
	pendingCount   atomic.Int64
	flushThreshold int64
	flushMu        sync.Mutex
}

// NewTimelineAccumulator creates a new TimelineAccumulator facade with destination Writer.
func NewTimelineAccumulator(idGen *id.Generator, clientPool *InternPool, serverPool *InternPool, writer *Writer) *TimelineAccumulator {
	pathPool := NewTimelinePathPool(idGen, clientPool)
	registry := NewTimelineRegistry(idGen, clientPool, serverPool)
	return &TimelineAccumulator{
		pathPool:       pathPool,
		registry:       registry,
		writer:         writer,
		flushThreshold: DefaultTimelineItemsFlushThreshold,
	}
}

// NewTestTimelineAccumulator creates a new TimelineAccumulator writing to an in-memory test writer.
func NewTestTimelineAccumulator(idGen *id.Generator, clientPool *InternPool, serverPool *InternPool) *TimelineAccumulator {
	return NewTimelineAccumulator(idGen, clientPool, serverPool, MustNewTestWriter())
}

// SetFlushThreshold sets the item count threshold for streaming TimelineItems chunks.
func (a *TimelineAccumulator) SetFlushThreshold(threshold int64) {
	a.flushMu.Lock()
	defer a.flushMu.Unlock()
	a.flushThreshold = threshold
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

// HasRevision reports whether the timeline at path has any accumulated revisions.
func (a *TimelineAccumulator) HasRevision(path *TimelinePath) bool {
	if b, ok := a.registry.GetBuilderIfExists(path); ok {
		return b.HasRevision()
	}
	return false
}

// HasEvent reports whether the timeline at path has any accumulated events.
func (a *TimelineAccumulator) HasEvent(path *TimelinePath) bool {
	if b, ok := a.registry.GetBuilderIfExists(path); ok {
		return b.HasEvent()
	}
	return false
}

// NotifyItemsAdded increments the count of accumulated items and triggers a flush if the threshold is met.
func (a *TimelineAccumulator) NotifyItemsAdded(count int) error {
	if count <= 0 {
		return nil
	}
	if a.pendingCount.Add(int64(count)) >= a.flushThreshold {
		return a.FlushPendingItems()
	}
	return nil
}

// FlushPendingItems drains accumulated items from all builders and streams them as TimelineChunk(s).
func (a *TimelineAccumulator) FlushPendingItems() error {
	a.flushMu.Lock()
	defer a.flushMu.Unlock()

	a.pendingCount.Store(0)

	var items []*pb.TimelineItems
	for builder := range a.registry.Builders() {
		if proto := builder.ExtractPendingProto(); proto != nil {
			items = append(items, proto)
		}
	}
	if len(items) == 0 {
		return nil
	}

	// Sort items by ID to ensure deterministic output within each chunk
	slices.SortFunc(items, func(a, b *pb.TimelineItems) int {
		return int(a.GetId()) - int(b.GetId())
	})

	gen := NewTimelineItemsGenerator(slices.Values(items))
	defer gen.Close()
	return a.writer.WriteGenerator(gen)
}

// Flush streams any remaining TimelineItems chunks, followed by the Timeline hierarchy chunks.
func (a *TimelineAccumulator) Flush() error {
	if err := a.FlushPendingItems(); err != nil {
		return err
	}

	a.flushMu.Lock()
	defer a.flushMu.Unlock()

	timelines := extractTimelines(a.pathPool, a.registry)
	if len(timelines) > 0 {
		gen := NewTimelineGenerator(slices.Values(timelines))
		defer gen.Close()
		if err := a.writer.WriteGenerator(gen); err != nil {
			return fmt.Errorf("failed to write timeline hierarchy chunks: %w", err)
		}
	}
	return nil
}

// AddTestRevision adds a dummy revision to the timeline builder at path for testing purposes.
func (a *TimelineAccumulator) AddTestRevision(path *TimelinePath) {
	b := a.GetBuilder(path)
	b.AddRevision(pendingRevision{})
}
