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

	"connectrpc.com/connect"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// WorkbenchServiceServer implements the apiv1connect.WorkbenchServiceHandler interface.
type WorkbenchServiceServer struct {
	manager *workbench.WorkbenchManager
}

var _ apiv1connect.WorkbenchServiceHandler = (*WorkbenchServiceServer)(nil)

// NewWorkbenchServiceServer creates a new WorkbenchServiceServer backed by the given manager.
func NewWorkbenchServiceServer(manager *workbench.WorkbenchManager) *WorkbenchServiceServer {
	return &WorkbenchServiceServer{
		manager: manager,
	}
}

// OpenWorkbench opens or attaches to an in-memory Workbench session, streaming progress stages back to the client.
func (s *WorkbenchServiceServer) OpenWorkbench(
	ctx context.Context,
	req *connect.Request[apiv1.OpenWorkbenchRequest],
	stream *connect.ServerStream[apiv1.OpenWorkbenchResponse],
) error {
	msg := req.Msg
	if msg.GetUserId() == "" || msg.GetSessionId() == "" || msg.GetInspectionId() == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("user_id, session_id, and inspection_id are required"))
	}

	workbenchID := fmt.Sprintf("%s-%s", msg.GetUserId(), msg.GetSessionId())

	wb, err := s.manager.GetOrOpen(ctx, workbenchID, msg.GetInspectionId(), func(stage apiv1.OpenWorkbenchResponse_Stage, progressPercentage float64, message string) error {
		res := &apiv1.OpenWorkbenchResponse{
			Stage:              stage.Enum(),
			ProgressPercentage: proto.Float64(progressPercentage),
			Message:            proto.String(message),
		}
		return stream.Send(res)
	})
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to open workbench: %w", err))
	}

	// Send final completion message with workbench ID
	finalRes := &apiv1.OpenWorkbenchResponse{
		Stage:              apiv1.OpenWorkbenchResponse_STAGE_READY.Enum(),
		ProgressPercentage: proto.Float64(100.0),
		Message:            proto.String("Workbench ready."),
		WorkbenchId:        proto.String(wb.ID()),
	}
	return stream.Send(finalRes)
}

// HeartbeatWorkbench refreshes the lease expiration time for an active Workbench session.
func (s *WorkbenchServiceServer) HeartbeatWorkbench(
	ctx context.Context,
	req *connect.Request[apiv1.HeartbeatWorkbenchRequest],
) (*connect.Response[apiv1.HeartbeatWorkbenchResponse], error) {
	msg := req.Msg
	if msg.GetWorkbenchId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workbench_id is required"))
	}

	_, expiresAt, err := s.manager.Heartbeat(msg.GetWorkbenchId())
	if err != nil {
		if errors.Is(err, workbench.ErrWorkbenchNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := &apiv1.HeartbeatWorkbenchResponse{
		Active:    proto.Bool(true),
		ExpiresAt: timestamppb.New(expiresAt),
	}
	return connect.NewResponse(res), nil
}

// CloseWorkbench explicitly closes and frees the specified Workbench session.
func (s *WorkbenchServiceServer) CloseWorkbench(
	ctx context.Context,
	req *connect.Request[apiv1.CloseWorkbenchRequest],
) (*connect.Response[apiv1.CloseWorkbenchResponse], error) {
	msg := req.Msg
	if msg.GetWorkbenchId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workbench_id is required"))
	}

	if err := s.manager.Close(msg.GetWorkbenchId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := &apiv1.CloseWorkbenchResponse{
		Closed: proto.Bool(true),
	}
	return connect.NewResponse(res), nil
}
