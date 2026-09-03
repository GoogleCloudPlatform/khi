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
	"cmp"
	"fmt"
	"math"
	"slices"
	"sync"
	"time"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// estimatedLogProtoSize is the estimated serialized protobuf byte size for a Log entry plus slice overhead.
const estimatedLogProtoSize = 48

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
	clientPool         *InternPool
	serverPool         *InternPool
	idGen              *id.Generator
	writer             *Writer
	chunkSizeLimit     int
	batchIDs           []uint32
	batchTimestamps    []int64 // Unix nanoseconds, or math.MinInt64 if zero
	batchLogTypeIDs    []uint32
	batchSeverityIDs   []uint32
	batchSummaryIDs    []uint32
	batchBodyStructIDs []uint32
	currentBatchSize   int
	parserIDToID       []uint32
	mu                 sync.Mutex
}

// NewLogAccumulator creates a new LogAccumulator with the provided client and server InternPools, IDGenerator, and destination Writer.
func NewLogAccumulator(clientPool *InternPool, serverPool *InternPool, idGen *id.Generator, writer *Writer) *LogAccumulator {
	if serverPool == nil {
		serverPool = clientPool
	}
	return &LogAccumulator{
		clientPool:         clientPool,
		serverPool:         serverPool,
		idGen:              idGen,
		writer:             writer,
		chunkSizeLimit:     DefaultChunkSizeLimit,
		batchIDs:           make([]uint32, 0),
		batchTimestamps:    make([]int64, 0),
		batchLogTypeIDs:    make([]uint32, 0),
		batchSeverityIDs:   make([]uint32, 0),
		batchSummaryIDs:    make([]uint32, 0),
		batchBodyStructIDs: make([]uint32, 0),
		parserIDToID:       make([]uint32, 0),
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

// AddLog converts a StagingLog into an interned log entry and adds it to the accumulator.
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
	summaryStrRef := a.clientPool.InternString(s.Summary)

	var tsNano int64 = math.MinInt64
	if !s.Timestamp.IsZero() {
		tsNano = s.Timestamp.UnixNano()
	}

	sevID := s.Severity.GetId()
	logTypeID := s.LogType.GetId()
	bodyStructID := internedBody.id
	summaryID := summaryStrRef.id

	a.mu.Lock()
	defer a.mu.Unlock()

	if needed := int(s.Log.ID) - len(a.parserIDToID); needed > 0 {
		a.parserIDToID = slices.Grow(a.parserIDToID, needed)[:int(s.Log.ID)]
	}
	a.parserIDToID[s.Log.ID-1] = logID

	if a.currentBatchSize+estimatedLogProtoSize > a.chunkSizeLimit && len(a.batchIDs) > 0 {
		if err := a.flushLocked(); err != nil {
			return err
		}
	}

	a.batchIDs = append(a.batchIDs, logID)
	a.batchTimestamps = append(a.batchTimestamps, tsNano)
	a.batchLogTypeIDs = append(a.batchLogTypeIDs, logTypeID)
	a.batchSeverityIDs = append(a.batchSeverityIDs, sevID)
	a.batchSummaryIDs = append(a.batchSummaryIDs, summaryID)
	a.batchBodyStructIDs = append(a.batchBodyStructIDs, bodyStructID)
	a.currentBatchSize += estimatedLogProtoSize

	return nil
}

// flushLocked sorts staged logs chronologically and writes them as a compressed LogChunk to the destination Writer.
func (a *LogAccumulator) flushLocked() error {
	n := len(a.batchIDs)
	if n == 0 {
		return nil
	}

	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	slices.SortStableFunc(indices, func(i, j int) int {
		return cmp.Compare(a.batchTimestamps[i], a.batchTimestamps[j])
	})

	logs := make([]*pb.Log, n)
	for outIdx, inIdx := range indices {
		var ts *timestamppb.Timestamp
		if a.batchTimestamps[inIdx] != math.MinInt64 {
			ts = timestamppb.New(time.Unix(0, a.batchTimestamps[inIdx]).UTC())
		}
		logs[outIdx] = &pb.Log{
			Id:              proto.Uint32(a.batchIDs[inIdx]),
			Ts:              ts,
			SeverityTypeId:  proto.Uint32(a.batchSeverityIDs[inIdx]),
			LogTypeId:       proto.Uint32(a.batchLogTypeIDs[inIdx]),
			BodyStructId:    proto.Uint32(a.batchBodyStructIDs[inIdx]),
			SummaryStringId: proto.Uint32(a.batchSummaryIDs[inIdx]),
		}
	}

	chunk := &pb.LogChunk{
		Logs: logs,
	}

	rawChunk, err := CompressChunk(ChunkTypeLog, chunk)
	if err != nil {
		return fmt.Errorf("failed to compress log chunk: %w", err)
	}

	if err := a.writer.WriteRawChunk(rawChunk); err != nil {
		return fmt.Errorf("failed to write log chunk: %w", err)
	}

	a.batchIDs = a.batchIDs[:0]
	a.batchTimestamps = a.batchTimestamps[:0]
	a.batchLogTypeIDs = a.batchLogTypeIDs[:0]
	a.batchSeverityIDs = a.batchSeverityIDs[:0]
	a.batchSummaryIDs = a.batchSummaryIDs[:0]
	a.batchBodyStructIDs = a.batchBodyStructIDs[:0]
	a.currentBatchSize = 0

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
