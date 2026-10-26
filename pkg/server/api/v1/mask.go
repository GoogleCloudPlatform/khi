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
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/streamingutil"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"google.golang.org/protobuf/proto"
)

// MaskServiceServer implements the apiv1connect.MaskServiceHandler interface.
type MaskServiceServer struct {
	manager *workbench.WorkbenchManager
}

var _ apiv1connect.MaskServiceHandler = (*MaskServiceServer)(nil)

// NewMaskServiceServer creates a new MaskServiceServer backed by the given manager.
func NewMaskServiceServer(manager *workbench.WorkbenchManager) *MaskServiceServer {
	return &MaskServiceServer{
		manager: manager,
	}
}

// WatchStringPoolIndexProgress streams construction progress of the on-demand string pool trigram index.
// The server terminates the stream every 30s to accommodate proxy timeouts, and clients are expected to reconnect.
func (s *MaskServiceServer) WatchStringPoolIndexProgress(
	ctx context.Context,
	req *connect.Request[apiv1.WatchStringPoolIndexProgressRequest],
	stream *connect.ServerStream[apiv1.WatchStringPoolIndexProgressResponse],
) error {
	msg := req.Msg
	if msg.GetWorkbenchId() == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("workbench_id is required"))
	}

	wb, err := s.manager.Get(msg.GetWorkbenchId(), workbench.AccessorBrowser)
	if err != nil {
		if errors.Is(err, workbench.ErrWorkbenchNotFound) || errors.Is(err, workbench.ErrWorkbenchClosed) {
			return connect.NewError(connect.CodeNotFound, err)
		}
		return connect.NewError(connect.CodeInternal, err)
	}

	eventCh, unsubscribe, err := wb.SubscribeStringPoolIndexProgress(ctx)
	if err != nil {
		if errors.Is(err, workbench.ErrWorkbenchClosed) {
			return connect.NewError(connect.CodeNotFound, err)
		}
		return connect.NewError(connect.CodeInternal, err)
	}
	defer unsubscribe()

	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		case event, ok := <-eventCh:
			if !ok {
				return nil
			}
			protoState := toProtoIndexState(event.State)
			res := &apiv1.WatchStringPoolIndexProgressResponse{
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

// PullStringPoolIndexProgress retrieves a snapshot of the on-demand string pool trigram index construction progress.
func (s *MaskServiceServer) PullStringPoolIndexProgress(
	ctx context.Context,
	req *connect.Request[apiv1.PullStringPoolIndexProgressRequest],
) (*connect.Response[apiv1.PullStringPoolIndexProgressResponse], error) {
	msg := req.Msg
	if msg.GetWorkbenchId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workbench_id is required"))
	}

	wb, err := s.manager.Get(msg.GetWorkbenchId(), workbench.AccessorBrowser)
	if err != nil {
		if errors.Is(err, workbench.ErrWorkbenchNotFound) || errors.Is(err, workbench.ErrWorkbenchClosed) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	state, pct, statusMsg, _ := wb.StringPoolIndexStatus()
	protoState := toProtoIndexState(state)
	return connect.NewResponse(&apiv1.PullStringPoolIndexProgressResponse{
		State:              protoState.Enum(),
		ProgressPercentage: proto.Float64(pct),
		Message:            proto.String(statusMsg),
	}), nil
}

// ListWorkbenchStrings returns a paginated list of unique strings contained in the Workbench dataset for masking.
func (s *MaskServiceServer) ListWorkbenchStrings(
	ctx context.Context,
	req *connect.Request[apiv1.ListWorkbenchStringsRequest],
) (*connect.Response[apiv1.ListWorkbenchStringsResponse], error) {
	msg := req.Msg
	if msg.GetWorkbenchId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workbench_id is required"))
	}

	wb, err := s.manager.Get(msg.GetWorkbenchId(), workbench.AccessorBrowser)
	if err != nil {
		if errors.Is(err, workbench.ErrWorkbenchNotFound) || errors.Is(err, workbench.ErrWorkbenchClosed) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	values, total, err := wb.ListStrings(msg.GetQuery(), int(msg.GetOffset()), int(msg.GetLimit()))
	if err != nil {
		if errors.Is(err, workbench.ErrInvalidRegexQuery) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		if errors.Is(err, workbench.ErrWorkbenchClosed) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list workbench strings: %w", err))
	}

	return connect.NewResponse(&apiv1.ListWorkbenchStringsResponse{
		Values:     values,
		TotalCount: proto.Uint32(uint32(total)),
	}), nil
}

// resolveMaskRules validates incoming StringMaskRule messages and assigns non-colliding "<masked_N>" defaults when replacement is unset.
func resolveMaskRules(protoRules []*apiv1.StringMaskRule) ([]khifilev6model.MaskRule, error) {
	if len(protoRules) == 0 {
		return nil, errors.New("at least one masking rule is required")
	}

	usedReplacements := make(map[string]struct{}, len(protoRules))
	for _, r := range protoRules {
		if r.Replacement != nil {
			usedReplacements[*r.Replacement] = struct{}{}
		}
	}

	nextDefaultIndex := 1
	assignedDefaultByTarget := make(map[string]string, len(protoRules))
	rules := make([]khifilev6model.MaskRule, 0, len(protoRules))
	for _, r := range protoRules {
		target := r.GetTarget()
		if target == "" {
			return nil, errors.New("masking rule target cannot be empty")
		}
		var replacement string
		if r.Replacement == nil {
			if existing, ok := assignedDefaultByTarget[target]; ok {
				replacement = existing
			} else {
				for {
					candidate := khifilev6model.DefaultMaskReplacement(nextDefaultIndex)
					nextDefaultIndex++
					if _, taken := usedReplacements[candidate]; !taken {
						replacement = candidate
						usedReplacements[candidate] = struct{}{}
						assignedDefaultByTarget[target] = candidate
						break
					}
				}
			}
		} else {
			replacement = *r.Replacement
		}
		rules = append(rules, khifilev6model.MaskRule{
			Target:      target,
			Replacement: replacement,
		})
	}
	return rules, nil
}

// runCreateMaskedInspection executes WorkbenchManager.CreateMaskedInspection and converts progress and result into Protobuf messages.
func (s *MaskServiceServer) runCreateMaskedInspection(
	ctx context.Context,
	workbenchID string,
	rules []khifilev6model.MaskRule,
	onProgress func(*apiv1.MaskedInspectionProgress) error,
) (*apiv1.MaskedInspectionResult, error) {
	result, err := s.manager.CreateMaskedInspection(ctx, workbenchID, rules, func(pct float64, message string) error {
		if onProgress == nil {
			return nil
		}
		return onProgress(&apiv1.MaskedInspectionProgress{
			ProgressPercentage: proto.Float64(pct),
			Message:            proto.String(message),
		})
	})
	if err != nil {
		return nil, err
	}
	return &apiv1.MaskedInspectionResult{
		InspectionId:      proto.String(result.InspectionID),
		InspectionName:    proto.String(result.InspectionName),
		SuggestedFilename: proto.String(result.SuggestedFilename),
		FileSizeBytes:     proto.Int64(result.FileSizeBytes),
	}, nil
}

// CreateMaskedInspection creates and registers a new inspection archive with string masking rules applied, streaming progress updates followed by the final metadata.
func (s *MaskServiceServer) CreateMaskedInspection(
	ctx context.Context,
	req *connect.Request[apiv1.CreateMaskedInspectionRequest],
	stream *connect.ServerStream[apiv1.CreateMaskedInspectionResponse],
) error {
	msg := req.Msg
	if msg.GetWorkbenchId() == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("workbench_id is required"))
	}
	rules, err := resolveMaskRules(msg.GetRules())
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	if _, err := s.manager.Get(msg.GetWorkbenchId(), workbench.AccessorBrowser); err != nil {
		if errors.Is(err, workbench.ErrWorkbenchNotFound) || errors.Is(err, workbench.ErrWorkbenchClosed) {
			return connect.NewError(connect.CodeNotFound, err)
		}
		return connect.NewError(connect.CodeInternal, err)
	}

	err = streamingutil.StreamJob(
		ctx,
		stream,
		func(jobCtx context.Context, onProgress func(*apiv1.MaskedInspectionProgress) error) (*apiv1.MaskedInspectionResult, error) {
			return s.runCreateMaskedInspection(jobCtx, msg.GetWorkbenchId(), rules, onProgress)
		},
		func(progress *apiv1.MaskedInspectionProgress) *apiv1.CreateMaskedInspectionResponse {
			return &apiv1.CreateMaskedInspectionResponse{
				Payload: &apiv1.CreateMaskedInspectionResponse_Progress{
					Progress: progress,
				},
			}
		},
		func(result *apiv1.MaskedInspectionResult) *apiv1.CreateMaskedInspectionResponse {
			return &apiv1.CreateMaskedInspectionResponse{
				Payload: &apiv1.CreateMaskedInspectionResponse_Result{
					Result: result,
				},
			}
		},
	)
	if err != nil {
		if errors.Is(err, workbench.ErrWorkbenchNotFound) || errors.Is(err, workbench.ErrWorkbenchClosed) {
			return connect.NewError(connect.CodeNotFound, err)
		}
		if errors.Is(err, context.Canceled) {
			return connect.NewError(connect.CodeCanceled, err)
		}
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create masked inspection: %w", err))
	}
	return nil
}

