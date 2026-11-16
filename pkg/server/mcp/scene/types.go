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

const (
	// MarkerStylePin renders a solid stem anchored to a target row circle.
	MarkerStylePin MarkerStyle = "pin"
	// MarkerStyleReference renders a dash-dot full-height external reference line.
	MarkerStyleReference MarkerStyle = "reference"
)

const (
	// SpanPatternSolid renders a solid translucent fill band.
	SpanPatternSolid SpanPattern = "solid"
	// SpanPatternHatched renders diagonal stripes with a dashed border for missing logs.
	SpanPatternHatched SpanPattern = "hatched"
	// SpanPatternOutline renders an unfilled border box.
	SpanPatternOutline SpanPattern = "outline"
)

const (
	// ConnectorStyleCurve renders a directed causal Bezier curve arrow.
	ConnectorStyleCurve ConnectorStyle = "curve"
	// ConnectorStyleStep renders a directed orthogonal elbow arrow.
	ConnectorStyleStep ConnectorStyle = "step"
	// ConnectorStyleCaliper renders a horizontal duration caliper line.
	ConnectorStyleCaliper ConnectorStyle = "caliper"
)

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

const (
	// RevisionEdgeStart selects the revision's ChangedTime.
	RevisionEdgeStart RevisionEdge = "start"
	// RevisionEdgeEnd selects the next revision's ChangedTime.
	RevisionEdgeEnd RevisionEdge = "end"
)

const (
	// ExcerptKindDiff identifies a resource manifest diff excerpt card.
	ExcerptKindDiff ExcerptKind = "diff"
	// ExcerptKindLog identifies a structured log excerpt card.
	ExcerptKindLog ExcerptKind = "log"
)

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

// AnnotationLayer identifies a named overlay layer in Annotations.ClearLayers.
type AnnotationLayer string

// MarkerStyle defines the visual style of a vertical marker annotation.
type MarkerStyle string

// SpanPattern defines the fill pattern of a time-range span annotation.
type SpanPattern string

// ConnectorStyle defines the routing geometry of a connector annotation.
type ConnectorStyle string

// CalloutPlacement defines the preferred side of an inline callout card relative to its anchor.
type CalloutPlacement string

// RevisionEdge selects either the start or end timestamp of a resource revision interval.
type RevisionEdge string

// ExcerptKind classifies an evidence deck card as a resource diff or structured log excerpt.
type ExcerptKind string

// ExcerptLineKind classifies a rendered row inside a diff or log excerpt card.
type ExcerptLineKind string

// PaletteEntry binds a reusable semantic role identifier to a curated preset name or #RRGGBB hex color.
type PaletteEntry struct {
	// ID is the semantic role identifier referenced by annotations via PaletteID.
	ID string `json:"id" jsonschema:"Palette entry identifier referenced via paletteId on annotations (e.g. 'trigger', 'victim', 'impact', 'recovery')."`
	// Color is a preset color name or a 6-digit #RRGGBB hex string.
	Color string `json:"color" jsonschema:"Preset color name ('blue', 'amber', 'red', 'green', 'purple', 'teal', 'indigo', 'pink', 'brown', 'slate') or 6-digit hex '#RRGGBB'."`
}

