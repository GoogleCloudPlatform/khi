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

package formtask

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// DefineFileForm defines a form task that receives a file uploaded by the user and returns the upload result.
// verifier checks the uploaded file. The form reads no input, so it declares nothing on the Binder.
func DefineFileForm(id taskid.TaskImplementationID[upload.UploadResult], priority int, label string, description string, verifier upload.UploadFileVerifier, labelOpts ...coretask.LabelOpt) coretask.Task[upload.UploadResult] {
	form := newFormTaskBase(id, priority, label, description)
	return coretask.Define(id, func(_ *coretask.Binder) func(ctx context.Context) (upload.UploadResult, error) {
		return func(ctx context.Context) (upload.UploadResult, error) {
			metadata := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)

			req := khictx.MustGetValue(ctx, inspectioncore.InspectionTaskInput)
			taskMode := khictx.MustGetValue(ctx, inspectioncore.InspectionTaskMode)

			fieldID := id.ReferenceIDString()
			token := upload.DefaultUploadFileStore.GetUploadToken(GenerateUploadIDWithTaskContext(ctx, fieldID), verifier, fieldID)

			var uploadResult upload.UploadResult
			var err error
			if taskMode == inspectioncore.TaskModeRun {
				uploadResult, err = upload.DefaultUploadFileStore.GetCompletedResult(ctx, token, req)
			} else {
				uploadResult, err = upload.DefaultUploadFileStore.GetResult(token, req)
			}
			if err != nil {
				return upload.UploadResult{}, err
			}
			field := inspectionmetadata.FileParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Type:     inspectionmetadata.File,
					HintType: inspectionmetadata.None,
					Hint:     "",
				},
				Token:     token,
				Status:    uploadResult.Status,
				FileName:  uploadResult.FileName,
				SizeBytes: uploadResult.SizeBytes,
			}
			form.setupBaseFormField(&field.ParameterFormFieldBase)

			field = setFormHintsFromUploadResult(uploadResult, field)
			formFields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
			if !found {
				return upload.UploadResult{}, fmt.Errorf("failed to get form fields from metadata")
			}
			err = formFields.SetField(field)
			if err != nil {
				return upload.UploadResult{}, fmt.Errorf("failed to configure the form metadata in task `%s`\n%v", id, err)
			}

			if taskMode == inspectioncore.TaskModeRun {
				if uploadResult.UploadError != nil {
					return upload.UploadResult{}, fmt.Errorf("file upload failed in task %s: %w", id, uploadResult.UploadError)
				}
				if uploadResult.VerificationError != nil {
					return upload.UploadResult{}, fmt.Errorf("file verification failed in task %s: %w", id, uploadResult.VerificationError)
				}
				if uploadResult.Status != upload.UploadStatusCompleted {
					return upload.UploadResult{}, fmt.Errorf("file upload is not completed in task %s (current status: %d)", id, uploadResult.Status)
				}
			}

			return uploadResult, nil
		}
	}, form.formLabelOpts(labelOpts)...)
}

// setFormHintsFromUploadResult sets the appropriate hint and hint type on a form field
// based on the upload result status and any errors encountered during the upload process.
func setFormHintsFromUploadResult(result upload.UploadResult, field inspectionmetadata.FileParameterFormField) inspectionmetadata.FileParameterFormField {
	switch {
	case result.UploadError != nil:
		field.Hint = result.UploadError.Error()
		field.HintType = inspectionmetadata.Error
	case result.VerificationError != nil:
		field.Hint = result.VerificationError.Error()
		field.HintType = inspectionmetadata.Error
	case result.Status == upload.UploadStatusWaiting:
		field.Hint = "Waiting a file to be uploaded."
		field.HintType = inspectionmetadata.Error
	case result.Status != upload.UploadStatusCompleted:
		field.Hint = "File is being processed. Please wait a moment."
		field.HintType = inspectionmetadata.Info
		field.Pending = true
	}
	return field
}

// GenerateUploadIDWithTaskContext generates the upload ID from form ID and task ID.
func GenerateUploadIDWithTaskContext(ctx context.Context, formId string) string {
	inspectionID := khictx.MustGetValue(ctx, inspectioncore.InspectionTaskInspectionID)
	taskID := khictx.MustGetValue(ctx, core_contract.TaskImplementationIDContextKey)
	return strings.ReplaceAll(fmt.Sprintf("%s_%s_%s", inspectionID, taskID.ReferenceIDString(), formId), "/", "_")
}
