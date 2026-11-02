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
	"encoding/json"
	"slices"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

// PaletteEntry binds a reusable semantic role identifier to a curated preset name or #RRGGBB hex color.
type PaletteEntry struct {
	ID    string `json:"id" jsonschema:"Palette entry identifier referenced via paletteId on annotations (e.g. 'trigger', 'victim', 'impact', 'recovery')."`
	Color string `json:"color" jsonschema:"Preset color name ('blue', 'amber', 'red', 'green', 'purple', 'teal', 'indigo', 'pink', 'brown', 'slate') or 6-digit hex '#RRGGBB'."`
}

// Annotations groups declarative visual overlays and patch directives for a rendered timeline scene.
type Annotations struct {
	ClearLayers     []string              `json:"clearLayers,omitempty" jsonschema:"Optional layer names to clear from parentSceneId before adding items: 'palette', 'markers', 'spans', 'rowHighlights', 'diffExcerpts', 'logExcerpts', 'connectors', 'callouts', or 'all'."`
	RemoveIDs       []string              `json:"removeIds,omitempty" jsonschema:"Optional badges, labels, or palette entry IDs (e.g. 'E2', 'ALERT', 'trigger') to remove from parentSceneId."`
	Palette         []PaletteEntry        `json:"palette,omitempty" jsonschema:"Named color definitions shared across markers, spans, rowHighlights, connectors, callouts, and evidence deck cards via paletteId (max 16)."`
	Markers         []MarkerAnnotation    `json:"markers,omitempty" jsonschema:"V1/V2: Vertical indicator lines with auto-staggered callout pills in the top annotation band (max 12)."`
	Spans           []SpanAnnotation      `json:"spans,omitempty" jsonschema:"V4: Time-range bands across the full chart or scoped via timelineQuery/timelineIds, including hatched missing-log gaps (max 6)."`
	RowHighlights   []RowHighlight        `json:"rowHighlights,omitempty" jsonschema:"V6: Highlights focal resource subtrees via timelineQuery or timelineIds with left-gutter role badges."`
	DimUnlistedRows *bool                 `json:"dimUnlistedRows,omitempty" jsonschema:"Softly dims background rows outside rowHighlights when true."`
	DiffExcerpts    []DiffExcerpt         `json:"diffExcerpts,omitempty" jsonschema:"V7: Resource YAML diff cards anchored to a revision and rendered in the bottom evidence deck (max 6 total excerpts)."`
	LogExcerpts     []LogExcerpt          `json:"logExcerpts,omitempty" jsonschema:"V8: Structured log excerpt cards anchored to a logId and rendered in the bottom evidence deck (max 6 total excerpts)."`
	Connectors      []ConnectorAnnotation `json:"connectors,omitempty" jsonschema:"V5: Directed causal arrows across rows or horizontal delta-time calipers between two anchors (max 8)."`
	Callouts        []CalloutAnnotation   `json:"callouts,omitempty" jsonschema:"V3: Compact 1-line inline callout cards pinned to a row event or revision (max 8)."`
}

// SliceWindow specifies a literal or regex search spot and surrounding context lines to retain.
type SliceWindow struct {
	Query        string `json:"query" jsonschema:"Target text to match (case-sensitive literal substring by default; RE2 regex when regex=true)."`
	ContextLines *int   `json:"contextLines,omitempty" jsonschema:"Symmetric number of surrounding lines to keep before and after each match (default 1, max 5)."`
	BeforeLines  *int   `json:"beforeLines,omitempty" jsonschema:"Optional asymmetric override for lines kept before each match (0..5)."`
	AfterLines   *int   `json:"afterLines,omitempty" jsonschema:"Optional asymmetric override for lines kept after each match (0..5)."`
	Regex        bool   `json:"regex,omitempty" jsonschema:"Interprets query as an RE2 regular expression when true."`
	MaxMatches   int    `json:"maxMatches,omitempty" jsonschema:"Maximum occurrences of this query to slice within the target diff or log (default 2, max 4)."`
}