// Annotations groups declarative visual overlays and patch directives for a rendered timeline scene.
type Annotations struct {
	// ClearLayers lists annotation layers to clear from the parent scene before applying additions.
	ClearLayers []AnnotationLayer `json:"clearLayers,omitempty" jsonschema:"Optional layer names to clear from parentSceneId before adding items: 'palette', 'markers', 'spans', 'rowHighlights', 'diffExcerpts', 'logExcerpts', 'connectors', 'callouts', or 'all'."`
	// RemoveIDs lists badges, labels, or palette entry IDs to remove from the parent scene.
	RemoveIDs []string `json:"removeIds,omitempty" jsonschema:"Optional badges, labels, or palette entry IDs (e.g. 'E2', 'ALERT', 'trigger') to remove from parentSceneId."`
	// Palette defines named colors shared across annotations and evidence deck cards.
	Palette []PaletteEntry `json:"palette,omitempty" jsonschema:"Named color definitions shared across markers, spans, rowHighlights, connectors, callouts, and evidence deck cards via paletteId (max 16)."`
	// Markers defines vertical indicator lines and top-band callout pills.
	Markers []MarkerAnnotation `json:"markers,omitempty" jsonschema:"Vertical indicator lines with auto-staggered callout pills in the top annotation band (max 12)."`
	// Spans defines full-height or row-scoped time-range bands.
	Spans []SpanAnnotation `json:"spans,omitempty" jsonschema:"Time-range bands across the full chart or scoped via timelineQuery/timelineIds, including hatched missing-log gaps (max 6)."`
	// RowHighlights defines focal resource subtree highlights and left-gutter role badges.
	RowHighlights []RowHighlight `json:"rowHighlights,omitempty" jsonschema:"Highlights focal resource subtrees via timelineQuery or timelineIds with left-gutter role badges."`
	// DimUnlistedRows dims background rows outside RowHighlights when true.
	DimUnlistedRows *bool `json:"dimUnlistedRows,omitempty" jsonschema:"Softly dims background rows outside rowHighlights when true."`
	// DiffExcerpts defines resource YAML diff cards in the bottom evidence deck.
	DiffExcerpts []DiffExcerpt `json:"diffExcerpts,omitempty" jsonschema:"Resource YAML diff cards anchored to a revision and rendered in the bottom evidence deck (max 6 total excerpts)."`
	// LogExcerpts defines structured log excerpt cards in the bottom evidence deck.
	LogExcerpts []LogExcerpt `json:"logExcerpts,omitempty" jsonschema:"Structured log excerpt cards anchored to a logId and rendered in the bottom evidence deck (max 6 total excerpts)."`
	// Connectors defines directed causal arrows or duration calipers between two anchors.
	Connectors []ConnectorAnnotation `json:"connectors,omitempty" jsonschema:"Directed causal arrows across rows or horizontal delta-time calipers between two anchors (max 8)."`
	// Callouts defines compact inline callout cards pinned to row events or revisions.
	Callouts []CalloutAnnotation `json:"callouts,omitempty" jsonschema:"Compact 1-line inline callout cards pinned to a row event or revision (max 8)."`
}

// SliceWindow specifies a literal search spot and surrounding context lines to retain.
type SliceWindow struct {
	// Query is the case-sensitive literal substring to match within the diff or log.
	Query string `json:"query" jsonschema:"Case-sensitive literal substring to match within the target diff or log."`
	// BeforeLines is the number of context lines retained before each match.
	BeforeLines int `json:"beforeLines" jsonschema:"Number of context lines to retain before each match (0..5)."`
	// AfterLines is the number of context lines retained after each match.
	AfterLines int `json:"afterLines" jsonschema:"Number of context lines to retain after each match (0..5)."`
	// MaxMatches caps the number of occurrences sliced for this window.
	MaxMatches int `json:"maxMatches,omitempty" jsonschema:"Maximum occurrences of this query to slice within the target diff or log (default 2, max 4)."`
}

// RevisionRef identifies a resource revision transition by timeline ID and 0-based revision index.
type RevisionRef struct {
	// TimelineID is the target resource timeline ID.
	TimelineID uint32 `json:"timelineId" jsonschema:"Target resource timeline ID from search_timelines."`
	// RevisionIndex is the 0-based revision index on the timeline.
	RevisionIndex int `json:"revisionIndex" jsonschema:"0-based revision index from get_resource_revisions."`
}

