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
	"slices"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

// AnnotationLayer identifies a named overlay layer in Annotations.ClearLayers.
type AnnotationLayer string

const (
	// AnnotationLayerAll clears all annotation layers on a parent scene.
	AnnotationLayerAll AnnotationLayer = "all"
	// AnnotationLayerPalette clears the named palette entries on a parent scene.
	AnnotationLayerPalette AnnotationLayer = "palette"
	// AnnotationLayerMarkers clears vertical marker annotations on a parent scene.
	AnnotationLayerMarkers AnnotationLayer = "markers"
	// AnnotationLayerSpans clears time-range span annotations on a parent scene.
	AnnotationLayerSpans AnnotationLayer = "spans"
	// AnnotationLayerRowHighlights clears row highlight annotations on a parent scene.
	AnnotationLayerRowHighlights AnnotationLayer = "rowHighlights"
	// AnnotationLayerDiffExcerpts clears resource diff excerpt cards on a parent scene.
	AnnotationLayerDiffExcerpts AnnotationLayer = "diffExcerpts"
	// AnnotationLayerLogExcerpts clears structured log excerpt cards on a parent scene.
	AnnotationLayerLogExcerpts AnnotationLayer = "logExcerpts"
	// AnnotationLayerConnectors clears connector annotations on a parent scene.
	AnnotationLayerConnectors AnnotationLayer = "connectors"
	// AnnotationLayerCallouts clears inline callout annotations on a parent scene.
	AnnotationLayerCallouts AnnotationLayer = "callouts"
)

// MarkerStyle defines the visual style of a vertical marker annotation.
type MarkerStyle string

const (
	// MarkerStylePin renders a solid stem anchored to a target row circle.
	MarkerStylePin MarkerStyle = "pin"
	// MarkerStyleReference renders a dash-dot full-height external reference line.
	MarkerStyleReference MarkerStyle = "reference"
)

// SpanPattern defines the fill pattern of a time-range span annotation.
type SpanPattern string

const (
	// SpanPatternSolid renders a solid translucent fill band.
	SpanPatternSolid SpanPattern = "solid"
	// SpanPatternHatched renders diagonal stripes with a dashed border for missing logs.
	SpanPatternHatched SpanPattern = "hatched"
	// SpanPatternOutline renders an unfilled border box.
	SpanPatternOutline SpanPattern = "outline"
)

// SpanLabelPlacement defines where a time-range span's header chip is positioned relative to its bounding box.
type SpanLabelPlacement string

const (
	// SpanLabelPlacementAuto places the label chip above compact row-scoped spans and inside tall or full-height spans.
	SpanLabelPlacementAuto SpanLabelPlacement = "auto"
	// SpanLabelPlacementAbove places the label chip directly above the top border of the span box.
	SpanLabelPlacementAbove SpanLabelPlacement = "above"
	// SpanLabelPlacementInside places the label chip inside the top edge of the span box.
	SpanLabelPlacementInside SpanLabelPlacement = "inside"
)

// ConnectorStyle defines the routing geometry of a connector annotation.
type ConnectorStyle string

const (
	// ConnectorStyleCurve renders a directed causal Bezier curve arrow.
	ConnectorStyleCurve ConnectorStyle = "curve"
	// ConnectorStyleStep renders a directed orthogonal elbow arrow.
	ConnectorStyleStep ConnectorStyle = "step"
	// ConnectorStyleCaliper renders a horizontal duration caliper line.
	ConnectorStyleCaliper ConnectorStyle = "caliper"
)

// CalloutPlacement defines the preferred side of an inline callout card relative to its anchor.
type CalloutPlacement string

const (
	// CalloutPlacementAuto automatically selects a collision-free side around the anchor.
	CalloutPlacementAuto CalloutPlacement = "auto"
	// CalloutPlacementLeft places the callout card to the left of the anchor.
	CalloutPlacementLeft CalloutPlacement = "left"
	// CalloutPlacementRight places the callout card to the right of the anchor.
	CalloutPlacementRight CalloutPlacement = "right"
	// CalloutPlacementAbove places the callout card above the anchor.
	CalloutPlacementAbove CalloutPlacement = "above"
	// CalloutPlacementBelow places the callout card below the anchor.
	CalloutPlacementBelow CalloutPlacement = "below"
)

