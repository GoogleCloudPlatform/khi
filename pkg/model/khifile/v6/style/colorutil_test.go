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

package style

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMustForceConvertSRGBHex(t *testing.T) {
	testCases := []struct {
		name      string
		hex       string
		want      Color
		wantPanic bool
	}{
		{
			name: "valid 7-char hex",
			hex:  "#CC33CC",
			want: Color{R: 204.0 / 255.0, G: 51.0 / 255.0, B: 204.0 / 255.0, A: 1.0},
		},
		{
			name: "valid 9-char hex with alpha",
			hex:  "#0000FF80",
			want: Color{R: 0.0, G: 0.0, B: 1.0, A: 128.0 / 255.0},
		},
		{
			name:      "invalid length 5-char",
			hex:       "#FFFF",
			wantPanic: true,
		},
		{
			name:      "invalid characters",
			hex:       "#GGGGGG",
			wantPanic: true,
		},
		{
			name:      "missing hash prefix",
			hex:       "CC33CC",
			wantPanic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("expected panic, but did not panic")
					}
				}()
			}
			got := MustForceConvertSRGBHex(tc.hex)
			if !tc.wantPanic {
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("MustForceConvertSRGBHex() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