// DiffExcerpt slices changed hunks from a resource revision transition into a bottom deck card.
type DiffExcerpt struct {
	// Revision identifies the target resource timeline and revision index.
	Revision RevisionRef `json:"revision" jsonschema:"Target resource timelineId and 0-based revisionIndex from get_resource_revisions."`
	// Badge is an optional short pin badge, defaulting to sequential E1..En labels.
	Badge string `json:"badge,omitempty" jsonschema:"Optional short pin badge such as 'E1'. Defaults to 'E1'..'En' ordered by timestamp."`
	// Windows specifies search spots within the unified diff.
	Windows []SliceWindow `json:"windows,omitempty" jsonschema:"Search spots within the managedFields-stripped unified diff. Disjoint matches are joined with 'N lines omitted' bands."`
	// Note is an optional one-line summary rendered in the card footer.
	Note string `json:"note,omitempty" jsonschema:"Optional 1-line human summary rendered in the card footer, visually separated from verbatim diff lines (max 80 chars)."`
	// PaletteID references a named entry in Annotations.Palette.
	PaletteID string `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Inherits automatically from a co-anchored item or enclosing rowHighlight when omitted."`
}

// LogExcerpt slices structured YAML or multiline payload lines from a log entry into a bottom deck card.
type LogExcerpt struct {
	// LogID is the target log ID.
	LogID uint32 `json:"logId" jsonschema:"Target log ID from get_timeline_logs or search_logs."`
	// TimelineID optionally selects the associated timeline when a log is linked to multiple timelines.
	TimelineID uint32 `json:"timelineId,omitempty" jsonschema:"Optional timeline ID when a single log entry is linked to multiple timelines."`
	// Badge is an optional short pin badge, defaulting to sequential E1..En labels.
	Badge string `json:"badge,omitempty" jsonschema:"Optional short pin badge such as 'E2'. Defaults to 'E1'..'En' ordered by timestamp."`
	// Windows specifies search spots across the expanded structured log YAML.
	Windows []SliceWindow `json:"windows,omitempty" jsonschema:"Search spots across line-expanded structured log YAML. The parser-distilled Summary is always included."`
	// Note is an optional one-line summary rendered in the card footer.
	Note string `json:"note,omitempty" jsonschema:"Optional 1-line human summary rendered in the card footer, visually separated from verbatim log lines (max 80 chars)."`
	// PaletteID references a named entry in Annotations.Palette.
	PaletteID string `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Inherits automatically from a co-anchored item or enclosing rowHighlight when omitted."`
}

// MarkerAnnotation defines a vertical event pin or global reference line.
type MarkerAnnotation struct {
	// Anchor pins the marker to a log event, resource revision, or fallback timestamp.
	Anchor Anchor `json:"anchor" jsonschema:"Target timeline event (timelineId + logId), revision (timelineId + revisionIndex), or fallback RFC3339 time."`
	// Label is the concise callout text shown in the top annotation band.
	Label string `json:"label" jsonschema:"Concise callout text shown in the top annotation band (max 80 chars)."`
	// Badge is an optional short badge text, defaulting to a 1-based sequence number.
	Badge string `json:"badge,omitempty" jsonschema:"Short badge text such as '1', '2', 'ALERT', or 'TICKET'. Defaults to 1-based index."`
	// Style selects between a row-anchored pin and a full-height reference line.
	Style MarkerStyle `json:"style,omitempty" jsonschema:"'pin' (default: solid stem to target row circle) or 'reference' (dash-dot full-height external reference line)."`
	// PaletteID references a named entry in Annotations.Palette.
	PaletteID string `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Inherits automatically from a co-anchored item or enclosing rowHighlight when omitted."`
}

