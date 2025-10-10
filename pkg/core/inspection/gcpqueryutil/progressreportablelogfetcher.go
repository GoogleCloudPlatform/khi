// Copyright 2025 Google LLC
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

package gcpqueryutil

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/logging/apiv2/loggingpb"
)

// LogFetchProgress represents the progress of a log fetching operation.
type LogFetchProgress struct {
	// LogCount is the total number of logs fetched so far.
	LogCount int
	// Progress indicates the completion status, ranging from 0.0 to 1.0.
	Progress float32
}

type ProgressReportableLogFetcher interface {
	// FetchLogsWithProgress fetches logs while periodically reporting its progress through a separate channel.
	// Implementations must close both the dest and progress channels upon completion.
	FetchLogsWithProgress(dest chan<- *loggingpb.LogEntry, progress chan<- LogFetchProgress, ctx context.Context, beginTime, endTime time.Time, filterWithoutTimeRange string, resourceContainers []string) error
}

// StandardProgressReportableLogFetcher is a decorator for a LogFetcher that adds the ability
// to report the progress of log fetching.
type StandardProgressReportableLogFetcher struct {
	fetcher        LogFetcher
	reportInterval time.Duration
}

// NewProgressReportableLogFetcher creates a new instance of ProgressReportableLogFetcher.
func NewStandardProgressReportableLogFetcher(fetcher LogFetcher, interval time.Duration) *StandardProgressReportableLogFetcher {
	return &StandardProgressReportableLogFetcher{
		fetcher:        fetcher,
		reportInterval: interval,
	}
}

// FetchLogsWithProgress implements FetchLogsWithProgress.
func (s *StandardProgressReportableLogFetcher) FetchLogsWithProgress(dest chan<- *loggingpb.LogEntry, progress chan<- LogFetchProgress, ctx context.Context, beginTime, endTime time.Time, filterWithoutTimeRange string, resourceContainers []string) error {
	defer close(dest)
	defer close(progress)

	ticker := time.NewTicker(s.reportInterval)
	defer ticker.Stop() // ticker must be closed before closing progress

	stubChan := make(chan *loggingpb.LogEntry)
	subroutineCtx, cancelSubroutine := context.WithCancel(ctx)

	filter := fmt.Sprintf("%s\n%s", filterWithoutTimeRange, TimeRangeQuerySection(beginTime, endTime, false))

	wg := sync.WaitGroup{}
	wg.Add(2)
	logCount := atomic.Int32{}
	latestLogTime := &beginTime
	totalDurationInSeconds := endTime.Sub(beginTime).Seconds()

	if totalDurationInSeconds == 0 {
		totalDurationInSeconds = 1
	}

	// Consume logs from log fetcher and record count and the latest time for reporting progress
	go func() {
		defer wg.Done() // fetcher.FetchLogs is expected to run in sync. But make sure all the logs are consumed in this go routine.
		for {
			select {
			case <-subroutineCtx.Done():
				return
			case logEntry, ok := <-stubChan:
				if !ok {
					return
				}
				logCount.Add(1)
				t := logEntry.Timestamp.AsTime()
				latestLogTime = &t
				select {
				case <-subroutineCtx.Done():
					return
				case dest <- logEntry:
				}
			}
		}
	}()

	// Report progress for every reportInterval
	go func() {
		defer wg.Done()
		// Send initial progress
		select {
		case progress <- LogFetchProgress{}:
		case <-subroutineCtx.Done():
			return
		}

		for {
			select {
			case <-subroutineCtx.Done():
				return
			case <-ticker.C:
				latestLogTimeFromBeginTimeInSeconds := latestLogTime.Sub(beginTime).Seconds()
				select {
				case progress <- LogFetchProgress{
					LogCount: int(logCount.Load()),
					Progress: float32(latestLogTimeFromBeginTimeInSeconds) / float32(totalDurationInSeconds),
				}:
				case <-subroutineCtx.Done():
					return
				}
			}
		}
	}()

	err := s.fetcher.FetchLogs(stubChan, ctx, filter, resourceContainers)
	if err != nil {
		cancelSubroutine()
		return err
	}

	cancelSubroutine()
	wg.Wait()

	// Send final progress report.
	select {
	case progress <- LogFetchProgress{LogCount: int(logCount.Load()), Progress: 1.0}:
	case <-ctx.Done():
	}
	return nil
}

var _ ProgressReportableLogFetcher = (*StandardProgressReportableLogFetcher)(nil)
