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
	"errors"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// mockUploadToken is a simple implementation of the UploadToken interface for testing
type mockUploadToken struct {
	id string
}

func (m mockUploadToken) GetID() string {
	return m.id
}

func (m mockUploadToken) GetHash() string {
	return "mock-hash"
}

func (m mockUploadToken) GetType() string {
	return "mock-type"
}

func TestSetFormHintsFromUploadResult(t *testing.T) {
	// Create mock token for testing
	mockToken := mockUploadToken{id: "test-token"}

	// Base field used in all test cases
	baseField := inspectionmetadata.FileParameterFormField{
		ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
			ID:       "test-field",
			Type:     inspectionmetadata.File,
			Label:    "Test File Field",
			Priority: 0,
			HintType: inspectionmetadata.None,
			Hint:     "",
		},
		Token:  mockToken,
		Status: upload.UploadStatusWaiting,
	}

	testCases := []struct {
		name          string
		uploadResult  upload.UploadResult
		expectedField inspectionmetadata.FileParameterFormField
	}{
		{
			name: "upload error case",
			uploadResult: upload.UploadResult{
				Status:            upload.UploadStatusWaiting,
				UploadError:       errors.New("upload failed: file too large"),
				VerificationError: nil,
			},
			expectedField: inspectionmetadata.FileParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:       "test-field",
					Type:     inspectionmetadata.File,
					Label:    "Test File Field",
					Priority: 0,
					HintType: inspectionmetadata.Error,
					Hint:     "upload failed: file too large",
				},
				Token:  mockToken,
				Status: upload.UploadStatusWaiting,
			},
		},
		{
			name: "verification error case",
			uploadResult: upload.UploadResult{
				Status:            upload.UploadStatusWaiting,
				UploadError:       nil,
				VerificationError: errors.New("invalid file format"),
			},
			expectedField: inspectionmetadata.FileParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:       "test-field",
					Type:     inspectionmetadata.File,
					Label:    "Test File Field",
					Priority: 0,
					HintType: inspectionmetadata.Error,
					Hint:     "invalid file format",
				},
				Token:  mockToken,
				Status: upload.UploadStatusWaiting,
			},
		},
		{
			name: "waiting status case",
			uploadResult: upload.UploadResult{
				Status:            upload.UploadStatusWaiting,
				UploadError:       nil,
				VerificationError: nil,
			},
			expectedField: inspectionmetadata.FileParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:       "test-field",
					Type:     inspectionmetadata.File,
					Label:    "Test File Field",
					Priority: 0,
					HintType: inspectionmetadata.Error,
					Hint:     "Waiting a file to be uploaded.",
				},
				Token:  mockToken,
				Status: upload.UploadStatusWaiting,
			},
		},
		{
			name: "processing status case",
			uploadResult: upload.UploadResult{
				Status:            upload.UploadStatusUploading,
				UploadError:       nil,
				VerificationError: nil,
			},
			expectedField: inspectionmetadata.FileParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:       "test-field",
					Type:     inspectionmetadata.File,
					Label:    "Test File Field",
					Priority: 0,
					HintType: inspectionmetadata.Info,
					Hint:     "File is being processed. Please wait a moment.",
					Pending:  true,
				},
				Token:  mockToken,
				Status: upload.UploadStatusWaiting,
			},
		},
		{
			name: "completed status case",
			uploadResult: upload.UploadResult{
				Status:            upload.UploadStatusCompleted,
				UploadError:       nil,
				VerificationError: nil,
			},
			expectedField: inspectionmetadata.FileParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:       "test-field",
					Type:     inspectionmetadata.File,
					Label:    "Test File Field",
					Priority: 0,
					HintType: inspectionmetadata.None,
					Hint:     "",
				},
				Token:  mockToken,
				Status: upload.UploadStatusWaiting,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := setFormHintsFromUploadResult(tc.uploadResult, baseField)

			if result.Hint != tc.expectedField.Hint || result.HintType != tc.expectedField.HintType || result.Pending != tc.expectedField.Pending {
				t.Errorf("setFormHintsFromUploadResult() unexpected result:\nwant: (hint=%s, hintType=%v, pending=%v)\ngot: (hint=%s, hintType=%v, pending=%v)",
					tc.expectedField.Hint, tc.expectedField.HintType, tc.expectedField.Pending, result.Hint, result.HintType, result.Pending)
			}
		})
	}
}

type mockFileFormStore struct {
	token                    upload.UploadToken
	result                   upload.UploadResult
	completedResult          upload.UploadResult
	getResultCalled          bool
	getCompletedResultCalled bool
}

func (m *mockFileFormStore) GetUploadToken(id string, verifier upload.UploadFileVerifier, fieldID string) upload.UploadToken {
	return m.token
}

func (m *mockFileFormStore) GetResult(token upload.UploadToken, req map[string]any) (upload.UploadResult, error) {
	m.getResultCalled = true
	return m.result, nil
}

func (m *mockFileFormStore) GetCompletedResult(ctx context.Context, token upload.UploadToken, req map[string]any) (upload.UploadResult, error) {
	m.getCompletedResultCalled = true
	return m.completedResult, nil
}

