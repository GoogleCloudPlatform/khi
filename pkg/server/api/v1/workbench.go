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

// OpenWorkbenchSync opens or polls loading progress of a Workbench session without streaming.
func (s *WorkbenchServiceServer) OpenWorkbenchSync(
	ctx context.Context,
	req *connect.Request[apiv1.OpenWorkbenchSyncRequest],
) (*connect.Response[apiv1.OpenWorkbenchSyncResponse], error) {
	msg := req.Msg
	if msg.GetJobId() == "" && (msg.GetUserId() == "" || msg.GetSessionId() == "" || msg.GetInspectionId() == "") {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user_id, session_id, and inspection_id are required"))
	}

	workbenchID := fmt.Sprintf("%s-%s", msg.GetUserId(), msg.GetSessionId())
	runner := func(jobCtx context.Context, onProgress func(*apiv1.OpenWorkbenchSyncResponse) error) (string, error) {
		wb, err := s.manager.GetOrOpen(jobCtx, workbenchID, msg.GetInspectionId(), func(stage apiv1.OpenWorkbenchResponse_Stage, progressPercentage float64, message string) error {
			return onProgress(&apiv1.OpenWorkbenchSyncResponse{
				Stage:              stage.Enum(),
				ProgressPercentage: proto.Float64(progressPercentage),
				Message:            proto.String(message),
			})
		})
		if err != nil {
			return "", err
		}
		return wb.ID(), nil
	}

	status, err := s.manager.OpenJobManager().Poll(ctx, msg.GetJobId(), 50*time.Millisecond, runner)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, connect.NewError(connect.CodeCanceled, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if status.IsDone {
		if status.Err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to open workbench: %w", status.Err))
		}
		return connect.NewResponse(&apiv1.OpenWorkbenchSyncResponse{
			JobId:              proto.String(status.JobID),
			Stage:              apiv1.OpenWorkbenchResponse_STAGE_READY.Enum(),
			ProgressPercentage: proto.Float64(100.0),
			Message:            proto.String("Workbench ready."),
			WorkbenchId:        proto.String(status.Result),
		}), nil
	}

	res := &apiv1.OpenWorkbenchSyncResponse{
		JobId: proto.String(status.JobID),
	}
	if status.HasProgress && status.Progress != nil {
		res.Stage = status.Progress.GetStage().Enum()
		res.ProgressPercentage = proto.Float64(status.Progress.GetProgressPercentage())
		res.Message = proto.String(status.Progress.GetMessage())
	}
	return connect.NewResponse(res), nil
}

// CancelOpenWorkbenchSync cancels an in-progress synchronous workbench open task.
func (s *WorkbenchServiceServer) CancelOpenWorkbenchSync(
	ctx context.Context,
	req *connect.Request[apiv1.CancelOpenWorkbenchSyncRequest],
) (*connect.Response[apiv1.CancelOpenWorkbenchSyncResponse], error) {
	msg := req.Msg
	if msg.GetJobId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("job_id is required"))
	}
	canceled := s.manager.OpenJobManager().Cancel(msg.GetJobId())
	return connect.NewResponse(&apiv1.CancelOpenWorkbenchSyncResponse{
		Canceled: proto.Bool(canceled),
	}), nil
}

// WatchIndexProgress streams the search index construction progress and status for an active Workbench session.
// The server terminates the stream every 30s to accommodate proxy timeouts, and clients are expected to reconnect.
func (s *WorkbenchServiceServer) WatchIndexProgress(
	ctx context.Context,
	req *connect.Request[apiv1.WatchIndexProgressRequest],
	stream *connect.ServerStream[apiv1.WatchIndexProgressResponse],
) error {
	msg := req.Msg
	if msg.GetWorkbenchId() == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("workbench_id is required"))
	}

	wb, err := s.manager.GetAndTouch(msg.GetWorkbenchId())
	if err != nil {
		if errors.Is(err, workbench.ErrWorkbenchNotFound) || errors.Is(err, workbench.ErrWorkbenchClosed) {
			return connect.NewError(connect.CodeNotFound, err)
		}
		return connect.NewError(connect.CodeInternal, err)
	}

	eventCh, unsubscribe := wb.SubscribeIndexProgress(ctx)
	defer unsubscribe()

	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	wb.StartAsyncIndexing(context.Background())
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			// Normal 30s stream cycle termination; the client will reconnect.
			return nil
		case event, ok := <-eventCh:
			if !ok {
				return nil
			}
			var protoState apiv1.WatchIndexProgressResponse_IndexState
			switch event.State {
			case workbench.IndexStateBuilding:
				protoState = apiv1.WatchIndexProgressResponse_INDEX_STATE_BUILDING
			case workbench.IndexStateReady:
				protoState = apiv1.WatchIndexProgressResponse_INDEX_STATE_READY
			case workbench.IndexStateFailed:
				protoState = apiv1.WatchIndexProgressResponse_INDEX_STATE_FAILED
			default:
				protoState = apiv1.WatchIndexProgressResponse_INDEX_STATE_UNSPECIFIED
			}

			res := &apiv1.WatchIndexProgressResponse{
				State:              protoState.Enum(),
				ProgressPercentage: proto.Float64(event.ProgressPercentage),
				Message:            proto.String(event.Message),
			}
			if err := stream.Send(res); err != nil {
				return err
			}

			if event.State == workbench.IndexStateReady || event.State == workbench.IndexStateFailed {
				return nil
			}
		}
	}
}

