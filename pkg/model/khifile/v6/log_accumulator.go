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
	"time"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// StagingLog represents a log entry and its associated metadata to be added to LogAccumulator.
type StagingLog struct {
	Log       *log.Log
	Summary   string
	Timestamp time.Time
	LogType   *pb.LogType
	Severity  *pb.Severity
}

// LogAccumulator facilitates the construction of Log chunks
// by interning their structured data using an InternPool and streaming
// completed chunks directly to the destination Writer.
// It is safe for concurrent use by multiple goroutines.
type LogAccumulator struct {
	clientPool       *InternPool
	serverPool       *InternPool
	idGen            *id.Generator
	writer           *Writer
	chunkSizeLimit   int
	currentBatch     []*pb.Log
	currentBatchSize int
	parserIDToID     []uint32
	mu               sync.Mutex
}

// NewLogAccumulator creates a new LogAccumulator with the provided client and server InternPools, IDGenerator, and destination Writer.
func NewLogAccumulator(clientPool *InternPool, serverPool *InternPool, idGen *id.Generator, writer *Writer) *LogAccumulator {
	if serverPool == nil {
		serverPool = clientPool
	}
	return &LogAccumulator{
		clientPool:     clientPool,
		serverPool:     serverPool,
		idGen:          idGen,
		writer:         writer,
		chunkSizeLimit: DefaultChunkSizeLimit,
		currentBatch:   make([]*pb.Log, 0),
		parserIDToID:   make([]uint32, 0),
	}
}

// NewTestLogAccumulator creates a LogAccumulator with MustNewTestWriter for testing purposes.
func NewTestLogAccumulator(clientPool *InternPool, serverPool *InternPool, idGen *id.Generator) *LogAccumulator {
	return NewLogAccumulator(clientPool, serverPool, idGen, MustNewTestWriter())
}

// SetChunkSizeLimit sets the maximum byte size limit for a single LogChunk.
func (a *LogAccumulator) SetChunkSizeLimit(limit int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.chunkSizeLimit = limit
}

// AddLog converts a StagingLog into a khifilev6.Log proto and adds it to the accumulator.
// It interns the log body to optimize storage and flushes chunks directly to the Writer when the batch size exceeds the chunk limit.
func (a *LogAccumulator) AddLog(s *StagingLog) error {
	if s.Severity == nil || s.Severity.Id == nil {
		return fmt.Errorf("severity or its ID is missing")
	}
	if s.LogType == nil || s.LogType.Id == nil {
		return fmt.Errorf("log type or its ID is missing")
	}

	internedBody, err := ToInternedStruct(s.Log.Node, a.serverPool)
	if err != nil {
		return fmt.Errorf("failed to intern log body: %w", err)
	}

	logID := a.idGen.New(id.Log)
	pbLog := &pb.Log{
		Id:           &logID,
		BodyStructId: &internedBody.id,
	}

	pbLog.Ts = timestamppb.New(s.Timestamp)
	pbLog.SeverityTypeId = s.Severity.Id

	summaryStrRef := a.clientPool.InternString(s.Summary)
	pbLog.SummaryStringId = &summaryStrRef.id

	pbLog.LogTypeId = s.LogType.Id

	itemSize := proto.Size(pbLog) + 8

	a.mu.Lock()
	defer a.mu.Unlock()

	if needed := int(s.Log.ID) - len(a.parserIDToID); needed > 0 {
		a.parserIDToID = slices.Grow(a.parserIDToID, needed)[:int(s.Log.ID)]
	}
	a.parserIDToID[s.Log.ID-1] = logID

	if a.currentBatchSize+itemSize > a.chunkSizeLimit && len(a.currentBatch) > 0 {
		if err := a.flushLocked(); err != nil {
			return err
		}
	}

	a.currentBatch = append(a.currentBatch, pbLog)
	a.currentBatchSize += itemSize

	return nil
}

func (a *LogAccumulator) flushLocked() error {
	if len(a.currentBatch) == 0 {
		return nil
	}

	batch := a.currentBatch
	a.currentBatch = make([]*pb.Log, 0)
	a.currentBatchSize = 0

	slices.SortStableFunc(batch, func(x, y *pb.Log) int {
		return x.GetTs().AsTime().Compare(y.GetTs().AsTime())
	})

	chunk := &pb.LogChunk{
		Logs: batch,
	}

	rawChunk, err := CompressChunk(ChunkTypeLog, chunk)
	if err != nil {
		return fmt.Errorf("failed to compress log chunk: %w", err)
	}

	if err := a.writer.WriteRawChunk(rawChunk); err != nil {
		return fmt.Errorf("failed to write log chunk: %w", err)
	}

	return nil
}

// Flush writes any pending buffered logs into a final LogChunk and outputs it to the destination Writer.
func (a *LogAccumulator) Flush() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.flushLocked()
}

// ResolveLogID returns the serialized log ID (uint32) for a given parser-side temporary log ID (uint32).
// It returns false if the log ID is not found.
func (a *LogAccumulator) ResolveLogID(parserID uint32) (uint32, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if parserID == 0 || int(parserID) > len(a.parserIDToID) {
		return 0, false
	}
	confirmedID := a.parserIDToID[parserID-1]
	if confirmedID == 0 {
		return 0, false
	}
	return confirmedID, true
}
