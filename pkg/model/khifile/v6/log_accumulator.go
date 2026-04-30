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
	"sync"
	"time"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
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
	pool  *InternPool
	idGen *IDGenerator
	logs  []*pb.Log
	mu    sync.RWMutex
}

// NewLogAccumulator creates a new LogAccumulator with the provided InternPool and IDGenerator.
func NewLogAccumulator(pool *InternPool, idGen *IDGenerator) *LogAccumulator {
	return &LogAccumulator{
		pool:  pool,
		idGen: idGen,
		logs:  make([]*pb.Log, 0),
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

	internedBody, err := ToInternedStruct(s.Log.Node, a.pool)
	if err != nil {
		return fmt.Errorf("failed to intern log body: %w", err)
	}

	id := a.idGen.New(IDLog)
	pbLog := &pb.Log{
		Id:   &id,
		Body: internedBody,
	}

	pbLog.Ts = timestamppb.New(s.Timestamp)
	pbLog.SeverityTypeId = s.Severity.Id

	summaryStrRef := a.pool.InternString(s.Summary)
	pbLog.SummaryStringId = &summaryStrRef.id

	pbLog.LogTypeId = s.LogType.Id

	a.mu.Lock()
	if needed := int(id) - len(a.logs); needed > 0 {
		// Note: Go compiler optimizes append(slice, make([]T, n)...) to avoid temporary allocation.
		a.logs = append(a.logs, make([]*pb.Log, needed)...)
	}
	a.logs[id-1] = pbLog
	a.mu.Unlock()

	return nil
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
// The returned list is naturally sorted by log ID.
func (a *LogAccumulator) Accumulate() []*pb.Log {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]*pb.Log, 0, len(a.logs))
	for _, l := range a.logs {
		if l != nil {
			result = append(result, l)
		}
	}
	return result
}
