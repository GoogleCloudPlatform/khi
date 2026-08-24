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
	"os"
	"strings"
	"testing"
	"time"

	logging "cloud.google.com/go/logging/apiv2"
	"cloud.google.com/go/logging/apiv2/loggingpb"
	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"github.com/GoogleCloudPlatform/khi/internal/testflags"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/iterator"
)

func TestStructuredLogQuery_Generation(t *testing.T) {
	tests := []struct {
		name                    string
		query                   *StructuredLogQuery
		wantLoggingQuery        string
		wantMetricFilters       []string
		wantAllFiltersSupported bool
	}{
		{
			name: "k8s_container query with pod substring and namespace exclusion",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_container"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("cluster_name", Exact("my-cluster")),
					ResourceLabel("location", Exact("us-central1")),
					ResourceLabel("namespace_name", FromSetFilter(&gcpqueryutil.SetFilterParseResult{
						SubtractMode: true,
						Subtractives: []string{"kube-system", "istio-system"},
					}, false)),
					ResourceLabel("pod_name", FromSetFilter(&gcpqueryutil.SetFilterParseResult{
						Additives: []string{"nginx", "redis"},
					}, true)),
					ResourceLabel("project_id", Exact("my-project")),
					LogID(NoneOf("server-accesslog-stackdriver", "client-accesslog-stackdriver")),
				},
			},
			wantLoggingQuery: `resource.type="k8s_container"
resource.labels.cluster_name="my-cluster"
resource.labels.location="us-central1"
-resource.labels.namespace_name=("kube-system" OR "istio-system")
resource.labels.pod_name:("nginx" OR "redis")
resource.labels.project_id="my-project"
-LOG_ID("server-accesslog-stackdriver")
-LOG_ID("client-accesslog-stackdriver")`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_container" AND resource.labels.cluster_name = "my-cluster" AND resource.labels.location = "us-central1" AND resource.labels.namespace_name != "kube-system" AND resource.labels.namespace_name != "istio-system" AND (resource.labels.pod_name = has_substring("nginx") OR resource.labels.pod_name = has_substring("redis")) AND resource.labels.project_id = "my-project" AND metric.labels.log != "server-accesslog-stackdriver" AND metric.labels.log != "client-accesslog-stackdriver"`,
			},
			wantAllFiltersSupported: true,
		},
		{
			name: "gke_audit query with multiple resource types and log IDs",
			query: &StructuredLogQuery{
				ResourceTypes:             []string{"gke_cluster", "gke_nodepool"},
				IgnoreMetricsResourceType: []string{"gke_cluster", "gke_nodepool"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("cluster_name", Exact("my-cluster")),
					ResourceLabel("location", Exact("us-central1")),
					ResourceLabel("project_id", Exact("my-project")),
					LogID(OneOf("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")),
					CustomFilter(`protoPayload.serviceName="container.googleapis.com"`),
				},
			},
			wantLoggingQuery: `resource.type=("gke_cluster" OR "gke_nodepool")
resource.labels.cluster_name="my-cluster"
resource.labels.location="us-central1"
resource.labels.project_id="my-project"
(LOG_ID("cloudaudit.googleapis.com/activity") OR LOG_ID("cloudaudit.googleapis.com/data_access"))
protoPayload.serviceName="container.googleapis.com"`,
			wantMetricFilters:       nil,
			wantAllFiltersSupported: false, // CustomFilter and IgnoreMetricsResourceType trigger probe
		},
		{
			name: "k8s_event query with log ID inclusion (100% pure metric)",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_cluster"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("cluster_name", Exact("my-cluster")),
					ResourceLabel("location", Exact("us-central1")),
					ResourceLabel("project_id", Exact("my-project")),
					LogID(Exact("events")),
				},
			},
			wantLoggingQuery: `resource.type="k8s_cluster"
resource.labels.cluster_name="my-cluster"
resource.labels.location="us-central1"
resource.labels.project_id="my-project"
LOG_ID("events")`,
			wantMetricFilters: []string{
				`metric.type = "logging.googleapis.com/log_entry_count" AND resource.type = "k8s_cluster" AND resource.labels.cluster_name = "my-cluster" AND resource.labels.location = "us-central1" AND resource.labels.project_id = "my-project" AND metric.labels.log = "events"`,
			},
			wantAllFiltersSupported: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLogging := tt.query.GenerateCloudLoggingQuery()
			if diff := cmp.Diff(tt.wantLoggingQuery, gotLogging); diff != "" {
				t.Errorf("GenerateCloudLoggingQuery() mismatch (-want +got):\n%s", diff)
			}

			if got := tt.query.AllFiltersSupportMetrics(); got != tt.wantAllFiltersSupported {
				t.Errorf("AllFiltersSupportMetrics() = %v, want %v", got, tt.wantAllFiltersSupported)
			}

			gotMetrics := tt.query.GenerateMonitoringMetricFilters()
			if diff := cmp.Diff(tt.wantMetricFilters, gotMetrics); diff != "" {
				t.Errorf("GenerateMonitoringMetricFilters() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

type mockMetricFetcher struct {
	count int64
	err   error
}

func (m *mockMetricFetcher) QueryMetricCount(ctx context.Context, container googlecloud.ResourceContainer, metricFilter string, startTime, endTime time.Time) (int64, error) {
	return m.count, m.err
}

type mockProbeFetcher struct {
	baseCount   int32
	customCount int32
	err         error
}

func (m *mockProbeFetcher) ProbeLogTimestamps(ctx context.Context, container googlecloud.ResourceContainer, filter string, maxEntries int32) ([]time.Time, error) {
	if m.err != nil {
		return nil, m.err
	}
	count := m.baseCount
	if strings.Contains(filter, "jsonPayload") || strings.Contains(filter, "protoPayload") {
		count = m.customCount
	}
	if count <= 0 {
		return nil, nil
	}
	now := time.Now()
	res := make([]time.Time, count)
	for i := int32(0); i < count; i++ {
		res[i] = now.Add(-time.Duration(i) * time.Second)
	}
	return res, nil
}

func TestStructuredLogEstimator_Estimate(t *testing.T) {
	tests := []struct {
		name          string
		query         *StructuredLogQuery
		metricFetcher MetricLogCountFetcher
		probeFetcher  LogSamplingProbeFetcher
		wantResult    *EstimateResult
		wantErr       bool
	}{
		{
			name: "exact count when metric has 0 logs",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_container"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("cluster_name", Exact("cluster-1")),
				},
			},
			metricFetcher: &mockMetricFetcher{count: 0},
			wantResult: &EstimateResult{
				MetricCount:       0,
				EstimatedCount:    0,
				CustomFilterRatio: 1.0,
				IsExact:           true,
			},
		},
		{
			name: "exact count when no custom filter and no exclusions",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_container"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("cluster_name", Exact("cluster-1")),
					LogID(Exact("stdout")),
				},
			},
			metricFetcher: &mockMetricFetcher{count: 500000},
			wantResult: &EstimateResult{
				MetricCount:       500000,
				EstimatedCount:    500000,
				CustomFilterRatio: 1.0,
				IsExact:           true,
			},
		},
		{
			name: "concurrent multi-resource metric summation",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_node", "k8s_pod"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("cluster_name", Exact("cluster-1")),
					LogID(Exact("events")),
				},
			},
			metricFetcher: &mockMetricFetcher{count: 3000}, // 2 resource types * 3000 = 6000
			wantResult: &EstimateResult{
				MetricCount:       6000,
				EstimatedCount:    6000,
				CustomFilterRatio: 1.0,
				IsExact:           true,
			},
		},
		{
			name: "sampled estimation with custom filter ratio",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_cluster"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("cluster_name", Exact("cluster-1")),
					LogID(Exact("events")),
					CustomFilter(`jsonPayload.involvedObject.namespace="default"`),
				},
			},
			metricFetcher: &mockMetricFetcher{count: 100000},
			probeFetcher: &mockProbeFetcher{
				baseCount:   50,
				customCount: 10, // ratio 10/50 = 0.20 -> 20000
			},
			wantResult: &EstimateResult{
				MetricCount:       100000,
				EstimatedCount:    20000,
				CustomFilterRatio: 0.2,
				IsExact:           false,
			},
		},
		{
			name: "fallback when metric is 0 but custom probe finds logs",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_cluster"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("cluster_name", Exact("cluster-1")),
					LogID(Exact("events")),
					CustomFilter(`jsonPayload.involvedObject.namespace="default"`),
				},
			},
			metricFetcher: &mockMetricFetcher{count: 0},
			probeFetcher: &mockProbeFetcher{
				baseCount:   0,
				customCount: 42,
			},
			wantResult: &EstimateResult{
				MetricCount:       0,
				EstimatedCount:    42,
				CustomFilterRatio: 1.0,
				IsExact:           false,
			},
		},
		{
			name: "metric 0 and custom probe 0 returns 0 exact",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_cluster"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("cluster_name", Exact("cluster-1")),
					LogID(Exact("events")),
					CustomFilter(`jsonPayload.involvedObject.namespace="default"`),
				},
			},
			metricFetcher: &mockMetricFetcher{count: 0},
			probeFetcher: &mockProbeFetcher{
				baseCount:   0,
				customCount: 0,
			},
			wantResult: &EstimateResult{
				MetricCount:       0,
				EstimatedCount:    0,
				CustomFilterRatio: 1.0,
				IsExact:           true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			estimator := NewStructuredLogEstimator(tt.metricFetcher, tt.probeFetcher)
			got, err := estimator.Estimate(context.Background(), googlecloud.Project("test-project"), tt.query, time.Now().Add(-time.Hour), time.Now())
			if (err != nil) != tt.wantErr {
				t.Fatalf("Estimate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.wantResult, got); diff != "" {
				t.Errorf("Estimate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

type realLogProbeFetcher struct {
	client *logging.Client
}

func (f *realLogProbeFetcher) ProbeLogTimestamps(ctx context.Context, container googlecloud.ResourceContainer, filter string, maxEntries int32) ([]time.Time, error) {
	req := &loggingpb.ListLogEntriesRequest{
		ResourceNames: []string{container.Identifier()},
		Filter:        filter,
		PageSize:      maxEntries,
	}
	it := f.client.ListLogEntries(ctx, req)
	var timestamps []time.Time
	for {
		entry, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if entry.GetTimestamp() != nil {
			timestamps = append(timestamps, entry.GetTimestamp().AsTime())
		} else {
			timestamps = append(timestamps, time.Now())
		}
		if int32(len(timestamps)) >= maxEntries {
			break
		}
	}
	return timestamps, nil
}

// TestLiveMetricEstimation runs against real GCP when RUN_LIVE_ESTIMATION_TESTS=true.
func TestLiveMetricEstimation(t *testing.T) {
	if os.Getenv("RUN_LIVE_ESTIMATION_TESTS") != "true" || *testflags.SkipCloudLogging {
		t.Skip("skipping live test: set RUN_LIVE_ESTIMATION_TESTS=true to run")
		return
	}

	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		projectID = "tse-kakeru"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	metricClient, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		t.Skipf("skipping live test: failed to create monitoring client: %v", err)
	}
	defer metricClient.Close()

	logClient, err := logging.NewClient(ctx)
	if err != nil {
		t.Skipf("skipping live test: failed to create logging client: %v", err)
	}
	defer logClient.Close()

	estimator := NewStructuredLogEstimator(
		&MonitoringClientFetcher{Client: metricClient},
		&realLogProbeFetcher{client: logClient},
	)

	now := time.Now()
	startTime := now.Add(-24 * time.Hour)
	endTime := now

	scenarios := []struct {
		name  string
		query *StructuredLogQuery
	}{
		{
			name: "Scenario 1: K8s Event logs (Pure Metric - 100% exact)",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_cluster"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("project_id", Exact(projectID)),
					LogID(Exact("events")),
				},
			},
		},
		{
			name: "Scenario 2: Container logs with namespace filter (Pure Metric)",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_container"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("project_id", Exact(projectID)),
					ResourceLabel("namespace_name", OneOf("default", "kube-system")),
				},
			},
		},
		{
			name: "Scenario 3: GKE Audit logs (Metric + Service Name Custom Filter)",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"audited_resource"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("project_id", Exact(projectID)),
					ResourceLabel("service", Exact("container.googleapis.com")),
					LogID(OneOf("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")),
				},
			},
		},
		{
			name: "Scenario 4: Container logs with Custom Filter (Sampling Probe Estimation)",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_container"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("project_id", Exact(projectID)),
					ResourceLabel("namespace_name", OneOf("kube-system")),
					CustomFilter(`jsonPayload.message:""`),
				},
			},
		},
		{
			name: "Scenario 5: Container logs with LogID NoneOf (Pure Metric)",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_container"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("project_id", Exact(projectID)),
					LogID(NoneOf("server-accesslog-stackdriver", "client-accesslog-stackdriver")),
				},
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			mFilters := s.query.GenerateMonitoringMetricFilters()
			t.Logf("Metric Filters: %v", mFilters)
			t.Logf("Logging Query:\n%s", s.query.GenerateCloudLoggingQuery())

			start := time.Now()
			res, err := estimator.Estimate(ctx, googlecloud.Project(projectID), s.query, startTime, endTime)
			elapsed := time.Since(start)

			if err != nil {
				t.Fatalf("Estimate failed: %v", err)
			}

			t.Logf("Result in %v: MetricCount=%d, EstimatedCount=%d, Ratio=%.4f, IsExact=%v",
				elapsed, res.MetricCount, res.EstimatedCount, res.CustomFilterRatio, res.IsExact)
		})
	}
}