// RevisionEdge selects either the start or end timestamp of a resource revision interval.
type RevisionEdge string

const (
	// RevisionEdgeStart selects the revision's ChangedTime.
	RevisionEdgeStart RevisionEdge = "start"
	// RevisionEdgeEnd selects the next revision's ChangedTime.
	RevisionEdgeEnd RevisionEdge = "end"
)

// ExcerptKind classifies an evidence deck card as a resource diff or structured log excerpt.
type ExcerptKind string

const (
	// ExcerptKindDiff identifies a resource manifest diff excerpt card.
	ExcerptKindDiff ExcerptKind = "diff"
	// ExcerptKindLog identifies a structured log excerpt card.
	ExcerptKindLog ExcerptKind = "log"
)

// PaletteEntry binds a reusable semantic role identifier to a curated preset name or #RRGGBB hex color.
type PaletteEntry struct {
	ID    string `json:"id" jsonschema:"Palette entry identifier referenced via paletteId on annotations (e.g. 'trigger', 'victim', 'impact', 'recovery')."`
	Color string `json:"color" jsonschema:"Preset color name ('blue', 'amber', 'red', 'green', 'purple', 'teal', 'indigo', 'pink', 'brown', 'slate') or 6-digit hex '#RRGGBB'."`
}

// Annotations groups declarative visual overlays and patch directives for a rendered timeline scene.
type Annotations struct {
	ClearLayers     []AnnotationLayer     `json:"clearLayers,omitempty" jsonschema:"Optional layer names to clear from parentSceneId before adding items: 'palette', 'markers', 'spans', 'rowHighlights', 'diffExcerpts', 'logExcerpts', 'connectors', 'callouts', or 'all'."`
	RemoveIDs       []string              `json:"removeIds,omitempty" jsonschema:"Optional badges, labels, or palette entry IDs (e.g. 'E2', 'ALERT', 'trigger') to remove from parentSceneId."`
	Palette         []PaletteEntry        `json:"palette,omitempty" jsonschema:"Named color definitions shared across markers, spans, rowHighlights, connectors, callouts, and evidence deck cards via paletteId (max 16)."`
	Markers         []MarkerAnnotation    `json:"markers,omitempty" jsonschema:"Vertical indicator lines with auto-staggered callout pills in the top annotation band (max 12)."`
	Spans           []SpanAnnotation      `json:"spans,omitempty" jsonschema:"Time-range bands across the full chart or scoped via timelineQuery/timelineIds, including hatched missing-log gaps (max 6)."`
	RowHighlights   []RowHighlight        `json:"rowHighlights,omitempty" jsonschema:"Highlights focal resource subtrees via timelineQuery or timelineIds with left-gutter role badges."`
	DimUnlistedRows *bool                 `json:"dimUnlistedRows,omitempty" jsonschema:"Softly dims background rows outside rowHighlights when true."`
	DiffExcerpts    []DiffExcerpt         `json:"diffExcerpts,omitempty" jsonschema:"Resource YAML diff cards anchored to a revision and rendered in the bottom evidence deck (max 6 total excerpts)."`
	LogExcerpts     []LogExcerpt          `json:"logExcerpts,omitempty" jsonschema:"Structured log excerpt cards anchored to a logId and rendered in the bottom evidence deck (max 6 total excerpts)."`
	Connectors      []ConnectorAnnotation `json:"connectors,omitempty" jsonschema:"Directed causal arrows across rows or horizontal delta-time calipers between two anchors (max 8)."`
	Callouts        []CalloutAnnotation   `json:"callouts,omitempty" jsonschema:"Compact 1-line inline callout cards pinned to a row event or revision (max 8)."`
}

