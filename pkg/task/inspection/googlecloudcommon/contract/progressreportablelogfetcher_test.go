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
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/logging/apiv2/loggingpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// mockLogFetcher is a mock implementation of the LogFetcher interface for testing purposes.
type mockLogFetcher struct {
	logUpstream func(filter string) chan *loggingpb.LogEntry
	errUpstream func(filter string) chan error
}

// FetchLogs simulates fetching logs from an upstream source.
// It sends logs from `m.logUpstream` and errors from `m.errUpstream` to the `dest` channel.
// It respects context cancellation.
func (m *mockLogFetcher) FetchLogs(dest chan<- *loggingpb.LogEntry, ctx context.Context, filter string, container googlecloud.ResourceContainer, resourceContainers []string) error {
	errUpstream := m.errUpstream(filter)
	logUpstream := m.logUpstream(filter)

	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case err, ok := <-errUpstream:
			if ok {
				return err
			}
		case l, ok := <-logUpstream:
			if !ok {
				close(dest)
				return nil
			}
			dest <- l
		}
	}
}

var _ LogFetcher = (*mockLogFetcher)(nil)

// fakeLogUpstreamPair represents a pair of channels (log and error) for a specific filter,
// used to simulate log fetching in tests.
type fakeLogUpstreamPair struct {
	filter    string
	logChan   chan *loggingpb.LogEntry
	errChan   chan error
	startLock chan struct{}
}

// getMockFetcherFromFakeLogUpstreamPairs creates a mockLogFetcher that uses the provided
// fakeLogUpstreamPair instances to simulate log and error streams based on the filter.
func getMockFetcherFromFakeLogUpstreamPairs(t *testing.T, fakeLogUpstreams []fakeLogUpstreamPair) *mockLogFetcher {
	logUpstream := func(filter string) chan *loggingpb.LogEntry {
		for _, pair := range fakeLogUpstreams {
			if pair.filter == filter {
				<-pair.startLock
				return pair.logChan
			}
		}
		t.Errorf("given filter is not matching any available fake log upstream: %v", filter)
		return nil
	}
	errUpstream := func(filter string) chan error {
		for _, pair := range fakeLogUpstreams {
			if pair.filter == filter {
				return pair.errChan
			}
		}
		t.Errorf("given filter is not matching any available fake log upstream: %v", filter)
		return nil
	}
	return &mockLogFetcher{
		logUpstream: logUpstream,
		errUpstream: errUpstream,
	}
}

// newFakeLogUpstreamPair creates a new fakeLogUpstreamPair.
// It takes a filter string and a function `f` that simulates the log fetching logic,
// sending logs and errors to the provided channels.
func newFakeLogUpstreamPair(filter string, f func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error)) fakeLogUpstreamPair {
	logSource := make(chan *loggingpb.LogEntry)
	errSource := make(chan error)
	startLock := make(chan struct{})
	go func() {
		startLock <- struct{}{}
		f(logSource, errSource)
		<-time.After(10 * time.Millisecond) // wait a short time to prevent to close the channel before the test target processes the last element
		close(logSource)
		close(errSource)
	}()
	return fakeLogUpstreamPair{
		filter:    filter,
		logChan:   logSource,
		errChan:   errSource,
		startLock: startLock,
	}
}

// channelToArrayParallel reads values from a channel of type T and appends them to a slice `resultTo`.
// It runs in a separate goroutine and stops when the context is canceled or the channel is closed.
// It uses a `sync.WaitGroup` to signal completion.
func channelToArrayParallel[T any](ctx context.Context, wg *sync.WaitGroup, resultTo *[]T) chan<- T {
	channel := make(chan T)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case v, open := <-channel:
				if !open {
					return
				}
				*resultTo = append(*resultTo, v)
			}
		}
	}()
	return channel
}

