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

import "fmt"

// MustHDRColor4FromHex parses a hex string (e.g., "#RRGGBBAA") and returns a new HDRColor4.
// It panics if the parsing fails.
func MustHDRColor4FromHex(hex string) *HDRColor4 {
	color, err := HDRColor4FromHex(hex)
	if err != nil {
		panic(err)
	}
	return color
}

// HDRColor4FromHex parses a hex string (e.g., "#RRGGBBAA") and returns a new HDRColor4.
// The hex string must be exactly 9 characters long including the '#'.
func HDRColor4FromHex(hex string) (*HDRColor4, error) {
	var r, g, b, a uint8
	_, err := fmt.Sscanf(hex, "#%02x%02x%02x%02x", &r, &g, &b, &a)
	if err != nil {
		return nil, err
	}
	return &HDRColor4{
		R: float32(r) / 255.0,
		G: float32(g) / 255.0,
		B: float32(b) / 255.0,
		A: float32(a) / 255.0,
	}, nil
}