// SliceWindow specifies a literal search spot and surrounding context lines to retain.
type SliceWindow struct {
	Query       string `json:"query" jsonschema:"Case-sensitive literal substring to match within the target diff or log."`
	BeforeLines int    `json:"beforeLines" jsonschema:"Number of context lines to retain before each match (0..5)."`
	AfterLines  int    `json:"afterLines" jsonschema:"Number of context lines to retain after each match (0..5)."`
	MaxMatches  int    `json:"maxMatches,omitempty" jsonschema:"Maximum occurrences of this query to slice within the target diff or log (default 2, max 4)."`
}

// RevisionRef identifies a resource revision transition by timeline ID and 0-based revision index.
type RevisionRef struct {
	TimelineID    uint32 `json:"timelineId" jsonschema:"Target resource timeline ID from search_timelines."`
	RevisionIndex int    `json:"revisionIndex" jsonschema:"0-based revision index from get_resource_revisions."`
}

// DiffExcerpt slices changed hunks from a resource revision transition into a bottom deck card.
type DiffExcerpt struct {
	Revision  RevisionRef   `json:"revision" jsonschema:"Target resource timelineId and 0-based revisionIndex from get_resource_revisions."`
	Badge     string        `json:"badge,omitempty" jsonschema:"Optional short pin badge such as 'E1'. Defaults to 'E1'..'En' ordered by timestamp."`
	Windows   []SliceWindow `json:"windows,omitempty" jsonschema:"Search spots within the managedFields-stripped unified diff. Disjoint matches are joined with 'N lines omitted' bands."`
	Note      string        `json:"note,omitempty" jsonschema:"Optional 1-line human summary rendered in the card footer, visually separated from verbatim diff lines (max 80 chars)."`
	PaletteID string        `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Inherits automatically from a co-anchored item or enclosing rowHighlight when omitted."`
}

// LogExcerpt slices structured YAML or multiline payload lines from a log entry into a bottom deck card.
type LogExcerpt struct {
	LogID      uint32        `json:"logId" jsonschema:"Target log ID from get_timeline_logs or search_logs."`
	TimelineID uint32        `json:"timelineId,omitempty" jsonschema:"Optional timeline ID when a single log entry is linked to multiple timelines."`
	Badge      string        `json:"badge,omitempty" jsonschema:"Optional short pin badge such as 'E2'. Defaults to 'E1'..'En' ordered by timestamp."`
	Windows    []SliceWindow `json:"windows,omitempty" jsonschema:"Search spots across line-expanded structured log YAML. The parser-distilled Summary is always included."`
	Note       string        `json:"note,omitempty" jsonschema:"Optional 1-line human summary rendered in the card footer, visually separated from verbatim log lines (max 80 chars)."`
	PaletteID  string        `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Inherits automatically from a co-anchored item or enclosing rowHighlight when omitted."`
}

// MarkerAnnotation defines a vertical event pin or global reference line.
type MarkerAnnotation struct {
	Anchor    Anchor      `json:"anchor" jsonschema:"Target timeline event (timelineId + logId), revision (timelineId + revisionIndex), or fallback RFC3339 time."`
	Label     string      `json:"label" jsonschema:"Concise callout text shown in the top annotation band (max 80 chars)."`
	Badge     string      `json:"badge,omitempty" jsonschema:"Short badge text such as '1', '2', 'ALERT', or 'TICKET'. Defaults to 1-based index."`
	Style     MarkerStyle `json:"style,omitempty" jsonschema:"'pin' (default: solid stem to target row circle) or 'reference' (dash-dot full-height external reference line)."`
	PaletteID string      `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Inherits automatically from a co-anchored item or enclosing rowHighlight when omitted."`
}

