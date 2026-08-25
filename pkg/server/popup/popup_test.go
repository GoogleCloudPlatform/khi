// Copyright 2024 Google LLC
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

package popup

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

type testPopupForm struct{}

// BuildProtoForm implements PopupForm.
func (t *testPopupForm) BuildProtoForm(id string) *apiv1.PopupForm {
	return &apiv1.PopupForm{
		Id:          proto.String(id),
		Title:       proto.String("foo"),
		Description: proto.String("baz"),
		Payload: &apiv1.PopupForm_Text{
			Text: &apiv1.TextPopupPayload{
				Placeholder: proto.String("qux"),
			},
		},
	}
}

// Validate implements PopupForm.
func (t *testPopupForm) Validate(_ context.Context, req *apiv1.ValidatePopupAnswerRequest) (*apiv1.ValidatePopupAnswerResponse, error) {
	errStr := ""
	if !strings.Contains(req.GetText().GetValue(), "ok") {
		errStr = "answer for test popup must contain ok"
	}
	return &apiv1.ValidatePopupAnswerResponse{
		Id:              proto.String(req.GetId()),
		ValidationError: proto.String(errStr),
	}, nil
}

// Answer implements PopupForm.
func (t *testPopupForm) Answer(_ context.Context, req *apiv1.SubmitPopupAnswerRequest) (string, error) {
	return req.GetText().GetValue(), nil
}

var _ PopupForm = (*testPopupForm)(nil)

