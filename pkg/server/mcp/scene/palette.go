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
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	maxScenePaletteEntries    = 16
	defaultPaletteID          = "default"
	defaultReferencePaletteID = "default-reference"
	maxPaletteIDRunes         = 40
	lightOnMainHex            = "#fafafa"
	darkOnMainHex             = "#212121"
)

type revAnchorKey struct {
	timelineID uint32
	revIndex   int
}

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

func assignInheritedPaletteIDs(overlay *Overlay, reg *paletteRegistry) {
	type scopedHighlight struct {
		paletteID string
		rowSet    map[uint32]bool
		rowCount  int
	}
	var explicitHighlights []scopedHighlight
	for _, rh := range overlay.RowHighlights {
		if rh.PaletteID == "" {
			continue
		}
		rowSet := make(map[uint32]bool, len(rh.TimelineIDs))
		for _, tid := range rh.TimelineIDs {
			rowSet[tid] = true
		}
		explicitHighlights = append(explicitHighlights, scopedHighlight{
			paletteID: rh.PaletteID,
			rowSet:    rowSet,
			rowCount:  len(rh.TimelineIDs),
		})
	}

	enclosingRowPalette := func(timelineID uint32) string {
		if timelineID == 0 {
			return ""
		}
		bestID := ""
		bestCount := math.MaxInt
		for _, h := range explicitHighlights {
			if h.rowSet[timelineID] && h.rowCount <= bestCount {
				bestID = h.paletteID
				bestCount = h.rowCount
			}
		}
		return bestID
	}

	byRev := make(map[revAnchorKey]string)
	byLog := make(map[uint32]string)
	recordAnchor := func(timelineID uint32, revIdx *int, logID uint32, pid string) {
		if pid == "" {
			return
		}
		if timelineID != 0 && revIdx != nil {
			key := revAnchorKey{timelineID: timelineID, revIndex: *revIdx}
			if _, exists := byRev[key]; !exists {
				byRev[key] = pid
			}
		}
		if logID != 0 {
			if _, exists := byLog[logID]; !exists {
				byLog[logID] = pid
			}
		}
	}

	for _, m := range overlay.Markers {
		recordAnchor(m.TimelineID, m.RevisionIndex, m.LogID, m.PaletteID)
	}
	for _, ex := range overlay.Excerpts {
		recordAnchor(ex.TimelineID, ex.RevisionIndex, ex.LogID, ex.PaletteID)
	}
	for _, ca := range overlay.Callouts {
		recordAnchor(ca.TimelineID, ca.RevisionIndex, ca.LogID, ca.PaletteID)
	}

	lookupCoAnchor := func(timelineID uint32, revIdx *int, logID uint32) string {
		if timelineID != 0 && revIdx != nil {
			if pid := byRev[revAnchorKey{timelineID: timelineID, revIndex: *revIdx}]; pid != "" {
				return pid
			}
		}
		if logID != 0 {
			if pid := byLog[logID]; pid != "" {
				return pid
			}
		}
		return ""
	}

	for i := range overlay.Markers {
		m := &overlay.Markers[i]
		if m.PaletteID != "" {
			continue
		}
		if pid := lookupCoAnchor(m.TimelineID, m.RevisionIndex, m.LogID); pid != "" {
			m.PaletteID = pid
			continue
		}
		if pid := enclosingRowPalette(m.TimelineID); pid != "" {
			m.PaletteID = pid
			recordAnchor(m.TimelineID, m.RevisionIndex, m.LogID, pid)
			continue
		}
		if m.Style == MarkerStyleReference {
			m.PaletteID = defaultReferencePaletteID
		} else {
			m.PaletteID = defaultPaletteID
		}
	}

	for i := range overlay.Excerpts {
		ex := &overlay.Excerpts[i]
		if ex.PaletteID != "" {
			continue
		}
		if pid := lookupCoAnchor(ex.TimelineID, ex.RevisionIndex, ex.LogID); pid != "" {
			ex.PaletteID = pid
			continue
		}
		if pid := enclosingRowPalette(ex.TimelineID); pid != "" {
			ex.PaletteID = pid
			recordAnchor(ex.TimelineID, ex.RevisionIndex, ex.LogID, pid)
			continue
		}
		ex.PaletteID = defaultPaletteID
	}

	for i := range overlay.Callouts {
		ca := &overlay.Callouts[i]
		if ca.PaletteID != "" {
			continue
		}
		if pid := lookupCoAnchor(ca.TimelineID, ca.RevisionIndex, ca.LogID); pid != "" {
			ca.PaletteID = pid
			continue
		}
		if pid := enclosingRowPalette(ca.TimelineID); pid != "" {
			ca.PaletteID = pid
			continue
		}
		ca.PaletteID = defaultPaletteID
	}

	for i := range overlay.Spans {
		sp := &overlay.Spans[i]
		if sp.PaletteID != "" {
			continue
		}
		if len(sp.TimelineIDs) > 0 {
			shared := ""
			consistent := true
			for _, tid := range sp.TimelineIDs {
				pid := enclosingRowPalette(tid)
				if pid == "" {
					consistent = false
					break
				}
				if shared == "" {
					shared = pid
				} else if shared != pid {
					consistent = false
					break
				}
			}
			if consistent && shared != "" {
				sp.PaletteID = shared
				continue
			}
		}
		sp.PaletteID = defaultPaletteID
	}

	for i := range overlay.Connectors {
		conn := &overlay.Connectors[i]
		if conn.PaletteID != "" {
			continue
		}
		if pid := enclosingRowPalette(conn.FromTimelineID); pid != "" {
			conn.PaletteID = pid
			continue
		}
		if pid := enclosingRowPalette(conn.ToTimelineID); pid != "" {
			conn.PaletteID = pid
			continue
		}
		conn.PaletteID = defaultPaletteID
	}

	used := make(map[string]bool)
	for _, m := range overlay.Markers {
		used[m.PaletteID] = true
	}
	for _, sp := range overlay.Spans {
		used[sp.PaletteID] = true
	}
	for _, rh := range overlay.RowHighlights {
		used[rh.PaletteID] = true
	}
	for _, ex := range overlay.Excerpts {
		used[ex.PaletteID] = true
	}
	for _, conn := range overlay.Connectors {
		used[conn.PaletteID] = true
	}
	for _, ca := range overlay.Callouts {
		used[ca.PaletteID] = true
	}

	resolvedPalette := slices.Clone(reg.ordered)
	if used[defaultPaletteID] {
		if _, exists := reg.byID[defaultPaletteID]; !exists {
			resolvedPalette = append(resolvedPalette, derivePaletteColors(defaultPaletteID, mustParseHexColor(presetPaletteColors["blue"])))
		}
	}
	if used[defaultReferencePaletteID] {
		if _, exists := reg.byID[defaultReferencePaletteID]; !exists {
			resolvedPalette = append(resolvedPalette, derivePaletteColors(defaultReferencePaletteID, mustParseHexColor(presetPaletteColors["purple"])))
		}
	}
	overlay.Palette = resolvedPalette
}
