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
	"net"
	"testing"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type mockMetricServer struct {
	monitoringpb.UnimplementedMetricServiceServer
	listTimeSeriesFunc func(context.Context, *monitoringpb.ListTimeSeriesRequest) (*monitoringpb.ListTimeSeriesResponse, error)
}

func (m *mockMetricServer) ListTimeSeries(ctx context.Context, req *monitoringpb.ListTimeSeriesRequest) (*monitoringpb.ListTimeSeriesResponse, error) {
	if m.listTimeSeriesFunc != nil {
		return m.listTimeSeriesFunc(ctx, req)
	}
	return &monitoringpb.ListTimeSeriesResponse{}, nil
}

func TestQueryResourceLabelsFromMetrics(t *testing.T) {
	tests := []struct {
		name           string
		listTimeSeries func(context.Context, *monitoringpb.ListTimeSeriesRequest) (*monitoringpb.ListTimeSeriesResponse, error)
		groupByKey     []string
		want           []map[string]string
		wantErr        bool
	}{
		{
			name: "success with results",
			listTimeSeries: func(ctx context.Context, req *monitoringpb.ListTimeSeriesRequest) (*monitoringpb.ListTimeSeriesResponse, error) {
				return &monitoringpb.ListTimeSeriesResponse{
					TimeSeries: []*monitoringpb.TimeSeries{
						{
							Resource: &monitoredres.MonitoredResource{
								Labels: map[string]string{"cluster_name": "cluster-1"},
							},
						},
						{
							Resource: &monitoredres.MonitoredResource{
								Labels: map[string]string{"cluster_name": "cluster-2"},
							},
						},
					},
				}, nil
			},
			groupByKey: []string{"resource.label.cluster_name"},
			want: []map[string]string{
				{"cluster_name": "cluster-1"},
				{"cluster_name": "cluster-2"},
			},
		},
		{
			name: "success with empty results",
			listTimeSeries: func(ctx context.Context, req *monitoringpb.ListTimeSeriesRequest) (*monitoringpb.ListTimeSeriesResponse, error) {
				return &monitoringpb.ListTimeSeriesResponse{}, nil
			},
			groupByKey: []string{"resource.label.cluster_name"},
			want:       []map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lis, err := net.Listen("tcp", "localhost:0")
			if err != nil {
				t.Fatalf("failed to listen: %v", err)
			}
			s := grpc.NewServer()
			mock := &mockMetricServer{
				listTimeSeriesFunc: tt.listTimeSeries,
			}
			monitoringpb.RegisterMetricServiceServer(s, mock)
			go func() {
				if err := s.Serve(lis); err != nil {
					// Server might be closed
				}
			}()
			defer s.Stop()

			ctx := context.Background()
			client, err := monitoring.NewMetricClient(ctx,
				option.WithEndpoint(lis.Addr().String()),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
				option.WithoutAuthentication(),
			)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}
			defer client.Close()

			got, err := QueryResourceLabelsFromMetrics(ctx, client, "test-project", "filter", time.Now(), time.Now().Add(time.Hour), tt.groupByKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryResourceLabelsFromMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("QueryResourceLabelsFromMetrics() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestQueryDistinctStringLabelValuesFromMetrics(t *testing.T) {
	tests := []struct {
		name           string
		listTimeSeries func(context.Context, *monitoringpb.ListTimeSeriesRequest) (*monitoringpb.ListTimeSeriesResponse, error)
		groupByKey     string
		resultLabelKey string
		want           []string
		wantErr        bool
	}{
		{
			name: "success with unique values",
			listTimeSeries: func(ctx context.Context, req *monitoringpb.ListTimeSeriesRequest) (*monitoringpb.ListTimeSeriesResponse, error) {
				return &monitoringpb.ListTimeSeriesResponse{
					TimeSeries: []*monitoringpb.TimeSeries{
						{
							Resource: &monitoredres.MonitoredResource{
								Labels: map[string]string{"cluster_name": "cluster-1"},
							},
						},
						{
							Resource: &monitoredres.MonitoredResource{
								Labels: map[string]string{"cluster_name": "cluster-1"},
							},
						},
						{
							Resource: &monitoredres.MonitoredResource{
								Labels: map[string]string{"cluster_name": "cluster-2"},
							},
						},
					},
				}, nil
			},
			groupByKey:     "resource.label.cluster_name",
			resultLabelKey: "cluster_name",
			want:           []string{"cluster-1", "cluster-2"},
		},
		{
			name: "success with no values",
			listTimeSeries: func(ctx context.Context, req *monitoringpb.ListTimeSeriesRequest) (*monitoringpb.ListTimeSeriesResponse, error) {
				return &monitoringpb.ListTimeSeriesResponse{
					TimeSeries: []*monitoringpb.TimeSeries{},
				}, nil
			},
			groupByKey:     "resource.label.cluster_name",
			resultLabelKey: "cluster_name",
			want:           []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lis, err := net.Listen("tcp", "localhost:0")
			if err != nil {
				t.Fatalf("failed to listen: %v", err)
			}
			s := grpc.NewServer()
			mock := &mockMetricServer{
				listTimeSeriesFunc: tt.listTimeSeries,
			}
			monitoringpb.RegisterMetricServiceServer(s, mock)
			go func() {
				if err := s.Serve(lis); err != nil {
					// Server might be closed
				}
			}()
			defer s.Stop()

			ctx := context.Background()
			client, err := monitoring.NewMetricClient(ctx,
				option.WithEndpoint(lis.Addr().String()),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
				option.WithoutAuthentication(),
			)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}
			defer client.Close()

			got, err := QueryDistinctStringLabelValuesFromMetrics(ctx, client, "test-project", "filter", time.Now(), time.Now().Add(time.Hour), tt.groupByKey, tt.resultLabelKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryDistinctStringLabelValuesFromMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.want, got, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
				t.Errorf("QueryDistinctStringLabelValuesFromMetrics() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
