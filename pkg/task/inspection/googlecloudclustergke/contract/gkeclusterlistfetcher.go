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

package googlecloudclustergke_contract

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ClusterListFetcher fetches the list of GKE cluster in the project.
type ClusterListFetcher interface {
	GetClusterNames(ctx context.Context, projectID string, startTime, endTime time.Time) ([]string, error)
}

// ClusterListFetcherImpl is the default implementation of ClusterListFetcher.
type ClusterListFetcherImpl struct{}

// GetClusterNames implements ClusterListFetcher.
// This expects the task googlecloudcommon_contract.APIClientFactoryTaskID is already resolved.
func (g *ClusterListFetcherImpl) GetClusterNames(ctx context.Context, projectID string, startTime, endTime time.Time) ([]string, error) {
	cf := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientFactoryTaskID.Ref())
	injector := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref())

	client, err := cf.MonitoringMetricClient(ctx, googlecloud.Project(projectID))
	if err != nil {
		return nil, fmt.Errorf("failed to create monitoring metric client: %w", err)
	}
	defer client.Close()

	ctx = injector.InjectToCallContext(ctx, googlecloud.Project(projectID))
	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + projectID,
		Filter: `resource.type="k8s_cluster" AND metric.type="logging.googleapis.com/log_entry_count"`,
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamppb.New(startTime),
			EndTime:   timestamppb.New(endTime),
		},
		View: monitoringpb.ListTimeSeriesRequest_HEADERS,
		Aggregation: &monitoringpb.Aggregation{
			AlignmentPeriod:    &durationpb.Duration{Seconds: int64(endTime.Sub(startTime).Seconds())},
			PerSeriesAligner:   monitoringpb.Aggregation_ALIGN_SUM,
			CrossSeriesReducer: monitoringpb.Aggregation_REDUCE_NONE,
			// Grouping by cluster_name to get unique clusters
			GroupByFields: []string{"resource.label.cluster_name"},
		},
	}

	it := client.ListTimeSeries(ctx, req)
	clusterNames := make(map[string]struct{})
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list time series: %w", err)
		}
		if name, ok := resp.GetResource().GetLabels()["cluster_name"]; ok {
			clusterNames[name] = struct{}{}
		}
	}

	result := make([]string, 0, len(clusterNames))
	for name := range clusterNames {
		result = append(result, name)
	}
	return result, nil
}

var _ ClusterListFetcher = (*ClusterListFetcherImpl)(nil)