// TestEstimateClusterLogs tests log estimation for a specific project and cluster.
func TestEstimateClusterLogs(t *testing.T) {
	if os.Getenv("RUN_LIVE_ESTIMATION_TESTS") != "true" || *testflags.SkipCloudLogging {
		t.Skip("skipping live test: set RUN_LIVE_ESTIMATION_TESTS=true to run")
		return
	}

	projectID := "khi-testing-with-auditlog"
	clusterName := "p0-gke-basic-1"

	startTime, err := time.Parse(time.RFC3339, "2026-02-18T05:53:08Z")
	if err != nil {
		t.Fatalf("failed to parse startTime: %v", err)
	}
	endTime, err := time.Parse(time.RFC3339, "2026-02-18T09:53:08Z")
	if err != nil {
		t.Fatalf("failed to parse endTime: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	metricClient, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		t.Skipf("skipping live test: failed to create monitoring client: %v", err)
	}
	defer metricClient.Close()

	logClient, err := logging.NewClient(ctx)
	if err != nil {
		t.Skipf("skipping live test: failed to create logging client: %v", err)
	}
	defer logClient.Close()

	estimator := NewStructuredLogEstimator(
		&MonitoringClientFetcher{Client: metricClient},
		&realLogProbeFetcher{client: logClient},
	)

	queries := []struct {
		logType string
		query   *StructuredLogQuery
	}{
		{
			logType: "1. K8s Event Log",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_cluster"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("project_id", Exact(projectID)),
					ResourceLabel("cluster_name", Exact(clusterName)),
					LogID(Exact("events")),
				},
			},
		},
		{
			logType: "2. K8s Audit Log (mutations on deployments/replicasets/pods/nodes)",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_cluster"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("project_id", Exact(projectID)),
					ResourceLabel("cluster_name", Exact(clusterName)),
					LogID(OneOf("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")),
					CustomFilter(`protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
protoPayload.methodName=~"\.(deployments|replicasets|pods|nodes)\."`),
				},
			},
		},
		{
			logType: "3. K8s Node Log",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_node"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("project_id", Exact(projectID)),
					ResourceLabel("cluster_name", Exact(clusterName)),
					LogID(NoneOf("events")),
				},
			},
		},
		{
			logType: "4. K8s Container Log",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_container"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("project_id", Exact(projectID)),
					ResourceLabel("cluster_name", Exact(clusterName)),
					LogID(NoneOf("server-accesslog-stackdriver", "client-accesslog-stackdriver")),
				},
			},
		},
	}

	for _, item := range queries {
		t.Run(item.logType, func(t *testing.T) {
			mFilters := item.query.GenerateMonitoringMetricFilters()
			t.Logf("Metric Filter: %v", mFilters)
			t.Logf("Logging Query:\n%s", item.query.GenerateCloudLoggingQuery())

			start := time.Now()
			res, err := estimator.Estimate(ctx, googlecloud.Project(projectID), item.query, startTime, endTime)
			elapsed := time.Since(start)

			if err != nil {
				t.Fatalf("Estimate failed: %v", err)
			}

			t.Logf(">>> [%s] Elapsed: %v | Base Metric: %d | Estimated: %d | Ratio: %.4f | IsExact: %v",
				item.logType, elapsed, res.MetricCount, res.EstimatedCount, res.CustomFilterRatio, res.IsExact)
		})
	}
}