// DiffExcerpt slices changed hunks from a resource revision transition into a bottom deck card.
type DiffExcerpt struct {
	Revision           RevisionAnchor `json:"revision" jsonschema:"Target resource timelineId and 0-based revisionIndex from get_resource_revisions."`
	Badge              string         `json:"badge,omitempty" jsonschema:"Optional short pin badge such as 'E1'. Defaults to 'E1'..'En' ordered by timestamp."`
	Windows            []SliceWindow  `json:"windows,omitempty" jsonschema:"Search spots within the managedFields-stripped unified diff. Disjoint matches are joined with 'N lines omitted' bands."`
	PreserveChangePair *bool          `json:"preserveChangePair,omitempty" jsonschema:"Defaults to true, keeping paired '-' and '+' lines of a matched replacement block intact."`
	MaxLines           int            `json:"maxLines,omitempty" jsonschema:"Maximum code lines rendered inside this deck card across all windows (default 10, max 16)."`
	Note               string         `json:"note,omitempty" jsonschema:"Optional 1-line human summary rendered in the card footer, visually separated from verbatim diff lines (max 80 chars)."`
	PaletteID          string         `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Inherits automatically from a co-anchored item or enclosing rowHighlight when omitted."`
}

// LogExcerpt slices structured YAML or multiline payload lines from a log entry into a bottom deck card.
type LogExcerpt struct {
	LogID          string        `json:"logId" jsonschema:"Target log ID from get_timeline_logs or search_logs."`
	TimelineID     string        `json:"timelineId,omitempty" jsonschema:"Optional timeline ID when a single log entry is linked to multiple timelines."`
	Badge          string        `json:"badge,omitempty" jsonschema:"Optional short pin badge such as 'E2'. Defaults to 'E1'..'En' ordered by timestamp."`
	Windows        []SliceWindow `json:"windows,omitempty" jsonschema:"Search spots across line-expanded structured log YAML. When omitted, renders only the parser-distilled Summary."`
	IncludeSummary *bool         `json:"includeSummary,omitempty" jsonschema:"Prepends the parser-distilled 1-line log Summary above sliced body windows when true."`
	MaxLines       int           `json:"maxLines,omitempty" jsonschema:"Maximum log lines rendered inside this deck card across all windows (default 10, max 16)."`
	Note           string        `json:"note,omitempty" jsonschema:"Optional 1-line human summary rendered in the card footer, visually separated from verbatim log lines (max 80 chars)."`
	PaletteID      string        `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Inherits automatically from a co-anchored item or enclosing rowHighlight when omitted."`
}

// MarkerAnnotation defines a vertical event pin (V1) or global reference line (V2).
type MarkerAnnotation struct {
	Anchor    Anchor `json:"anchor" jsonschema:"Timestamp and optional row anchor resolved from logId, revision, or RFC3339 time."`
	Label     string `json:"label" jsonschema:"Concise callout text shown in the top annotation band (max 80 chars)."`
	Badge     string `json:"badge,omitempty" jsonschema:"Short badge text such as '1', '2', 'ALERT', or 'TICKET'. Defaults to 1-based index."`
	Style     string `json:"style,omitempty" jsonschema:"'pin' (default: solid stem to target row circle) or 'reference' (dash-dot full-height external reference line)."`
	PaletteID string `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Inherits automatically from a co-anchored item or enclosing rowHighlight when omitted."`
}

// SpanAnnotation defines a full-height or row-scoped time-range band (V4).
type SpanAnnotation struct {
	Start          Anchor   `json:"start" jsonschema:"Inclusive start anchor of the interval."`
	End            Anchor   `json:"end" jsonschema:"Inclusive end anchor of the interval."`
	Label          string   `json:"label" jsonschema:"Concise label for the span or region box (max 80 chars)."`
	TimelineQuery  string   `json:"timelineQuery,omitempty" jsonschema:"Optional Timeline CEL query scoping the box vertically to matching rows."`
	TimelineIDs    []string `json:"timelineIds,omitempty" jsonschema:"Optional timeline IDs scoping the box vertically. Omit both for a full-height chart span."`
	Pattern        string   `json:"pattern,omitempty" jsonschema:"Fill style: 'solid' (default), 'hatched' (diagonal stripes + dashed border for missing logs), or 'outline'."`
	IncludeSubtree *bool    `json:"includeSubtree,omitempty" jsonschema:"Defaults to true when scoped to specific timelines."`
	PaletteID      string   `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Inherits from an enclosing rowHighlight when scoped to rows, or uses built-in 'default' when omitted."`
}

