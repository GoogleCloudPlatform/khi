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

package logestimator

import (
	"context"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	logging "cloud.google.com/go/logging/apiv2"
	"cloud.google.com/go/logging/apiv2/loggingpb"
	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MetricLogCountFetcher is a function or client capable of querying Cloud Monitoring for log entry counts.
type MetricLogCountFetcher interface {
	QueryMetricCount(ctx context.Context, container googlecloud.ResourceContainer, metricFilter string, startTime, endTime time.Time) (int64, error)
}

// MonitoringClientFetcher implements MetricLogCountFetcher using *monitoring.MetricClient.
type MonitoringClientFetcher struct {
	Client *monitoring.MetricClient
}

// QueryMetricCount queries Cloud Monitoring for logging.googleapis.com/log_entry_count.
func (f *MonitoringClientFetcher) QueryMetricCount(ctx context.Context, container googlecloud.ResourceContainer, metricFilter string, startTime, endTime time.Time) (int64, error) {
	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   container.Identifier(),
		Filter: metricFilter,
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(startTime),
			EndTime:   timestamppb.New(endTime),
		},
		View: monitoringpb.ListTimeSeriesRequest_FULL,
		Aggregation: &monitoringpb.Aggregation{
			CrossSeriesReducer: monitoringpb.Aggregation_REDUCE_SUM,
			PerSeriesAligner:   monitoringpb.Aggregation_ALIGN_SUM,
			AlignmentPeriod:    durationpb.New(60 * time.Second),
		},
	}

	it := f.Client.ListTimeSeries(ctx, req)
	var totalCount int64

	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("failed to query time series: %w", err)
		}
		for _, pt := range resp.GetPoints() {
			if pt == nil || pt.GetValue() == nil {
				continue
			}
			totalCount += pt.GetValue().GetInt64Value()
		}
	}

	return totalCount, nil
}

// LogSamplingProbeFetcher is an interface for probing Cloud Logging sample logs.
type LogSamplingProbeFetcher interface {
	// ProbeLogTimestamps returns timestamps of matching log entries (up to maxEntries, ordered by timestamp desc).
	ProbeLogTimestamps(ctx context.Context, container googlecloud.ResourceContainer, filter string, maxEntries int32) ([]time.Time, error)
}

// LoggingClientProbeFetcher implements LogSamplingProbeFetcher using *logging.Client.
type LoggingClientProbeFetcher struct {
	Client *logging.Client
}

// ProbeLogTimestamps queries recent matching log entries from Cloud Logging to extract timestamps.
func (f *LoggingClientProbeFetcher) ProbeLogTimestamps(ctx context.Context, container googlecloud.ResourceContainer, filter string, maxEntries int32) ([]time.Time, error) {
	req := &loggingpb.ListLogEntriesRequest{
		ResourceNames: []string{container.Identifier()},
		Filter:        filter,
		OrderBy:       "timestamp desc",
		PageSize:      maxEntries,
	}
	it := f.Client.ListLogEntries(ctx, req)
	var timestamps []time.Time
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to probe log entries: %w", err)
		}
		timestamps = append(timestamps, resp.GetTimestamp().AsTime())
		if int32(len(timestamps)) >= maxEntries {
			break
		}
	}
	return timestamps, nil
}

// StructuredLogEstimator estimates log volume for StructuredLogQuery.
type StructuredLogEstimator struct {
	MetricFetcher MetricLogCountFetcher
	ProbeFetcher  LogSamplingProbeFetcher
	TargetSamples int32
}

// NewStructuredLogEstimator creates a StructuredLogEstimator with defaults.
func NewStructuredLogEstimator(metricFetcher MetricLogCountFetcher, probeFetcher LogSamplingProbeFetcher) *StructuredLogEstimator {
	return &StructuredLogEstimator{
		MetricFetcher: metricFetcher,
		ProbeFetcher:  probeFetcher,
		TargetSamples: 50,
	}
}

// NewStructuredLogEstimatorFromClients constructs a StructuredLogEstimator directly from Google Cloud SDK clients.
func NewStructuredLogEstimatorFromClients(loggingClient *logging.Client, metricClient *monitoring.MetricClient) *StructuredLogEstimator {
	var metricFetcher MetricLogCountFetcher
	if metricClient != nil {
		metricFetcher = &MonitoringClientFetcher{Client: metricClient}
	}
	var probeFetcher LogSamplingProbeFetcher
	if loggingClient != nil {
		probeFetcher = &LoggingClientProbeFetcher{Client: loggingClient}
	}
	return NewStructuredLogEstimator(metricFetcher, probeFetcher)
}

