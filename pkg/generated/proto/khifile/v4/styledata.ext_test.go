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

package khifilev4

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestHDRColor4FromHex(t *testing.T) {
	testcases := []struct {
		name      string
		hex       string
		wantColor *HDRColor4
		wantErr   bool
	}{
		{
			name:      "valid #RRGGBBAA",
			hex:       "#ff8000ff",
			wantColor: &HDRColor4{R: float32(0xff) / 255.0, G: float32(0x80) / 255.0, B: float32(0x00) / 255.0, A: float32(0xff) / 255.0},
			wantErr:   false,
		},
		{
			name:    "invalid format (too short)",
			hex:     "#ff8000",
			wantErr: true,
		},
		{
			name:    "invalid characters",
			hex:     "#zzzzzzzz",
			wantErr: true,
		},
		{
			name:    "missing hash",
			hex:     "ff8000ff",
			wantErr: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := HDRColor4FromHex(tc.hex)
			if (err != nil) != tc.wantErr {
				t.Errorf("HDRColor4FromHex(%q) error = %v, wantErr %v", tc.hex, err, tc.wantErr)
			}
			if !tc.wantErr {
				if diff := cmp.Diff(tc.wantColor, got, cmpopts.IgnoreUnexported(HDRColor4{})); diff != "" {
					t.Errorf("HDRColor4FromHex(%q) mismatch (-want +got):\n%s", tc.hex, diff)
				}
			}
		})
	}
}

func TestMustHDRColor4FromHex(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("MustHDRColor4FromHex panicked unexpectedly: %v", r)
			}
		}()
		got := MustHDRColor4FromHex("#11223344")
		want := &HDRColor4{R: float32(0x11) / 255.0, G: float32(0x22) / 255.0, B: float32(0x33) / 255.0, A: float32(0x44) / 255.0}
		if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(HDRColor4{})); diff != "" {
			t.Errorf("MustHDRColor4FromHex mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("panic on error", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("MustHDRColor4FromHex did not panic on invalid input")
			}
		}()
		MustHDRColor4FromHex("invalid")
	})
}
