// Copyright 2026 Google LLC
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

package structured

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseFieldPath(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple path",
			input:    "a.b.c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty path",
			input:    "",
			expected: []string{""},
		},
		{
			name:     "path with single segment",
			input:    "single",
			expected: []string{"single"},
		},
		{
			name:     "escaped dots",
			input:    "a\\.b.c",
			expected: []string{"a.b", "c"},
		},
		{
			name:     "backslash following non dot char",
			input:    "a\\_b.c",
			expected: []string{"a\\_b", "c"},
		},
		{
			name:     "multiple escaped dots",
			input:    "a\\.b\\.c.d",
			expected: []string{"a.b.c", "d"},
		},
		{
			name:     "trailing escape character",
			input:    "a.b\\",
			expected: []string{"a", "b\\"},
		},
		{
			name:     "trailing escaped dot",
			input:    "a.b\\.",
			expected: []string{"a", "b."},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ParseFieldPath(tc.input)
			if diff := cmp.Diff(tc.expected, result); diff != "" {
				t.Errorf("ParseFieldPath() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCompileFieldPath(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		wantSegments []string
		wantIsEmpty  bool
	}{
		{
			name:         "empty path",
			input:        "",
			wantSegments: nil,
			wantIsEmpty:  true,
		},
		{
			name:         "single segment",
			input:        "message",
			wantSegments: []string{"message"},
			wantIsEmpty:  false,
		},
		{
			name:         "nested segments",
			input:        "resource.labels.node_name",
			wantSegments: []string{"resource", "labels", "node_name"},
			wantIsEmpty:  false,
		},
		{
			name:         "escaped dot segment",
			input:        "labels.k8s\\.io/app",
			wantSegments: []string{"labels", "k8s.io/app"},
			wantIsEmpty:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := CompileFieldPath(tc.input)
			if diff := cmp.Diff(tc.wantSegments, p.Segments()); diff != "" {
				t.Errorf("Segments() mismatch (-want +got):\n%s", diff)
			}
			if p.IsEmpty() != tc.wantIsEmpty {
				t.Errorf("IsEmpty() got %v, want %v", p.IsEmpty(), tc.wantIsEmpty)
			}
		})
	}
}

func TestEmptyFieldPath(t *testing.T) {
	if !EmptyFieldPath.IsEmpty() {
		t.Errorf("EmptyFieldPath.IsEmpty() got false, want true")
	}
	if len(EmptyFieldPath.Segments()) != 0 {
		t.Errorf("EmptyFieldPath.Segments() got %v, want empty slice", EmptyFieldPath.Segments())
	}
}