// SpanAnnotation defines a full-height or row-scoped time-range band.
type SpanAnnotation struct {
	// Start is the inclusive start anchor of the time-range band.
	Start Anchor `json:"start" jsonschema:"Inclusive start anchor of the interval."`
	// End is the inclusive end anchor of the time-range band.
	End Anchor `json:"end" jsonschema:"Inclusive end anchor of the interval."`
	// Label is the concise text displayed for the span band.
	Label string `json:"label" jsonschema:"Concise label for the span or region box (max 80 chars)."`
	// TimelineQuery optionally scopes the span vertically using a Timeline CEL query.
	TimelineQuery string `json:"timelineQuery,omitempty" jsonschema:"Optional Timeline CEL query scoping the box vertically to matching rows."`
	// TimelineIDs optionally scopes the span vertically to the specified timeline IDs.
	TimelineIDs []uint32 `json:"timelineIds,omitempty" jsonschema:"Optional timeline IDs scoping the box vertically. Omit both for a full-height chart span."`
	// Pattern selects the visual fill style of the span band.
	Pattern SpanPattern `json:"pattern,omitempty" jsonschema:"Fill style: 'solid' (default), 'hatched' (diagonal stripes + dashed border for missing logs), or 'outline'."`
	// IncludeSubtree controls whether descendants of matched timelines are included when scoped.
	IncludeSubtree *bool `json:"includeSubtree,omitempty" jsonschema:"Defaults to true when scoped to specific timelines."`
	// PaletteID references a named entry in Annotations.Palette.
	PaletteID string `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Inherits from an enclosing rowHighlight when scoped to rows, or uses built-in 'default' when omitted."`
}

// RowHighlight highlights resource subtrees selected by Timeline CEL or explicit timeline IDs.
type RowHighlight struct {
	// TimelineQuery is an optional Timeline CEL query selecting resource subtrees to highlight.
	TimelineQuery string `json:"timelineQuery,omitempty" jsonschema:"Optional Timeline CEL query selecting the resource subtree(s) to highlight."`
	// TimelineIDs lists explicit root timeline IDs of resource subtrees to highlight.
	TimelineIDs []uint32 `json:"timelineIds,omitempty" jsonschema:"Optional root timeline IDs of the resource subtree(s) to highlight."`
	// RoleLabel is a concise role badge rendered in the left gutter.
	RoleLabel string `json:"roleLabel,omitempty" jsonschema:"Concise role badge shown in the left margin (max 28 chars), e.g. 'TRIGGER · Node', 'VICTIM · Pod-1'."`
	// IncludeSubtree controls whether descendant rows under matched timelines are highlighted.
	IncludeSubtree *bool `json:"includeSubtree,omitempty" jsonschema:"Defaults to true, covering child rows under matched timelines."`
	// PaletteID references a named entry in Annotations.Palette.
	PaletteID string `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Uses built-in 'default' when omitted."`
}

// ConnectorAnnotation defines a directed causal arrow or horizontal duration caliper between two anchors.
type ConnectorAnnotation struct {
	// From is the origin anchor of the connector.
	From Anchor `json:"from" jsonschema:"Origin anchor (cause event/revision or interval start)."`
	// To is the destination anchor of the connector.
	To Anchor `json:"to" jsonschema:"Destination anchor (effect event/revision or interval end)."`
	// Label is an optional label shown on the connector pill, defaulting to elapsed duration.
	Label string `json:"label,omitempty" jsonschema:"Optional concise label on the connector pill. Defaults to formatted elapsed delta."`
	// Badge is an optional step badge shown on causal arrows.
	Badge string `json:"badge,omitempty" jsonschema:"Optional step number badge on causal arrows such as '1', '2'."`
	// Style selects the connector geometry (curve, step, or caliper).
	Style ConnectorStyle `json:"style,omitempty" jsonschema:"'curve' (default causal Bezier arrow), 'step' (orthogonal elbow arrow), or 'caliper' (horizontal duration line)."`
	// PaletteID references a named entry in Annotations.Palette.
	PaletteID string `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Inherits from an enclosing rowHighlight when omitted."`
}

