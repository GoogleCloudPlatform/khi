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

	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

// BuilderProgressReporter is an interface to report progress during KHI file generation.
type BuilderProgressReporter interface {
	// ReportProgress reports the current progress with a percentage (0.0 to 1.0) and status message.
	ReportProgress(progress float32, status string)
}

// Builder orchestrates the accumulators, pools, and final file generation for KHI v6 format.
type Builder struct {
	idGenerator         *id.Generator
	writer              *Writer
	internPool          *InternPool
	serverInternPool    *InternPool
	TimelineAccumulator *TimelineAccumulator
	LogAccumulator      *LogAccumulator
	MetadataAccumulator *MetadataAccumulator
}

// NewBuilder initializes a new v6 Builder with all necessary accumulators and pools.
func NewBuilder(gen *id.Generator, writer *Writer) *Builder {
	internPool := NewInternPool(gen, writer)
	serverPool := NewServerInternPool(internPool, gen, writer)
	logAcc := NewLogAccumulator(internPool, serverPool, gen, writer)

	return &Builder{
		idGenerator:         gen,
		writer:              writer,
		internPool:          internPool,
		serverInternPool:    serverPool,
		TimelineAccumulator: NewTimelineAccumulator(gen, internPool, serverPool),
		LogAccumulator:      logAcc,
		MetadataAccumulator: NewMetadataAccumulator(),
	}
}

// NewTestBuilder creates a Builder for testing purposes with an in-memory test writer.
func NewTestBuilder(gen *id.Generator) *Builder {
	return NewBuilder(gen, MustNewTestWriter())
}

// Build writes the accumulated metadata, timeline chunks, and flushes remaining log and intern pool chunks.
func (b *Builder) Build(reporter BuilderProgressReporter) (err error) {
	defer func() {
		if closeErr := b.writer.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close writer: %w", closeErr)
		}
	}()

	report := func(progress float32, status string) {
		if reporter != nil {
			reporter.ReportProgress(progress, status)
		}
	}

	report(0.1, "Writing timeline style chunk")
	// 1. Write TimelineStyleChunk directly (no generator needed since it's a single chunk)
	styleChunk := style.GenerateChunk()
	if err := b.writer.WriteChunk(ChunkTypeTimelineStyle, styleChunk); err != nil {
		return fmt.Errorf("failed to write timeline style chunk: %w", err)
	}

	report(0.2, "Writing metadata chunk")
	// 2. Write MetadataChunk
	metadataList := b.MetadataAccumulator.Accumulate()
	if len(metadataList) > 0 {
		metadataGen := NewMetadataGenerator(slices.Values(metadataList))
		defer metadataGen.Close()
		if err := b.writer.WriteGenerator(metadataGen); err != nil {
			return fmt.Errorf("failed to write metadata chunk: %w", err)
		}
	}

	report(0.4, "Flushing log chunks")
	// 3. Flush remaining LogChunks
	if err := b.LogAccumulator.Flush(); err != nil {
		return fmt.Errorf("failed to flush log accumulator: %w", err)
	}

	report(0.6, "Writing timeline chunks")
	// 4. Write TimelineChunks
	timelines, timelineItems := b.TimelineAccumulator.Accumulate()

	if len(timelines) > 0 {
		timelineGen := NewTimelineGenerator(slices.Values(timelines))
		defer timelineGen.Close()
		if err := b.writer.WriteGenerator(timelineGen); err != nil {
			return fmt.Errorf("failed to write timeline chunks: %w", err)
		}
	}

	if len(timelineItems) > 0 {
		timelineItemsGen := NewTimelineItemsGenerator(slices.Values(timelineItems))
		defer timelineItemsGen.Close()
		if err := b.writer.WriteGenerator(timelineItemsGen); err != nil {
			return fmt.Errorf("failed to write timeline items chunks: %w", err)
		}
	}

	report(0.8, "Flushing intern pool chunks")
	// 5. Flush client and server intern pool chunks
	if err := b.internPool.Flush(); err != nil {
		return fmt.Errorf("failed to flush client intern pool: %w", err)
	}
	if err := b.serverInternPool.Flush(); err != nil {
		return fmt.Errorf("failed to flush server intern pool: %w", err)
	}

	report(1.0, "Done")
	return nil
}
