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

package error

import (
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

const ErrorMessageSetMetadataKey = "error"

type ErrorMessage struct {
	ErrorId int    `json:"errorId"`
	Message string `json:"message"`
	Link    string `json:"link"`
}

// ErrorMessageSet is a metadata type containing errors exposed to frontend.
type ErrorMessageSet struct {
	ErrorMessages []*ErrorMessage `json:"errorMessages"`
}

// Labels implements metadata.Metadata.
func (e *ErrorMessageSet) Labels() *task.LabelSet {
	return task.NewLabelSet(metadata.IncludeInRunResult(), metadata.IncludeInTaskList())
}

// ToSerializable implements metadata.Metadata.
func (e *ErrorMessageSet) ToSerializable() interface{} {
	return e
}

var _ metadata.Metadata = (*ErrorMessageSet)(nil)

// AddErrorMessage stores a new ErrorMessage. Duplicated error message will be ignored.
func (e *ErrorMessageSet) AddErrorMessage(newError *ErrorMessage) {
	for _, msg := range e.ErrorMessages {
		if msg.ErrorId == newError.ErrorId {
			return // Skip adding duplicated error
		}
	}
	e.ErrorMessages = append(e.ErrorMessages, newError)
}

func NewPermissionErrorMessage(projectId string) *ErrorMessage {
	return &ErrorMessage{
		ErrorId: 0,
		Message: fmt.Sprintf("Permission error to read logs from project `%s`", projectId),
	}
}

func NewNotFoundErrorMessage(projectId string) *ErrorMessage {
	return &ErrorMessage{
		ErrorId: 1,
		Message: fmt.Sprintf("Project `%s` not found", projectId),
	}
}

func NewUnauthorizedErrorMessage() *ErrorMessage {
	return &ErrorMessage{
		ErrorId: 2,
		Message: "Access token is not authorized. (Token expired?)",
	}
}

var _ metadata.MetadataFactory = (*ErrorMessageSetFactory)(nil)

type ErrorMessageSetFactory struct {
}

// Instanciate implements metadata.MetadataFactory.
func (e *ErrorMessageSetFactory) Instanciate() metadata.Metadata {
	return &ErrorMessageSet{[]*ErrorMessage{}}
}
