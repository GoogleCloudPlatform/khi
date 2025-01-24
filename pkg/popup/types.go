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
	"fmt"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
)

var PopupOptionRedirectTargetKey = "redirectTo"

var NoCurrentPopup = fmt.Errorf("no active current popup")
var CurrentPopupIsntMatchingWithGivenId = fmt.Errorf("given id is not matching with the current popup")

// PopupForm is an abstract interface to represent the type to define the form shown from backend to frontend.
type PopupForm interface {
	// GetMetadata return the metadata type needed to show the form to frontend
	GetMetadata() PopupFormMetadata
	// Validate receives the input from frontend and returns validation result
	Validate(req *PopupAnswerResponse) string
}

// PopupFormMetadata contains data needed for showing input ui on frontend side.
type PopupFormMetadata struct {
	// The title of this form
	Title string
	// Type of input field. Currently, only `text` or `popup_redirect` is the supported value.
	Type string
	// Description of this form.
	Description string
	Placeholder string

	// The other option values of the request.
	Options map[string]string `json:"options"`
}

// PopupFormRequest is a popup display request that is actually passed to the frontend.
type PopupFormRequest struct {
	Id          string            `json:"id"`
	Title       string            `json:"title"`
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Placeholder string            `json:"placeholder"`
	Options     map[string]string `json:"options"`
}

// PopupAnswerResponse is the container of the data to validate/answer shown popup form.
type PopupAnswerResponse struct {
	Id    string `json:"id"`
	Value string `json:"value"`
}

// PopupAnswerValidationResult is the type passed from the frontend to validate the popup.
type PopupAnswerValidationResult struct {
	Id              string `json:"id"`
	ValidationError string `json:"validationError"`
}

// PopupManager receives questions shown to user from frontend.
type PopupManager struct {
	newPopupLock        sync.Mutex
	popupWaiter         sync.WaitGroup
	popupResult         string
	currentPopupRequest *PopupFormRequest
	currentPopup        PopupForm
}

func NewPopupManager() *PopupManager {
	return &PopupManager{
		newPopupLock:        sync.Mutex{},
		popupWaiter:         sync.WaitGroup{},
		popupResult:         "",
		currentPopupRequest: nil,
		currentPopup:        nil,
	}
}

// ShowPopup shows the popup UI on frontend side and wait until receiving the input.
func (p *PopupManager) ShowPopup(popup PopupForm) (string, error) {
	id := common.NewUUID()
	metadata := popup.GetMetadata()
	p.newPopupLock.Lock()
	defer p.newPopupLock.Unlock()
	p.currentPopup = popup
	p.currentPopupRequest = &PopupFormRequest{
		Id:          id,
		Title:       metadata.Title,
		Type:        metadata.Type,
		Description: metadata.Description,
		Placeholder: metadata.Placeholder,
		Options:     metadata.Options,
	}
	p.popupWaiter = sync.WaitGroup{}
	p.popupWaiter.Add(1)
	p.popupWaiter.Wait()
	return p.popupResult, nil
}

// GetCurrentPopup returns currently active popup request data needed in frontend side to show the popup
func (p *PopupManager) GetCurrentPopup() *PopupFormRequest {
	return p.currentPopupRequest
}

// Validate receives form input and check if the request is valid to receive. If it was not valid, it returns validation error in string.
func (p *PopupManager) Validate(request *PopupAnswerResponse) (*PopupAnswerValidationResult, error) {
	if p.currentPopupRequest == nil {
		return nil, NoCurrentPopup
	}
	if p.currentPopupRequest.Id != request.Id {
		return nil, CurrentPopupIsntMatchingWithGivenId
	}
	return &PopupAnswerValidationResult{
		Id:              request.Id,
		ValidationError: p.currentPopup.Validate(request),
	}, nil
}

// Answer determine the result of the form. This method assume the request is already validated before.
func (p *PopupManager) Answer(request *PopupAnswerResponse) error {
	if p.currentPopupRequest == nil {
		return NoCurrentPopup
	}
	if p.currentPopupRequest.Id != request.Id {
		return CurrentPopupIsntMatchingWithGivenId
	}
	p.popupResult = request.Value
	p.currentPopup = nil
	p.currentPopupRequest = nil
	p.popupWaiter.Done()
	return nil
}

var Instance *PopupManager = NewPopupManager()
