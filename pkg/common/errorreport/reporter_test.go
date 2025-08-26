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

package errorreport

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCloudErrorReportWriterKeepWorkingWhenReporterFailsToWrite(t *testing.T) {
	reporter, err := NewCloudErrorReportWriter("kubernetes-history-inspector", "An invalid api key")

	if err != nil {
		t.Fatalf("Failed to initialize CloudErrorReporter: %v", err)
	}
	reporter.WriteReportSync(context.Background(), errors.New("a test value"))
}

func TestReporterMetadata(t *testing.T) {
	testCases := []struct {
		name           string
		inputMetadata  map[string]string
		expectedOutput map[string]string
	}{
		{
			name:           "no metadata",
			inputMetadata:  map[string]string{},
			expectedOutput: map[string]string{},
		},
		{
			name: "with metadata",
			inputMetadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expectedOutput: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reporter := NewReporter(nil)
			for key, value := range tc.inputMetadata {
				reporter.SetMetadataEntry(key, value)
			}
			output := reporter.GetMetadata()
			if diff := cmp.Diff(tc.expectedOutput, output); diff != "" {
				t.Errorf("Unexpected metadata(-want,+got) = %s", diff)
			}
		})
	}
}

func TestGetErrorMessageWithMetadata(t *testing.T) {
	testCases := []struct {
		name           string
		inputErr       error
		inputMetadata  map[string]string
		expectedOutput string
	}{
		{
			name:           "no metadata",
			inputErr:       errors.New("test error"),
			inputMetadata:  map[string]string{},
			expectedOutput: "test error",
		},
		{
			name:     "with metadata",
			inputErr: errors.New("test error"),
			inputMetadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expectedOutput: "test error\n  Metadata:\n    * key1: value1\n    * key2: value2\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reporter := &Reporter{
				metadata: tc.inputMetadata,
			}
			err := reporter.getErrorMessageWithMetadata(tc.inputErr)
			if err.Error() != tc.expectedOutput {
				t.Errorf("Unexpected error message:\nExpected: %s\nActual: %s", tc.expectedOutput, err.Error())
			}
		})
	}
}