// CalloutAnnotation places a compact single-line card near the target point on the chart body.
type CalloutAnnotation struct {
	// Anchor pins the callout leader line to a timeline event, revision, or fallback timestamp.
	Anchor Anchor `json:"anchor" jsonschema:"Target timeline event (timelineId + logId), revision (timelineId + revisionIndex), or fallback (time + timelineId) to point the leader line at."`
	// Label is the concise text rendered inside the callout card.
	Label string `json:"label" jsonschema:"Concise annotation text shown inside the card (max 80 chars)."`
	// Placement specifies the preferred card placement relative to Anchor.
	Placement CalloutPlacement `json:"placement,omitempty" jsonschema:"Preferred side relative to anchor: 'auto' (default), 'left', 'right', 'above', or 'below'."`
	// Dashed renders a dashed border and leader line when true.
	Dashed bool `json:"dashed,omitempty" jsonschema:"Renders a dashed border and leader line for secondary/non-causal context when true."`
	// PaletteID references a named entry in Annotations.Palette.
	PaletteID string `json:"paletteId,omitempty" jsonschema:"Optional palette entry ID from annotations.palette. Inherits automatically from a co-anchored item or enclosing rowHighlight when omitted."`
}

// Anchor pins an annotation to a timeline event (TimelineID + LogID), a resource revision (TimelineID + RevisionIndex), or a fallback timestamp.
type Anchor struct {
	// TimelineID is the target timeline ID, required for revisionIndex and optional for logId or time.
	TimelineID uint32 `json:"timelineId,omitempty" jsonschema:"Target timeline ID from search_timelines or search_logs. Required when specifying logId or revisionIndex; optional when attaching a row to time."`
	// LogID anchors to a specific log entry timestamp.
	LogID uint32 `json:"logId,omitempty" jsonschema:"Log ID on timelineId from get_timeline_logs or search_logs. Preferred for event anchors to avoid timestamp hallucination."`
	// RevisionIndex anchors to a 0-based resource revision on TimelineID.
	RevisionIndex *int `json:"revisionIndex,omitempty" jsonschema:"0-based revision index on timelineId from get_resource_revisions. Preferred for revision anchors to avoid timestamp hallucination."`
	// RevisionEdge selects the start or end timestamp of the revision specified by RevisionIndex.
	RevisionEdge RevisionEdge `json:"revisionEdge,omitempty" jsonschema:"Revision boundary when revisionIndex is set: 'start' (default, revision ChangedTime) or 'end' (next revision ChangedTime)."`
	// Time provides a fallback RFC3339 timestamp when no log or revision anchor applies.
	Time *time.Time `json:"time,omitempty" jsonschema:"Fallback RFC3339 timestamp only when no log or revision exists on the timeline. Prefer timelineId + logId or timelineId + revisionIndex whenever possible to reduce timestamp hallucinations."`
}

// Spec captures the canonical, immutable scene parameters so child scenes can inherit and patch them.
type Spec struct {
	// InspectionID is the unique identifier of the target inspection.
	InspectionID string
	// Filter holds the resolved timeline and log filter parameters.
	Filter workbench.Filter
	// TimelineIDs lists explicit timeline IDs to include in the rendered scene.
	TimelineIDs []uint32
	// MaxDepth limits the rendered timeline tree depth.
	MaxDepth int
	// MaxNodes caps the number of rendered timeline rows.
	MaxNodes int
	// TimezoneShiftHours specifies the UTC offset in hours for the timeline ruler.
	TimezoneShiftHours float64
	// Annotations holds the declarative visual overlay and evidence card specifications.
	Annotations *Annotations
}

// LinePart represents a contiguous substring of an excerpt line and whether it matched a query.
type LinePart struct {
	// Text is the substring content of this line segment.
	Text string `json:"text"`
	// Matched reports whether this segment matched a slice window query.
	Matched bool `json:"matched,omitempty"`
}

// ExcerptLine represents one display row in a diff or log excerpt card.
type ExcerptLine struct {
	// Kind classifies the row as context, added, deleted, summary, or omitted.
	Kind ExcerptLineKind `json:"kind"`
	// OldLineNum is the 1-based line number in the previous revision, when applicable.
	OldLineNum int `json:"oldLineNum,omitempty"`
	// NewLineNum is the 1-based line number in the target revision or log body, when applicable.
	NewLineNum int `json:"newLineNum,omitempty"`
	// Text is the full text of the line.
	Text string `json:"text"`
	// Parts splits Text into highlighted and non-highlighted segments when a query matched.
	Parts []LinePart `json:"parts,omitempty"`
	// Omitted is the number of folded lines when Kind is ExcerptLineKindOmitted.
	Omitted int `json:"omitted,omitempty"`
}

