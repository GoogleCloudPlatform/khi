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
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"golang.org/x/sync/errgroup"
)

// LogCELFilter evaluates a CEL query against candidate logs associated with retained timelines.
type LogCELFilter struct {
	query string
}

var _ TimelineFilter = (*LogCELFilter)(nil)

// NewLogCELFilter creates a new LogCELFilter with the specified CEL expression.
func NewLogCELFilter(query string) *LogCELFilter {
	return &LogCELFilter{query: query}
}

// Name returns the display name of this filter stage.
func (f *LogCELFilter) Name() string {
	return "Log CEL filter"
}

// Process gathers candidate logs from all currently matching timelines and evaluates the log CEL expression concurrently across workers.
func (f *LogCELFilter) Process(
	ctx context.Context,
	filterCtx *FilterContext,
	index *SearchIndex,
	report ProgressReporter,
) error {
	candidateLogIDsMap := make(map[uint32]struct{})
	for id := range filterCtx.TimelineIDs {
		if tl, ok := index.TimelineMap[id]; ok {
			tl.ForEachLogID(func(logID uint32) bool {
				candidateLogIDsMap[logID] = struct{}{}
				return true
			})
		}
	}

	candidateLogIDs := make([]uint32, 0, len(candidateLogIDsMap))
	for id := range candidateLogIDsMap {
		candidateLogIDs = append(candidateLogIDs, id)
	}

	totalCandidateLogs := uint32(len(candidateLogIDs))
	if totalCandidateLogs == 0 {
		return nil
	}

	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > int(totalCandidateLogs) {
		numWorkers = int(totalCandidateLogs)
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	g, groupCtx := errgroup.WithContext(ctx)
	results := make([][]uint32, numWorkers)
	chunkSize := (int(totalCandidateLogs) + numWorkers - 1) / numWorkers

	var processedCount uint32
	var reportMu sync.Mutex

	for w := 0; w < numWorkers; w++ {
		workerIdx := w
		start := workerIdx * chunkSize
		end := start + chunkSize
		if end > int(totalCandidateLogs) {
			end = int(totalCandidateLogs)
		}
		if start >= end {
			continue
		}

		g.Go(func() error {
			logEval, err := cel.NewLogEvaluator()
			if err != nil {
				return fmt.Errorf("failed to initialize log evaluator: %w", err)
			}
			logEval.SetInternPool(index.InternPool)
			logEval.SetTrigramIndex(index.TrigramIndex)
			logEval.SetStructYAMLs(index.StructYAMLs)
			if err := logEval.Compile(f.query); err != nil {
				return fmt.Errorf("invalid log query: %w", err)
			}

			localMatched := make([]uint32, 0, (end-start)/4)
			for i := start; i < end; i++ {
				select {
				case <-groupCtx.Done():
					return groupCtx.Err()
				default:
				}

				logID := candidateLogIDs[i]
				if logObj := index.GetLog(logID); logObj != nil {
					matched, err := logEval.Evaluate(groupCtx, logObj)
					if err != nil {
						return fmt.Errorf("error evaluating log query: %w", err)
					}
					if matched {
						localMatched = append(localMatched, logID)
					}
				}

				currentProcessed := atomic.AddUint32(&processedCount, 1)
				if currentProcessed%1000 == 0 || currentProcessed == totalCandidateLogs {
					if report != nil {
						reportMu.Lock()
						_ = report(f.Name(), currentProcessed, totalCandidateLogs)
						reportMu.Unlock()
					}
				}
			}
			results[workerIdx] = localMatched
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	for _, localMatched := range results {
		for _, id := range localMatched {
			filterCtx.LogIDs[id] = struct{}{}
		}
	}

	return nil
}
