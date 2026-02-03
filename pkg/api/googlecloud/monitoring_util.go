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
	"google.golang.org/protobuf/types/known/timestamppb"
)

// QueryDistinctStringLabelValuesFromMetrics queries Cloud Monitoring for TimeSeries matching the filter and interval,
// and returns unique values for the specified label key.
//
// groupByKey: The full label key to group by (e.g. "resource.label.cluster_name").
// resultLabelKey: The simple label key to extract from the result (e.g. "cluster_name").
func QueryDistinctStringLabelValuesFromMetrics(ctx context.Context, client *monitoring.MetricClient, projectID string, filter string, startTime, endTime time.Time, groupByKey, resultLabelKey string) ([]string, error) {
	labels, err := QueryResourceLabelsFromMetrics(ctx, client, projectID, filter, startTime, endTime, []string{groupByKey})
	if err != nil {
		return nil, err
	}
	uniqueValues := make(map[string]struct{})
	for _, label := range labels {
		if val, ok := label[resultLabelKey]; ok {
			uniqueValues[val] = struct{}{}
		}
	}
	result := make([]string, 0, len(uniqueValues))
	for v := range uniqueValues {
		result = append(result, v)
	}
	return result, nil
}

// QueryResourceLabelsFromMetrics queries Cloud Monitoring for TimeSeries matching the filter and interval,
// and returns the list of resource labels map.
//
// groupByKey: The full label key to group by (e.g. "resource.label.cluster_name").
// resultLabelKey: The simple label key to extract from the result (e.g. "cluster_name").
func QueryResourceLabelsFromMetrics(ctx context.Context, client *monitoring.MetricClient, projectID string, filter string, startTime, endTime time.Time, groupByKey []string) ([]map[string]string, error) {
	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + projectID,
		Filter: filter,
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(startTime),
			EndTime:   timestamppb.New(endTime),
		},
		View: monitoringpb.ListTimeSeriesRequest_HEADERS,
		Aggregation: &monitoringpb.Aggregation{
			CrossSeriesReducer: monitoringpb.Aggregation_REDUCE_SUM,
			GroupByFields:      groupByKey,
		},
	}
	it := client.ListTimeSeries(ctx, req)
	resultValues := make([]map[string]string, 0)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list time series: %w", err)
		}
		resultValues = append(resultValues, resp.GetResource().GetLabels())
	}
	return resultValues, nil
}