// ResolvedPaletteColors holds derived foreground, badge text, soft fill, and surface tint CSS colors for a palette entry.
type ResolvedPaletteColors struct {
	// ID is the palette entry identifier.
	ID string `json:"id"`
	// Main is the primary #RRGGBB foreground and accent color.
	Main string `json:"main"`
	// OnMain is the high-contrast text color (#fafafa or #212121) placed over Main.
	OnMain string `json:"onMain"`
	// Soft is a translucent rgba fill derived from Main.
	Soft string `json:"soft"`
	// Surface is a light-tinted #RRGGBB surface background derived from Main.
	Surface string `json:"surface"`
}

// Overlay holds the resolved coordinates, matched timeline IDs, and sliced evidence cards for frontend rendering.
type Overlay struct {
	// AutoIncludeTimelineIDs lists timeline IDs referenced by annotations to include automatically.
	AutoIncludeTimelineIDs []uint32 `json:"autoIncludeTimelineIds,omitempty"`
	// Palette holds the resolved CSS color entries used by annotations in this overlay.
	Palette []ResolvedPaletteColors `json:"palette,omitempty"`
	// Markers holds the resolved vertical marker annotations.
	Markers []ResolvedMarker `json:"markers,omitempty"`
	// Spans holds the resolved time-range span annotations.
	Spans []ResolvedSpan `json:"spans,omitempty"`
	// RowHighlights holds the resolved row subtree highlights.
	RowHighlights []ResolvedRowHighlight `json:"rowHighlights,omitempty"`
	// DimUnlistedRows dims background rows outside RowHighlights when true.
	DimUnlistedRows bool `json:"dimUnlistedRows,omitempty"`
	// Excerpts holds the resolved diff and log excerpt cards sorted chronologically.
	Excerpts []ResolvedExcerptCard `json:"excerpts,omitempty"`
	// Connectors holds the resolved causal arrows and duration calipers.
	Connectors []ResolvedConnector `json:"connectors,omitempty"`
	// Callouts holds the resolved inline callout cards.
	Callouts []ResolvedCallout `json:"callouts,omitempty"`
}

// ResolvedMarker represents a resolved vertical event pin or global reference line.
type ResolvedMarker struct {
	// ID is the generated marker element identifier.
	ID string `json:"id"`
	// Badge is the short badge text rendered on the marker pill.
	Badge string `json:"badge"`
	// Label is the callout label displayed in the top annotation band.
	Label string `json:"label"`
	// Style is the resolved marker style (pin or reference).
	Style MarkerStyle `json:"style"`
	// PaletteID is the resolved palette entry ID coloring this marker.
	PaletteID string `json:"paletteId"`
	// TimeMs is the resolved marker timestamp in Unix milliseconds.
	TimeMs int64 `json:"timeMs"`
	// TimeLabel is the formatted clock string rendered on the marker pill.
	TimeLabel string `json:"timeLabel"`
	// TimelineID is the associated timeline ID, if any.
	TimelineID uint32 `json:"timelineId,omitempty"`
	// LogID is the anchored log ID, if any.
	LogID uint32 `json:"logId,omitempty"`
	// RevisionIndex is the anchored 0-based revision index, if any.
	RevisionIndex *int `json:"revisionIndex,omitempty"`
}

// ResolvedSpan represents a resolved full-height or row-scoped interval band.
type ResolvedSpan struct {
	// ID is the generated span element identifier.
	ID string `json:"id"`
	// Label is the text label displayed for the span band.
	Label string `json:"label"`
	// StartTimeMs is the inclusive start timestamp in Unix milliseconds.
	StartTimeMs int64 `json:"startTimeMs"`
	// EndTimeMs is the inclusive end timestamp in Unix milliseconds.
	EndTimeMs int64 `json:"endTimeMs"`
	// DurationLabel is the formatted elapsed duration between StartTimeMs and EndTimeMs.
	DurationLabel string `json:"durationLabel"`
	// TimelineIDs restricts the span vertically to these timelines when non-empty.
	TimelineIDs []uint32 `json:"timelineIds,omitempty"`
	// Pattern is the resolved fill pattern of the span band.
	Pattern SpanPattern `json:"pattern"`
	// PaletteID is the resolved palette entry ID coloring this span.
	PaletteID string `json:"paletteId"`
}