// CreateMaskedInspectionSync starts or polls an asynchronous masked inspection creation task without opening a persistent stream.
func (s *MaskServiceServer) CreateMaskedInspectionSync(
	ctx context.Context,
	req *connect.Request[apiv1.CreateMaskedInspectionSyncRequest],
) (*connect.Response[apiv1.CreateMaskedInspectionSyncResponse], error) {
	msg := req.Msg
	if msg.GetWorkbenchId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workbench_id is required"))
	}

	wb, err := s.manager.Get(msg.GetWorkbenchId(), workbench.AccessorBrowser)
	if err != nil {
		if errors.Is(err, workbench.ErrWorkbenchNotFound) || errors.Is(err, workbench.ErrWorkbenchClosed) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var rules []khifilev6model.MaskRule
	if msg.GetJobId() == "" {
		rules, err = resolveMaskRules(msg.GetRules())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}

	runner := func(jobCtx context.Context, onProgress func(*apiv1.MaskedInspectionProgress) error) (*apiv1.MaskedInspectionResult, error) {
		return s.runCreateMaskedInspection(jobCtx, msg.GetWorkbenchId(), rules, onProgress)
	}

	status, err := wb.MaskedExportJobManager().Poll(ctx, msg.GetJobId(), 50*time.Millisecond, runner)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, connect.NewError(connect.CodeCanceled, err)
		}
		return nil, err
	}

	if status.IsDone {
		if status.Err != nil {
			return connect.NewResponse(&apiv1.CreateMaskedInspectionSyncResponse{
				JobId:        proto.String(status.JobID),
				IsDone:       proto.Bool(true),
				ErrorMessage: proto.String(status.Err.Error()),
			}), nil
		}
		return connect.NewResponse(&apiv1.CreateMaskedInspectionSyncResponse{
			JobId:  proto.String(status.JobID),
			IsDone: proto.Bool(true),
			Result: status.Result,
		}), nil
	}

	res := &apiv1.CreateMaskedInspectionSyncResponse{
		JobId:  proto.String(status.JobID),
		IsDone: proto.Bool(false),
	}
	if status.HasProgress && status.Progress != nil {
		res.Progress = status.Progress
	}
	return connect.NewResponse(res), nil
}

