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
	"errors"
	"testing"

	form_metadata "github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/form"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
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
	baseField := form_metadata.FileParameterFormField{
		ParameterFormFieldBase: form_metadata.ParameterFormFieldBase{
			ID:       "test-field",
			Type:     form_metadata.File,
			Label:    "Test File Field",
			Priority: 0,
			HintType: form_metadata.None,
			Hint:     "",
		},
		Token:  mockToken,
		Status: upload.UploadStatusWaiting,
	}

	testCases := []struct {
		name          string
		uploadResult  upload.UploadResult
		expectedField form_metadata.FileParameterFormField
	}{
		{
			name: "upload error case",
			uploadResult: upload.UploadResult{
				Status:            upload.UploadStatusWaiting,
				UploadError:       errors.New("upload failed: file too large"),
				VerificationError: nil,
			},
			expectedField: form_metadata.FileParameterFormField{
				ParameterFormFieldBase: form_metadata.ParameterFormFieldBase{
					ID:       "test-field",
					Type:     form_metadata.File,
					Label:    "Test File Field",
					Priority: 0,
					HintType: form_metadata.Error,
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
			expectedField: form_metadata.FileParameterFormField{
				ParameterFormFieldBase: form_metadata.ParameterFormFieldBase{
					ID:       "test-field",
					Type:     form_metadata.File,
					Label:    "Test File Field",
					Priority: 0,
					HintType: form_metadata.Error,
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
			expectedField: form_metadata.FileParameterFormField{
				ParameterFormFieldBase: form_metadata.ParameterFormFieldBase{
					ID:       "test-field",
					Type:     form_metadata.File,
					Label:    "Test File Field",
					Priority: 0,
					HintType: form_metadata.Error,
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
			expectedField: form_metadata.FileParameterFormField{
				ParameterFormFieldBase: form_metadata.ParameterFormFieldBase{
					ID:       "test-field",
					Type:     form_metadata.File,
					Label:    "Test File Field",
					Priority: 0,
					HintType: form_metadata.Error,
					Hint:     "File is being processed. Please wait a moment.",
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
			expectedField: form_metadata.FileParameterFormField{
				ParameterFormFieldBase: form_metadata.ParameterFormFieldBase{
					ID:       "test-field",
					Type:     form_metadata.File,
					Label:    "Test File Field",
					Priority: 0,
					HintType: form_metadata.None,
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

			if result.Hint != tc.expectedField.Hint || result.HintType != tc.expectedField.HintType {
				t.Errorf("setFormHintsFromUploadResult() unexpected result:\nwant: (hint=%s, hintType=%v)\ngot: (hint=%s, hintType=%v)",
					tc.expectedField.Hint, tc.expectedField.HintType, result.Hint, result.HintType)
			}
		})
	}
}