// SpanAnnotation defines a full-height or row-scoped time-range band.
type SpanAnnotation struct {
	Start          Anchor             `json:"start" jsonschema:"Inclusive start anchor of the interval."`
	End            Anchor             `json:"end" jsonschema:"Inclusive end anchor of the interval."`
	Label          string             `json:"label" jsonschema:"Concise label for the span or region box (max 80 chars)."`
	TimelineQuery  string             `json:"timelineQuery,omitempty" jsonschema:"Optional Timeline CEL query scoping the box vertically to matching rows."`
	TimelineIDs    []uint32           `json:"timelineIds,omitempty" jsonschema:"Optional timeline IDs scoping the box vertically. Omit both for a full-height chart span."`
	Pattern        SpanPattern        `json:"pattern,omitempty" jsonschema:"Fill style: 'solid' (default), 'hatched' (diagonal stripes + dashed border for missing logs), or 'outline'."`
	LabelPlacement SpanLabelPlacement `json:"labelPlacement,omitempty" jsonschema:"Header chip placement: 'auto' (default: 'above' for compact row-scoped spans and 'inside' for tall or full-height spans), 'above', or 'inside'."`
	HideDuration   bool               `json:"hideDuration,omitempty" jsonschema:"Omits the automatic '(duration)' suffix from the span header chip when true."`
	IncludeSubtree *bool              `json:"includeSubtree,omitempty" jsonschema:"Defaults to true when scoped to specific timelines."`
	PaletteID      string             `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Inherits from an enclosing rowHighlight when scoped to rows, or uses built-in 'default' when omitted."`
}

// RowHighlight highlights resource subtrees selected by Timeline CEL or explicit timeline IDs.
type RowHighlight struct {
	TimelineQuery  string   `json:"timelineQuery,omitempty" jsonschema:"Optional Timeline CEL query selecting the resource subtree(s) to highlight."`
	TimelineIDs    []uint32 `json:"timelineIds,omitempty" jsonschema:"Optional root timeline IDs of the resource subtree(s) to highlight."`
	RoleLabel      string   `json:"roleLabel,omitempty" jsonschema:"Concise role badge shown in the left margin (max 28 chars), e.g. 'TRIGGER · Node', 'VICTIM · Pod-1'."`
	IncludeSubtree *bool    `json:"includeSubtree,omitempty" jsonschema:"Defaults to true, covering child rows under matched timelines."`
	PaletteID      string   `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Uses built-in 'default' when omitted."`
}

// ConnectorAnnotation defines a directed causal arrow or horizontal duration caliper between two anchors.
type ConnectorAnnotation struct {
	From      Anchor         `json:"from" jsonschema:"Origin anchor (cause event/revision or interval start)."`
	To        Anchor         `json:"to" jsonschema:"Destination anchor (effect event/revision or interval end)."`
	Label     string         `json:"label,omitempty" jsonschema:"Optional concise label on the connector pill. Defaults to formatted elapsed delta."`
	Badge     string         `json:"badge,omitempty" jsonschema:"Optional step number badge on causal arrows such as '1', '2'."`
	Style     ConnectorStyle `json:"style,omitempty" jsonschema:"'curve' (default causal Bezier arrow), 'step' (orthogonal elbow arrow), or 'caliper' (horizontal duration line)."`
	PaletteID string         `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Inherits from an enclosing rowHighlight when omitted."`
}

// CalloutAnnotation places a compact single-line card near the target point on the chart body.
type CalloutAnnotation struct {
	Anchor    Anchor           `json:"anchor" jsonschema:"Target timeline event (timelineId + logId), revision (timelineId + revisionIndex), or fallback (time + timelineId) to point the leader line at."`
	Label     string           `json:"label" jsonschema:"Concise annotation text shown inside the card (max 80 chars)."`
	Placement CalloutPlacement `json:"placement,omitempty" jsonschema:"Preferred side relative to anchor: 'auto' (default), 'left', 'right', 'above', or 'below'."`
	Dashed    bool             `json:"dashed,omitempty" jsonschema:"Renders a dashed border and leader line for secondary/non-causal context when true."`
	PaletteID string           `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Inherits automatically from a co-anchored item or enclosing rowHighlight when omitted."`
}

