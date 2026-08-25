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
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	"github.com/GoogleCloudPlatform/khi/pkg/server/popup"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

type testForm struct {
	title string
}

func (f *testForm) BuildProtoForm(id string) *v1.PopupForm {
	return &v1.PopupForm{
		Id:          proto.String(id),
		Title:       proto.String(f.title),
		Description: proto.String("test desc"),
		Payload: &v1.PopupForm_Text{
			Text: &v1.TextPopupPayload{Placeholder: proto.String("test placeholder")},
		},
	}
}

func (f *testForm) Validate(_ context.Context, req *v1.ValidatePopupAnswerRequest) (*v1.ValidatePopupAnswerResponse, error) {
	errStr := ""
	if !strings.Contains(req.GetText().GetValue(), "valid") {
		errStr = "must contain valid"
	}
	return &v1.ValidatePopupAnswerResponse{
		Id:              proto.String(req.GetId()),
		ValidationError: proto.String(errStr),
	}, nil
}

func (f *testForm) Answer(_ context.Context, req *v1.SubmitPopupAnswerRequest) (string, error) {
	return req.GetText().GetValue(), nil
}

var _ popup.PopupForm = (*testForm)(nil)

type oauthTestForm struct {
	title string
}

func (f *oauthTestForm) BuildProtoForm(id string) *v1.PopupForm {
	return &v1.PopupForm{
		Id:          proto.String(id),
		Title:       proto.String(f.title),
		Description: proto.String("test desc"),
		Payload: &v1.PopupForm_OauthLogin{
			OauthLogin: &v1.OAuthLoginPopupPayload{AuthUrl: proto.String("http://example.com/oauth")},
		},
	}
}

func (f *oauthTestForm) Validate(_ context.Context, req *v1.ValidatePopupAnswerRequest) (*v1.ValidatePopupAnswerResponse, error) {
	return &v1.ValidatePopupAnswerResponse{
		Id: proto.String(req.GetId()),
	}, nil
}

func (f *oauthTestForm) Answer(_ context.Context, _ *v1.SubmitPopupAnswerRequest) (string, error) {
	return "", nil
}

var _ popup.PopupForm = (*oauthTestForm)(nil)