var _ upload.Store = (*mockFileFormStore)(nil)

func TestFileFormTaskBuilder_Build(t *testing.T) {
	mockToken := mockUploadToken{id: "test-token"}
	verifyErr := errors.New("invalid file format")
	uploadErr := errors.New("network disconnect during upload")

	testCases := []struct {
		name                    string
		mode                    inspectioncore.InspectionTaskModeType
		storeResult             upload.UploadResult
		wantErr                 bool
		wantErrSubstring        string
		wantPending             bool
		wantResultCall          bool
		wantCompletedResultCall bool
	}{
		{
			name: "TaskModeDryRun with status UploadStatusUploading succeeds and sets Pending true",
			mode: inspectioncore.TaskModeDryRun,
			storeResult: upload.UploadResult{
				Status: upload.UploadStatusUploading,
				Token:  mockToken,
			},
			wantErr:                 false,
			wantPending:             true,
			wantResultCall:          true,
			wantCompletedResultCall: false,
		},
		{
			name: "TaskModeRun with status UploadStatusWaiting returns error",
			mode: inspectioncore.TaskModeRun,
			storeResult: upload.UploadResult{
				Status: upload.UploadStatusWaiting,
				Token:  mockToken,
			},
			wantErr:                 true,
			wantErrSubstring:        "file upload is not completed",
			wantResultCall:          false,
			wantCompletedResultCall: true,
		},
		{
			name: "TaskModeRun with status UploadStatusUploading returns error",
			mode: inspectioncore.TaskModeRun,
			storeResult: upload.UploadResult{
				Status: upload.UploadStatusUploading,
				Token:  mockToken,
			},
			wantErr:                 true,
			wantErrSubstring:        "file upload is not completed",
			wantResultCall:          false,
			wantCompletedResultCall: true,
		},
		{
			name: "TaskModeRun when upload fails returns error containing upload error",
			mode: inspectioncore.TaskModeRun,
			storeResult: upload.UploadResult{
				Status:      upload.UploadStatusWaiting,
				UploadError: uploadErr,
				Token:       mockToken,
			},
			wantErr:                 true,
			wantErrSubstring:        "file upload failed in task",
			wantResultCall:          false,
			wantCompletedResultCall: true,
		},
		{
			name: "TaskModeRun after upload completes returns UploadStatusCompleted without error",
			mode: inspectioncore.TaskModeRun,
			storeResult: upload.UploadResult{
				Status: upload.UploadStatusCompleted,
				Token:  mockToken,
			},
			wantErr:                 false,
			wantResultCall:          false,
			wantCompletedResultCall: true,
		},
		{
			name: "TaskModeRun when verification fails returns error containing verification error",
			mode: inspectioncore.TaskModeRun,
			storeResult: upload.UploadResult{
				Status:            upload.UploadStatusCompleted,
				VerificationError: verifyErr,
				Token:             mockToken,
			},
			wantErr:                 true,
			wantErrSubstring:        "invalid file format",
			wantResultCall:          false,
			wantCompletedResultCall: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prevStore := upload.DefaultUploadFileStore
			defer func() { upload.DefaultUploadFileStore = prevStore }()

			store := &mockFileFormStore{
				token:           mockToken,
				result:          tc.storeResult,
				completedResult: tc.storeResult,
			}
			upload.DefaultUploadFileStore = store

			taskId := taskid.NewDefaultImplementationID[upload.UploadResult]("test-field")
			builder := NewFileFormTaskBuilder(taskId, 10, "Test File", nil)
			task := builder.Build()

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			result, metadata, err := inspectiontest.RunInspectionTaskWithDependency(
				ctx,
				task,
				nil,
				tc.mode,
				map[string]any{},
			)

			if tc.wantResultCall != store.getResultCalled {
				t.Errorf("getResultCalled = %v, want %v", store.getResultCalled, tc.wantResultCall)
			}
			if tc.wantCompletedResultCall != store.getCompletedResultCalled {
				t.Errorf("getCompletedResultCalled = %v, want %v", store.getCompletedResultCalled, tc.wantCompletedResultCall)
			}

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.wantErrSubstring != "" && !strings.Contains(err.Error(), tc.wantErrSubstring) {
					t.Errorf("error %q does not contain expected substring %q", err.Error(), tc.wantErrSubstring)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.mode == inspectioncore.TaskModeRun && result.Status != upload.UploadStatusCompleted {
				t.Errorf("expected result.Status UploadStatusCompleted, got %v", result.Status)
			}

			formFields, found := typedmap.Get(metadata, inspectionmetadata.FormFieldSetMetadataKey)
			if !found {
				t.Fatalf("FormFieldSetMetadataKey not found in metadata")
			}
			field := formFields.DangerouslyGetField(taskId.ReferenceIDString())
			fileField, ok := field.(inspectionmetadata.FileParameterFormField)
			if !ok {
				t.Fatalf("field is not FileParameterFormField")
			}

			if fileField.Pending != tc.wantPending {
				t.Errorf("field.Pending = %v, want %v", fileField.Pending, tc.wantPending)
			}
		})
	}
}
