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

package googlecloud

import (
	"context"
	"fmt"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// QueryDistinctLabelValuesFromMetrics queries Cloud Monitoring for TimeSeries matching the filter and interval,
// and returns unique values for the specified label key.
//
// groupByKey: The full label key to group by (e.g. "resource.label.cluster_name").
// resultLabelKey: The simple label key to extract from the result (e.g. "cluster_name").
func QueryDistinctLabelValuesFromMetrics(ctx context.Context, client *monitoring.MetricClient, projectID string, filter string, startTime, endTime time.Time, groupByKey, resultLabelKey string) ([]string, error) {
	d := endTime.Sub(startTime)
	if d < 60*time.Second {
		d = 60 * time.Second
	}
	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + projectID,
		Filter: filter,
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(startTime),
			EndTime:   timestamppb.New(endTime),
		},
		View: monitoringpb.ListTimeSeriesRequest_HEADERS,
		Aggregation: &monitoringpb.Aggregation{
			AlignmentPeriod:    &durationpb.Duration{Seconds: int64(d.Seconds())},
			PerSeriesAligner:   monitoringpb.Aggregation_ALIGN_SUM,
			CrossSeriesReducer: monitoringpb.Aggregation_REDUCE_NONE,
			GroupByFields:      []string{groupByKey},
		},
	}

	it := client.ListTimeSeries(ctx, req)
	uniqueValues := make(map[string]struct{})
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list time series: %w", err)
		}

		// Attempt to find the label in Resource or Metric labels
		val, ok := resp.GetResource().GetLabels()[resultLabelKey]
		if !ok {
			val, ok = resp.GetMetric().GetLabels()[resultLabelKey]
		}
		if ok {
			uniqueValues[val] = struct{}{}
		}
	}

	result := make([]string, 0, len(uniqueValues))
	for v := range uniqueValues {
		result = append(result, v)
	}
	return result, nil
}
