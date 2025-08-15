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

package progressutil

import (
	"errors"
	"testing"

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
)

// updateRecord records a single progress update
type updateRecord struct {
	percentage float32
	message    string
}

// TestReportProgressFromArraySync tests the ReportProgressFromArraySync function
func TestReportProgressFromArraySync(t *testing.T) {
	tests := []struct {
		name          string
		source        []int
		process       func(int, int) error
		expectedError bool
		expectedCalls int
		checkUpdates  func([]updateRecord) bool
	}{
		{
			name:   "Normal case - all items processed successfully",
			source: []int{1, 2, 3, 4, 5},
			process: func(i int, val int) error {
				return nil
			},
			expectedError: false,
			expectedCalls: 6, // Initial 0% + 5 updates
			checkUpdates: func(updates []updateRecord) bool {
				if updates[0].percentage != 0 || updates[0].message != "0/5" {
					return false
				}

				lastIdx := len(updates) - 1
				if updates[lastIdx].percentage != 1 || updates[lastIdx].message != "5/5" {
					return false
				}

				return true
			},
		},
		{
			name:   "Error case - processing fails at specific index",
			source: []int{1, 2, 3, 4, 5},
			process: func(i int, val int) error {
				if i == 1 {
					return errors.New("error at index 1")
				}
				return nil
			},
			expectedError: true,
			expectedCalls: 3, // Initial 0% + 2 successful updates
			checkUpdates: func(updates []updateRecord) bool {
				// Only processed up to index 2 (error occurs here)
				return len(updates) == 3
			},
		},
		{
			name:   "Edge case - empty array",
			source: []int{},
			process: func(i int, val int) error {
				return nil
			},
			expectedError: false,
			expectedCalls: 1, // Just initial 0%
			checkUpdates: func(updates []updateRecord) bool {
				return len(updates) == 1 && updates[0].message == "0/0"
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a real TaskProgress
			progress := inspectionmetadata.NewTaskProgressMetadata("test")

			// Set up a recorder to capture updates
			var updates []updateRecord

			err := ReportProgressFromArraySync(progress, tc.source, func(i int, val int) error {
				updates = append(updates, updateRecord{
					percentage: progress.Percentage,
					message:    progress.Message,
				})
				return tc.process(i, val)
			})
			if (err != nil) != tc.expectedError {
				t.Errorf("Expected error: %v, got: %v", tc.expectedError, err != nil)
			}

			updates = append(updates, updateRecord{
				percentage: progress.Percentage,
				message:    progress.Message,
			})

			// Check number of progress updates
			if len(updates) != tc.expectedCalls {
				t.Errorf("Expected %d progress updates, got: %d", tc.expectedCalls, len(updates))
				for i, u := range updates {
					t.Logf("Update %d: percentage=%f, message=%s", i, u.percentage, u.message)
				}
			}

			// Check update content
			if !tc.checkUpdates(updates) {
				t.Errorf("Progress updates did not match expected pattern")
				for i, u := range updates {
					t.Logf("Update %d: percentage=%f, message=%s", i, u.percentage, u.message)
				}
			}
		})
	}
}
