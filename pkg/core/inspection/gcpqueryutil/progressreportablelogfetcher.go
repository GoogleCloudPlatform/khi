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

type TimePartitioningProgressReportableLogFetcher struct {
	client         *StandardProgressReportableLogFetcher
	partitionCount int
	maxParallelism int
	reportInterval time.Duration
}

func NewTimePartitioningProgressReportableLogFetcher(fetcher LogFetcher, interval time.Duration, partitionCount int, maxParallelism int) *TimePartitioningProgressReportableLogFetcher {
	return &TimePartitioningProgressReportableLogFetcher{
		client:         NewStandardProgressReportableLogFetcher(fetcher, interval),
		partitionCount: partitionCount,
		maxParallelism: maxParallelism,
		reportInterval: interval,
	}
}

// FetchLogsWithProgress implements ProgressReportableLogFetcher.
func (t *TimePartitioningProgressReportableLogFetcher) FetchLogsWithProgress(logChan chan<- *loggingpb.LogEntry, progressChan chan<- LogFetchProgress, ctx context.Context, beginTime time.Time, endTime time.Time, filterWithoutTimeRange string, resourceContainers []string) error {
	defer close(logChan)
	defer close(progressChan)

	ticker := time.NewTicker(t.reportInterval)
	defer ticker.Stop()

	subErrChan := make(chan error, 1)
	semaphore := make(chan struct{}, t.maxParallelism)
	defer close(subErrChan)
	defer close(semaphore)

	select {
	case progressChan <- LogFetchProgress{
		LogCount: 0,
		Progress: 0,
	}:
	case <-ctx.Done():
		return ctx.Err()
	}

	subProgresses := make([]LogFetchProgress, t.partitionCount)
	cancellableCtx, cancel := context.WithCancel(ctx)
	rootGoroutineWaitGroup := sync.WaitGroup{}
	rootGoroutineWaitGroup.Add(1)

	go func() {
		defer rootGoroutineWaitGroup.Done()
		for {
			select {
			case <-cancellableCtx.Done():
				return
			case <-ticker.C:
				result := LogFetchProgress{}
				for _, subProgress := range subProgresses {
					result.LogCount += subProgress.LogCount
					result.Progress += subProgress.Progress / float32(t.partitionCount)
				}
				progressChan <- result
			}
		}
	}()

	times := t.getPartitionedTimes(beginTime, endTime)

	wg := sync.WaitGroup{}
	wg.Add(t.partitionCount)
	for i := 0; i < t.partitionCount; i++ {
		partitionBeginTime := times[i]
		partitionEndTime := times[i+1]

		select {
		case <-cancellableCtx.Done():
			wg.Done()
			continue
		case semaphore <- struct{}{}:
		}

		go func(subProgressIndex int) {
			defer wg.Done()
			defer func() {
				<-semaphore
			}()
			childWg := sync.WaitGroup{}
			childWg.Add(2)

			subLogChan := make(chan *loggingpb.LogEntry)
			subProgressChan := make(chan LogFetchProgress)

			// Consume the subLogChan and route the log to the parent channel.
			go func() {
				defer childWg.Done()
				for {
					select {
					case <-cancellableCtx.Done():
						return
					case logEntry, ok := <-subLogChan:
						if !ok {
							return
						}
						select {
						case logChan <- logEntry:
						case <-cancellableCtx.Done():
							return
						}
					}
				}
			}()

			// Consume the subProgressChan and store it to the progress array.
			go func(subProgressIndex int) {
				defer childWg.Done()
				for {
					select {
					case <-cancellableCtx.Done():
						return
					case progress, ok := <-subProgressChan:
						if !ok {
							return
						}
						subProgresses[subProgressIndex] = progress
					}
				}
			}(subProgressIndex)

			err := t.client.FetchLogsWithProgress(subLogChan, subProgressChan, cancellableCtx, partitionBeginTime, partitionEndTime, filterWithoutTimeRange, resourceContainers)
			if err != nil {
				select {
				case subErrChan <- err:
				default: // When an error happens, the other subroutine will finish with cancelled error. And receiver only receive the first error, thus ignore the error if no receiver active.
				}
				cancel()
				return
			}
			childWg.Wait()
		}(i)
	}

	wg.Wait()
	cancel()
	rootGoroutineWaitGroup.Wait()
	if len(subErrChan) > 0 {
		return <-subErrChan
	}
	sumLog := 0
	for _, subProgress := range subProgresses {
		sumLog += subProgress.LogCount
	}
	select {
	case progressChan <- LogFetchProgress{
		LogCount: sumLog,
		Progress: 1,
	}:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func (t *TimePartitioningProgressReportableLogFetcher) getPartitionedTimes(beginTime, endTime time.Time) []time.Time {
	return divideTimeSegments(beginTime, endTime, t.partitionCount)
}

var _ ProgressReportableLogFetcher = (*TimePartitioningProgressReportableLogFetcher)(nil)
