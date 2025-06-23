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

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectioncontract "github.com/GoogleCloudPlatform/khi/pkg/inspection/contract"
	form_metadata "github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/form"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	common_task "github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

type FileFormTaskBuilder struct {
	FormTaskBuilderBase[upload.UploadResult]
	verifier upload.UploadFileVerifier
}

func NewFileFormTaskBuilder(id taskid.TaskImplementationID[upload.UploadResult], priority int, label string, verifier upload.UploadFileVerifier) *FileFormTaskBuilder {
	return &FileFormTaskBuilder{
		FormTaskBuilderBase: NewFormTaskBuilderBase(id, priority, label),
		verifier:            verifier,
	}
}

// WithDependencies sets the task dependencies
func (b *FileFormTaskBuilder) WithDependencies(dependencies []taskid.UntypedTaskReference) *FileFormTaskBuilder {
	b.FormTaskBuilderBase.WithDependencies(dependencies)
	return b
}

// WithDescription sets the description for the form field
func (b *FileFormTaskBuilder) WithDescription(description string) *FileFormTaskBuilder {
	b.FormTaskBuilderBase.WithDescription(description)
	return b
}

func (b *FileFormTaskBuilder) Build(labelOpts ...common_task.LabelOpt) common_task.Task[upload.UploadResult] {
	return common_task.NewTask(b.id, b.dependencies, func(ctx context.Context) (upload.UploadResult, error) {
		metadata := khictx.MustGetValue(ctx, inspectioncontract.InspectionRunMetadata)

		token := upload.DefaultUploadFileStore.GetUploadToken(upload.GenerateUploadIDWithTaskContext(ctx, b.id.ReferenceIDString()), b.verifier)
		uploadResult, err := upload.DefaultUploadFileStore.GetResult(token)
		if err != nil {
			return upload.UploadResult{}, err
		}
		field := form_metadata.FileParameterFormField{
			ParameterFormFieldBase: form_metadata.ParameterFormFieldBase{
				Type:     form_metadata.File,
				HintType: form_metadata.None,
				Hint:     "",
			},
			Token:  token,
			Status: uploadResult.Status,
		}
		b.SetupBaseFormField(&field.ParameterFormFieldBase)

		field = setFormHintsFromUploadResult(uploadResult, field)
		formFields, found := typedmap.Get(metadata, form_metadata.FormFieldSetMetadataKey)
		if !found {
			return upload.UploadResult{}, fmt.Errorf("failed to get form fields from metadata")
		}
		err = formFields.SetField(field)
		if err != nil {
			return upload.UploadResult{}, fmt.Errorf("failed to configure the form metadata in task `%s`\n%v", b.id, err)
		}

		return uploadResult, nil
	}, labelOpts...)
}

// setFormHintsFromUploadResult sets the appropriate hint and hint type on a form field
// based on the upload result status and any errors encountered during the upload process.
func setFormHintsFromUploadResult(result upload.UploadResult, field form_metadata.FileParameterFormField) form_metadata.FileParameterFormField {
	switch {
	case result.UploadError != nil:
		field.Hint = result.UploadError.Error()
		field.HintType = form_metadata.Error
	case result.VerificationError != nil:
		field.Hint = result.VerificationError.Error()
		field.HintType = form_metadata.Error
	case result.Status == upload.UploadStatusWaiting:
		field.Hint = "Waiting a file to be uploaded."
		field.HintType = form_metadata.Error
	case result.Status != upload.UploadStatusCompleted:
		field.Hint = "File is being processed. Please wait a moment."
		field.HintType = form_metadata.Error
	}
	return field
}
