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

package enum

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestHexToHDRColor4(t *testing.T) {
	tests := []struct {
		name    string
		hex     string
		want    HDRColor4
		wantErr bool
	}{
		{
			name: "Valid 7-char hex",
			hex:  "#ffffff",
			want: HDRColor4{1, 1, 1, 1},
		},
		{
			name: "Valid 7-char hex black",
			hex:  "#000000",
			want: HDRColor4{0, 0, 0, 1},
		},
		{
			name: "Valid 9-char hex with alpha",
			hex:  "#ffffff80",
			want: HDRColor4{1, 1, 1, float32(0x80) / 255.0},
		},
		{
			name:    "Invalid length",
			hex:     "#fff",
			wantErr: true,
		},
		{
			name:    "Invalid hex content",
			hex:     "#zzzzzz",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hexToHDRColor4(tt.hex)
			if (err != nil) != tt.wantErr {
				t.Errorf("hexToHDRColor4() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("hexToHDRColor4() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestColorToHexRGB(t *testing.T) {
	tests := []struct {
		name  string
		color HDRColor4
		want  string
	}{
		{
			name:  "White",
			color: HDRColor4{1, 1, 1, 1},
			want:  "#ffffff",
		},
		{
			name:  "Black",
			color: HDRColor4{0, 0, 0, 1},
			want:  "#000000",
		},
		{
			name:  "Red",
			color: HDRColor4{1, 0, 0, 1},
			want:  "#ff0000",
		},
		{
			name:  "Green",
			color: HDRColor4{0, 1, 0, 1},
			want:  "#00ff00",
		},
		{
			name:  "Blue",
			color: HDRColor4{0, 0, 1, 1},
			want:  "#0000ff",
		},
		{
			name:  "Single digit components",
			color: HDRColor4{float32(10) / 255.0, float32(10) / 255.0, float32(10) / 255.0, 1},
			want:  "#0a0a0a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ColorToHexRGB(tt.color); got != tt.want {
				t.Errorf("ColorToHexRGB() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestColorToHexRGBA(t *testing.T) {
	tests := []struct {
		name  string
		color HDRColor4
		want  string
	}{
		{
			name:  "White Opaque",
			color: HDRColor4{1, 1, 1, 1},
			want:  "#ffffffff",
		},
		{
			name:  "Black Transparent",
			color: HDRColor4{0, 0, 0, 0},
			want:  "#00000000",
		},
		{
			name:  "Red Semi-Transparent",
			color: HDRColor4{1, 0, 0, 0.5},
			// 0.5 * 255 = 127.5 -> 127 -> 7f
			want: "#ff00007f",
		},
		{
			name:  "Single digit components",
			color: HDRColor4{float32(10) / 255.0, float32(10) / 255.0, float32(10) / 255.0, float32(10) / 255.0},
			want:  "#0a0a0a0a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ColorToHexRGBA(tt.color); got != tt.want {
				t.Errorf("ColorToHexRGBA() = %v, want %v", got, tt.want)
			}
		})
	}
}
