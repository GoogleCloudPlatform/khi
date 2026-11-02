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
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestBuildPaletteRegistry_PresetsAndHex(t *testing.T) {
	testCases := []struct {
		name  string
		entry PaletteEntry
		want  ResolvedPaletteColors
	}{
		{
			name:  "preset blue",
			entry: PaletteEntry{ID: "info", Color: "blue"},
			want: ResolvedPaletteColors{
				ID:      "info",
				Main:    "#1976d2",
				OnMain:  "#fafafa",
				Soft:    "rgba(25, 118, 210, 0.13)",
				Surface: "#e8f1fb",
			},
		},
		{
			name:  "preset amber",
			entry: PaletteEntry{ID: "trigger", Color: "amber"},
			want: ResolvedPaletteColors{
				ID:      "trigger",
				Main:    "#b45309",
				OnMain:  "#fafafa",
				Soft:    "rgba(180, 83, 9, 0.13)",
				Surface: "#f8eee6",
			},
		},
		{
			name:  "preset red",
			entry: PaletteEntry{ID: "victim", Color: "red"},
			want: ResolvedPaletteColors{
				ID:      "victim",
				Main:    "#d32f2f",
				OnMain:  "#fafafa",
				Soft:    "rgba(211, 47, 47, 0.13)",
				Surface: "#fbeaea",
			},
		},
		{
			name:  "preset green",
			entry: PaletteEntry{ID: "recovery", Color: "green"},
			want: ResolvedPaletteColors{
				ID:      "recovery",
				Main:    "#2e7d32",
				OnMain:  "#fafafa",
				Soft:    "rgba(46, 125, 50, 0.13)",
				Surface: "#eaf2eb",
			},
		},
		{
			name:  "preset purple",
			entry: PaletteEntry{ID: "external", Color: "purple"},
			want: ResolvedPaletteColors{
				ID:      "external",
				Main:    "#5e35b1",
				OnMain:  "#fafafa",
				Soft:    "rgba(94, 53, 177, 0.13)",
				Surface: "#efebf7",
			},
		},
		{
			name:  "preset teal",
			entry: PaletteEntry{ID: "net", Color: "teal"},
			want: ResolvedPaletteColors{
				ID:      "net",
				Main:    "#00796b",
				OnMain:  "#fafafa",
				Soft:    "rgba(0, 121, 107, 0.13)",
				Surface: "#e6f2f0",
			},
		},
		{
			name:  "preset indigo",
			entry: PaletteEntry{ID: "ctrl", Color: "indigo"},
			want: ResolvedPaletteColors{
				ID:      "ctrl",
				Main:    "#3949ab",
				OnMain:  "#fafafa",
				Soft:    "rgba(57, 73, 171, 0.13)",
				Surface: "#ebedf7",
			},
		},
		{
			name:  "preset pink",
			entry: PaletteEntry{ID: "oom", Color: "pink"},
			want: ResolvedPaletteColors{
				ID:      "oom",
				Main:    "#c2185b",
				OnMain:  "#fafafa",
				Soft:    "rgba(194, 24, 91, 0.13)",
				Surface: "#f9e8ef",
			},
		},
		{
			name:  "preset brown",
			entry: PaletteEntry{ID: "disk", Color: "brown"},
			want: ResolvedPaletteColors{
				ID:      "disk",
				Main:    "#5d4037",
				OnMain:  "#fafafa",
				Soft:    "rgba(93, 64, 55, 0.13)",
				Surface: "#efeceb",
			},
		},
		{
			name:  "preset slate",
			entry: PaletteEntry{ID: "bg", Color: "slate"},
			want: ResolvedPaletteColors{
				ID:      "bg",
				Main:    "#546e7a",
				OnMain:  "#fafafa",
				Soft:    "rgba(84, 110, 122, 0.13)",
				Surface: "#eef1f2",
			},
		},
		{
			name:  "uppercase bright yellow hex selects dark onMain for WCAG contrast",
			entry: PaletteEntry{ID: "highlight", Color: "#FFEB3B"},
			want: ResolvedPaletteColors{
				ID:      "highlight",
				Main:    "#ffeb3b",
				OnMain:  "#212121",
				Soft:    "rgba(255, 235, 59, 0.13)",
				Surface: "#fffdeb",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reg, errRes := buildPaletteRegistry([]PaletteEntry{tc.entry})
			if errRes != nil {
				t.Fatalf("buildPaletteRegistry() unexpected error = %+v", errRes)
			}
			if len(reg.ordered) != 1 {
				t.Fatalf("len(reg.ordered) = %d, want 1", len(reg.ordered))
			}
			if diff := cmp.Diff(tc.want, reg.ordered[0]); diff != "" {
				t.Errorf("resolved palette colors mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildPaletteRegistry_ValidationErrors(t *testing.T) {
	tooMany := make([]PaletteEntry, maxScenePaletteEntries+1)
	for i := range tooMany {
		tooMany[i] = PaletteEntry{ID: fmt.Sprintf("p-%d", i), Color: "blue"}
	}

	testCases := []struct {
		name           string
		entries        []PaletteEntry
		wantErrSubstrs []string
	}{
		{
			name:           "empty id",
			entries:        []PaletteEntry{{ID: "   ", Color: "red"}},
			wantErrSubstrs: []string{"Error: INVALID_ARGUMENT", "`annotations.palette[0].id` is required."},
		},
		{
			name: "id exceeds max length",
			entries: []PaletteEntry{{
				ID:    strings.Repeat("a", maxPaletteIDRunes+1),
				Color: "red",
			}},
			wantErrSubstrs: []string{"Error: INVALID_ARGUMENT", "exceeds maximum length"},
		},
		{
			name: "duplicate id",
			entries: []PaletteEntry{
				{ID: "victim", Color: "red"},
				{ID: "victim", Color: "amber"},
			},
			wantErrSubstrs: []string{"Error: INVALID_ARGUMENT", "Duplicate palette ID `victim`"},
		},
		{
			name:           "unsupported color name",
			entries:        []PaletteEntry{{ID: "victim", Color: "chartreuse"}},
			wantErrSubstrs: []string{"Error: INVALID_ARGUMENT", "Unsupported color `chartreuse`"},
		},
		{
			name:           "short hex #RGB rejected",
			entries:        []PaletteEntry{{ID: "victim", Color: "#f00"}},
			wantErrSubstrs: []string{"Error: INVALID_ARGUMENT", "Unsupported color `#f00`"},
		},
		{
			name:           "non-hex #GGGGGG rejected",
			entries:        []PaletteEntry{{ID: "victim", Color: "#GGGGGG"}},
			wantErrSubstrs: []string{"Error: INVALID_ARGUMENT", "Unsupported color `#GGGGGG`"},
		},
		{
			name:           "exceeds maxScenePaletteEntries",
			entries:        tooMany,
			wantErrSubstrs: []string{"Error: INVALID_ARGUMENT", "exceeds maximum of 16 entries"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, errRes := buildPaletteRegistry(tc.entries)
			if errRes == nil || !errRes.IsError || len(errRes.Content) == 0 {
				t.Fatalf("buildPaletteRegistry() errRes = %+v, want error", errRes)
			}
			gotText := errRes.Content[0].(*mcpsdk.TextContent).Text
			for _, sub := range tc.wantErrSubstrs {
				if !strings.Contains(gotText, sub) {
					t.Errorf("error text = %q, missing substring %q", gotText, sub)
				}
			}
		})
	}
}

func TestValidateExplicitPaletteID(t *testing.T) {
	emptyReg, _ := buildPaletteRegistry(nil)
	populatedReg, _ := buildPaletteRegistry([]PaletteEntry{
		{ID: "trigger", Color: "amber"},
		{ID: "victim", Color: "red"},
	})

	t.Run("empty paletteId returns empty string without error", func(t *testing.T) {
		got, errRes := validateExplicitPaletteID(emptyReg, "annotations.markers[0]", "  ")
		if errRes != nil || got != "" {
			t.Fatalf("validateExplicitPaletteID() = (%q, %+v), want (\"\", nil)", got, errRes)
		}
	})

	t.Run("unknown paletteId with empty registry explains how to register", func(t *testing.T) {
		_, errRes := validateExplicitPaletteID(emptyReg, "annotations.markers[0]", "victim")
		if errRes == nil || !errRes.IsError {
			t.Fatalf("expected UNKNOWN_PALETTE_ID error, got %+v", errRes)
		}
		gotText := errRes.Content[0].(*mcpsdk.TextContent).Text
		if !strings.HasPrefix(gotText, "Error: UNKNOWN_PALETTE_ID") || !strings.Contains(gotText, "`annotations.palette` is empty") {
			t.Errorf("unexpected error text:\n%s", gotText)
		}
	})

	t.Run("unknown paletteId with populated registry lists registered IDs", func(t *testing.T) {
		_, errRes := validateExplicitPaletteID(populatedReg, "annotations.spans[1]", "typo-id")
		if errRes == nil || !errRes.IsError {
			t.Fatalf("expected UNKNOWN_PALETTE_ID error, got %+v", errRes)
		}
		gotText := errRes.Content[0].(*mcpsdk.TextContent).Text
		if !strings.HasPrefix(gotText, "Error: UNKNOWN_PALETTE_ID") || !strings.Contains(gotText, "Registered palette IDs: `trigger`, `victim`") {
			t.Errorf("unexpected error text:\n%s", gotText)
		}
	})
}

func TestSpec_Clone_DeepCopiesMutableFields(t *testing.T) {
	start := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	excludeNoLogs := true
	dim := true
	orig := Spec{
		InspectionID: "insp-1",
		Filter: workbench.Filter{
			TimelineQuery:               "kind == 'pod'",
			StartTime:                   &start,
			ExcludeTimelinesWithoutLogs: &excludeNoLogs,
		},
		TimelineIDs: []string{"10", "20"},
		Annotations: &Annotations{
			Palette:         []PaletteEntry{{ID: "victim", Color: "red"}},
			DimUnlistedRows: &dim,
		},
	}

	cloned := orig.Clone()
	cloned.TimelineIDs[0] = "99"
	*cloned.Filter.ExcludeTimelinesWithoutLogs = false
	cloned.Annotations.Palette[0].Color = "blue"
	*cloned.Annotations.DimUnlistedRows = false

	if orig.TimelineIDs[0] != "10" {
		t.Errorf("orig.TimelineIDs[0] mutated to %q, want \"10\"", orig.TimelineIDs[0])
	}
	if !*orig.Filter.ExcludeTimelinesWithoutLogs {
		t.Error("orig.Filter.ExcludeTimelinesWithoutLogs mutated to false, want true")
	}
	if orig.Annotations.Palette[0].Color != "red" {
		t.Errorf("orig.Annotations.Palette[0].Color mutated to %q, want \"red\"", orig.Annotations.Palette[0].Color)
	}
	if !*orig.Annotations.DimUnlistedRows {
		t.Error("orig.Annotations.DimUnlistedRows mutated to false, want true")
	}
}
