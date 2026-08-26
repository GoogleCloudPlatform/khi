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
	"time"

	"connectrpc.com/connect"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	"github.com/GoogleCloudPlatform/khi/pkg/server"
	"google.golang.org/protobuf/proto"
)

// ServerStatusServiceServer implements the apiv1connect.ServerStatusServiceHandler interface.
type ServerStatusServiceServer struct {
	monitor             server.ResourceMonitor
	streamCycleDuration time.Duration
	updateInterval      time.Duration
}

var _ apiv1connect.ServerStatusServiceHandler = (*ServerStatusServiceServer)(nil)

// NewServerStatusServiceServer creates a new ServerStatusServiceServer with default 30s stream cycle and 1s interval.
func NewServerStatusServiceServer(monitor server.ResourceMonitor) *ServerStatusServiceServer {
	return NewServerStatusServiceServerWithIntervals(monitor, 30*time.Second, 1*time.Second)
}

// NewServerStatusServiceServerWithIntervals creates a new ServerStatusServiceServer with custom intervals for testing.
func NewServerStatusServiceServerWithIntervals(
	monitor server.ResourceMonitor,
	streamCycleDuration time.Duration,
	updateInterval time.Duration,
) *ServerStatusServiceServer {
	return &ServerStatusServiceServer{
		monitor:             monitor,
		streamCycleDuration: streamCycleDuration,
		updateInterval:      updateInterval,
	}
}

// PullServerStat pulls current server resource usage statistics as a one-off snapshot.
func (s *ServerStatusServiceServer) PullServerStat(
	ctx context.Context,
	req *connect.Request[apiv1.PullServerStatRequest],
) (*connect.Response[apiv1.PullServerStatResponse], error) {
	stat := s.buildServerStat()
	return connect.NewResponse(&apiv1.PullServerStatResponse{
		ServerStat: stat,
	}), nil
}

// WatchServerStat streams real-time server resource usage statistics, terminating after streamCycleDuration.
func (s *ServerStatusServiceServer) WatchServerStat(
	ctx context.Context,
	req *connect.Request[apiv1.WatchServerStatRequest],
	stream *connect.ServerStream[apiv1.WatchServerStatResponse],
) error {
	// Send initial snapshot immediately upon connection.
	if err := stream.Send(&apiv1.WatchServerStatResponse{ServerStat: s.buildServerStat()}); err != nil {
		return err
	}

	ticker := time.NewTicker(s.updateInterval)
	defer ticker.Stop()

	cycleTimer := time.NewTimer(s.streamCycleDuration)
	defer cycleTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-cycleTimer.C:
			// Gracefully close the stream when cycle duration expires.
			return nil
		case <-ticker.C:
			if err := stream.Send(&apiv1.WatchServerStatResponse{ServerStat: s.buildServerStat()}); err != nil {
				return err
			}
		}
	}
}

func (s *ServerStatusServiceServer) buildServerStat() *apiv1.ServerStat {
	return &apiv1.ServerStat{
		CurrentMemoryUsage: proto.Uint64(s.monitor.GetUsedMemory()),
		TotalMemory:        proto.Uint64(s.monitor.GetTotalMemory()),
		CpuUsagePercentage: proto.Float64(s.monitor.GetCPUUsage()),
	}
}
