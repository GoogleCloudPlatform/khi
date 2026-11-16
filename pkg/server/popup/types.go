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
	"fmt"
	"sync"

	"google.golang.org/protobuf/proto"

	"github.com/GoogleCloudPlatform/khi/pkg/common/idgenerator"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
)

const (
	// PopupEventTypeOpened indicates a new popup has been displayed.
	PopupEventTypeOpened PopupEventType = iota
	// PopupEventTypeDismissed indicates the active popup has been closed or answered.
	PopupEventTypeDismissed
)

var (
	// NoCurrentPopup indicates that an operation was attempted when no popup was active.
	NoCurrentPopup = fmt.Errorf("no active current popup")
	// CurrentPopupIsntMatchingWithGivenId indicates that the requested popup ID does not match the active popup.
	CurrentPopupIsntMatchingWithGivenId = fmt.Errorf("given id is not matching with the current popup")
	// Instance is the singleton instance of PopupManager.
	Instance *PopupManager = NewPopupManager()
)

var popupIDGenerator = idgenerator.NewPrefixIDGenerator("popup-")

var _ PopupForm = (*TextPopupForm)(nil)

// PopupForm is a polymorphic interface for backend popup implementations.
type PopupForm interface {
	// BuildProtoForm constructs the proto message for this popup with the given ID.
	BuildProtoForm(id string) *apiv1.PopupForm
	// Validate handles answer validation for this popup.
	Validate(ctx context.Context, req *apiv1.ValidatePopupAnswerRequest) (*apiv1.ValidatePopupAnswerResponse, error)
	// Answer handles answer submission for this popup. Returns the resolved result string for ShowPopup.
	Answer(ctx context.Context, req *apiv1.SubmitPopupAnswerRequest) (string, error)
}

// TextPopupForm represents a standard text input popup.
type TextPopupForm struct {
	// Title is the heading displayed in the popup dialog.
	Title string
	// Description is the explanatory text shown below the title.
	Description string
	// Placeholder is the hint text displayed inside the empty text input.
	Placeholder string
	// Validator validates user input and returns an error message when invalid.
	Validator func(value string) string
}

// PopupEventType represents the lifecycle state change of a popup form.
type PopupEventType int

// PopupEvent represents a popup lifecycle event delivered to subscribers.
type PopupEvent struct {
	// Type indicates whether the popup was opened or dismissed.
	Type PopupEventType
	// Form holds the active popup definition when Type is PopupEventTypeOpened.
	Form *apiv1.PopupForm
}

// PopupManager manages questions shown to user from frontend.
type PopupManager struct {
	mu           sync.Mutex
	showPopupMu  sync.Mutex
	popupWaiter  chan struct{}
	popupResult  string
	currentProto *apiv1.PopupForm
	currentForm  PopupForm
	subscribers  map[chan PopupEvent]struct{}
}

// BuildProtoForm implements PopupForm.
func (t *TextPopupForm) BuildProtoForm(id string) *apiv1.PopupForm {
	return &apiv1.PopupForm{
		Id:          proto.String(id),
		Title:       proto.String(t.Title),
		Description: proto.String(t.Description),
		Payload: &apiv1.PopupForm_Text{
			Text: &apiv1.TextPopupPayload{
				Placeholder: proto.String(t.Placeholder),
			},
		},
	}
}

// Validate implements PopupForm.
func (t *TextPopupForm) Validate(_ context.Context, req *apiv1.ValidatePopupAnswerRequest) (*apiv1.ValidatePopupAnswerResponse, error) {
	var errStr string
	if t.Validator != nil {
		errStr = t.Validator(req.GetText().GetValue())
	}
	return &apiv1.ValidatePopupAnswerResponse{
		Id:              proto.String(req.GetId()),
		ValidationError: proto.String(errStr),
	}, nil
}

// Answer implements PopupForm.
func (t *TextPopupForm) Answer(_ context.Context, req *apiv1.SubmitPopupAnswerRequest) (string, error) {
	return req.GetText().GetValue(), nil
}

