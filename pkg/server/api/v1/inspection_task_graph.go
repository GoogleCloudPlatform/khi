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
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
)

// InspectionTaskGraphServer implements the apiv1connect.InspectionTaskGraphServiceHandler interface.
type InspectionTaskGraphServer struct {
	inspectionServer    *coreinspection.InspectionTaskServer
	streamCycleDuration time.Duration
	updateInterval      time.Duration
}

var _ apiv1connect.InspectionTaskGraphServiceHandler = (*InspectionTaskGraphServer)(nil)

const (
	// DefaultStreamCycleDuration is the default duration before a watch stream closes and prompts the client to reconnect.
	DefaultStreamCycleDuration = 30 * time.Second
	// DefaultUpdateInterval is the default interval between task graph snapshot emissions.
	DefaultUpdateInterval = 1 * time.Second
)

// NewInspectionTaskGraphServer creates a new InspectionTaskGraphServer with the given stream intervals.
func NewInspectionTaskGraphServer(
	inspectionServer *coreinspection.InspectionTaskServer,
	streamCycleDuration time.Duration,
	updateInterval time.Duration,
) *InspectionTaskGraphServer {
	return &InspectionTaskGraphServer{
		inspectionServer:    inspectionServer,
		streamCycleDuration: streamCycleDuration,
		updateInterval:      updateInterval,
	}
}

// GetInspectionTaskRegistry returns all registered tasks grouped by reference identifier and all inspection types.
func (s *InspectionTaskGraphServer) GetInspectionTaskRegistry(
	ctx context.Context,
	req *connect.Request[apiv1.GetInspectionTaskRegistryRequest],
) (*connect.Response[apiv1.GetInspectionTaskRegistryResponse], error) {
	resp, err := coreinspection.InspectRegistry(s.inspectionServer)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

// ResolveInspectionTaskGraph resolves task filtering against an inspection type and feature configuration,
// and returns step-by-step filtering results alongside the resolved execution DAG.
func (s *InspectionTaskGraphServer) ResolveInspectionTaskGraph(
	ctx context.Context,
	req *connect.Request[apiv1.ResolveInspectionTaskGraphRequest],
) (*connect.Response[apiv1.ResolveInspectionTaskGraphResponse], error) {
	if req.Msg.GetInspectionTypeId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("inspection_type_id must not be empty"))
	}

	resp, err := coreinspection.InspectResolution(
		s.inspectionServer,
		req.Msg.GetInspectionTypeId(),
		req.Msg.GetFeatureOverrides(),
	)
	if err != nil {
		if errors.Is(err, coreinspection.ErrInspectionTypeNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(resp), nil
}

// WatchInspectionRunTaskGraph streams progress snapshots of an inspection run. The stream closes when
// the run reaches its terminal state, or after streamCycleDuration to prompt the client to reconnect.
func (s *InspectionTaskGraphServer) WatchInspectionRunTaskGraph(
	ctx context.Context,
	req *connect.Request[apiv1.WatchInspectionRunTaskGraphRequest],
	stream *connect.ServerStream[apiv1.WatchInspectionRunTaskGraphResponse],
) error {
	inspectionID := req.Msg.GetInspectionId()

	// Send initial snapshot immediately upon connection.
	snapshot, err := s.inspectRunTaskGraph(inspectionID)
	if err != nil {
		return err
	}
	if err := stream.Send(&apiv1.WatchInspectionRunTaskGraphResponse{Snapshot: snapshot}); err != nil {
		return err
	}
	if snapshot.GetIsRunFinished() {
		return nil
	}

	cycleTimer := time.NewTimer(s.streamCycleDuration)
	defer cycleTimer.Stop()

	ticker := time.NewTicker(s.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-cycleTimer.C:
			// Gracefully close stream after cycle duration expires to prompt client reconnect.
			return nil
		case <-ticker.C:
			snapshot, err := s.inspectRunTaskGraph(inspectionID)
			if err != nil {
				return err
			}
			if err := stream.Send(&apiv1.WatchInspectionRunTaskGraphResponse{Snapshot: snapshot}); err != nil {
				return err
			}
			if snapshot.GetIsRunFinished() {
				return nil
			}
		}
	}
}

// PullInspectionRunTaskGraph returns a single progress snapshot of an inspection run without opening a stream.
func (s *InspectionTaskGraphServer) PullInspectionRunTaskGraph(
	ctx context.Context,
	req *connect.Request[apiv1.PullInspectionRunTaskGraphRequest],
) (*connect.Response[apiv1.PullInspectionRunTaskGraphResponse], error) {
	snapshot, err := s.inspectRunTaskGraph(req.Msg.GetInspectionId())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&apiv1.PullInspectionRunTaskGraphResponse{
		Snapshot: snapshot,
	}), nil
}

// inspectRunTaskGraph builds a run snapshot and translates core errors into Connect error codes.
func (s *InspectionTaskGraphServer) inspectRunTaskGraph(inspectionID string) (*apiv1.InspectionRunTaskGraphSnapshot, error) {
	if inspectionID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("inspection_id must not be empty"))
	}
	snapshot, err := coreinspection.InspectRunTaskGraph(s.inspectionServer, inspectionID)
	if err != nil {
		switch {
		case errors.Is(err, coreinspection.ErrInspectionNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err)
		case errors.Is(err, coreinspection.ErrRunTaskGraphNotStarted):
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return snapshot, nil
}
