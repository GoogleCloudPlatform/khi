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

package lifecycle

import "os"

var Default = NewLifecycleEventNotifier()

// LifecycleHandlerInit is the handler on initializing application.
type LifecycleHandlerInit = func()

// LifecycleHandlerTerminate is the handler on terminating application
type LifecycleHandlerTerminate = func(s os.Signal)

// LifecycleHandlerInspectionStart is the handler on starting a new inspection task.
type LifecycleHandlerInspectionStart = func(runId string, inspectionType string)

// LifecycleHandlerInspectionEnd is the handler on finishing a new inspection task.
type LifecycleHandlerInspectionEnd = func(runId string, inspectionType string, status string, size int)

// LifecycleEventHandler is a collection of event handlers called at the event on KHI backend.
type LifecycleEventHandler struct {
	OnInit            LifecycleHandlerInit
	OnTerminate       LifecycleHandlerTerminate
	OnInspectionStart LifecycleHandlerInspectionStart
	OnInspectionEnd   LifecycleHandlerInspectionEnd
}

// LifecycleEventNotifier provides api to call registered lifecycle event handlers.
type LifecycleEventNotifier struct {
	handlers []*LifecycleEventHandler
}

func NewLifecycleEventNotifier() *LifecycleEventNotifier {
	return &LifecycleEventNotifier{
		handlers: make([]*LifecycleEventHandler, 0),
	}
}

// AddHandler adds the given handler to be called on events.
func (n *LifecycleEventNotifier) AddHandler(handler *LifecycleEventHandler) {
	n.handlers = append(n.handlers, handler)
}

// NotifyInit calls OnInit function for each extensions if it was given in it's definition.
func (n *LifecycleEventNotifier) NotifyInit() {
	for _, handler := range n.handlers {
		if handler.OnInit != nil {
			handler.OnInit()
		}
	}
}

// NotifyTerminate calls OnTerminate function for each extensions if it was given in it's definition.
func (n *LifecycleEventNotifier) NotifyTerminate(s os.Signal) {
	for _, handler := range n.handlers {
		if handler.OnTerminate != nil {
			handler.OnTerminate(s)
		}
	}
}

// NotifyInspectionStart calls OnInspectionStart function for each extensions if it was given in it's definition.
func (n *LifecycleEventNotifier) NotifyInspectionStart(runId string, inspectionType string) {
	for _, handler := range n.handlers {
		if handler.OnInspectionStart != nil {
			handler.OnInspectionStart(runId, inspectionType)
		}
	}
}

// NotifyInspectionEnd calls OnInspectionEnd function for each extensions if it was given in it's definition.
func (n *LifecycleEventNotifier) NotifyInspectionEnd(runId string, inspectionType string, status string, size int) {
	for _, handler := range n.handlers {
		if handler.OnInspectionEnd != nil {
			handler.OnInspectionEnd(runId, inspectionType, status, size)
		}
	}
}