// RowHighlight highlights resource subtrees selected by Timeline CEL or explicit timeline IDs (V6).
type RowHighlight struct {
	TimelineQuery  string   `json:"timelineQuery,omitempty" jsonschema:"Optional Timeline CEL query selecting the resource subtree(s) to highlight."`
	TimelineIDs    []string `json:"timelineIds,omitempty" jsonschema:"Optional root timeline IDs of the resource subtree(s) to highlight."`
	RoleLabel      string   `json:"roleLabel,omitempty" jsonschema:"Concise role badge shown in the left margin (max 28 chars), e.g. 'TRIGGER · Node', 'VICTIM · Pod-1'."`
	IncludeSubtree *bool    `json:"includeSubtree,omitempty" jsonschema:"Defaults to true, covering child rows under matched timelines."`
	PaletteID      string   `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Uses built-in 'default' when omitted."`
}

// ConnectorAnnotation defines a directed causal arrow or horizontal duration caliper between two anchors (V5).
type ConnectorAnnotation struct {
	From      Anchor `json:"from" jsonschema:"Origin anchor (cause event/revision or interval start)."`
	To        Anchor `json:"to" jsonschema:"Destination anchor (effect event/revision or interval end)."`
	Label     string `json:"label,omitempty" jsonschema:"Optional concise label on the connector pill. Defaults to formatted elapsed delta."`
	Badge     string `json:"badge,omitempty" jsonschema:"Optional step number badge on causal arrows such as '1', '2'."`
	Style     string `json:"style,omitempty" jsonschema:"'curve' (default causal Bezier arrow), 'step' (orthogonal elbow arrow), or 'caliper' (horizontal duration line)."`
	PaletteID string `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Inherits from an enclosing rowHighlight when omitted."`
}

// CalloutAnnotation places a compact single-line card near the target point on the chart body (V3).
type CalloutAnnotation struct {
	Anchor    Anchor `json:"anchor" jsonschema:"Target logId, revision, or (time + timelineId) to point the leader line at."`
	Label     string `json:"label" jsonschema:"Concise annotation text shown inside the card (max 80 chars)."`
	Placement string `json:"placement,omitempty" jsonschema:"Preferred side relative to anchor: 'auto' (default), 'left', 'right', 'above', or 'below'."`
	Dashed    bool   `json:"dashed,omitempty" jsonschema:"Renders a dashed border and leader line for secondary/non-causal context when true."`
	PaletteID string `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Inherits automatically from a co-anchored item or enclosing rowHighlight when omitted."`
}

// Anchor specifies exactly one of LogID, Revision, or Time.
type Anchor struct {
	LogID      string          `json:"logId,omitempty" jsonschema:"Log ID from get_timeline_logs or search_logs."`
	TimelineID string          `json:"timelineId,omitempty" jsonschema:"Optional timeline ID to disambiguate multi-timeline logs or attach a row to an explicit RFC3339 time."`
	Revision   *RevisionAnchor `json:"revision,omitempty" jsonschema:"Resource revision boundary from get_resource_revisions."`
	Time       *time.Time      `json:"time,omitempty" jsonschema:"Explicit RFC3339 timestamp for external alerts, customer reports, or silent-gap boundaries."`
}

// RevisionAnchor pins an anchor to a specific resource revision transition.
type RevisionAnchor struct {
	TimelineID    string `json:"timelineId" jsonschema:"Target resource timeline ID from search_timelines."`
	RevisionIndex int    `json:"revisionIndex" jsonschema:"0-based revision index from get_resource_revisions."`
	Edge          string `json:"edge,omitempty" jsonschema:"'start' (default, revision ChangedTime) or 'end' (next revision ChangedTime)."`
}

// Spec captures the canonical, immutable scene parameters so child scenes can inherit and patch them.
type Spec struct {
	InspectionID       string
	Filter             workbench.Filter
	TimelineIDs        []string
	MaxDepth           int
	MaxNodes           int
	Width              int
	TimezoneShiftHours float64
	Annotations        *Annotations
}

// Clone returns a deep copy of Spec so caller mutations never alter stored scene history.
func (s Spec) Clone() Spec {
	cloned := s
	cloned.Filter = cloneFilter(s.Filter)
	cloned.TimelineIDs = slices.Clone(s.TimelineIDs)
	cloned.Annotations = cloneAnnotations(s.Annotations)
	return cloned
}

// ExcerptLineKind classifies a rendered row inside a diff or log excerpt card.
type ExcerptLineKind string

