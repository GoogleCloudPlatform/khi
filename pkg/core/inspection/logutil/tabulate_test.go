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

package logutil

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTabulateReader(t *testing.T) {
	testCases := []struct {
		name           string
		lines          []string
		expected       []*TabulateParseResult
		expectedErrors []bool
	}{
		{
			name: "Normal table parsing with spaces in values",
			lines: []string{
				"Some stray log before table",                              // 0
				"File Path                               PID    Runtime",   // 1
				"--------------------------------------  -----  ---------", // 2
				"test/path/to/file.py                    12345  0.10s",     // 3
				"path with spaces.py                       999  10.0s",     // 4
			},
			expected: []*TabulateParseResult{
				{Type: TabulateLineTypeHeaderCandidate, Columns: []string{"Some stray log before table"}}, // 0
				{Type: TabulateLineTypeHeaderCandidate, Columns: []string{"File Path", "PID", "Runtime"}}, // 1
				{Type: TabulateLineTypeSeparator, Columns: []string{"File Path", "PID", "Runtime"}},       // 2
				{Type: TabulateLineTypeBody, Columns: []string{"File Path", "PID", "Runtime"}, Values: map[string]string{
					"File Path": "test/path/to/file.py",
					"PID":       "12345",
					"Runtime":   "0.10s",
				}}, // 3
				{Type: TabulateLineTypeBody, Columns: []string{"File Path", "PID", "Runtime"}, Values: map[string]string{
					"File Path": "path with spaces.py",
					"PID":       "999",
					"Runtime":   "10.0s",
				}}, // 4
			},
			expectedErrors: []bool{false, false, false, false, false},
		},
		{
			name: "Format error crosses boundaries and resets state",
			lines: []string{
				"Col1   Col2   Col3",
				"----   ----   ----",
				"val1   val2   val3",
				"this long string crosses boundary",
				"another stray string",
			},
			expected: []*TabulateParseResult{
				{Type: TabulateLineTypeHeaderCandidate, Columns: []string{"Col1", "Col2", "Col3"}},
				{Type: TabulateLineTypeSeparator, Columns: []string{"Col1", "Col2", "Col3"}},
				{Type: TabulateLineTypeBody, Columns: []string{"Col1", "Col2", "Col3"}, Values: map[string]string{
					"Col1": "val1", "Col2": "val2", "Col3": "val3",
				}},
				nil, // Error expected, returns nil result
				{Type: TabulateLineTypeHeaderCandidate, Columns: []string{"another stray string"}}, // Since state was reset, this is parsed as header candidate again
			},
			expectedErrors: []bool{false, false, false, true, false},
		},
		{
			name: "Separator without header defaults to generic names",
			lines: []string{
				"----   ----",
				"v1     v2",
			},
			expected: []*TabulateParseResult{
				{Type: TabulateLineTypeSeparator, Columns: nil},
				{Type: TabulateLineTypeBody, Columns: []string{"Column_0", "Column_1"}, Values: map[string]string{
					"Column_0": "v1", "Column_1": "v2",
				}},
			},
			expectedErrors: []bool{false, false},
		},
		{
			name: "Missing columns in data row",
			lines: []string{
				"A      B      C",
				"----   ----   ----",
				"v1", // Missing cols B and C
			},
			expected: []*TabulateParseResult{
				{Type: TabulateLineTypeHeaderCandidate, Columns: []string{"A", "B", "C"}},
				{Type: TabulateLineTypeSeparator, Columns: []string{"A", "B", "C"}},
				{Type: TabulateLineTypeBody, Columns: []string{"A", "B", "C"}, Values: map[string]string{
					"A": "v1", "B": "", "C": "",
				}},
			},
			expectedErrors: []bool{false, false, false},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := NewTabulateReader()
			for i, line := range tc.lines {
				res, err := reader.ParseLine(line)

				if tc.expectedErrors[i] {
					if err == nil {
						t.Errorf("line %d: expected error but got nil", i)
					}
					continue
				}

				if err != nil {
					t.Errorf("line %d: unexpected error: %v", i, err)
				}

				expectedRes := tc.expected[i]
				if expectedRes == nil {
					continue
				}

				if res == nil {
					t.Fatalf("line %d: got nil result but expected %+v", i, expectedRes)
				}

				if res.Type != expectedRes.Type {
					t.Errorf("line %d: expected type %v, got %v", i, expectedRes.Type, res.Type)
				}

				if diff := cmp.Diff(expectedRes.Columns, res.Columns); diff != "" {
					t.Errorf("line %d: expected columns diff (-want +got):\n%s", i, diff)
				}

				if diff := cmp.Diff(expectedRes.Values, res.Values); diff != "" {
					t.Errorf("line %d: expected values diff (-want +got):\n%s", i, diff)
				}
			}
		})
	}
}
