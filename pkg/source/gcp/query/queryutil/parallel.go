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

package queryutil

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/adapter"
	"github.com/GoogleCloudPlatform/khi/pkg/parser/yaml/yamlutil"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/api"
	gcp_log "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/log"
)

type ParallelQueryWorker struct {
	workerCount int
	baseQuery   string
	startTime   time.Time
	endTime     time.Time
	apiClient   api.GCPClient
	pool        *worker.Pool
}

func NewParallelQueryWorker(pool *worker.Pool, apiClient api.GCPClient, baseQuery string, startTime time.Time, endTime time.Time, workerCount int) *ParallelQueryWorker {
	return &ParallelQueryWorker{
		baseQuery:   baseQuery,
		startTime:   startTime,
		endTime:     endTime,
		workerCount: workerCount,
		apiClient:   apiClient,
		pool:        pool,
	}
}

func (p *ParallelQueryWorker) Query(ctx context.Context, readerFactory *structure.ReaderFactory, projectId string, progress *progress.TaskProgress) ([]*log.LogEntity, error) {
	timeSegments := divideTimeSegments(p.startTime, p.endTime, p.workerCount)
	percentages := make([]float32, p.workerCount)
	logSink := make(chan *log.LogEntity)
	logEntries := []*log.LogEntity{}
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
		subLogSink := make(chan any)
		wg.Add(1)
		p.pool.Run(func() {
			defer wg.Done()
			go func() {
				threadCount.Add(1)
				err := p.apiClient.ListLogEntries(cancellableCtx, projectId, query, subLogSink)
				if err != nil && !errors.Is(err, context.Canceled) {
					slog.WarnContext(cancellableCtx, fmt.Sprintf("query thread failed with an error\n%s", err))
					cancel(err)
				}
			}()
			for logEntryAny := range subLogSink {
				yamlString, err := yamlutil.MarshalToYamlString(logEntryAny)
				if err != nil {
					slog.WarnContext(ctx, "failed to parse a log as YAML. Skipping.")
					continue
				}
				logReader, err := readerFactory.NewReader(adapter.Yaml(yamlString))
				if err != nil {
					slog.WarnContext(ctx, fmt.Sprintf("failed to create reader for log entry\n%s", err))
					continue
				}
				commonLogFieldCache := log.NewCachedLogFieldExtractor(gcp_log.GCPCommonFieldExtractor{})
				commonLogFieldCache.SetLogBodyCacheDirect(yamlString)
				logEntry := log.NewLogEntity(logReader, commonLogFieldCache)
				percentages[workerIndex] = float32(logEntry.Timestamp().Sub(begin)) / float32(end.Sub(begin))
				logSink <- logEntry
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
