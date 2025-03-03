// Copyright 2025 Google LLC
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

package form

import (
	"context"
	"fmt"

	form_metadata "github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/form"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	common_task "github.com/GoogleCloudPlatform/khi/pkg/task"
)

type FileFormTaskBuilder struct {
	id           string
	label        string
	priority     int
	dependencies []string
	verifier     upload.UploadFileVerifier
}

func NewFileFormTaskBuilder(id string, priority int, label string, verifier upload.UploadFileVerifier) *FileFormTaskBuilder {
	return &FileFormTaskBuilder{
		id:       id,
		priority: priority,
		label:    label,
		verifier: verifier,
	}
}

func (b *FileFormTaskBuilder) Build() common_task.Definition {
	return common_task.NewProcessorTask(b.id, b.dependencies, func(ctx context.Context, taskMode int, v *common_task.VariableSet) (any, error) {
		m, err := task.GetMetadataSetFromVariable(v)
		if err != nil {
			return nil, err
		}

		token := upload.DefaultUploadFileStore.GetUploadToken(upload.GenerateUploadIDWithTaskContext(ctx, b.id), b.verifier)
		status, err := upload.DefaultUploadFileStore.GetResult(token)
		if err != nil {
			return nil, err
		}
		field := form_metadata.FileParameterFormField{
			ParameterFormFieldBase: form_metadata.ParameterFormFieldBase{
				ID:       b.id,
				Type:     form_metadata.File,
				Label:    b.label,
				Priority: b.priority,
				HintType: form_metadata.None,
				Hint:     "",
			},
			Token:  token,
			Status: status.Status,
		}

		if status.UploadError != nil {
			field.Hint = status.UploadError.Error()
			field.HintType = form_metadata.Error
		} else if status.VerificationError != nil {
			field.Hint = status.VerificationError.Error()
			field.HintType = form_metadata.Error
		} else if status.Status == upload.UploadStatusWaiting {
			field.Hint = "Waiting a file to be uploaded."
			field.HintType = form_metadata.Error
		} else if status.Status != upload.UploadStatusCompleted {
			field.Hint = "File is being processed. Please wait a moment."
			field.HintType = form_metadata.Error
		}

		formFields := m.LoadOrStore(form_metadata.FormFieldSetMetadataKey, &form_metadata.FormFieldSetMetadataFactory{}).(*form_metadata.FormFieldSet)
		err = formFields.SetField(field)
		if err != nil {
			return nil, fmt.Errorf("failed to configure the form metadata in task `%s`\n%v", b.id, err)
		}
		return nil, nil
	})
}