func TestProgressReportableLogFetcher(t *testing.T) {
	tick := 100 * time.Millisecond
	beginTime := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	testErr := errors.New("test error")
	testCases := []struct {
		desc             string
		fetcherFactory   func(t *testing.T) *mockLogFetcher
		duration         time.Duration
		cancelAfter      time.Duration
		wantCompleteTime time.Duration
		wantErr          error
		wantLogs         []*loggingpb.LogEntry
		wantProgresses   []LogFetchProgress
	}{
		{
			desc: "standard result",
			fetcherFactory: func(t *testing.T) *mockLogFetcher {
				return getMockFetcherFromFakeLogUpstreamPairs(t, []fakeLogUpstreamPair{
					newFakeLogUpstreamPair(`test filter
timestamp >= "2025-01-01T00:00:00+0000"
timestamp < "2025-01-01T01:00:00+0000"`, func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
						<-time.After(tick / 2) // delta to prevent freaky test
						<-time.After(tick)     // <- 1st progress tick
						logSource <- &loggingpb.LogEntry{LogName: "foo", Timestamp: timestamppb.New(beginTime.Add(15 * time.Minute))}
						<-time.After(tick) // <- 2nd progress tick
						logSource <- &loggingpb.LogEntry{LogName: "bar", Timestamp: timestamppb.New(beginTime.Add(30 * time.Minute))}
						<-time.After(2 * tick) // <- 3rd & 4th progress tick
					}),
				})
			},
			duration:         time.Hour,
			wantCompleteTime: 5 * tick,
			wantErr:          nil,
			wantLogs: []*loggingpb.LogEntry{
				{
					LogName: "foo", Timestamp: timestamppb.New(beginTime.Add(15 * time.Minute)),
				},
				{
					LogName: "bar", Timestamp: timestamppb.New(beginTime.Add(30 * time.Minute)),
				},
			},
			wantProgresses: []LogFetchProgress{
				{},
				{},
				{LogCount: 1, Progress: 0.25},
				{LogCount: 2, Progress: 0.50},
				{LogCount: 2, Progress: 0.50},
				{LogCount: 2, Progress: 1},
			},
		},
		{
			desc: "with error",
			fetcherFactory: func(t *testing.T) *mockLogFetcher {
				return getMockFetcherFromFakeLogUpstreamPairs(t, []fakeLogUpstreamPair{
					newFakeLogUpstreamPair(`test filter
timestamp >= "2025-01-01T00:00:00+0000"
timestamp < "2025-01-01T01:00:00+0000"`, func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
						<-time.After(tick / 2) // delta to prevent freaky test
						<-time.After(tick)     // <- 1st progress tick
						logSource <- &loggingpb.LogEntry{LogName: "foo", Timestamp: timestamppb.New(beginTime.Add(15 * time.Minute))}
						<-time.After(tick) // <- 2nd progress tick
						errSource <- testErr
						<-time.After(2 * tick)
					}),
				})
			},
			duration:         time.Hour,
			wantCompleteTime: 3 * tick,
			wantErr:          testErr,
			wantLogs: []*loggingpb.LogEntry{
				{
					LogName: "foo", Timestamp: timestamppb.New(beginTime.Add(15 * time.Minute)),
				},
			},
			wantProgresses: []LogFetchProgress{
				{},
				{},
				{LogCount: 1, Progress: 0.25},
			},
		},
		{
			desc: "with cancel",
			fetcherFactory: func(t *testing.T) *mockLogFetcher {
				return getMockFetcherFromFakeLogUpstreamPairs(t, []fakeLogUpstreamPair{
					newFakeLogUpstreamPair(`test filter
timestamp >= "2025-01-01T00:00:00+0000"
timestamp < "2025-01-01T01:00:00+0000"`, func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
						<-time.After(tick / 2) // delta to prevent freaky test
						<-time.After(tick)     // <- 1st progress tick
						logSource <- &loggingpb.LogEntry{LogName: "foo", Timestamp: timestamppb.New(beginTime.Add(15 * time.Minute))}
						<-time.After(tick) // <- 2nd progress tick
						// cancel happens here
						<-time.After(10 * tick)
					}),
				})
			},
			duration:         time.Hour,
			wantCompleteTime: 4 * tick,
			cancelAfter:      time.Duration(2.5 * float64(tick)),
			wantErr:          context.Canceled,
			wantLogs: []*loggingpb.LogEntry{
				{
					LogName: "foo", Timestamp: timestamppb.New(beginTime.Add(15 * time.Minute)),
				},
			},
			wantProgresses: []LogFetchProgress{
				{},
				{},
				{LogCount: 1, Progress: 0.25},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			wg := sync.WaitGroup{}
			endTime := beginTime.Add(tc.duration)
			fetcher := tc.fetcherFactory(t)

			var logs []*loggingpb.LogEntry
			var progresses []LogFetchProgress
			logReceiveChan := channelToArrayParallel(t.Context(), &wg, &logs)
			progressReceiveChan := channelToArrayParallel(t.Context(), &wg, &progresses)

			progressReportableFetcher := NewStandardProgressReportableLogFetcher(fetcher, tick)

			afterFetchDone := make(chan struct{})
			go func() {
				select {
				case <-time.After(tc.wantCompleteTime):
					t.Errorf("FetchLogWithProgress didn't return within expected completion time %d ms. ", tc.wantCompleteTime.Microseconds())
				case <-afterFetchDone:
					return
				}
			}()

			cancellableCtx, cancel := context.WithCancel(t.Context())
			if tc.cancelAfter != 0 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-time.After(tc.cancelAfter)
					cancel()
				}()
			}

			err := progressReportableFetcher.FetchLogsWithProgress(logReceiveChan, progressReceiveChan, cancellableCtx, beginTime, endTime, "test filter", googlecloud.Project("foobar"), []string{})
			if tc.wantErr == nil && err != nil {
				t.Errorf("FetchLogsWithProgress() returned unexpected error: %v", err)
			}
			if tc.wantErr != nil && err == nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("FetchLogsWithProgress() didn't return expected error. Returned error: %v", err)
			}
			wg.Wait()
			close(afterFetchDone)

			slices.SortFunc(logs, func(a, b *loggingpb.LogEntry) int { return a.Timestamp.AsTime().Compare(b.Timestamp.AsTime()) })

			if diff := cmp.Diff(tc.wantLogs, logs, protocmp.Transform(), cmpopts.IgnoreUnexported()); diff != "" {
				t.Errorf("FetchLogsWithProgress() produced non expected result: (-want, +got):\n%v", diff)
			}
			if diff := cmp.Diff(tc.wantProgresses, progresses); diff != "" {
				t.Errorf("FetchLogsWithProgress() produced non expected progress: (-want, +got):\n%v", diff)
			}
			cancel()
		})
	}
}