// TestEstimateTseKakeruClusterLogs tests log volume estimation for yesterday on tse-kakeru clusters.
func TestEstimateTseKakeruClusterLogs(t *testing.T) {
	if os.Getenv("RUN_LIVE_ESTIMATION_TESTS") != "true" || *testflags.SkipCloudLogging {
		t.Skip("skipping live test: set RUN_LIVE_ESTIMATION_TESTS=true to run")
		return
	}

	projectID := "tse-kakeru"
	clusterName := "keigof-0804-cluster"
	location := "asia-northeast1"

	// Yesterday: 2026-08-21 00:00:00 UTC to 2026-08-21 23:59:59 UTC
	startTime, err := time.Parse(time.RFC3339, "2026-08-21T00:00:00Z")
	if err != nil {
		t.Fatalf("failed to parse startTime: %v", err)
	}
	endTime, err := time.Parse(time.RFC3339, "2026-08-21T23:59:59Z")
	if err != nil {
		t.Fatalf("failed to parse endTime: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	metricClient, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		t.Skipf("skipping live test: failed to create monitoring client: %v", err)
	}
	defer metricClient.Close()

	logClient, err := logging.NewClient(ctx)
	if err != nil {
		t.Skipf("skipping live test: failed to create logging client: %v", err)
	}
	defer logClient.Close()

	estimator := NewStructuredLogEstimator(
		&MonitoringClientFetcher{Client: metricClient},
		&realLogProbeFetcher{client: logClient},
	)

	queries := []struct {
		logType string
		query   *StructuredLogQuery
	}{
		{
			logType: "1. K8s Event Log",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_cluster"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("project_id", Exact(projectID)),
					ResourceLabel("location", Exact(location)),
					ResourceLabel("cluster_name", Exact(clusterName)),
					LogID(Exact("events")),
				},
			},
		},
		{
			logType: "2. K8s Audit Log (mutations on deployments/replicasets/pods/nodes)",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_cluster"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("project_id", Exact(projectID)),
					ResourceLabel("location", Exact(location)),
					ResourceLabel("cluster_name", Exact(clusterName)),
					LogID(OneOf("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")),
					CustomFilter(`protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
protoPayload.methodName=~"\.(deployments|replicasets|pods|nodes)\."`),
				},
			},
		},
		{
			logType: "3. K8s Node Log",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_node"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("project_id", Exact(projectID)),
					ResourceLabel("location", Exact(location)),
					ResourceLabel("cluster_name", Exact(clusterName)),
					LogID(NoneOf("events")),
				},
			},
		},
		{
			logType: "4. K8s Container Log",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_container"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("project_id", Exact(projectID)),
					ResourceLabel("location", Exact(location)),
					ResourceLabel("cluster_name", Exact(clusterName)),
					LogID(NoneOf("server-accesslog-stackdriver", "client-accesslog-stackdriver")),
				},
			},
		},
		{
			logType: "5. K8s Control Plane Log",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_control_plane_component"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("project_id", Exact(projectID)),
					ResourceLabel("location", Exact(location)),
					ResourceLabel("cluster_name", Exact(clusterName)),
					CustomFilter(`-sourceLocation.file="httplog.go"`),
				},
			},
		},
	}

	for _, item := range queries {
		t.Run(item.logType, func(t *testing.T) {
			mFilters := item.query.GenerateMonitoringMetricFilters()
			t.Logf("Metric Filter: %v", mFilters)
			t.Logf("Logging Query:\n%s", item.query.GenerateCloudLoggingQuery())

			start := time.Now()
			res, err := estimator.Estimate(ctx, googlecloud.Project(projectID), item.query, startTime, endTime)
			elapsed := time.Since(start)

			if err != nil {
				t.Fatalf("Estimate failed: %v", err)
			}

			t.Logf(">>> [%s] Elapsed: %v | Base Metric: %d | Estimated: %d | Ratio: %.4f | IsExact: %v",
				item.logType, elapsed, res.MetricCount, res.EstimatedCount, res.CustomFilterRatio, res.IsExact)
		})
	}
}