// PullIndexProgress retrieves the current search index construction progress and status snapshot for an active Workbench session.
func (s *WorkbenchServiceServer) PullIndexProgress(
	ctx context.Context,
	req *connect.Request[apiv1.PullIndexProgressRequest],
) (*connect.Response[apiv1.PullIndexProgressResponse], error) {
	msg := req.Msg
	if msg.GetWorkbenchId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workbench_id is required"))
	}

	wb, err := s.manager.GetAndTouch(msg.GetWorkbenchId())
	if err != nil {
		if errors.Is(err, workbench.ErrWorkbenchNotFound) || errors.Is(err, workbench.ErrWorkbenchClosed) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	wb.StartAsyncIndexing(context.Background())
	indexState, pct, statusMsg, _ := wb.IndexStatus()

	var protoState apiv1.WatchIndexProgressResponse_IndexState
	switch indexState {
	case workbench.IndexStateBuilding:
		protoState = apiv1.WatchIndexProgressResponse_INDEX_STATE_BUILDING
	case workbench.IndexStateReady:
		protoState = apiv1.WatchIndexProgressResponse_INDEX_STATE_READY
	case workbench.IndexStateFailed:
		protoState = apiv1.WatchIndexProgressResponse_INDEX_STATE_FAILED
	default:
		protoState = apiv1.WatchIndexProgressResponse_INDEX_STATE_UNSPECIFIED
	}

	return connect.NewResponse(&apiv1.PullIndexProgressResponse{
		State:              protoState.Enum(),
		ProgressPercentage: proto.Float64(pct),
		Message:            proto.String(statusMsg),
	}), nil
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

// ReadStructYAML decodes an interned struct by ID and returns its YAML representation.
func (s *WorkbenchServiceServer) ReadStructYAML(
	ctx context.Context,
	req *connect.Request[apiv1.ReadStructYAMLRequest],
) (*connect.Response[apiv1.ReadStructYAMLResponse], error) {
	msg := req.Msg
	if msg.GetWorkbenchId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workbench_id is required"))
	}
	if msg.GetStructId() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("struct_id must be greater than 0"))
	}

	wb, err := s.manager.GetAndTouch(msg.GetWorkbenchId())
	if err != nil {
		if errors.Is(err, workbench.ErrWorkbenchNotFound) || errors.Is(err, workbench.ErrWorkbenchClosed) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	yamlStr, err := wb.ReadStructYAML(msg.GetStructId())
	if err != nil {
		if errors.Is(err, workbench.ErrStructNotFound) || errors.Is(err, workbench.ErrWorkbenchClosed) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read struct YAML: %w", err))
	}

	res := &apiv1.ReadStructYAMLResponse{
		Yaml: proto.String(yamlStr),
	}
	return connect.NewResponse(res), nil
}