const (
	// ExcerptLineKindContext represents an unchanged context line in a unified diff.
	ExcerptLineKindContext ExcerptLineKind = "context"
	// ExcerptLineKindAdded represents an added line in a unified diff.
	ExcerptLineKindAdded ExcerptLineKind = "added"
	// ExcerptLineKindDeleted represents a deleted line in a unified diff.
	ExcerptLineKindDeleted ExcerptLineKind = "deleted"
	// ExcerptLineKindLog represents a structured YAML or expanded multiline log line.
	ExcerptLineKindLog ExcerptLineKind = "log"
	// ExcerptLineKindSummary represents a parser-distilled single-line log summary.
	ExcerptLineKindSummary ExcerptLineKind = "summary"
	// ExcerptLineKindOmitted represents a folded gap of omitted lines.
	ExcerptLineKindOmitted ExcerptLineKind = "omitted"
)

// LinePart represents a contiguous substring of an excerpt line and whether it matched a query.
type LinePart struct {
	Text    string `json:"text"`
	Matched bool   `json:"matched,omitempty"`
}

// ExcerptLine represents one display row in a diff or log excerpt card.
type ExcerptLine struct {
	Kind       ExcerptLineKind `json:"kind"`
	OldLineNum int             `json:"oldLineNum,omitempty"`
	NewLineNum int             `json:"newLineNum,omitempty"`
	Text       string          `json:"text"`
	Parts      []LinePart      `json:"parts,omitempty"`
	Omitted    int             `json:"omitted,omitempty"`
}

// ResolvedPaletteColors holds derived foreground, badge text, soft fill, and surface tint CSS colors for a palette entry.
type ResolvedPaletteColors struct {
	ID      string `json:"id"`
	Main    string `json:"main"`
	OnMain  string `json:"onMain"`
	Soft    string `json:"soft"`
	Surface string `json:"surface"`
}

// Overlay holds the resolved coordinates, matched timeline IDs, and sliced evidence cards for frontend rendering.
type Overlay struct {
	AutoIncludeTimelineIDs []uint32                `json:"autoIncludeTimelineIds,omitempty"`
	Palette                []ResolvedPaletteColors `json:"palette,omitempty"`
	Markers                []ResolvedMarker        `json:"markers,omitempty"`
	Spans                  []ResolvedSpan          `json:"spans,omitempty"`
	RowHighlights          []ResolvedRowHighlight  `json:"rowHighlights,omitempty"`
	DimUnlistedRows        bool                    `json:"dimUnlistedRows,omitempty"`
	Excerpts               []ResolvedExcerptCard   `json:"excerpts,omitempty"`
	Connectors             []ResolvedConnector     `json:"connectors,omitempty"`
	Callouts               []ResolvedCallout       `json:"callouts,omitempty"`
}

// ResolvedMarker represents a resolved V1 vertical pin or V2 reference line.
type ResolvedMarker struct {
	ID            string `json:"id"`
	Badge         string `json:"badge"`
	Label         string `json:"label"`
	Style         string `json:"style"`
	PaletteID     string `json:"paletteId"`
	TimeMs        int64  `json:"timeMs"`
	TimeLabel     string `json:"timeLabel"`
	TimelineID    uint32 `json:"timelineId,omitempty"`
	LogID         uint32 `json:"logId,omitempty"`
	RevisionIndex *int   `json:"revisionIndex,omitempty"`
}

// ResolvedSpan represents a resolved V4 full-height or row-scoped interval band.
type ResolvedSpan struct {
	ID            string   `json:"id"`
	Label         string   `json:"label"`
	StartTimeMs   int64    `json:"startTimeMs"`
	EndTimeMs     int64    `json:"endTimeMs"`
	DurationLabel string   `json:"durationLabel"`
	TimelineIDs   []uint32 `json:"timelineIds,omitempty"`
	Pattern       string   `json:"pattern"`
	PaletteID     string   `json:"paletteId"`
}

// ResolvedRowHighlight represents a resolved V6 focal subtree highlight and left-gutter role badge.
type ResolvedRowHighlight struct {
	ID          string   `json:"id"`
	RoleLabel   string   `json:"roleLabel,omitempty"`
	TimelineIDs []uint32 `json:"timelineIds"`
	PaletteID   string   `json:"paletteId"`
}