// Anchor pins an annotation to a timeline event (TimelineID + LogID), a resource revision (TimelineID + RevisionIndex), or a fallback timestamp.
type Anchor struct {
	TimelineID    uint32       `json:"timelineId,omitempty" jsonschema:"Target timeline ID from search_timelines or search_logs. Required when specifying logId or revisionIndex; optional when attaching a row to time."`
	LogID         uint32       `json:"logId,omitempty" jsonschema:"Log ID on timelineId from get_timeline_logs or search_logs. Preferred for event anchors to avoid timestamp hallucination."`
	RevisionIndex *int         `json:"revisionIndex,omitempty" jsonschema:"0-based revision index on timelineId from get_resource_revisions. Preferred for revision anchors to avoid timestamp hallucination."`
	RevisionEdge  RevisionEdge `json:"revisionEdge,omitempty" jsonschema:"Revision boundary when revisionIndex is set: 'start' (default, revision ChangedTime) or 'end' (next revision ChangedTime)."`
	Time          *time.Time   `json:"time,omitempty" jsonschema:"Fallback RFC3339 timestamp only when no log or revision exists on the timeline. Prefer timelineId + logId or timelineId + revisionIndex whenever possible to reduce timestamp hallucinations."`
}

// Spec captures the canonical, immutable scene parameters so child scenes can inherit and patch them.
type Spec struct {
	InspectionID       string
	Filter             workbench.Filter
	TimelineIDs        []uint32
	MaxDepth           int
	MaxNodes           int
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
	// ExcerptLineKindContext represents an unchanged context line in a unified diff or structured log.
	ExcerptLineKindContext ExcerptLineKind = "context"
	// ExcerptLineKindAdded represents an added line in a unified diff.
	ExcerptLineKindAdded ExcerptLineKind = "added"
	// ExcerptLineKindDeleted represents a deleted line in a unified diff.
	ExcerptLineKindDeleted ExcerptLineKind = "deleted"
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

// ResolvedMarker represents a resolved vertical event pin or global reference line.
type ResolvedMarker struct {
	ID            string      `json:"id"`
	Badge         string      `json:"badge"`
	Label         string      `json:"label"`
	Style         MarkerStyle `json:"style"`
	PaletteID     string      `json:"paletteId"`
	TimeMs        int64       `json:"timeMs"`
	TimeLabel     string      `json:"timeLabel"`
	TimelineID    uint32      `json:"timelineId,omitempty"`
	LogID         uint32      `json:"logId,omitempty"`
	RevisionIndex *int        `json:"revisionIndex,omitempty"`
}

// ResolvedSpan represents a resolved full-height or row-scoped interval band.
type ResolvedSpan struct {
	ID             string             `json:"id"`
	Label          string             `json:"label"`
	StartTimeMs    int64              `json:"startTimeMs"`
	EndTimeMs      int64              `json:"endTimeMs"`
	DurationLabel  string             `json:"durationLabel"`
	TimelineIDs    []uint32           `json:"timelineIds,omitempty"`
	Pattern        SpanPattern        `json:"pattern"`
	LabelPlacement SpanLabelPlacement `json:"labelPlacement,omitempty"`
	HideDuration   bool               `json:"hideDuration,omitempty"`
	PaletteID      string             `json:"paletteId"`
}

// ResolvedRowHighlight represents a resolved focal subtree highlight and left-gutter role badge.
type ResolvedRowHighlight struct {
	ID          string   `json:"id"`
	RoleLabel   string   `json:"roleLabel,omitempty"`
	TimelineIDs []uint32 `json:"timelineIds"`
	PaletteID   string   `json:"paletteId"`
}

// ResolvedExcerptCard represents a resolved diff card or log card in the bottom evidence deck.
type ResolvedExcerptCard struct {
	ID            string        `json:"id"`
	Kind          ExcerptKind   `json:"kind"`
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

// ResolvedConnector represents a resolved causal arrow or horizontal duration caliper.
type ResolvedConnector struct {
	ID             string         `json:"id"`
	Badge          string         `json:"badge,omitempty"`
	Label          string         `json:"label,omitempty"`
	Style          ConnectorStyle `json:"style"`
	PaletteID      string         `json:"paletteId"`
	FromTimeMs     int64          `json:"fromTimeMs"`
	FromTimelineID uint32         `json:"fromTimelineId,omitempty"`
	ToTimeMs       int64          `json:"toTimeMs"`
	ToTimelineID   uint32         `json:"toTimelineId,omitempty"`
}

// ResolvedCallout represents a resolved compact single-line callout card.
type ResolvedCallout struct {
	ID            string           `json:"id"`
	Label         string           `json:"label"`
	Placement     CalloutPlacement `json:"placement"`
	Dashed        bool             `json:"dashed,omitempty"`
	PaletteID     string           `json:"paletteId"`
	TimeMs        int64            `json:"timeMs"`
	TimelineID    uint32           `json:"timelineId,omitempty"`
	LogID         uint32           `json:"logId,omitempty"`
	RevisionIndex *int             `json:"revisionIndex,omitempty"`
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
	TimezoneShiftHours     float64   `json:"timezoneShiftHours,omitempty"`
	Overlay                *Overlay  `json:"overlay,omitempty"`
}

func isAnnotationsEmpty(a *Annotations) bool {
	if a == nil {
		return true
	}
	return len(a.ClearLayers) == 0 &&
		len(a.RemoveIDs) == 0 &&
		len(a.Palette) == 0 &&
		len(a.Markers) == 0 &&
		len(a.Spans) == 0 &&
		len(a.RowHighlights) == 0 &&
		a.DimUnlistedRows == nil &&
		len(a.DiffExcerpts) == 0 &&
		len(a.LogExcerpts) == 0 &&
		len(a.Connectors) == 0 &&
		len(a.Callouts) == 0
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

func clonePtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func cloneAnchor(a Anchor) Anchor {
	a.RevisionIndex = clonePtr(a.RevisionIndex)
	a.Time = clonePtr(a.Time)
	return a
}

func cloneAnnotations(a *Annotations) *Annotations {
	if a == nil {
		return nil
	}
	out := *a
	out.ClearLayers = slices.Clone(a.ClearLayers)
	out.RemoveIDs = slices.Clone(a.RemoveIDs)
	out.Palette = slices.Clone(a.Palette)
	out.DimUnlistedRows = clonePtr(a.DimUnlistedRows)
	out.Markers = slices.Clone(a.Markers)
	for i := range out.Markers {
		out.Markers[i].Anchor = cloneAnchor(out.Markers[i].Anchor)
	}
	out.Spans = slices.Clone(a.Spans)
	for i := range out.Spans {
		out.Spans[i].Start = cloneAnchor(out.Spans[i].Start)
		out.Spans[i].End = cloneAnchor(out.Spans[i].End)
		out.Spans[i].TimelineIDs = slices.Clone(out.Spans[i].TimelineIDs)
		out.Spans[i].IncludeSubtree = clonePtr(out.Spans[i].IncludeSubtree)
	}
	out.RowHighlights = slices.Clone(a.RowHighlights)
	for i := range out.RowHighlights {
		out.RowHighlights[i].TimelineIDs = slices.Clone(out.RowHighlights[i].TimelineIDs)
		out.RowHighlights[i].IncludeSubtree = clonePtr(out.RowHighlights[i].IncludeSubtree)
	}
	out.DiffExcerpts = slices.Clone(a.DiffExcerpts)
	for i := range out.DiffExcerpts {
		out.DiffExcerpts[i].Windows = slices.Clone(out.DiffExcerpts[i].Windows)
	}
	out.LogExcerpts = slices.Clone(a.LogExcerpts)
	for i := range out.LogExcerpts {
		out.LogExcerpts[i].Windows = slices.Clone(out.LogExcerpts[i].Windows)
	}
	out.Connectors = slices.Clone(a.Connectors)
	for i := range out.Connectors {
		out.Connectors[i].From = cloneAnchor(out.Connectors[i].From)
		out.Connectors[i].To = cloneAnchor(out.Connectors[i].To)
	}
	out.Callouts = slices.Clone(a.Callouts)
	for i := range out.Callouts {
		out.Callouts[i].Anchor = cloneAnchor(out.Callouts[i].Anchor)
	}
	return &out
}
