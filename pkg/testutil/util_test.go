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

package testutil

import (
	"io"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestRemoveSlogTimestampFromLine(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "time=2024-02-08T10:48:10.459+09:00 level=INFO msg=\"inspection1 task1 info\"",
			expected: "level=INFO msg=\"inspection1 task1 info\"",
		},
		{
			input:    "time=2024-02-08T10:48:10.459Z level=INFO msg=\"inspection1 task1 info\"",
			expected: "level=INFO msg=\"inspection1 task1 info\"",
		},
		{
			input:    "", // Empty input
			expected: "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.input, func(t *testing.T) {
			result := RemoveSlogTimestampFromLine(testCase.input)
			if result != testCase.expected {
				t.Errorf("RemoveSlogTimestampFromLine failed. Input: %s, Expected: %s, Got: %s", testCase.input, testCase.expected, result)
			}
		})
	}
}

func TestResponseFromString(t *testing.T) {
	t.Parallel()
	type args struct {
		code     int
		response string
	}
	tests := []struct {
		name     string
		args     args
		wantBody string
		wantCode int
	}{
		{
			name: "ResponseFromString should return a http.Response with the given code and response",
			args: args{
				code:     http.StatusOK,
				response: "test-response",
			},
			wantBody: "test-response",
			wantCode: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResponseFromString(tt.args.code, tt.args.response)
			if got.StatusCode != tt.wantCode {
				t.Errorf("ResponseFromString() = %v, want %v", got.StatusCode, tt.wantCode)
			}
			gotBody, _ := io.ReadAll(got.Body)
			if diff := cmp.Diff(string(gotBody), tt.wantBody); diff != "" {
				t.Errorf("ResponseFromString() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
