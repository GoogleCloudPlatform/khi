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

package resourcepath

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

func TestResourcePath_GetParentPathString(t *testing.T) {
	testCases := []struct {
		name     string
		path     ResourcePath
		expected string
	}{
		{
			name: "path with multiple parts",
			path: ResourcePath{
				Path:               "A#B#C",
				ParentRelationship: enum.RelationshipChild,
			},
			expected: "A#B",
		},
		{
			name: "path with single part",
			path: ResourcePath{
				Path:               "A",
				ParentRelationship: enum.RelationshipChild,
			},
			expected: "",
		},
		{
			name: "empty path",
			path: ResourcePath{
				Path:               "",
				ParentRelationship: enum.RelationshipChild,
			},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.path.GetParentPathString()
			if actual != tc.expected {
				t.Errorf("unexpected parent path: got %q, want %q", actual, tc.expected)
			}
		})
	}
}
