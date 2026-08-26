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

package apiv1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	"github.com/GoogleCloudPlatform/khi/pkg/server"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

func setupTestServerStatusServer(
	t *testing.T,
	monitor server.ResourceMonitor,
	cycleDuration time.Duration,
	updateInterval time.Duration,
) (*httptest.Server, apiv1connect.ServerStatusServiceClient) {
	serverImpl := NewServerStatusServiceServerWithIntervals(monitor, cycleDuration, updateInterval)
	mux := http.NewServeMux()
	path, handler := apiv1connect.NewServerStatusServiceHandler(serverImpl)
	mux.Handle(path, handler)

	ts := httptest.NewServer(mux)
	client := apiv1connect.NewServerStatusServiceClient(ts.Client(), ts.URL)
	return ts, client
}

func TestServerStatusServiceServer_PullServerStat(t *testing.T) {
	testCases := []struct {
		name     string
		mock     *server.ResourceMonitorMock
		wantStat *apiv1.ServerStat
	}{
		{
			name: "pulls resource monitor snapshot",
			mock: &server.ResourceMonitorMock{
				UsedMemory:  1024 * 1024 * 64,
				TotalMemory: 1024 * 1024 * 1024 * 32,
				CPUUsage:    75.2,
			},
			wantStat: &apiv1.ServerStat{
				CurrentMemoryUsage: proto.Uint64(1024 * 1024 * 64),
				TotalMemory:        proto.Uint64(1024 * 1024 * 1024 * 32),
				CpuUsagePercentage: proto.Float64(75.2),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts, client := setupTestServerStatusServer(t, tc.mock, 30*time.Second, 1*time.Second)
			defer ts.Close()

			res, err := client.PullServerStat(context.Background(), connect.NewRequest(&apiv1.PullServerStatRequest{}))
			if err != nil {
				t.Fatalf("PullServerStat() unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.wantStat, res.Msg.GetServerStat(), protocmp.Transform()); diff != "" {
				t.Errorf("PullServerStat() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestServerStatusServiceServer_WatchServerStat(t *testing.T) {
	testCases := []struct {
		name          string
		mock          *server.ResourceMonitorMock
		cycleDuration time.Duration
		interval      time.Duration
		cancelEarly   bool
		wantMinCount  int
		wantStat      *apiv1.ServerStat
	}{
		{
			name: "streams initial snapshot and ticks then completes on cycle expiration",
			mock: &server.ResourceMonitorMock{
				UsedMemory:  2048,
				TotalMemory: 4096,
				CPUUsage:    15.0,
			},
			cycleDuration: 50 * time.Millisecond,
			interval:      10 * time.Millisecond,
			cancelEarly:   false,
			wantMinCount:  2,
			wantStat: &apiv1.ServerStat{
				CurrentMemoryUsage: proto.Uint64(2048),
				TotalMemory:        proto.Uint64(4096),
				CpuUsagePercentage: proto.Float64(15.0),
			},
		},
		{
			name: "terminates immediately when context is cancelled",
			mock: &server.ResourceMonitorMock{
				UsedMemory:  100,
				TotalMemory: 200,
				CPUUsage:    5.0,
			},
			cycleDuration: 5 * time.Second,
			interval:      100 * time.Millisecond,
			cancelEarly:   true,
			wantMinCount:  1,
			wantStat: &apiv1.ServerStat{
				CurrentMemoryUsage: proto.Uint64(100),
				TotalMemory:        proto.Uint64(200),
				CpuUsagePercentage: proto.Float64(5.0),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts, client := setupTestServerStatusServer(t, tc.mock, tc.cycleDuration, tc.interval)
			defer ts.Close()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			stream, err := client.WatchServerStat(ctx, connect.NewRequest(&apiv1.WatchServerStatRequest{}))
			if err != nil {
				t.Fatalf("WatchServerStat() unexpected error: %v", err)
			}

			receivedCount := 0
			var firstStat *apiv1.ServerStat
			for stream.Receive() {
				receivedCount++
				if firstStat == nil {
					firstStat = stream.Msg().GetServerStat()
				}
				if tc.cancelEarly {
					cancel()
				}
			}

			if receivedCount < tc.wantMinCount {
				t.Errorf("received count = %d, want at least %d", receivedCount, tc.wantMinCount)
			}

			if diff := cmp.Diff(tc.wantStat, firstStat, protocmp.Transform()); diff != "" {
				t.Errorf("initial snapshot mismatch (-want +got):\n%s", diff)
			}

			if !tc.cancelEarly && stream.Err() != nil {
				t.Errorf("stream.Err() = %v, want nil for clean cycle expiration", stream.Err())
			}
		})
	}
}