// NewPopupManager creates an initialized instance of PopupManager.
func NewPopupManager() *PopupManager {
	return &PopupManager{
		mu:           sync.Mutex{},
		showPopupMu:  sync.Mutex{},
		popupWaiter:  nil,
		popupResult:  "",
		currentProto: nil,
		currentForm:  nil,
		subscribers:  make(map[chan PopupEvent]struct{}),
	}
}

// Subscribe registers a listener for popup lifecycle events.
// It returns a receive-only channel and an unsubscribe function to clean up resources.
func (p *PopupManager) Subscribe() (<-chan PopupEvent, func()) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ch := make(chan PopupEvent, 10)
	if p.subscribers == nil {
		p.subscribers = make(map[chan PopupEvent]struct{})
	}
	p.subscribers[ch] = struct{}{}

	if p.currentProto != nil {
		ch <- PopupEvent{
			Type: PopupEventTypeOpened,
			Form: p.currentProto,
		}
	} else {
		ch <- PopupEvent{
			Type: PopupEventTypeDismissed,
		}
	}

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			p.mu.Lock()
			defer p.mu.Unlock()
			delete(p.subscribers, ch)
			close(ch)
		})
	}
	return ch, unsubscribe
}

// ShowPopup shows the popup UI on frontend side and waits until receiving the input.
func (p *PopupManager) ShowPopup(popup PopupForm) (string, error) {
	p.showPopupMu.Lock()
	defer p.showPopupMu.Unlock()

	id := popupIDGenerator.Generate()
	protoForm := popup.BuildProtoForm(id)

	waiter := make(chan struct{})

	p.mu.Lock()
	p.currentForm = popup
	p.currentProto = protoForm
	p.popupWaiter = waiter
	p.broadcastLocked(PopupEvent{
		Type: PopupEventTypeOpened,
		Form: p.currentProto,
	})
	p.mu.Unlock()

	<-waiter

	p.mu.Lock()
	result := p.popupResult
	p.mu.Unlock()

	return result, nil
}

// GetCurrentPopup returns currently active popup proto data needed in frontend side to show the popup.
func (p *PopupManager) GetCurrentPopup() *apiv1.PopupForm {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentProto
}

// Validate receives form input and checks if the request is valid to receive.
func (p *PopupManager) Validate(ctx context.Context, request *apiv1.ValidatePopupAnswerRequest) (*apiv1.ValidatePopupAnswerResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentProto == nil || p.currentForm == nil {
		return nil, NoCurrentPopup
	}
	if p.currentProto.GetId() != request.GetId() {
		return nil, CurrentPopupIsntMatchingWithGivenId
	}
	return p.currentForm.Validate(ctx, request)
}

// Answer processes the finalized answer for the active popup.
func (p *PopupManager) Answer(ctx context.Context, request *apiv1.SubmitPopupAnswerRequest) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentProto == nil || p.currentForm == nil {
		return NoCurrentPopup
	}
	if p.currentProto.GetId() != request.GetId() {
		return CurrentPopupIsntMatchingWithGivenId
	}
	res, err := p.currentForm.Answer(ctx, request)
	if err != nil {
		return err
	}
	p.popupResult = res
	p.currentForm = nil
	p.currentProto = nil
	if p.popupWaiter != nil {
		close(p.popupWaiter)
		p.popupWaiter = nil
	}
	p.broadcastLocked(PopupEvent{
		Type: PopupEventTypeDismissed,
	})
	return nil
}

// DismissActivePopup dismisses the active popup from the server side without waiting for user submission.
func (p *PopupManager) DismissActivePopup(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentProto == nil || p.currentForm == nil {
		return NoCurrentPopup
	}
	if p.currentProto.GetId() != id {
		return CurrentPopupIsntMatchingWithGivenId
	}
	p.currentForm = nil
	p.currentProto = nil
	if p.popupWaiter != nil {
		close(p.popupWaiter)
		p.popupWaiter = nil
	}
	p.broadcastLocked(PopupEvent{
		Type: PopupEventTypeDismissed,
	})
	return nil
}

func (p *PopupManager) broadcastLocked(ev PopupEvent) {
	for ch := range p.subscribers {
		select {
		case ch <- ev:
		default:
		}
	}
}
