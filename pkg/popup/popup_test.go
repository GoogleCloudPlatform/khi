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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

type testPopupForm struct{}

// GetMetadata implements PopupForm.
func (t *testPopupForm) GetMetadata() PopupFormMetadata {
	return PopupFormMetadata{
		Title:       "foo",
		Type:        "bar",
		Description: "baz",
		Placeholder: "qux",
	}
}

// Validate implements PopupForm.
func (t *testPopupForm) Validate(req *PopupAnswerResponse) string {
	if strings.Contains(req.Value, "ok") {
		return ""
	} else {
		return "answer for test popup must contain ok"
	}
}

var _ PopupForm = &testPopupForm{}

func TestPopupManager(t *testing.T) {
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
		<-time.After(time.Second)
		cp := pm.GetCurrentPopup()
		if diff := cmp.Diff(cp, &PopupFormRequest{
			Title:       "foo",
			Type:        "bar",
			Description: "baz",
			Placeholder: "qux",
		}, cmpopts.IgnoreFields(PopupFormRequest{}, "Id")); diff != "" {
			t.Error(diff)
		}
		if cp.Id == "" {
			t.Error("Id is empty")
		}
		pm.Answer(&PopupAnswerResponse{
			Id:    cp.Id,
			Value: "ok",
		})
		wg.Wait()
	})

	t.Run("Validate returns the result obtained from the Validate method on PopupForm", func(t *testing.T) {
		go func() {
			<-time.After(time.Second)
			p := pm.GetCurrentPopup()
			result, err := pm.Validate(&PopupAnswerResponse{
				Id:    p.Id,
				Value: "ng",
			})
			if err != nil {
				t.Errorf("%s", err.Error())
			}
			if result.ValidationError != "answer for test popup must contain ok" {
				t.Errorf("expected answer for test popup must contain ok but got %s", result.ValidationError)
			}

			result, err = pm.Validate(&PopupAnswerResponse{
				Id:    p.Id,
				Value: "ok",
			})
			if err != nil {
				t.Errorf("%s", err.Error())
			}
			if result.ValidationError != "" {
				t.Errorf("expected empty but got %s", result.ValidationError)
			}

			pm.Answer(&PopupAnswerResponse{
				Id:    p.Id,
				Value: "ok",
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
			<-time.After(time.Second)
			p := pm.GetCurrentPopup()
			_, err := pm.Validate(&PopupAnswerResponse{
				Id:    "foo",
				Value: "ok",
			})
			if err != CurrentPopupIsntMatchingWithGivenId {
				t.Errorf("%s", err.Error())
			}
			pm.Answer(&PopupAnswerResponse{
				Id:    p.Id,
				Value: "ok",
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
			<-time.After(time.Second)
			p := pm.GetCurrentPopup()
			err := pm.Answer(&PopupAnswerResponse{
				Id:    "foo",
				Value: "ok",
			})
			if err != CurrentPopupIsntMatchingWithGivenId {
				t.Errorf("expected %s but got %s", CurrentPopupIsntMatchingWithGivenId, err)
			}
			pm.Answer(&PopupAnswerResponse{
				Id:    p.Id,
				Value: "ok",
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
}
