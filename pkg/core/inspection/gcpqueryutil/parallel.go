// Copyright 2024 Google LLC
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
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	googlecloudapi "github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

// ParallelCloudLoggingListWorker executes a Cloud Logging query in parallel by dividing the time range.
type ParallelCloudLoggingListWorker struct {
	workerCount int
	baseQuery   string
	startTime   time.Time
	endTime     time.Time
	apiClient   googlecloudapi.GCPClient
	pool        *worker.Pool
}

// NewParallelCloudLoggingListWorker creates a new ParallelCloudLoggingListWorker.
func NewParallelCloudLoggingListWorker(pool *worker.Pool, apiClient googlecloudapi.GCPClient, baseQuery string, startTime time.Time, endTime time.Time, workerCount int) *ParallelCloudLoggingListWorker {
	return &ParallelCloudLoggingListWorker{
		baseQuery:   baseQuery,
		startTime:   startTime,
		endTime:     endTime,
		workerCount: workerCount,
		apiClient:   apiClient,
		pool:        pool,
	}
}

// Query executes the query and returns the log entries.
func (p *ParallelCloudLoggingListWorker) Query(ctx context.Context, resourceNames []string, progress *inspectionmetadata.TaskProgressMetadata) ([]*log.Log, error) {
	timeSegments := divideTimeSegments(p.startTime, p.endTime, p.workerCount)
	percentages := make([]float32, p.workerCount)
	logSink := make(chan *log.Log)
	logEntries := []*log.Log{}
	wg := sync.WaitGroup{}
	queryStartTime := time.Now()
	threadCount := atomic.Int32{}
	threadCount.Add(1)
	go func() {
		cancellable, cancel := context.WithCancel(ctx)
		go func() {
			for {
				select {
				case <-cancellable.Done():
					return
				case <-time.After(time.Second):
					currentTime := time.Now()

					speed := float64(len(logEntries)) / currentTime.Sub(queryStartTime).Seconds()
					s := float32(0)
					for _, p := range percentages {
						s += p
					}
					progressRatio := s / float32(len(percentages))
					progress.Update(progressRatio, fmt.Sprintf("%.2f lps(concurrency %d)", speed, threadCount.Load()))
				}
			}
		}()
		for logEntry := range logSink {
			logEntries = append(logEntries, logEntry)
		}
		cancel()
	}()

	cancellableCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(errors.New("query completed"))

	for i := 0; i < len(timeSegments)-1; i++ {
		workerIndex := i
		begin := timeSegments[i]
		end := timeSegments[i+1]
		includeEnd := i == len(timeSegments)-1
		query := fmt.Sprintf("%s\n%s", p.baseQuery, TimeRangeQuerySection(begin, end, includeEnd))
		subLogSink := make(chan *log.Log)
		wg.Add(1)
		p.pool.Run(func() {
			defer wg.Done()
			go func() {
				threadCount.Add(1)
				err := p.apiClient.ListLogEntries(cancellableCtx, resourceNames, query, subLogSink)
				if err != nil && !errors.Is(err, context.Canceled) {
					slog.WarnContext(cancellableCtx, fmt.Sprintf("query thread failed with an error\n%s", err))
					cancel(err)
				}
			}()
			for l := range subLogSink {
				err := l.SetFieldSetReader(&GCPCommonFieldSetReader{})
				if err != nil {
					slog.WarnContext(ctx, fmt.Sprintf("failed to read CommonFieldSet from obtained log %s", err.Error()))
					continue
				}
				err = l.SetFieldSetReader(&GCPMainMessageFieldSetReader{})
				if err != nil {
					slog.WarnContext(ctx, fmt.Sprintf("failed to read MainMessageFieldSet from obtained log %s", err.Error()))
					continue
				}
				commonFieldSet, err := log.GetFieldSet(l, &log.CommonFieldSet{})
				if err != nil {
					slog.WarnContext(ctx, fmt.Sprintf("failed to read GCPCommonFieldSet from obtained log %s", err.Error()))
					continue
				}
				percentages[workerIndex] = float32(commonFieldSet.Timestamp.Sub(begin)) / float32(end.Sub(begin))
				logSink <- l
			}
			percentages[workerIndex] = 1
			threadCount.Add(-1)
		})
		if errors.Is(cancellableCtx.Err(), context.Canceled) {
			break
		}
		// To avoid being rate limited by accessing all at once, the access timing is shifted by 3000ms.
		<-time.After(time.Second * 3)
	}
	wg.Wait()
	close(logSink)
	err := context.Cause(cancellableCtx)
	if err != nil {
		cancel(err)
		return nil, err
	}
	cancel(nil)
	return logEntries, nil
}

func divideTimeSegments(startTime time.Time, endTime time.Time, count int) []time.Time {
	duration := endTime.Sub(startTime)
	sub_interval_duration := duration / time.Duration(count)

	sub_intervals := make([]time.Time, count+1)
	current_start := startTime
	for i := range sub_intervals {
		sub_intervals[i] = current_start
		current_start = current_start.Add(sub_interval_duration)
	}
	sub_intervals[len(sub_intervals)-1] = endTime
	return sub_intervals
}