func TestPopupManager(t *testing.T) {
	ctx := context.Background()
	pm := NewPopupManager()

	t.Run("GetCurrentPopup returns nil when no popup shown", func(t *testing.T) {
		cp := pm.GetCurrentPopup()
		if cp != nil {
			t.Error("expected nil but something returned")
		}
	})

	t.Run("ShowPopupRequest must be included in the GetCurrentPopup", func(t *testing.T) {
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			popupResult, err := pm.ShowPopup(&testPopupForm{})
			if err != nil {
				t.Errorf("%s", err.Error())
			}
			if popupResult != "ok" {
				t.Errorf("expected ok but got %s", popupResult)
			}
			wg.Done()
		}()
		<-time.After(50 * time.Millisecond)
		cp := pm.GetCurrentPopup()
		if cp == nil {
			t.Fatal("expected non-nil popup form")
		}
		if diff := cmp.Diff(&apiv1.PopupForm{
			Id:          cp.Id,
			Title:       proto.String("foo"),
			Description: proto.String("baz"),
			Payload: &apiv1.PopupForm_Text{
				Text: &apiv1.TextPopupPayload{
					Placeholder: proto.String("qux"),
				},
			},
		}, cp, protocmp.Transform()); diff != "" {
			t.Errorf("popup form mismatch (-want +got):\n%s", diff)
		}
		if cp.GetId() == "" {
			t.Error("id is empty")
		}
		_ = pm.Answer(ctx, &apiv1.SubmitPopupAnswerRequest{
			Id: cp.Id,
			Payload: &apiv1.SubmitPopupAnswerRequest_Text{
				Text: &apiv1.TextPopupAnswer{Value: proto.String("ok")},
			},
		})
		wg.Wait()
	})

	t.Run("Validate returns the result obtained from the Validate method on PopupForm", func(t *testing.T) {
		go func() {
			<-time.After(50 * time.Millisecond)
			p := pm.GetCurrentPopup()
			result, err := pm.Validate(ctx, &apiv1.ValidatePopupAnswerRequest{
				Id: p.Id,
				Payload: &apiv1.ValidatePopupAnswerRequest_Text{
					Text: &apiv1.TextPopupValidationRequest{Value: proto.String("ng")},
				},
			})
			if err != nil {
				t.Errorf("%s", err.Error())
			}
			if result.GetValidationError() != "answer for test popup must contain ok" {
				t.Errorf("expected answer for test popup must contain ok but got %s", result.GetValidationError())
			}

			result, err = pm.Validate(ctx, &apiv1.ValidatePopupAnswerRequest{
				Id: p.Id,
				Payload: &apiv1.ValidatePopupAnswerRequest_Text{
					Text: &apiv1.TextPopupValidationRequest{Value: proto.String("ok")},
				},
			})
			if err != nil {
				t.Errorf("%s", err.Error())
			}
			if result.GetValidationError() != "" {
				t.Errorf("expected empty but got %s", result.GetValidationError())
			}

			_ = pm.Answer(ctx, &apiv1.SubmitPopupAnswerRequest{
				Id: p.Id,
				Payload: &apiv1.SubmitPopupAnswerRequest_Text{
					Text: &apiv1.TextPopupAnswer{Value: proto.String("ok")},
				},
			})
		}()
		result, err := pm.ShowPopup(&testPopupForm{})
		if err != nil {
			t.Errorf("expected nil but got %s", err.Error())
		}
		if result != "ok" {
			t.Errorf("expected ok but got %s", result)
		}
	})

	t.Run("Validate returns an error when it got request for non current popup", func(t *testing.T) {
		go func() {
			<-time.After(50 * time.Millisecond)
			p := pm.GetCurrentPopup()
			_, err := pm.Validate(ctx, &apiv1.ValidatePopupAnswerRequest{
				Id: proto.String("foo"),
				Payload: &apiv1.ValidatePopupAnswerRequest_Text{
					Text: &apiv1.TextPopupValidationRequest{Value: proto.String("ok")},
				},
			})
			if err != CurrentPopupIsntMatchingWithGivenId {
				t.Errorf("%s", err.Error())
			}
			_ = pm.Answer(ctx, &apiv1.SubmitPopupAnswerRequest{
				Id: p.Id,
				Payload: &apiv1.SubmitPopupAnswerRequest_Text{
					Text: &apiv1.TextPopupAnswer{Value: proto.String("ok")},
				},
			})
		}()
		result, err := pm.ShowPopup(&testPopupForm{})
		if err != nil {
			t.Errorf("expected nil but got %s", err.Error())
		}
		if result != "ok" {
			t.Errorf("expected ok but got %s", result)
		}
	})

	t.Run("Answer returns an error when it got a request for non current popup", func(t *testing.T) {
		go func() {
			<-time.After(50 * time.Millisecond)
			p := pm.GetCurrentPopup()
			err := pm.Answer(ctx, &apiv1.SubmitPopupAnswerRequest{
				Id: proto.String("foo"),
				Payload: &apiv1.SubmitPopupAnswerRequest_Text{
					Text: &apiv1.TextPopupAnswer{Value: proto.String("ok")},
				},
			})
			if err != CurrentPopupIsntMatchingWithGivenId {
				t.Errorf("expected %s but got %s", CurrentPopupIsntMatchingWithGivenId, err)
			}
			_ = pm.Answer(ctx, &apiv1.SubmitPopupAnswerRequest{
				Id: p.Id,
				Payload: &apiv1.SubmitPopupAnswerRequest_Text{
					Text: &apiv1.TextPopupAnswer{Value: proto.String("ok")},
				},
			})
		}()
		result, err := pm.ShowPopup(&testPopupForm{})
		if err != nil {
			t.Errorf("expected nil but got %s", err.Error())
		}
		if result != "ok" {
			t.Errorf("expected ok but got %s", result)
		}
	})

	t.Run("Subscribe receives events for open and dismissed", func(t *testing.T) {
		events, unsubscribe := pm.Subscribe()
		defer unsubscribe()

		// Initial event when no popup is active must be dismissed.
		ev0 := <-events
		if ev0.Type != PopupEventTypeDismissed {
			t.Errorf("ev0.Type = %v, want %v", ev0.Type, PopupEventTypeDismissed)
		}

		go func() {
			time.Sleep(50 * time.Millisecond)
			p := pm.GetCurrentPopup()
			_ = pm.Answer(ctx, &apiv1.SubmitPopupAnswerRequest{
				Id: p.Id,
				Payload: &apiv1.SubmitPopupAnswerRequest_Text{
					Text: &apiv1.TextPopupAnswer{Value: proto.String("ok")},
				},
			})
		}()

		go func() {
			_, _ = pm.ShowPopup(&testPopupForm{})
		}()

		ev1 := <-events
		if ev1.Type != PopupEventTypeOpened {
			t.Errorf("ev1.Type = %v, want %v", ev1.Type, PopupEventTypeOpened)
		}
		if ev1.Form == nil || ev1.Form.GetTitle() != "foo" {
			t.Errorf("ev1.Form mismatch: %+v", ev1.Form)
		}

		ev2 := <-events
		if ev2.Type != PopupEventTypeDismissed {
			t.Errorf("ev2.Type = %v, want %v", ev2.Type, PopupEventTypeDismissed)
		}
	})

	t.Run("DismissActivePopup dismisses the current popup", func(t *testing.T) {
		events, unsubscribe := pm.Subscribe()
		defer unsubscribe()

		// Initial event when no popup is active must be dismissed.
		ev0 := <-events
		if ev0.Type != PopupEventTypeDismissed {
			t.Errorf("ev0.Type = %v, want %v", ev0.Type, PopupEventTypeDismissed)
		}

		go func() {
			time.Sleep(50 * time.Millisecond)
			p := pm.GetCurrentPopup()
			if p != nil {
				_ = pm.DismissActivePopup(p.GetId())
			}
		}()

		go func() {
			_, _ = pm.ShowPopup(&testPopupForm{})
		}()

		ev1 := <-events
		if ev1.Type != PopupEventTypeOpened {
			t.Errorf("ev1.Type = %v, want %v", ev1.Type, PopupEventTypeOpened)
		}

		ev2 := <-events
		if ev2.Type != PopupEventTypeDismissed {
			t.Errorf("ev2.Type = %v, want %v", ev2.Type, PopupEventTypeDismissed)
		}
	})

	t.Run("Subscribe emits initial event based on active popup state", func(t *testing.T) {
		// When no popup active, Subscribe emits PopupEventTypeDismissed.
		events1, unsub1 := pm.Subscribe()
		defer unsub1()
		ev := <-events1
		if ev.Type != PopupEventTypeDismissed {
			t.Errorf("expected initial dismissed event, got %v", ev.Type)
		}

		// Start a popup in background
		go func() {
			_, _ = pm.ShowPopup(&testPopupForm{})
		}()
		time.Sleep(50 * time.Millisecond)

		// When popup active, Subscribe emits PopupEventTypeOpened.
		events2, unsub2 := pm.Subscribe()
		defer unsub2()
		evActive := <-events2
		if evActive.Type != PopupEventTypeOpened || evActive.Form == nil || evActive.Form.GetTitle() != "foo" {
			t.Errorf("expected initial opened event with form, got %+v", evActive)
		}

		// Clean up active popup
		p := pm.GetCurrentPopup()
		if p != nil {
			_ = pm.DismissActivePopup(p.GetId())
		}
	})

	t.Run("Subscribe unsubscribe is idempotent", func(t *testing.T) {
		_, unsubscribe := pm.Subscribe()
		// Calling unsubscribe multiple times must not panic.
		unsubscribe()
		unsubscribe()
	})

	t.Run("ShowPopup serializes concurrent calls", func(t *testing.T) {
		seq := make([]string, 0, 2)
		var seqMu sync.Mutex

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			res, err := pm.ShowPopup(&testPopupForm{})
			if err != nil {
				t.Errorf("showPopup 1 error: %v", err)
			}
			seqMu.Lock()
			seq = append(seq, "1:"+res)
			seqMu.Unlock()
		}()

		// Wait briefly so goroutine 1 acquires showPopupMu first.
		time.Sleep(20 * time.Millisecond)

		go func() {
			defer wg.Done()
			res, err := pm.ShowPopup(&testPopupForm{})
			if err != nil {
				t.Errorf("showPopup 2 error: %v", err)
			}
			seqMu.Lock()
			seq = append(seq, "2:"+res)
			seqMu.Unlock()
		}()

		// Answer popup 1 first
		time.Sleep(30 * time.Millisecond)
		p1 := pm.GetCurrentPopup()
		if p1 == nil {
			t.Fatal("expected popup 1 to be active")
		}
		_ = pm.Answer(ctx, &apiv1.SubmitPopupAnswerRequest{
			Id: p1.Id,
			Payload: &apiv1.SubmitPopupAnswerRequest_Text{
				Text: &apiv1.TextPopupAnswer{Value: proto.String("ans1")},
			},
		})

		// Answer popup 2 second
		time.Sleep(30 * time.Millisecond)
		p2 := pm.GetCurrentPopup()
		if p2 == nil {
			t.Fatal("expected popup 2 to be active")
		}
		_ = pm.Answer(ctx, &apiv1.SubmitPopupAnswerRequest{
			Id: p2.Id,
			Payload: &apiv1.SubmitPopupAnswerRequest_Text{
				Text: &apiv1.TextPopupAnswer{Value: proto.String("ans2")},
			},
		})

		wg.Wait()

		wantSeq := []string{"1:ans1", "2:ans2"}
		if diff := cmp.Diff(wantSeq, seq); diff != "" {
			t.Errorf("ShowPopup sequence mismatch (-want +got):\n%s", diff)
		}
	})
}
