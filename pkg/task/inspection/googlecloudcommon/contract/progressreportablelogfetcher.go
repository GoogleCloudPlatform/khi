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

package googlecloudcommon_contract

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/logging/apiv2/loggingpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logconvert"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"golang.org/x/sync/errgroup"
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
	// It closes the progress channel upon completion and returns timestamp-sorted logs.
	FetchLogsWithProgress(progress chan<- LogFetchProgress, ctx context.Context, beginTime, endTime time.Time, filterWithoutTimeRange string, container googlecloud.ResourceContainer, resourceContainers []string) ([]*log.Log, error)
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
func (s *StandardProgressReportableLogFetcher) FetchLogsWithProgress(dest chan<- *loggingpb.LogEntry, progress chan<- LogFetchProgress, ctx context.Context, beginTime, endTime time.Time, filterWithoutTimeRange string, container googlecloud.ResourceContainer, resourceContainers []string) error {
	defer close(dest)
	defer close(progress)

	ticker := time.NewTicker(s.reportInterval)
	defer ticker.Stop() // ticker must be closed before closing progress

	stubChan := make(chan *loggingpb.LogEntry)
	subroutineCtx, cancelSubroutine := context.WithCancel(ctx)

	filter := fmt.Sprintf("%s\n%s", filterWithoutTimeRange, gcpqueryutil.TimeRangeQuerySection(beginTime, endTime, false))

	wg := sync.WaitGroup{}
	wg.Add(2)
	logCount := atomic.Int32{}
	latestLogTime := atomic.Pointer[time.Time]{}
	latestLogTime.Store(&beginTime)
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
				latestLogTime.Store(&t)
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
				latest := latestLogTime.Load()
				latestLogTimeFromBeginTimeInSeconds := latest.Sub(beginTime).Seconds()
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

	err := s.fetcher.FetchLogs(stubChan, ctx, filter, container, resourceContainers)
	if err != nil {
		cancelSubroutine()
		wg.Wait()
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
func (t *TimePartitioningProgressReportableLogFetcher) FetchLogsWithProgress(progressChan chan<- LogFetchProgress, ctx context.Context, beginTime time.Time, endTime time.Time, filterWithoutTimeRange string, container googlecloud.ResourceContainer, resourceContainers []string) ([]*log.Log, error) {
	defer close(progressChan)

	ticker := time.NewTicker(t.reportInterval)
	defer ticker.Stop()

	select {
	case progressChan <- LogFetchProgress{
		LogCount: 0,
		Progress: 0,
	}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	var subProgressMu sync.Mutex
	idGen, idErr := khictx.GetValue(ctx, inspectioncore_contract.IDGenerator)
	if idErr != nil {
		idGen = id.NewGenerator()
	}
	subProgresses := make([]LogFetchProgress, t.partitionCount)
	partitionLogs := make([][]*log.Log, t.partitionCount)
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
				subProgressMu.Lock()
				for _, subProgress := range subProgresses {
					result.LogCount += subProgress.LogCount
					result.Progress += subProgress.Progress / float32(t.partitionCount)
				}
				subProgressMu.Unlock()
				progressChan <- result
			}
		}
	}()

	times := t.getPartitionedTimes(beginTime, endTime)

	wg, groupCtx := errgroup.WithContext(cancellableCtx)
	wg.SetLimit(t.maxParallelism)

	for i := 0; i < t.partitionCount; i++ {
		subProgressIndex := i
		wg.Go(func() error {
			select {
			case <-groupCtx.Done():
				return groupCtx.Err()
			default:
			}
			partitionBeginTime := times[subProgressIndex]
			partitionEndTime := times[subProgressIndex+1]

			childWg := sync.WaitGroup{}
			childWg.Add(2)

			subLogChan := make(chan *loggingpb.LogEntry)
			subProgressChan := make(chan LogFetchProgress)
			subLogs := make([]*log.Log, 0)

			// Consume the subLogChan, convert proto to *log.Log in parallel, and append to subLogs.
			go func() {
				defer childWg.Done()
				for {
					select {
					case <-groupCtx.Done():
						return
					case logEntry, ok := <-subLogChan:
						if !ok {
							return
						}
						node, err := logconvert.LogEntryToNode(logEntry)
						if err != nil {
							slog.WarnContext(groupCtx, fmt.Sprintf("failed to convert loggingpb.LogEntry (insertId: %s, timestamp: %v) to structured.Node %v", logEntry.InsertId, logEntry.Timestamp, err))
							continue
						}
						ts := time.Time{}
						if logEntry.Timestamp != nil {
							ts = logEntry.Timestamp.AsTime()
						}
						khiLog := log.NewLogWithTimestamp(idGen, structured.NewNodeReader(structured.WithKeyOrder(node, logconvert.GCPLogEntryKeyOrder...)), ts)
						subLogs = append(subLogs, khiLog)
					}
				}
			}()

			// Consume the subProgressChan and store it to the progress array.
			go func() {
				defer childWg.Done()
				for {
					select {
					case <-groupCtx.Done():
						return
					case progress, ok := <-subProgressChan:
						if !ok {
							return
						}
						subProgressMu.Lock()
						subProgresses[subProgressIndex] = progress
						subProgressMu.Unlock()
					}
				}
			}()

			err := t.client.FetchLogsWithProgress(subLogChan, subProgressChan, cancellableCtx, partitionBeginTime, partitionEndTime, filterWithoutTimeRange, container, resourceContainers)
			childWg.Wait()
			partitionLogs[subProgressIndex] = subLogs
			if err != nil {
				cancel()
				return err
			}
			return nil
		})
	}

	err := wg.Wait()
	cancel()
	rootGoroutineWaitGroup.Wait()

	totalLogs := 0
	for _, slice := range partitionLogs {
		totalLogs += len(slice)
	}
	mergedLogs := make([]*log.Log, 0, totalLogs)
	for _, slice := range partitionLogs {
		mergedLogs = append(mergedLogs, slice...)
	}

	if err != nil {
		return mergedLogs, err
	}
	sumLog := 0
	subProgressMu.Lock()
	for _, subProgress := range subProgresses {
		sumLog += subProgress.LogCount
	}
	subProgressMu.Unlock()
	select {
	case progressChan <- LogFetchProgress{
		LogCount: sumLog,
		Progress: 1,
	}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return mergedLogs, nil
}

func (t *TimePartitioningProgressReportableLogFetcher) getPartitionedTimes(beginTime, endTime time.Time) []time.Time {
	return divideTimeSegments(beginTime, endTime, t.partitionCount)
}

var _ ProgressReportableLogFetcher = (*TimePartitioningProgressReportableLogFetcher)(nil)

// divideTimeSegments divides a given time range into count of partitioned time segments.
// First element is the begin time of the first segment, the last element is the end time of the last segment, otherwise the nth time.Time is (n-1)th begin time and n th end time.
func divideTimeSegments(startTime time.Time, endTime time.Time, count int) []time.Time {
	duration := endTime.Sub(startTime)
	subIntervalDuration := duration / time.Duration(count)
	subIntervals := make([]time.Time, count+1)
	currentStart := startTime
	for i := range subIntervals {
		subIntervals[i] = currentStart
		currentStart = currentStart.Add(subIntervalDuration)
	}
	subIntervals[len(subIntervals)-1] = endTime
	return subIntervals
}
