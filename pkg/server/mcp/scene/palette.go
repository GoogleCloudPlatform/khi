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

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	maxScenePaletteEntries = 16
	maxPaletteIDRunes      = 40
	lightOnMainHex         = "#fafafa"
	darkOnMainHex          = "#212121"
)

type rgbColor struct {
	r uint8
	g uint8
	b uint8
}

var presetPaletteColors = map[string]string{
	"blue":   "#1976d2",
	"amber":  "#b45309",
	"red":    "#d32f2f",
	"green":  "#2e7d32",
	"purple": "#5e35b1",
	"teal":   "#00796b",
	"indigo": "#3949ab",
	"pink":   "#c2185b",
	"brown":  "#5d4037",
	"slate":  "#546e7a",
}

var (
	lightOnMainLuminance = relativeLuminance(mustParseHexColor(lightOnMainHex))
	darkOnMainLuminance  = relativeLuminance(mustParseHexColor(darkOnMainHex))
)

type paletteRegistry struct {
	ordered       []ResolvedPaletteColors
	byID          map[string]ResolvedPaletteColors
	registeredIDs []string
}

func buildPaletteRegistry(entries []PaletteEntry) (*paletteRegistry, *mcpsdk.CallToolResult) {
	if len(entries) > maxScenePaletteEntries {
		res, _, _ := mdtemplate.ErrorResult(
			"INVALID_ARGUMENT",
			fmt.Sprintf("`annotations.palette` exceeds maximum of %d entries (got %d).", maxScenePaletteEntries, len(entries)),
		)
		return nil, res
	}

	reg := &paletteRegistry{
		ordered:       make([]ResolvedPaletteColors, 0, len(entries)),
		byID:          make(map[string]ResolvedPaletteColors, len(entries)),
		registeredIDs: make([]string, 0, len(entries)),
	}
	for i, pe := range entries {
		fieldPath := fmt.Sprintf("annotations.palette[%d]", i)
		id := strings.TrimSpace(pe.ID)
		if id == "" {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.id` is required.", fieldPath))
			return nil, res
		}
		if utf8.RuneCountInString(id) > maxPaletteIDRunes {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.id` exceeds maximum length of %d characters.", fieldPath, maxPaletteIDRunes))
			return nil, res
		}
		if _, dup := reg.byID[id]; dup {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("Duplicate palette ID `%s` in `%s.id`.", id, fieldPath))
			return nil, res
		}
		rgb, ok := parsePaletteColor(pe.Color)
		if !ok {
			res, _, _ := mdtemplate.ErrorResult(
				"INVALID_ARGUMENT",
				fmt.Sprintf("Unsupported color `%s` in `%s.color`.", pe.Color, fieldPath),
				"Use a preset name (`blue`, `amber`, `red`, `green`, `purple`, `teal`, `indigo`, `pink`, `brown`, `slate`) or 6-digit hex `#RRGGBB`.",
			)
			return nil, res
		}
		resolved := derivePaletteColors(id, rgb)
		reg.ordered = append(reg.ordered, resolved)
		reg.byID[id] = resolved
		reg.registeredIDs = append(reg.registeredIDs, id)
	}
	return reg, nil
}

func validateExplicitPaletteID(reg *paletteRegistry, fieldPath, rawID string) (string, *mcpsdk.CallToolResult) {
	pid := strings.TrimSpace(rawID)
	if pid == "" {
		return "", nil
	}
	if _, ok := reg.byID[pid]; !ok {
		if len(reg.registeredIDs) == 0 {
			res, _, _ := mdtemplate.ErrorResult(
				"UNKNOWN_PALETTE_ID",
				fmt.Sprintf("`%s.paletteId` references unregistered palette ID `%s` (`annotations.palette` is empty).", fieldPath, pid),
				"Register `{ id, color }` in `annotations.palette` (presets: `blue`, `amber`, `red`, `green`, `purple`, `teal`, `indigo`, `pink`, `brown`, `slate`, or `#RRGGBB`), or omit `paletteId` to inherit automatically.",
			)
			return "", res
		}
		res, _, _ := mdtemplate.ErrorResult(
			"UNKNOWN_PALETTE_ID",
			fmt.Sprintf("`%s.paletteId` references unregistered palette ID `%s`.", fieldPath, pid),
			fmt.Sprintf("Registered palette IDs: `%s`. Add `%s` to `annotations.palette` or use one of the registered IDs.", strings.Join(reg.registeredIDs, "`, `"), pid),
		)
		return "", res
	}
	return pid, nil
}

func parsePaletteColor(raw string) (rgbColor, bool) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if presetHex, ok := presetPaletteColors[s]; ok {
		return parseHexColor(presetHex)
	}
	return parseHexColor(s)
}

func parseHexColor(s string) (rgbColor, bool) {
	if len(s) != 7 || s[0] != '#' {
		return rgbColor{}, false
	}
	r, errR := strconv.ParseUint(s[1:3], 16, 8)
	g, errG := strconv.ParseUint(s[3:5], 16, 8)
	b, errB := strconv.ParseUint(s[5:7], 16, 8)
	if errR != nil || errG != nil || errB != nil {
		return rgbColor{}, false
	}
	return rgbColor{r: uint8(r), g: uint8(g), b: uint8(b)}, true
}

func mustParseHexColor(s string) rgbColor {
	c, ok := parseHexColor(s)
	if !ok {
		panic(fmt.Sprintf("invalid hex color constant %q", s))
	}
	return c
}

func derivePaletteColors(id string, c rgbColor) ResolvedPaletteColors {
	main := fmt.Sprintf("#%02x%02x%02x", c.r, c.g, c.b)
	lum := relativeLuminance(c)
	onMain := lightOnMainHex
	if contrastRatio(lum, darkOnMainLuminance) > contrastRatio(lum, lightOnMainLuminance) {
		onMain = darkOnMainHex
	}

	blendSurface := func(ch uint8) uint8 {
		return uint8(math.Round(255.0*0.90 + float64(ch)*0.10))
	}
	surface := fmt.Sprintf("#%02x%02x%02x", blendSurface(c.r), blendSurface(c.g), blendSurface(c.b))
	soft := fmt.Sprintf("rgba(%d, %d, %d, 0.13)", c.r, c.g, c.b)

	return ResolvedPaletteColors{
		ID:      id,
		Main:    main,
		OnMain:  onMain,
		Soft:    soft,
		Surface: surface,
	}
}

func relativeLuminance(c rgbColor) float64 {
	linearize := func(ch uint8) float64 {
		v := float64(ch) / 255.0
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*linearize(c.r) + 0.7152*linearize(c.g) + 0.0722*linearize(c.b)
}

func contrastRatio(l1, l2 float64) float64 {
	lighter := max(l1, l2)
	darker := min(l1, l2)
	return (lighter + 0.05) / (darker + 0.05)
}
