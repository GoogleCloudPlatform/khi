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
	"errors"
	"fmt"
	"strconv"
)

type HDRColor4 = [4]float32
type HDRColor3 = [3]float32

// hexToHDRColor4 converts sRGB hex to HDRColor4.
// Deprecated:
// Please specify HDRColor4 directly in future.
func hexToHDRColor4(hex string) (HDRColor4, error) {
	if len(hex) != 7 && len(hex) != 9 {
		return HDRColor4{}, errors.New("hex color must be 7 or 9 characters long")
	}
	var aStr string
	rStr := hex[1:3]
	gStr := hex[3:5]
	bStr := hex[5:7]
	if len(hex) == 9 {
		aStr = hex[7:9]
	}
	r, err := strconv.ParseInt(rStr, 16, 32)
	if err != nil {
		return HDRColor4{}, err
	}
	g, err := strconv.ParseInt(gStr, 16, 32)
	if err != nil {
		return HDRColor4{}, err
	}
	b, err := strconv.ParseInt(bStr, 16, 32)
	if err != nil {
		return HDRColor4{}, err
	}
	if len(hex) == 9 {
		a, err := strconv.ParseInt(aStr, 16, 32)
		if err != nil {
			return HDRColor4{}, err
		}
		return HDRColor4{float32(r) / 255, float32(g) / 255, float32(b) / 255, float32(a) / 255}, nil
	}
	return HDRColor4{float32(r) / 255, float32(g) / 255, float32(b) / 255, 1}, nil
}

func mustHexToHDRColor4(hex string) HDRColor4 {
	color, err := hexToHDRColor4(hex)
	if err != nil {
		panic(err)
	}
	return color
}

// ColorToHexRGB converts HDRColor4 to sRGB hex.
func ColorToHexRGB(color HDRColor4) string {
	return fmt.Sprintf("#%02x%02x%02x", int64(color[0]*255), int64(color[1]*255), int64(color[2]*255))
}

// ColorToHexRGBA converts HDRColor4 to sRGB hex with alpha.
func ColorToHexRGBA(color HDRColor4) string {
	return fmt.Sprintf("#%02x%02x%02x%02x", int64(color[0]*255), int64(color[1]*255), int64(color[2]*255), int64(color[3]*255))
}