// Estimate estimates the total log volume for the given StructuredLogQuery over the time interval.
// Cloud Monitoring queries for all resource types are executed concurrently.
// If custom filters require sampling, a time-window bounded sampling probe is executed.
func (e *StructuredLogEstimator) Estimate(ctx context.Context, container googlecloud.ResourceContainer, query *StructuredLogQuery, startTime, endTime time.Time) (*EstimateResult, error) {
	allSupported := query.AllFiltersSupportMetrics()
	metricFilters := query.GenerateMonitoringMetricFilters()

	sampleSize := e.TargetSamples
	if sampleSize <= 0 {
		sampleSize = 50
	}

	timeFilter := fmt.Sprintf("timestamp >= \"%s\"\ntimestamp <= \"%s\"", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))

	g, groupCtx := errgroup.WithContext(ctx)

	// 1. Fetch Cloud Monitoring metrics concurrently for all resource types.
	var metricCount int64
	for _, filter := range metricFilters {
		f := filter
		g.Go(func() error {
			cnt, err := e.MetricFetcher.QueryMetricCount(groupCtx, container, f, startTime, endTime)
			if err != nil {
				return fmt.Errorf("failed to query metric count for filter %s: %w", f, err)
			}
			atomic.AddInt64(&metricCount, cnt)
			return nil
		})
	}

	// 2. If sampling is needed, concurrently fetch base probe logs (up to sampleSize).
	var baseTimestamps []time.Time
	if !allSupported && e.ProbeFetcher != nil {
		baseQuery := query.GenerateBaseLoggingQuery()
		baseFilter := fmt.Sprintf("%s\n%s", baseQuery, timeFilter)
		g.Go(func() error {
			ts, err := e.ProbeFetcher.ProbeLogTimestamps(groupCtx, container, baseFilter, sampleSize)
			if err != nil {
				return fmt.Errorf("failed to fetch base sample probe: %w", err)
			}
			baseTimestamps = ts
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// 3. Pure metric query (no custom filter, all matchers natively supported in Cloud Monitoring)
	if allSupported {
		return &EstimateResult{
			MetricCount:       metricCount,
			EstimatedCount:    metricCount,
			CustomFilterRatio: 1.0,
			IsExact:           true,
		}, nil
	}

	// 4. Custom filter query but probe fetcher is not configured
	if e.ProbeFetcher == nil {
		return &EstimateResult{
			MetricCount:       metricCount,
			EstimatedCount:    metricCount,
			CustomFilterRatio: 1.0,
			IsExact:           false,
		}, nil
	}

	// 5. If base probe returned 0 logs in the whole interval
	if len(baseTimestamps) == 0 {
		// Probe custom query over the full interval to check if logs exist (e.g. retention > 42 days).
		customQuery := query.GenerateCloudLoggingQuery()
		customFilter := fmt.Sprintf("%s\n%s", customQuery, timeFilter)
		customTimestamps, err := e.ProbeFetcher.ProbeLogTimestamps(ctx, container, customFilter, sampleSize)
		if err != nil {
			return nil, fmt.Errorf("failed to probe custom log count: %w", err)
		}
		if len(customTimestamps) > 0 {
			return &EstimateResult{
				MetricCount:       0,
				EstimatedCount:    int64(len(customTimestamps)),
				CustomFilterRatio: 1.0,
				IsExact:           false,
			}, nil
		}
		return &EstimateResult{
			MetricCount:       0,
			EstimatedCount:    0,
			CustomFilterRatio: 1.0,
			IsExact:           true,
		}, nil
	}

	// 6. Time-window bounded sampling: query custom logs in the exact window [cutoffTime, endTime].
	cutoffTime := baseTimestamps[len(baseTimestamps)-1]
	customQuery := query.GenerateCloudLoggingQuery()
	customWindowFilter := fmt.Sprintf("%s\ntimestamp >= \"%s\"\ntimestamp <= \"%s\"",
		customQuery, cutoffTime.Format(time.RFC3339Nano), endTime.Format(time.RFC3339Nano))

	customTimestamps, err := e.ProbeFetcher.ProbeLogTimestamps(ctx, container, customWindowFilter, sampleSize)
	if err != nil {
		return nil, fmt.Errorf("failed to probe custom window logs: %w", err)
	}

	ratio := float64(len(customTimestamps)) / float64(len(baseTimestamps))
	if ratio > 1.0 {
		ratio = 1.0
	}

	estimatedCount := int64(math.Round(float64(metricCount) * ratio))
	if metricCount == 0 && len(customTimestamps) > 0 {
		estimatedCount = int64(len(customTimestamps))
	}

	return &EstimateResult{
		MetricCount:       metricCount,
		EstimatedCount:    estimatedCount,
		CustomFilterRatio: ratio,
		IsExact:           false,
	}, nil
}
