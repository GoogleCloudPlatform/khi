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
	"strconv"
)

// ColorWhite represents the white color (1.0, 1.0, 1.0, 1.0).
var ColorWhite = Color{R: 1, G: 1, B: 1, A: 1}

// ColorBlack represents the black color (0.0, 0.0, 0.0, 1.0).
var ColorBlack = Color{R: 0, G: 0, B: 0, A: 1}

// MustForceConvertSRGBHex converts an sRGB hex color string to a Color.
//
// Deprecated:
// The timeline style uses the display-p3 color space. Directly mapping sRGB
// hex values to the 0.0-1.0 range in display-p3 is mathematically incorrect.
// Ideally, you should define a Color struct directly with proper display-p3 values.
func MustForceConvertSRGBHex(hex string) Color {
	if len(hex) != 7 && len(hex) != 9 {
		panic("invalid hex color length: " + hex)
	}
	if hex[0] != '#' {
		panic("invalid hex color format (must start with '#'): " + hex)
	}
	r, err := strconv.ParseInt(hex[1:3], 16, 32)
	if err != nil {
		panic(err)
	}
	g, err := strconv.ParseInt(hex[3:5], 16, 32)
	if err != nil {
		panic(err)
	}
	b, err := strconv.ParseInt(hex[5:7], 16, 32)
	if err != nil {
		panic(err)
	}
	var a int64 = 255
	if len(hex) == 9 {
		a, err = strconv.ParseInt(hex[7:9], 16, 32)
		if err != nil {
			panic(err)
		}
	}
	return Color{
		R: float32(r) / 255.0,
		G: float32(g) / 255.0,
		B: float32(b) / 255.0,
		A: float32(a) / 255.0,
	}
}