func TestTimePartitioningProgressReportableLogFetcher_FetchLogsWithProgress(t *testing.T) {
	beginTime := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	tick := 300 * time.Millisecond
	testErr := errors.New("test error")
	testCases := []struct {
		desc             string
		fetcherFactory   func(t *testing.T) *mockLogFetcher
		duration         time.Duration
		partitionCount   int
		maxParallelism   int
		cancelAfter      time.Duration
		wantCompleteTime time.Duration
		wantProgress     []LogFetchProgress
		wantLogs         []*loggingpb.LogEntry
		wantErr          error
	}{
		{
			desc: "2 partition with 2 parallelism",
			fetcherFactory: func(t *testing.T) *mockLogFetcher {
				return getMockFetcherFromFakeLogUpstreamPairs(t, []fakeLogUpstreamPair{
					newFakeLogUpstreamPair(`test filter
timestamp >= "2025-01-01T00:30:00+0000"
timestamp < "2025-01-01T01:00:00+0000"`, func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
						<-time.After(tick / 4)
						<-time.After(tick)
						logSource <- &loggingpb.LogEntry{LogName: "foo", Timestamp: timestamppb.New(beginTime.Add(time.Minute * 45))}
						<-time.After(tick)
					}),
					newFakeLogUpstreamPair(`test filter
timestamp >= "2025-01-01T00:00:00+0000"
timestamp < "2025-01-01T00:30:00+0000"`, func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
						<-time.After(tick / 4)
						<-time.After(tick)
						logSource <- &loggingpb.LogEntry{LogName: "bar", Timestamp: timestamppb.New(beginTime.Add(time.Minute * 15))}
						<-time.After(tick)
					}),
				})
			},
			duration:         time.Hour,
			partitionCount:   2,
			maxParallelism:   2,
			wantCompleteTime: 3 * tick,
			wantProgress: []LogFetchProgress{
				{},
				{},
				{
					LogCount: 2,
					Progress: 0.5, // still the both channels are not closed. progress is calculated from log timestamp.
				},
				{
					LogCount: 2,
					Progress: 1, // both channels are closed. all sub progresses are 1.
				},
			},
			wantLogs: []*loggingpb.LogEntry{
				{
					LogName: "bar", Timestamp: timestamppb.New(beginTime.Add(time.Minute * 15)),
				},
				{
					LogName: "foo", Timestamp: timestamppb.New(beginTime.Add(time.Minute * 45)),
				},
			},
		},
		{
			desc: "2 partition with 1 parallelism",
			fetcherFactory: func(t *testing.T) *mockLogFetcher {
				return getMockFetcherFromFakeLogUpstreamPairs(t, []fakeLogUpstreamPair{
					newFakeLogUpstreamPair(`test filter
timestamp >= "2025-01-01T00:30:00+0000"
timestamp < "2025-01-01T01:00:00+0000"`, func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
						<-time.After(tick / 8)
						<-time.After(tick)
						logSource <- &loggingpb.LogEntry{LogName: "foo", Timestamp: timestamppb.New(beginTime.Add(time.Minute * 45))}
						<-time.After(tick)
					}),
					newFakeLogUpstreamPair(`test filter
timestamp >= "2025-01-01T00:00:00+0000"
timestamp < "2025-01-01T00:30:00+0000"`, func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
						<-time.After(tick / 8)
						<-time.After(tick)
						logSource <- &loggingpb.LogEntry{LogName: "bar", Timestamp: timestamppb.New(beginTime.Add(time.Minute * 15))}
						<-time.After(tick)
					}),
				})
			},
			duration:         time.Hour,
			partitionCount:   2,
			maxParallelism:   1,
			wantCompleteTime: 5 * tick,
			wantProgress: []LogFetchProgress{
				{},
				{},
				{LogCount: 1, Progress: 0.25},
				{LogCount: 1, Progress: 0.5},
				{LogCount: 2, Progress: 0.75},
				{LogCount: 2, Progress: 1},
			},
			wantLogs: []*loggingpb.LogEntry{
				{
					LogName: "bar", Timestamp: timestamppb.New(beginTime.Add(time.Minute * 15)),
				},
				{
					LogName: "foo", Timestamp: timestamppb.New(beginTime.Add(time.Minute * 45)),
				},
			},
		},
		{
			desc: "2 partition with 2 parallelism with error",
			fetcherFactory: func(t *testing.T) *mockLogFetcher {
				return getMockFetcherFromFakeLogUpstreamPairs(t, []fakeLogUpstreamPair{
					newFakeLogUpstreamPair(`test filter
timestamp >= "2025-01-01T00:30:00+0000"
timestamp < "2025-01-01T01:00:00+0000"`, func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
						<-time.After(tick / 4)
						<-time.After(tick)
						logSource <- &loggingpb.LogEntry{LogName: "foo", Timestamp: timestamppb.New(beginTime.Add(time.Minute * 45))}
						<-time.After(tick)
						errSource <- testErr
						<-time.After(10 * tick) // shouldn't wait channel close after error happen
					}),
					newFakeLogUpstreamPair(`test filter
timestamp >= "2025-01-01T00:00:00+0000"
timestamp < "2025-01-01T00:30:00+0000"`, func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
						<-time.After(tick / 4)
						<-time.After(tick)
						logSource <- &loggingpb.LogEntry{LogName: "bar", Timestamp: timestamppb.New(beginTime.Add(time.Minute * 15))}
						<-time.After(2 * tick)
						logSource <- &loggingpb.LogEntry{LogName: "never", Timestamp: timestamppb.New(beginTime.Add(time.Minute * 20))}
					}),
				})
			},
			duration:         time.Hour,
			partitionCount:   2,
			maxParallelism:   2,
			wantCompleteTime: 5 * tick,
			wantErr:          testErr,
			wantProgress: []LogFetchProgress{
				{},
				{},
				{LogCount: 2, Progress: 0.5},
			},
			wantLogs: []*loggingpb.LogEntry{
				{
					LogName: "bar", Timestamp: timestamppb.New(beginTime.Add(time.Minute * 15)),
				},
				{
					LogName: "foo", Timestamp: timestamppb.New(beginTime.Add(time.Minute * 45)),
				},
			},
		},
		{
			desc: "3 partition with 2 parallelism with cancel",
			fetcherFactory: func(t *testing.T) *mockLogFetcher {
				return getMockFetcherFromFakeLogUpstreamPairs(t, []fakeLogUpstreamPair{
					newFakeLogUpstreamPair(`test filter
timestamp >= "2025-01-01T00:20:00+0000"
timestamp < "2025-01-01T00:40:00+0000"`, func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
						<-time.After(tick / 4)
						<-time.After(tick)
						logSource <- &loggingpb.LogEntry{LogName: "foo", Timestamp: timestamppb.New(beginTime.Add(time.Minute * 30))}
						<-time.After(tick)
						<-time.After(10 * tick) // shouldn't wait channel close after cancel happen
					}),
					newFakeLogUpstreamPair(`test filter
timestamp >= "2025-01-01T00:00:00+0000"
timestamp < "2025-01-01T00:20:00+0000"`, func(logSource chan<- *loggingpb.LogEntry, errSource chan<- error) {
						<-time.After(tick / 4)
						<-time.After(tick)
						logSource <- &loggingpb.LogEntry{LogName: "bar", Timestamp: timestamppb.New(beginTime.Add(time.Minute * 10))}
						<-time.After(2 * tick)
						logSource <- &loggingpb.LogEntry{LogName: "never", Timestamp: timestamppb.New(beginTime.Add(time.Minute * 20))}
					}),
				})
			},
			duration:         time.Hour,
			partitionCount:   3,
			maxParallelism:   2,
			cancelAfter:      time.Duration(2*tick + tick/4),
			wantCompleteTime: 5 * tick,
			wantErr:          context.Canceled,
			wantProgress: []LogFetchProgress{
				{},
				{},
				{LogCount: 2, Progress: float32(1) / float32(3)},
			},
			wantLogs: []*loggingpb.LogEntry{
				{
					LogName: "bar", Timestamp: timestamppb.New(beginTime.Add(time.Minute * 10)),
				},
				{
					LogName: "foo", Timestamp: timestamppb.New(beginTime.Add(time.Minute * 30)),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			wg := sync.WaitGroup{}
			endTime := beginTime.Add(tc.duration)
			fetcher := tc.fetcherFactory(t)

			var logs []*loggingpb.LogEntry
			var progresses []LogFetchProgress
			logReceiveChan := channelToArrayParallel(t.Context(), &wg, &logs)
			progressReceiveChan := channelToArrayParallel(t.Context(), &wg, &progresses)

			progressReportableFetcher := NewTimePartitioningProgressReportableLogFetcher(fetcher, tick/2, tc.partitionCount, tc.maxParallelism)
			progressReportableFetcher.reportInterval = tick // To make this test stable, the parent progress reporter tick interval is 2 times longer than its child reporter.

			afterFetchDone := make(chan struct{})
			go func() {
				select {
				case <-time.After(tc.wantCompleteTime):
					t.Errorf("FetchLogWithProgress didn't return within expected completion time %d ms. ", tc.wantCompleteTime.Microseconds())
				case <-afterFetchDone:
					return
				}
			}()

			cancellableCtx, cancel := context.WithCancel(t.Context())
			if tc.cancelAfter != 0 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-time.After(tc.cancelAfter)
					cancel()
				}()
			}

			err := progressReportableFetcher.FetchLogsWithProgress(logReceiveChan, progressReceiveChan, cancellableCtx, beginTime, endTime, "test filter", googlecloud.Project("foobar"), []string{})
			if tc.wantErr != nil {
				if err == nil {
					t.Errorf("FetchLogsWithProgress() expected error, but got nil")
				} else if !errors.Is(err, tc.wantErr) && err.Error() != tc.wantErr.Error() {
					t.Errorf("FetchLogsWithProgress() returned wrong error type. got: %v, want: %v", err, tc.wantErr)
				}
			} else if err != nil {
				t.Errorf("FetchLogsWithProgress() returned unexpected error: %v", err)
			}
			wg.Wait()
			close(afterFetchDone)

			slices.SortFunc(logs, func(a, b *loggingpb.LogEntry) int { return a.Timestamp.AsTime().Compare(b.Timestamp.AsTime()) })

			if diff := cmp.Diff(tc.wantLogs, logs, protocmp.Transform(), cmpopts.IgnoreUnexported()); diff != "" {
				t.Errorf("FetchLogsWithProgress() produced non expected result: (-want, +got):\n%v", diff)
			}
			if diff := cmp.Diff(tc.wantProgress, progresses); diff != "" {
				t.Errorf("FetchLogsWithProgress() produced non expected progress: (-want, +got):\n%v", diff)
			}
			cancel()
		})
	}
}
