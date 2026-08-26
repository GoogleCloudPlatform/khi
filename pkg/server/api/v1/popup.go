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
	"time"

	"connectrpc.com/connect"
	v1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	"github.com/GoogleCloudPlatform/khi/pkg/server/popup"
)

// PopupServer implements the PopupService Connect-RPC handler.
type PopupServer struct {
	manager *popup.PopupManager
}

var _ apiv1connect.PopupServiceHandler = (*PopupServer)(nil)

// NewPopupServer creates a new instance of PopupServer.
func NewPopupServer(manager *popup.PopupManager) *PopupServer {
	if manager == nil {
		manager = popup.Instance
	}
	return &PopupServer{manager: manager}
}

// WatchPopup streams real-time popup events and closes every 30s to prevent proxy timeouts.
func (s *PopupServer) WatchPopup(
	ctx context.Context,
	req *connect.Request[v1.WatchPopupRequest],
	stream *connect.ServerStream[v1.WatchPopupResponse],
) error {
	events, unsubscribe := s.manager.Subscribe()
	defer unsubscribe()

	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			// Normal 30s stream cycle termination; the client will reconnect immediately.
			return nil
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			var resp *v1.WatchPopupResponse
			if ev.Type == popup.PopupEventTypeOpened && ev.Form != nil {
				resp = &v1.WatchPopupResponse{
					Event: &v1.WatchPopupResponse_Popup{
						Popup: ev.Form,
					},
				}
			} else if ev.Type == popup.PopupEventTypeDismissed {
				resp = &v1.WatchPopupResponse{
					Event: &v1.WatchPopupResponse_Dismissed{
						Dismissed: true,
					},
				}
			}
			if resp != nil {
				if err := stream.Send(resp); err != nil {
					return err
				}
			}
		}
	}
}

// PullPopup returns the currently active popup form snapshot.
func (s *PopupServer) PullPopup(
	ctx context.Context,
	req *connect.Request[v1.PullPopupRequest],
) (*connect.Response[v1.PullPopupResponse], error) {
	current := s.manager.GetCurrentPopup()
	return connect.NewResponse(&v1.PullPopupResponse{
		Popup: current,
	}), nil
}

// ValidatePopupAnswer validates in-progress input against the active popup.
func (s *PopupServer) ValidatePopupAnswer(
	ctx context.Context,
	req *connect.Request[v1.ValidatePopupAnswerRequest],
) (*connect.Response[v1.ValidatePopupAnswerResponse], error) {
	result, err := s.manager.Validate(ctx, req.Msg)
	if errors.Is(err, popup.NoCurrentPopup) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if errors.Is(err, popup.CurrentPopupIsntMatchingWithGivenId) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(result), nil
}

// SubmitPopupAnswer submits the finalized answer for the active popup.
func (s *PopupServer) SubmitPopupAnswer(
	ctx context.Context,
	req *connect.Request[v1.SubmitPopupAnswerRequest],
) (*connect.Response[v1.SubmitPopupAnswerResponse], error) {
	err := s.manager.Answer(ctx, req.Msg)
	if errors.Is(err, popup.NoCurrentPopup) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if errors.Is(err, popup.CurrentPopupIsntMatchingWithGivenId) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.SubmitPopupAnswerResponse{}), nil
}
