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

package merger

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestKeyDiff(t *testing.T) {
	testCase := []struct {
		name      string
		prevKeys  []string
		patchKeys []string
		expected  *keyDiff
	}{
		{
			name:      "standard merge",
			prevKeys:  []string{"foo", "bar"},
			patchKeys: []string{"foo", "qux"},
			expected: &keyDiff{
				OnlyInPrev:  []string{"bar"},
				OnlyInPatch: []string{"qux"},
				Both:        []string{"foo"},
			},
		},
		{
			name:      "must retain the order",
			prevKeys:  []string{"foo-2", "foo-1", "bar-2", "bar-1"},
			patchKeys: []string{"foo-1", "foo-2", "qux-1", "qux-2"},
			expected: &keyDiff{
				OnlyInPrev:  []string{"bar-2", "bar-1"},
				OnlyInPatch: []string{"qux-1", "qux-2"},
				Both:        []string{"foo-2", "foo-1"}, // When a key found in the both array, the order will be same as the sub array of prevKeys
			},
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			diff := NewKeyDiff(tc.prevKeys, tc.patchKeys)
			if diff := cmp.Diff(diff, tc.expected); diff != "" {
				t.Errorf("result is not matching with the expected\n%s", diff)
			}
		})
	}
}