func TestPopupServer_WatchPopup(t *testing.T) {
	testCases := []struct {
		name      string
		form      popup.PopupForm
		wantPopup *v1.PopupForm
	}{
		{
			name: "text popup streamed",
			form: &testForm{title: "text popup"},
			wantPopup: &v1.PopupForm{
				Title:       proto.String("text popup"),
				Description: proto.String("test desc"),
				Payload: &v1.PopupForm_Text{
					Text: &v1.TextPopupPayload{Placeholder: proto.String("test placeholder")},
				},
			},
		},
		{
			name: "oauth login popup streamed",
			form: &oauthTestForm{title: "oauth popup"},
			wantPopup: &v1.PopupForm{
				Title:       proto.String("oauth popup"),
				Description: proto.String("test desc"),
				Payload: &v1.PopupForm_OauthLogin{
					OauthLogin: &v1.OAuthLoginPopupPayload{AuthUrl: proto.String("http://example.com/oauth")},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pm := popup.NewPopupManager()
			server := NewPopupServer(pm)

			mux := http.NewServeMux()
			path, handler := apiv1connect.NewPopupServiceHandler(server)
			mux.Handle(path, handler)
			ts := httptest.NewServer(mux)
			defer ts.Close()

			client := apiv1connect.NewPopupServiceClient(ts.Client(), ts.URL)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// Start popup before connecting stream
			go func() {
				_, _ = pm.ShowPopup(tc.form)
			}()

			time.Sleep(50 * time.Millisecond)

			// Trigger popup answer or dismissal later
			go func() {
				time.Sleep(100 * time.Millisecond)
				p := pm.GetCurrentPopup()
				if p != nil {
					_ = pm.DismissActivePopup(p.GetId())
				}
			}()

			stream, err := client.WatchPopup(ctx, connect.NewRequest(&v1.WatchPopupRequest{}))
			if err != nil {
				t.Fatalf("failed to open watch stream: %v", err)
			}

			// Expect opened event
			if !stream.Receive() {
				t.Fatalf("stream closed before receiving popup opened event: %v", stream.Err())
			}
			openedMsg := stream.Msg()
			if openedMsg.GetPopup() == nil {
				t.Fatalf("expected popup event, got %v", openedMsg)
			}
			if diff := cmp.Diff(tc.wantPopup, openedMsg.GetPopup(), protocmp.Transform(), protocmp.IgnoreFields(&v1.PopupForm{}, "id")); diff != "" {
				t.Errorf("popup mismatch (-want +got):\n%s", diff)
			}

			// Expect dismissed event
			if !stream.Receive() {
				t.Fatalf("stream closed before receiving dismissed event: %v", stream.Err())
			}
			dismissedMsg := stream.Msg()
			if !dismissedMsg.GetDismissed() {
				t.Errorf("expected dismissed event, got %v", dismissedMsg)
			}
		})
	}

	t.Run("watch popup stream with no active popup sends initial dismissed event", func(t *testing.T) {
		pm := popup.NewPopupManager()
		server := NewPopupServer(pm)

		mux := http.NewServeMux()
		path, handler := apiv1connect.NewPopupServiceHandler(server)
		mux.Handle(path, handler)
		ts := httptest.NewServer(mux)
		defer ts.Close()

		client := apiv1connect.NewPopupServiceClient(ts.Client(), ts.URL)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		stream, err := client.WatchPopup(ctx, connect.NewRequest(&v1.WatchPopupRequest{}))
		if err != nil {
			t.Fatalf("failed to open watch stream: %v", err)
		}

		if !stream.Receive() {
			t.Fatalf("stream closed before receiving initial dismissed event: %v", stream.Err())
		}
		msg := stream.Msg()
		if !msg.GetDismissed() {
			t.Errorf("expected initial dismissed event, got %v", msg)
		}
	})
}

func TestPopupServer_ValidatePopupAnswer(t *testing.T) {
	testCases := []struct {
		name       string
		inputValue string
		wantError  string
	}{
		{
			name:       "valid input",
			inputValue: "valid-cluster",
			wantError:  "",
		},
		{
			name:       "invalid input",
			inputValue: "wrong-input",
			wantError:  "must contain valid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pm := popup.NewPopupManager()
			server := NewPopupServer(pm)

			go func() {
				time.Sleep(200 * time.Millisecond)
				p := pm.GetCurrentPopup()
				if p != nil {
					_ = pm.DismissActivePopup(p.GetId())
				}
			}()

			go func() {
				_, _ = pm.ShowPopup(&testForm{title: "title"})
			}()

			time.Sleep(50 * time.Millisecond)
			current := pm.GetCurrentPopup()
			if current == nil {
				t.Fatal("expected active popup")
			}

			req := connect.NewRequest(&v1.ValidatePopupAnswerRequest{
				Id: current.Id,
				Payload: &v1.ValidatePopupAnswerRequest_Text{
					Text: &v1.TextPopupValidationRequest{Value: proto.String(tc.inputValue)},
				},
			})

			res, err := server.ValidatePopupAnswer(context.Background(), req)
			if err != nil {
				t.Fatalf("ValidatePopupAnswer failed: %v", err)
			}

			if diff := cmp.Diff(tc.wantError, res.Msg.GetValidationError()); diff != "" {
				t.Errorf("validation error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPopupServer_SubmitPopupAnswer(t *testing.T) {
	testCases := []struct {
		name       string
		inputValue string
		wantResult string
	}{
		{
			name:       "submit valid answer",
			inputValue: "valid-answer",
			wantResult: "valid-answer",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pm := popup.NewPopupManager()
			server := NewPopupServer(pm)

			resultChan := make(chan string)
			go func() {
				res, _ := pm.ShowPopup(&testForm{title: "title"})
				resultChan <- res
			}()

			time.Sleep(50 * time.Millisecond)
			current := pm.GetCurrentPopup()
			if current == nil {
				t.Fatal("expected active popup")
			}

			req := connect.NewRequest(&v1.SubmitPopupAnswerRequest{
				Id: current.Id,
				Payload: &v1.SubmitPopupAnswerRequest_Text{
					Text: &v1.TextPopupAnswer{Value: proto.String(tc.inputValue)},
				},
			})

			_, err := server.SubmitPopupAnswer(context.Background(), req)
			if err != nil {
				t.Fatalf("SubmitPopupAnswer failed: %v", err)
			}

			select {
			case got := <-resultChan:
				if diff := cmp.Diff(tc.wantResult, got); diff != "" {
					t.Errorf("answer result mismatch (-want +got):\n%s", diff)
				}
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for popup result")
			}
		})
	}
}