// CancelCreateMaskedInspectionSync cancels an in-progress asynchronous masked inspection creation task.
func (s *MaskServiceServer) CancelCreateMaskedInspectionSync(
	ctx context.Context,
	req *connect.Request[apiv1.CancelCreateMaskedInspectionSyncRequest],
) (*connect.Response[apiv1.CancelCreateMaskedInspectionSyncResponse], error) {
	msg := req.Msg
	if msg.GetWorkbenchId() == "" || msg.GetJobId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workbench_id and job_id are required"))
	}

	wb, err := s.manager.Get(msg.GetWorkbenchId(), workbench.AccessorBrowser)
	if err != nil {
		if errors.Is(err, workbench.ErrWorkbenchNotFound) || errors.Is(err, workbench.ErrWorkbenchClosed) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	canceled := wb.MaskedExportJobManager().Cancel(msg.GetJobId())
	return connect.NewResponse(&apiv1.CancelCreateMaskedInspectionSyncResponse{
		Canceled: proto.Bool(canceled),
	}), nil
}

func toProtoIndexState(state workbench.IndexState) apiv1.WatchIndexProgressResponse_IndexState {
	switch state {
	case workbench.IndexStateBuilding:
		return apiv1.WatchIndexProgressResponse_INDEX_STATE_BUILDING
	case workbench.IndexStateReady:
		return apiv1.WatchIndexProgressResponse_INDEX_STATE_READY
	case workbench.IndexStateFailed:
		return apiv1.WatchIndexProgressResponse_INDEX_STATE_FAILED
	default:
		return apiv1.WatchIndexProgressResponse_INDEX_STATE_UNSPECIFIED
	}
}
