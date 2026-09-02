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

// LogAccumulator facilitates the construction of a list of Log proto messages
// by interning their structured data using an InternPool.
// It is safe for concurrent use by multiple goroutines.
type LogAccumulator struct {
	clientPool   *InternPool
	serverPool   *InternPool
	idGen        *id.Generator
	logs         []*pb.Log
	parserIDToID []uint32
	mu           sync.RWMutex
}

// NewLogAccumulator creates a new LogAccumulator with the provided client and server InternPools and IDGenerator.
func NewLogAccumulator(clientPool *InternPool, serverPool *InternPool, idGen *id.Generator) *LogAccumulator {
	if serverPool == nil {
		serverPool = clientPool
	}
	return &LogAccumulator{
		clientPool:   clientPool,
		serverPool:   serverPool,
		idGen:        idGen,
		logs:         make([]*pb.Log, 0),
		parserIDToID: make([]uint32, 0),
	}
}

// AddLog converts a StagingLog into a khifilev6.Log proto and adds it to the accumulator.
// It interns the log body to optimize storage.
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

	a.mu.Lock()
	if needed := int(logID) - len(a.logs); needed > 0 {
		a.logs = slices.Grow(a.logs, needed)[:int(logID)]
	}
	a.logs[logID-1] = pbLog
	if needed := int(s.Log.ID) - len(a.parserIDToID); needed > 0 {
		a.parserIDToID = slices.Grow(a.parserIDToID, needed)[:int(s.Log.ID)]
	}
	a.parserIDToID[s.Log.ID-1] = logID
	a.mu.Unlock()

	return nil
}

// ResolveLogID returns the serialized log ID (uint32) for a given parser-side temporary log ID (uint32).
// It returns false if the log ID is not found.
func (a *LogAccumulator) ResolveLogID(parserID uint32) (uint32, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if parserID == 0 || int(parserID) > len(a.parserIDToID) {
		return 0, false
	}
	confirmedID := a.parserIDToID[parserID-1]
	if confirmedID == 0 {
		return 0, false
	}
	return confirmedID, true
}

// GetLog returns a log by its ID. Returns nil if the log is not found.
func (a *LogAccumulator) GetLog(id uint32) *pb.Log {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if id == 0 || int(id) > len(a.logs) {
		return nil
	}
	return a.logs[id-1]
}

// Accumulate returns the accumulated list of Log proto messages.
// The returned list is sorted chronologically by their timestamp.
func (a *LogAccumulator) Accumulate() []*pb.Log {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]*pb.Log, 0, len(a.logs))
	for _, l := range a.logs {
		if l != nil {
			result = append(result, l)
		}
	}
	slices.SortStableFunc(result, func(a, b *pb.Log) int {
		return a.GetTs().AsTime().Compare(b.GetTs().AsTime())
	})
	return result
}