// TestAccuracyOneHourComparison tests accuracy by comparing estimated vs actual counts over a time window.
func TestAccuracyOneHourComparison(t *testing.T) {
	if os.Getenv("RUN_LIVE_ESTIMATION_TESTS") != "true" || *testflags.SkipCloudLogging {
		t.Skip("skipping live test: set RUN_LIVE_ESTIMATION_TESTS=true to run")
		return
	}

	projectID := "tse-kakeru"
	clusterName := os.Getenv("TEST_CLUSTER_NAME")
	if clusterName == "" {
		clusterName = "csm-cluster"
	}
	location := os.Getenv("TEST_CLUSTER_LOCATION")
	if location == "" {
		location = "us-central1"
	}

	// 15-minute window: 2026-08-21 06:00:00 UTC to 2026-08-21 06:15:00 UTC
	startTime, _ := time.Parse(time.RFC3339, "2026-08-21T06:00:00Z")
	endTime, _ := time.Parse(time.RFC3339, "2026-08-21T06:15:00Z")

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	metricClient, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		t.Skipf("skipping live test: failed to create monitoring client: %v", err)
	}
	defer metricClient.Close()

	logClient, err := logging.NewClient(ctx)
	if err != nil {
		t.Skipf("skipping live test: failed to create logging client: %v", err)
	}
	defer logClient.Close()

	estimator := NewStructuredLogEstimator(
		&MonitoringClientFetcher{Client: metricClient},
		&realLogProbeFetcher{client: logClient},
	)

	queries := []struct {
		logType string
		query   *StructuredLogQuery
	}{
		{
			logType: "1. K8s Event Log",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_cluster"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("project_id", Exact(projectID)),
					ResourceLabel("location", Exact(location)),
					ResourceLabel("cluster_name", Exact(clusterName)),
					LogID(Exact("events")),
				},
			},
		},
		{
			logType: "2. K8s Audit Log (Mutations)",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_cluster"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("project_id", Exact(projectID)),
					ResourceLabel("location", Exact(location)),
					ResourceLabel("cluster_name", Exact(clusterName)),
					LogID(OneOf("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")),
					CustomFilter(`protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
protoPayload.methodName=~"\.(deployments|replicasets|pods|nodes)\."`),
				},
			},
		},
		{
			logType: "3. K8s Node Log",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_node"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("project_id", Exact(projectID)),
					ResourceLabel("location", Exact(location)),
					ResourceLabel("cluster_name", Exact(clusterName)),
					LogID(NoneOf("events")),
				},
			},
		},
		{
			logType: "4. K8s Container Log",
			query: &StructuredLogQuery{
				ResourceTypes: []string{"k8s_container"},
				Filters: []LoggingMonitoringMatcher{
					ResourceLabel("project_id", Exact(projectID)),
					ResourceLabel("location", Exact(location)),
					ResourceLabel("cluster_name", Exact(clusterName)),
					LogID(NoneOf("server-accesslog-stackdriver", "client-accesslog-stackdriver")),
				},
			},
		},
	}

	for _, item := range queries {
		t.Run(item.logType, func(t *testing.T) {
			time.Sleep(5 * time.Second)

			// 1. Estimation
			estStart := time.Now()
			res, err := estimator.Estimate(ctx, googlecloud.Project(projectID), item.query, startTime, endTime)
			estElapsed := time.Since(estStart)
			if err != nil {
				t.Fatalf("Estimate failed: %v", err)
			}

			// 2. Actual Count from Cloud Logging with rate pacing
			time.Sleep(5 * time.Second)
			actStart := time.Now()
			fullQuery := fmt.Sprintf("%s\ntimestamp >= \"%s\"\ntimestamp <= \"%s\"",
				item.query.GenerateCloudLoggingQuery(),
				startTime.Format(time.RFC3339),
				endTime.Format(time.RFC3339),
			)
			req := &loggingpb.ListLogEntriesRequest{
				ResourceNames: []string{"projects/" + projectID},
				Filter:        fullQuery,
				PageSize:      1000,
			}
			it := logClient.ListLogEntries(ctx, req)
			var actualCount int64
			var pageCount int
			for {
				_, err := it.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					t.Fatalf("ListLogEntries error: %v", err)
				}
				actualCount++
				if actualCount%1000 == 0 {
					pageCount++
					time.Sleep(600 * time.Millisecond)
				}
			}
			actElapsed := time.Since(actStart)

			diff := res.EstimatedCount - actualCount
			errPct := 0.0
			if actualCount > 0 {
				errPct = math.Abs(float64(diff)) / float64(actualCount) * 100.0
			}

			t.Logf(">>> [%s]\n    Estimated: %d (in %v, MetricBase: %d, Ratio: %.4f)\n    Actual:    %d (in %v, Pages: %d)\n    Error:     %.2f%% (diff: %d)",
				item.logType, res.EstimatedCount, estElapsed, res.MetricCount, res.CustomFilterRatio, actualCount, actElapsed, pageCount, errPct, diff)
		})
	}
}