// ResolvedRowHighlight represents a resolved focal subtree highlight and left-gutter role badge.
type ResolvedRowHighlight struct {
	// ID is the generated row highlight identifier.
	ID string `json:"id"`
	// RoleLabel is the optional role badge shown in the left margin.
	RoleLabel string `json:"roleLabel,omitempty"`
	// TimelineIDs lists the resolved timeline IDs included in the highlight.
	TimelineIDs []uint32 `json:"timelineIds"`
	// PaletteID is the resolved palette entry ID coloring this highlight.
	PaletteID string `json:"paletteId"`
}

// ResolvedExcerptCard represents a resolved diff card or log card in the bottom evidence deck.
type ResolvedExcerptCard struct {
	// ID is the generated excerpt card identifier.
	ID string `json:"id"`
	// Kind classifies the card as a diff or log excerpt.
	Kind ExcerptKind `json:"kind"`
	// Badge is the short pin badge connecting the card to chart markers.
	Badge string `json:"badge"`
	// PaletteID is the resolved palette entry ID coloring this card.
	PaletteID string `json:"paletteId"`
	// TimeMs is the card anchor timestamp in Unix milliseconds.
	TimeMs int64 `json:"timeMs"`
	// TimeLabel is the formatted clock string rendered in the card header.
	TimeLabel string `json:"timeLabel"`
	// TimelineID is the associated timeline ID.
	TimelineID uint32 `json:"timelineId"`
	// ResourcePath is the compacted resource path displayed in the card header.
	ResourcePath string `json:"resourcePath"`
	// Title is the primary header title of the excerpt card.
	Title string `json:"title"`
	// Subtitle summarizes line or diff change counts for the excerpt card.
	Subtitle string `json:"subtitle"`
	// Lines holds the sliced diff or log lines rendered in the card body.
	Lines []ExcerptLine `json:"lines"`
	// Note is the optional footer summary note.
	Note string `json:"note,omitempty"`
	// LogTimestamp is the RFC3339Nano timestamp used for Cloud Logging deep links.
	LogTimestamp string `json:"logTimestamp,omitempty"`
	// InsertID is the Cloud Logging insertId used for deep links.
	InsertID string `json:"insertId,omitempty"`
	// LogID is the associated log ID, if any.
	LogID uint32 `json:"logId,omitempty"`
	// RevisionIndex is the anchored 0-based revision index, if any.
	RevisionIndex *int `json:"revisionIndex,omitempty"`
}

// ResolvedConnector represents a resolved causal arrow or horizontal duration caliper.
type ResolvedConnector struct {
	// ID is the generated connector element identifier.
	ID string `json:"id"`
	// Badge is the optional step badge displayed on the connector.
	Badge string `json:"badge,omitempty"`
	// Label is the label rendered on the connector pill.
	Label string `json:"label,omitempty"`
	// Style is the resolved connector routing style.
	Style ConnectorStyle `json:"style"`
	// PaletteID is the resolved palette entry ID coloring this connector.
	PaletteID string `json:"paletteId"`
	// FromTimeMs is the origin timestamp in Unix milliseconds.
	FromTimeMs int64 `json:"fromTimeMs"`
	// FromTimelineID is the origin timeline ID, if any.
	FromTimelineID uint32 `json:"fromTimelineId,omitempty"`
	// ToTimeMs is the destination timestamp in Unix milliseconds.
	ToTimeMs int64 `json:"toTimeMs"`
	// ToTimelineID is the destination timeline ID, if any.
	ToTimelineID uint32 `json:"toTimelineId,omitempty"`
}

