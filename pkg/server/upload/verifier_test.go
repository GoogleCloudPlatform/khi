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

package upload

import (
	"strings"
	"testing"
)

func TestJSONLineUploadFileVerifier(t *testing.T) {
	tests := []struct {
		name        string
		data        string
		expectedErr string
	}{
		{
			name: "Valid JSON Lines",
			data: `{"name": "Alice", "age": 30}
{"name": "Bob", "age": 25}
{"name": "Charlie", "age": 40}`,
			expectedErr: "",
		},
		{
			name:        "Empty File",
			data:        "",
			expectedErr: "",
		},
		{
			name:        "Single Valid Line",
			data:        `{"name": "David", "age": 50}`,
			expectedErr: "",
		},
		{
			name: "Invalid JSON",
			data: `{"name": "Eve", "age": 35}
{invalid json}
{"name": "Frank", "age": 45}`,
			expectedErr: "invalid JSON on line 2: invalid character 'i' looking for beginning of object key string",
		},
		{
			name: "Empty Lines and Whitespace",
			data: `{"name": "Grace", "age": 55}

{"name": "Hank", "age": 60}
   `, expectedErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := &JSONLineUploadFileVerifier{MaxLineSizeInBytes: 1024 * 1024}
			provider := &MockLocalUploadFileStoreProvider{Data: tt.data}
			err := verifier.Verify(provider, &DirectUploadToken{ID: "test"})

			if tt.expectedErr == "" {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error, but got nil")
				} else if !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("Expected error to contain: %q, but got: %v", tt.expectedErr, err)
				}
			}
		})
	}
}
