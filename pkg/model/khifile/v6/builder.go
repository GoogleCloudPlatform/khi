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
	"io"
	"iter"
	"slices"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
)

// BuilderProgressReporter is an interface to report progress during KHI file generation.
type BuilderProgressReporter interface {
	// ReportProgress reports the current progress with a percentage (0.0 to 1.0) and status message.
	ReportProgress(progress float32, status string)
}

// Builder orchestrates the accumulators, pools, and final file generation for KHI v6 format.
type Builder struct {
	idGenerator         *IDGenerator
	internPool          *InternPool
	TimelineAccumulator *TimelineAccumulator
	LogAccumulator      *LogAccumulator
	MetadataAccumulator *MetadataAccumulator
}

// NewBuilder initializes a new v6 Builder with all necessary accumulators and pools.
func NewBuilder() *Builder {
	gen := &IDGenerator{}
	internPool := NewInternPool(gen)
	logAcc := NewLogAccumulator(internPool, gen)

	return &Builder{
		idGenerator:         gen,
		internPool:          internPool,
		TimelineAccumulator: NewTimelineAccumulator(gen, internPool, logAcc),
		LogAccumulator:      logAcc,
		MetadataAccumulator: NewMetadataAccumulator(),
	}
}

// Build writes the accumulated data to the provided io.Writer in KHI v6 format.
func (b *Builder) Build(w io.Writer, reporter BuilderProgressReporter) error {
	report := func(progress float32, status string) {
		if reporter != nil {
			reporter.ReportProgress(progress, status)
		}
	}

	report(0.0, "Initializing KHI writer")
	writer, err := NewWriter(w)
	if err != nil {
		return fmt.Errorf("failed to create writer: %w", err)
	}

	report(0.1, "Writing timeline style chunk")
	// 1. Write TimelineStyleChunk directly (no generator needed since it's a single chunk)
	styleChunk := style.GenerateChunk()
	if err := writer.WriteChunk(ChunkTypeTimelineStyle, styleChunk); err != nil {
		return fmt.Errorf("failed to write timeline style chunk: %w", err)
	}

	report(0.2, "Writing metadata chunk")
	// 2. Write MetadataChunk
	metadataList := b.MetadataAccumulator.Accumulate()
	if len(metadataList) > 0 {
		metadataGen := NewMetadataGenerator(slices.Values(metadataList))
		defer metadataGen.Close()
		if err := writer.WriteGenerator(metadataGen); err != nil {
			return fmt.Errorf("failed to write metadata chunk: %w", err)
		}
	}

	report(0.4, "Writing log chunks")
	// 3. Write LogChunks
	logs := b.LogAccumulator.Accumulate()
	if len(logs) > 0 {
		logGen := NewLogGenerator(slices.Values(logs))
		defer logGen.Close()
		if err := writer.WriteGenerator(logGen); err != nil {
			return fmt.Errorf("failed to write log chunks: %w", err)
		}
	}

	report(0.6, "Writing timeline chunks")
	// 4. Write TimelineChunks
	timelines, timelineItems := b.TimelineAccumulator.Accumulate()

	if len(timelines) > 0 {
		timelineGen := NewTimelineGenerator(slices.Values(timelines))
		defer timelineGen.Close()
		if err := writer.WriteGenerator(timelineGen); err != nil {
			return fmt.Errorf("failed to write timeline chunks: %w", err)
		}
	}

	if len(timelineItems) > 0 {
		timelineItemsGen := NewTimelineItemsGenerator(slices.Values(timelineItems))
		defer timelineItemsGen.Close()
		if err := writer.WriteGenerator(timelineItemsGen); err != nil {
			return fmt.Errorf("failed to write timeline items chunks: %w", err)
		}
	}

	report(0.8, "Writing intern pool chunks")
	// 5. Write InternPoolChunk (Strings and FieldPathSets)
	stringSeq := mapSeq(b.internPool.SortedStringRefs(), func(ref *InternStringRef) *pb.InternString {
		return ref.ToProto()
	})
	stringGen := NewInternPoolGenerator(stringSeq)
	if err := writer.WriteGenerator(stringGen); err != nil {
		return fmt.Errorf("failed to write intern string chunks: %w", err)
	}

	fieldSetSeq := mapSeq(b.internPool.FieldSetRefs(), func(ref *FieldPathSetRef) *pb.InternFieldPathSet {
		return ref.ToProto()
	})
	fieldPathSetGen := NewInternFieldPathSetGenerator(fieldSetSeq)
	if err := writer.WriteGenerator(fieldPathSetGen); err != nil {
		return fmt.Errorf("failed to write intern field path set chunks: %w", err)
	}

	report(1.0, "Done")
	return nil
}

func mapSeq[T any, U any](seq iter.Seq[T], f func(T) U) iter.Seq[U] {
	return func(yield func(U) bool) {
		for v := range seq {
			if !yield(f(v)) {
				return
			}
		}
	}
}