// ResolvedCallout represents a resolved compact single-line callout card.
type ResolvedCallout struct {
	// ID is the generated callout element identifier.
	ID string `json:"id"`
	// Label is the text rendered inside the callout card.
	Label string `json:"label"`
	// Placement is the resolved preferred side relative to the anchor.
	Placement CalloutPlacement `json:"placement"`
	// Dashed renders a dashed border and leader line when true.
	Dashed bool `json:"dashed,omitempty"`
	// PaletteID is the resolved palette entry ID coloring this callout.
	PaletteID string `json:"paletteId"`
	// TimeMs is the anchor timestamp in Unix milliseconds.
	TimeMs int64 `json:"timeMs"`
	// TimelineID is the anchored timeline ID, if any.
	TimelineID uint32 `json:"timelineId,omitempty"`
	// LogID is the anchored log ID, if any.
	LogID uint32 `json:"logId,omitempty"`
	// RevisionIndex is the anchored 0-based revision index, if any.
	RevisionIndex *int `json:"revisionIndex,omitempty"`
}

// Document is the self-contained JSON payload served at /api/v1/mcp-scenes/:sceneId for interactive viewing.
type Document struct {
	// SceneID is the unique identifier of this scene document.
	SceneID string `json:"sceneId"`
	// ParentSceneID is the parent scene ID from which this scene was derived, if any.
	ParentSceneID string `json:"parentSceneId,omitempty"`
	// ExpiresAt is the UTC timestamp after which this scene document expires.
	ExpiresAt time.Time `json:"expiresAt"`
	// InspectionID is the target inspection ID.
	InspectionID string `json:"inspectionId"`
	// TimelineQuery is the Timeline CEL filter expression.
	TimelineQuery string `json:"timelineQuery,omitempty"`
	// TimelineExclusionQuery is the Timeline CEL exclusion filter expression.
	TimelineExclusionQuery string `json:"timelineExclusionQuery,omitempty"`
	// LogQuery is the Log CEL filter expression.
	LogQuery string `json:"logQuery,omitempty"`
	// ExcludeNoLogs controls whether timelines without matching logs are omitted.
	ExcludeNoLogs bool `json:"excludeNoLogs,omitempty"`
	// FilterStartTimeNs is the inclusive filter start timestamp in Unix nanoseconds as a string.
	FilterStartTimeNs string `json:"filterStartTimeNs,omitempty"`
	// FilterEndTimeNs is the inclusive filter end timestamp in Unix nanoseconds as a string.
	FilterEndTimeNs string `json:"filterEndTimeNs,omitempty"`
	// ViewStartTimeMs is the optional viewport start timestamp in Unix milliseconds.
	ViewStartTimeMs *int64 `json:"viewStartTimeMs,omitempty"`
	// ViewEndTimeMs is the optional viewport end timestamp in Unix milliseconds.
	ViewEndTimeMs *int64 `json:"viewEndTimeMs,omitempty"`
	// TimelineIDs restricts rendering to these timelines and their ancestors when non-empty.
	TimelineIDs []uint32 `json:"timelineIds,omitempty"`
	// MaxDepth limits the rendered timeline tree depth.
	MaxDepth int `json:"maxDepth,omitempty"`
	// MaxNodes caps the number of rendered timeline rows.
	MaxNodes int `json:"maxNodes,omitempty"`
	// TimezoneShiftHours specifies the UTC offset in hours for the timeline ruler.
	TimezoneShiftHours float64 `json:"timezoneShiftHours,omitempty"`
	// Overlay holds the resolved visual annotations and evidence deck cards.
	Overlay *Overlay `json:"overlay,omitempty"`
}

// Clone returns a deep copy of Spec so caller mutations never alter stored scene history.
func (s Spec) Clone() Spec {
	cloned := s
	cloned.Filter = cloneFilter(s.Filter)
	cloned.TimelineIDs = slices.Clone(s.TimelineIDs)
	cloned.Annotations = cloneAnnotations(s.Annotations)
	return cloned
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
