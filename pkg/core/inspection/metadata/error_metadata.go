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

package inspectionmetadata

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

type ErrorMessage struct {
	ErrorId int    `json:"errorId"`
	Message string `json:"message"`
	Link    string `json:"link"`
}

// ErrorMessageSetMetadata is a metadata type containing errors exposed to frontend.
type ErrorMessageSetMetadata struct {
	ErrorMessages []*ErrorMessage `json:"errorMessages"`
}

// Labels implements metadata.Metadata.
func (e *ErrorMessageSetMetadata) Labels() *typedmap.ReadonlyTypedMap {
	return NewLabelSet(IncludeInRunResult(), IncludeInTaskList())
}

// ToSerializable implements metadata.Metadata.
func (e *ErrorMessageSetMetadata) ToSerializable() interface{} {
	return e
}

var _ Metadata = (*ErrorMessageSetMetadata)(nil)

// AddErrorMessage stores a new ErrorMessage. Duplicated error message will be ignored.
func (e *ErrorMessageSetMetadata) AddErrorMessage(newError *ErrorMessage) {
	for _, msg := range e.ErrorMessages {
		if msg.ErrorId == newError.ErrorId {
			return // Skip adding duplicated error
		}
	}
	e.ErrorMessages = append(e.ErrorMessages, newError)
}

func NewUnauthorizedErrorMessage() *ErrorMessage {
	return &ErrorMessage{
		ErrorId: 2,
		Message: "Access token is not authorized. (Token expired?)",
	}
}

func NewErrorMessageSetMetadata() *ErrorMessageSetMetadata {
	return &ErrorMessageSetMetadata{
		ErrorMessages: []*ErrorMessage{},
	}
}
