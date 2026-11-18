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

package scene

import "testing"

func TestInvalidSliceWindowError_Error(t *testing.T) {
	testCases := []struct {
		name string
		err  *InvalidSliceWindowError
		want string
	}{
		{
			name: "negative window index formats plural windows message",
			err: &InvalidSliceWindowError{
				WindowIndex: -1,
				Reason:      "windows must contain at least one path or query selector",
			},
			want: "invalid slice windows: windows must contain at least one path or query selector",
		},
		{
			name: "zero window index formats indexed window message",
			err: &InvalidSliceWindowError{
				WindowIndex: 0,
				Reason:      "specify path, query, or both",
			},
			want: "invalid slice window 0: specify path, query, or both",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSliceQueryNotFoundError_Error(t *testing.T) {
	testCases := []struct {
		name string
		err  *SliceQueryNotFoundError
		want string
	}{
		{
			name: "path and query",
			err: &SliceQueryNotFoundError{
				WindowIndex: 1,
				Path:        "$.textPayload",
				Query:       "OOM",
			},
			want: `slice window 1 query "OOM" within path "$.textPayload" matched no lines`,
		},
		{
			name: "path only",
			err: &SliceQueryNotFoundError{
				WindowIndex: 0,
				Path:        "$.status.ready",
			},
			want: `slice path 0 "$.status.ready" matched no YAML nodes`,
		},
		{
			name: "query only",
			err: &SliceQueryNotFoundError{
				WindowIndex: 2,
				Query:       "panic",
			},
			want: `slice window 2 query "panic" matched no lines`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}