// ResolvedExcerptCard represents a resolved V7 diff card or V8 log card in the bottom evidence deck.
type ResolvedExcerptCard struct {
	ID            string        `json:"id"`
	Kind          string        `json:"kind"`
	Badge         string        `json:"badge"`
	PaletteID     string        `json:"paletteId"`
	TimeMs        int64         `json:"timeMs"`
	TimeLabel     string        `json:"timeLabel"`
	TimelineID    uint32        `json:"timelineId"`
	ResourcePath  string        `json:"resourcePath"`
	Title         string        `json:"title"`
	Subtitle      string        `json:"subtitle"`
	Lines         []ExcerptLine `json:"lines"`
	Note          string        `json:"note,omitempty"`
	LogTimestamp  string        `json:"logTimestamp,omitempty"`
	InsertID      string        `json:"insertId,omitempty"`
	LogID         uint32        `json:"logId,omitempty"`
	RevisionIndex *int          `json:"revisionIndex,omitempty"`
}

// ResolvedConnector represents a resolved V5 causal arrow or horizontal duration caliper.
type ResolvedConnector struct {
	ID             string `json:"id"`
	Badge          string `json:"badge,omitempty"`
	Label          string `json:"label,omitempty"`
	Style          string `json:"style"`
	PaletteID      string `json:"paletteId"`
	FromTimeMs     int64  `json:"fromTimeMs"`
	FromTimelineID uint32 `json:"fromTimelineId,omitempty"`
	ToTimeMs       int64  `json:"toTimeMs"`
	ToTimelineID   uint32 `json:"toTimelineId,omitempty"`
}

// ResolvedCallout represents a resolved V3 compact single-line callout card.
type ResolvedCallout struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	Placement     string `json:"placement"`
	Dashed        bool   `json:"dashed,omitempty"`
	PaletteID     string `json:"paletteId"`
	TimeMs        int64  `json:"timeMs"`
	TimelineID    uint32 `json:"timelineId,omitempty"`
	LogID         uint32 `json:"logId,omitempty"`
	RevisionIndex *int   `json:"revisionIndex,omitempty"`
}

// Document is the self-contained JSON payload served at /api/v1/mcp-scenes/:sceneId for interactive viewing.
type Document struct {
	SceneID                string    `json:"sceneId"`
	ParentSceneID          string    `json:"parentSceneId,omitempty"`
	ExpiresAt              time.Time `json:"expiresAt"`
	InspectionID           string    `json:"inspectionId"`
	TimelineQuery          string    `json:"timelineQuery,omitempty"`
	TimelineExclusionQuery string    `json:"timelineExclusionQuery,omitempty"`
	LogQuery               string    `json:"logQuery,omitempty"`
	ExcludeNoLogs          bool      `json:"excludeNoLogs,omitempty"`
	FilterStartTimeNs      string    `json:"filterStartTimeNs,omitempty"`
	FilterEndTimeNs        string    `json:"filterEndTimeNs,omitempty"`
	ViewStartTimeMs        *int64    `json:"viewStartTimeMs,omitempty"`
	ViewEndTimeMs          *int64    `json:"viewEndTimeMs,omitempty"`
	TimelineIDs            []uint32  `json:"timelineIds,omitempty"`
	MaxDepth               int       `json:"maxDepth,omitempty"`
	MaxNodes               int       `json:"maxNodes,omitempty"`
	Width                  int       `json:"width,omitempty"`
	TimezoneShiftHours     float64   `json:"timezoneShiftHours,omitempty"`
	Overlay                *Overlay  `json:"overlay,omitempty"`
}

func cloneFilter(f workbench.Filter) workbench.Filter {
	out := f
	if f.StartTime != nil {
		t := f.StartTime.UTC()
		out.StartTime = &t
	}
	if f.EndTime != nil {
		t := f.EndTime.UTC()
		out.EndTime = &t
	}
	if f.ExcludeTimelinesWithoutLogs != nil {
		v := *f.ExcludeTimelinesWithoutLogs
		out.ExcludeTimelinesWithoutLogs = &v
	}
	return out
}

func cloneAnnotations(a *Annotations) *Annotations {
	if a == nil {
		return nil
	}
	data, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}
	var out Annotations
	if err := json.Unmarshal(data, &out); err != nil {
		panic(err)
	}
	return &out
}