// FilterTimeline executes a timeline and log filtering pipeline on the server and streams progress updates followed by the final matched ID sets.
func (s *WorkbenchServiceServer) FilterTimeline(
	ctx context.Context,
	req *connect.Request[apiv1.FilterTimelineRequest],
	stream *connect.ServerStream[apiv1.FilterTimelineResponse],
) error {
	msg := req.Msg
	if msg.GetWorkbenchId() == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("workbench_id is required"))
	}

	wb, err := s.manager.GetAndTouch(msg.GetWorkbenchId())
	if err != nil {
		if errors.Is(err, workbench.ErrWorkbenchNotFound) || errors.Is(err, workbench.ErrWorkbenchClosed) {
			return connect.NewError(connect.CodeNotFound, err)
		}
		return connect.NewError(connect.CodeInternal, err)
	}

	params := workbench.FilterPipelineParams{
		TimelineQuery:          msg.GetTimelineQuery(),
		TimelineExclusionQuery: msg.GetTimelineExclusionQuery(),
		LogQuery:               msg.GetLogQuery(),
		ExcludeNoLogs:          msg.GetExcludeNoLogs(),
	}

	result, err := wb.FilterTimeline(ctx, params, func(progress *apiv1.FilterProgress) error {
		res := &apiv1.FilterTimelineResponse{
			Payload: &apiv1.FilterTimelineResponse_Progress{
				Progress: progress,
			},
		}
		return stream.Send(res)
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return connect.NewError(connect.CodeCanceled, err)
		}
		return connect.NewError(connect.CodeInternal, fmt.Errorf("filter pipeline failed: %w", err))
	}

	finalRes := &apiv1.FilterTimelineResponse{
		Payload: &apiv1.FilterTimelineResponse_Result{
			Result: result,
		},
	}
	return stream.Send(finalRes)
}

// FilterTimelineSync evaluates a timeline and log filtering pipeline synchronously or polls its in-progress status.
func (s *WorkbenchServiceServer) FilterTimelineSync(
	ctx context.Context,
	req *connect.Request[apiv1.FilterTimelineSyncRequest],
) (*connect.Response[apiv1.FilterTimelineSyncResponse], error) {
	msg := req.Msg
	if msg.GetWorkbenchId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workbench_id is required"))
	}

	wb, err := s.manager.GetAndTouch(msg.GetWorkbenchId())
	if err != nil {
		if errors.Is(err, workbench.ErrWorkbenchNotFound) || errors.Is(err, workbench.ErrWorkbenchClosed) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	params := workbench.FilterPipelineParams{
		TimelineQuery:          msg.GetTimelineQuery(),
		TimelineExclusionQuery: msg.GetTimelineExclusionQuery(),
		LogQuery:               msg.GetLogQuery(),
		ExcludeNoLogs:          msg.GetExcludeNoLogs(),
	}

	runner := func(jobCtx context.Context, onProgress func(*apiv1.FilterProgress) error) (*apiv1.FilterResult, error) {
		return wb.FilterTimeline(jobCtx, params, onProgress)
	}

	status, err := wb.FilterJobManager().Poll(ctx, msg.GetJobId(), 50*time.Millisecond, runner)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, connect.NewError(connect.CodeCanceled, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if status.IsDone {
		if status.Err != nil {
			return connect.NewResponse(&apiv1.FilterTimelineSyncResponse{
				JobId:        proto.String(status.JobID),
				IsDone:       proto.Bool(true),
				ErrorMessage: proto.String(status.Err.Error()),
			}), nil
		}
		return connect.NewResponse(&apiv1.FilterTimelineSyncResponse{
			JobId:  proto.String(status.JobID),
			IsDone: proto.Bool(true),
			Result: status.Result,
		}), nil
	}

	res := &apiv1.FilterTimelineSyncResponse{
		JobId:  proto.String(status.JobID),
		IsDone: proto.Bool(false),
	}
	if status.HasProgress && status.Progress != nil {
		res.Progress = status.Progress
	}
	return connect.NewResponse(res), nil
}

// CancelFilterTimelineSync cancels an in-progress synchronous timeline filtering task.
func (s *WorkbenchServiceServer) CancelFilterTimelineSync(
	ctx context.Context,
	req *connect.Request[apiv1.CancelFilterTimelineSyncRequest],
) (*connect.Response[apiv1.CancelFilterTimelineSyncResponse], error) {
	msg := req.Msg
	if msg.GetWorkbenchId() == "" || msg.GetJobId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workbench_id and job_id are required"))
	}

	wb, err := s.manager.GetAndTouch(msg.GetWorkbenchId())
	if err != nil {
		if errors.Is(err, workbench.ErrWorkbenchNotFound) || errors.Is(err, workbench.ErrWorkbenchClosed) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	canceled := wb.FilterJobManager().Cancel(msg.GetJobId())
	return connect.NewResponse(&apiv1.CancelFilterTimelineSyncResponse{
		Canceled: proto.Bool(canceled),
	}), nil
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
