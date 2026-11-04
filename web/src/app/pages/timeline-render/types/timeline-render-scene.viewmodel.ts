/**
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import {
  TimelineCalloutPlacement,
  TimelineConnectorStyle,
  TimelineExcerptKind,
  TimelineExcerptLine,
  TimelineExcerptLineKind,
  TimelineExcerptLinePart,
  TimelineMarkerStyle,
  TimelineResolvedCallout,
  TimelineResolvedConnector,
  TimelineResolvedExcerptCard,
  TimelineResolvedMarker,
  TimelineResolvedPaletteColors,
  TimelineResolvedRowHighlight,
  TimelineResolvedSpan,
  TimelineSceneOverlay,
  TimelineSpanPattern,
} from 'src/app/common/schema/timeline-render-types';

/**
 * Layout preset identifier for the bottom evidence deck CSS grid.
 */
export enum ExcerptDeckGridPreset {
  EMPTY = 'empty',
  COLS_1 = 'cols-1',
  COLS_2 = 'cols-2',
  COLS_3 = 'cols-3',
  COLS_4 = 'cols-4',
  GRID_2X2 = 'grid-2x2',
  GRID_3X2 = 'grid-3x2',
}

/**
 * Target resource timeline, revision, or log entry to focus when opening a full KHI session.
 */
export interface TimelineSessionJumpTarget {
  /** Target timeline domain ID when navigating to a specific resource row. */
  readonly timelineId?: number;
  /** Target revision index within the resource timeline when focusing a state diff. */
  readonly revisionIndex?: number;
  /** Target log entry ID when focusing a specific log line in KHI. */
  readonly logId?: number;
}

/**
 * Minimal height metric for a single rendered timeline row.
 */
export interface TimelineRowMetric {
  /** Unique numeric timeline domain ID for the rendered row. */
  readonly id: number;
  /** Rendered vertical row height in CSS pixels. */
  readonly heightPx: number;
}

/**
 * Input geometry and overlay payload used to compute the rendered scene ViewModel.
 */
export interface TimelineSceneLayoutInput {
  /** Fitted width of the left resource hierarchy index pane in CSS pixels. */
  readonly indexPaneWidthPx: number;
  /** Inclusive left edge timestamp of the chart viewport in epoch milliseconds. */
  readonly viewportLeftTimeMs: number;
  /** Horizontal chart scale in CSS pixels per millisecond. */
  readonly pixelsPerMs: number;
  /** Rendered timeline row height metrics ordered from top to bottom. */
  readonly rows: readonly TimelineRowMetric[];
  /** Resolved visual overlay payload, or null when rendering an unannotated timeline. */
  readonly overlay: TimelineSceneOverlay | null;
}

/**
 * Positioned vertical marker pill, stem, and optional target dot for timeline markers.
 */
export interface MarkerItemViewModel {
  /** Unique scene identifier for the marker annotation. */
  readonly id: string;
  /** Short badge token displayed at the leading edge of the marker pill. */
  readonly badge: string;
  /** Human-readable annotation label inside the marker pill. */
  readonly label: string;
  /** Formatted timestamp string displayed inside the marker pill. */
  readonly timeLabel: string;
  /** Visual marker presentation style controlling stem termination. */
  readonly style: TimelineMarkerStyle;
  /** Palette identifier referenced by this marker. */
  readonly paletteId: string;
  /** Resolved semantic foreground and surface colors for rendering the marker. */
  readonly colors: TimelineResolvedPaletteColors;
  /** Left X coordinate of the marker pill in stage CSS pixels. */
  readonly pillLeftPx: number;
  /** Top Y coordinate of the marker pill in stage CSS pixels. */
  readonly pillTopPx: number;
  /** Horizontal width of the marker pill in CSS pixels. */
  readonly pillWidthPx: number;
  /** Vertical height of the marker pill in CSS pixels. */
  readonly pillHeightPx: number;
  /** Horizontal X coordinate of the vertical marker stem in stage CSS pixels. */
  readonly stemX: number;
  /** Top Y coordinate where the vertical marker stem begins in stage CSS pixels. */
  readonly stemTopPx: number;
  /** Vertical length of the marker stem extending down into the chart in CSS pixels. */
  readonly stemHeightPx: number;
  /** True when a target dot should be rendered at the row intersection point. */
  readonly hasTargetDot: boolean;
  /** Vertical Y coordinate of the target dot in stage CSS pixels. */
  readonly dotY: number;
  /** Optional session jump target when clicking the marker in interactive mode. */
  readonly jumpTarget: TimelineSessionJumpTarget | null;
}

/**
 * Positioned full-height or row-scoped time-range band for timeline spans.
 */
export interface SpanItemViewModel {
  /** Unique scene identifier for the span annotation. */
  readonly id: string;
  /** Human-readable label displayed inside the span header chip. */
  readonly headerText: string;
  /** Visual fill pattern for the span body. */
  readonly pattern: TimelineSpanPattern;
  /** Palette identifier referenced by this span. */
  readonly paletteId: string;
  /** Resolved semantic stroke and surface colors for rendering the span. */
  readonly colors: TimelineResolvedPaletteColors;
  /** True when the span is bounded to specific timeline rows rather than full chart height. */
  readonly isRowScoped: boolean;
  /** True when corner bracket accents should be rendered on the span body. */
  readonly showCornerBrackets: boolean;
  /** True when a vertical marker stem coincides with the span's left boundary. */
  readonly coincidentLeft: boolean;
  /** True when a vertical marker stem coincides with the span's right boundary. */
  readonly coincidentRight: boolean;
  /** Horizontal offset of the header chip relative to span left in pixels. */
  readonly labelLeftPx: number;
  /** Left X coordinate of the header chip in stage CSS pixels. */
  readonly chipStageLeftPx: number;
  /** Top Y coordinate of the header chip in stage CSS pixels. */
  readonly chipStageTopPx: number;
  /** Maximum allowable width of the header chip in CSS pixels before truncation. */
  readonly chipMaxWidthPx: number;
  /** Left X coordinate of the span body in stage CSS pixels. */
  readonly leftPx: number;
  /** Top Y coordinate of the span body in stage CSS pixels. */
  readonly topPx: number;
  /** Horizontal width of the span body in CSS pixels. */
  readonly widthPx: number;
  /** Vertical height of the span body in CSS pixels. */
  readonly heightPx: number;
}

/**
 * Positioned focal subtree highlight band and left-gutter role badge for row highlights.
 */
export interface RowHighlightBandViewModel {
  /** Unique scene identifier for this highlight run. */
  readonly id: string;
  /** Formatted role label displayed in the left gutter, or empty string. */
  readonly roleLabel: string;
  /** Palette identifier referenced by this row highlight. */
  readonly paletteId: string;
  /** Resolved semantic background, border, and badge colors. */
  readonly colors: TimelineResolvedPaletteColors;
  /** Stage-relative top Y coordinate of the highlight band in CSS pixels. */
  readonly topPx: number;
  /** Vertical height of the highlight band covering contiguous rows in CSS pixels. */
  readonly heightPx: number;
  /** Stage-relative right X coordinate where the role badge chip ends in CSS pixels. */
  readonly badgeRightX: number;
}

/**
 * Positioned translucent mask over contiguous background rows when dimUnlistedRows is active.
 */
export interface DimmedRowMaskViewModel {
  /** Unique identifier for this contiguous dimmed background row mask. */
  readonly id: string;
  /** Stage-relative top Y coordinate of the dimmed row mask in CSS pixels. */
  readonly topPx: number;
  /** Vertical height of the dimmed row mask in CSS pixels. */
  readonly heightPx: number;
}

/**
 * Positioned SVG path, polygon arrowhead, and midpoint label chip for timeline connectors.
 */
export interface ConnectorItemViewModel {
  /** Unique scene identifier for the causal connector or duration caliper. */
  readonly id: string;
  /** Optional short badge token rendered in the midpoint connector chip. */
  readonly badge: string;
  /** Optional descriptive or delta-time label rendered at the connector midpoint. */
  readonly label: string;
  /** Geometric path style controlling curved, stepped, or caliper rendering. */
  readonly style: TimelineConnectorStyle;
  /** Palette identifier referenced by this connector. */
  readonly paletteId: string;
  /** Resolved semantic stroke and label colors for rendering the connector. */
  readonly colors: TimelineResolvedPaletteColors;
  /** SVG path data string tracing the connector curve, step, or caliper ticks. */
  readonly svgPath: string;
  /** True when a directional polygon arrowhead is rendered at the destination endpoint. */
  readonly hasArrowHead: boolean;
  /** Space-separated SVG polygon vertex coordinates for the arrowhead, or empty when omitted. */
  readonly arrowHeadPoints: string;
  /** Stage-relative center X coordinate of the midpoint label chip in CSS pixels. */
  readonly labelCenterX: number;
  /** Stage-relative center Y coordinate of the midpoint label chip in CSS pixels. */
  readonly labelCenterY: number;
}

/**
 * Positioned compact single-line inline callout card and leader line for timeline callouts.
 */
export interface CalloutItemViewModel {
  /** Unique scene identifier for the inline callout annotation. */
  readonly id: string;
  /** Single-line text label displayed inside the callout card. */
  readonly label: string;
  /** True when the leader line uses a dashed stroke style. */
  readonly dashed: boolean;
  /** Palette identifier referenced by this callout. */
  readonly paletteId: string;
  /** Resolved semantic border, stroke, and surface colors for the callout. */
  readonly colors: TimelineResolvedPaletteColors;
  /** Stage-relative X coordinate of the target timeline anchor point in CSS pixels. */
  readonly anchorX: number;
  /** Stage-relative Y coordinate of the target timeline anchor point in CSS pixels. */
  readonly anchorY: number;
  /** SVG path data string connecting the anchor dot to the callout card edge. */
  readonly leaderPath: string;
  /** Stage-relative left X coordinate of the callout card in CSS pixels. */
  readonly cardLeftPx: number;
  /** Stage-relative top Y coordinate of the callout card in CSS pixels. */
  readonly cardTopPx: number;
  /** Horizontal width of the callout card in CSS pixels. */
  readonly cardWidthPx: number;
  /** Vertical height of the callout card in CSS pixels. */
  readonly cardHeightPx: number;
  /** Optional session jump target when clicking the callout in interactive mode. */
  readonly jumpTarget: TimelineSessionJumpTarget | null;
}

/**
 * Positioned badge pin and vertical drop stem on the timeline canvas for evidence excerpts.
 */
export interface ExcerptPinViewModel {
  /** Unique scene identifier matching the paired bottom evidence deck card. */
  readonly id: string;
  /** Short badge token tying the timeline pin to its deck card and S-curve tether. */
  readonly badge: string;
  /** Evidence kind distinguishing resource manifest diffs from raw log excerpts. */
  readonly kind: TimelineExcerptKind;
  /** Palette identifier referenced by this excerpt pin. */
  readonly paletteId: string;
  /** Resolved semantic pin and drop-stem colors. */
  readonly colors: TimelineResolvedPaletteColors;
  /** Stage-relative center X coordinate of the timeline pin in CSS pixels. */
  readonly pinX: number;
  /** Stage-relative center Y coordinate of the timeline pin in CSS pixels. */
  readonly pinY: number;
  /** Stage-relative top Y coordinate where the vertical drop stem starts below the pin. */
  readonly stemTopPx: number;
  /** Vertical length of the drop stem reaching the bottom tether channel in CSS pixels. */
  readonly stemHeightPx: number;
  /** Session jump target pointing to the backing timeline revision or log entry. */
  readonly jumpTarget: TimelineSessionJumpTarget;
}

/**
 * Formatted display row inside an evidence deck card with normalized highlight parts.
 */
export interface ExcerptLineViewModel {
  /** Zero-based vertical line index within the card body. */
  readonly index: number;
  /** Visual diff kind or structured log category. */
  readonly kind: TimelineExcerptLineKind;
  /** Optional old line number for diff gutter rendering. */
  readonly gutterOld: string;
  /** Optional new line number for diff gutter rendering. */
  readonly gutterNew: string;
  /** Leading diff prefix symbol (+, -, or space). */
  readonly prefixSymbol: string;
  /** Raw line text without diff prefix. */
  readonly text: string;
  /** Highlight parts splitting text into matched and unmatched segments. */
  readonly parts: readonly TimelineExcerptLinePart[];
}

/**
 * ViewModel for a single bottom evidence deck diff or log card.
 */
export interface ExcerptCardViewModel {
  /** Unique scene identifier matching the source excerpt card payload. */
  readonly id: string;
  /** CSS grid area identifier assigned to this card. */
  readonly gridArea: string;
  /** Short badge token rendered on the card header matching its timeline pin. */
  readonly badge: string;
  /** Evidence kind distinguishing resource manifest diffs from raw log excerpts. */
  readonly kind: TimelineExcerptKind;
  /** Palette identifier referenced by this card. */
  readonly paletteId: string;
  /** Resolved semantic card border, header, and badge colors. */
  readonly colors: TimelineResolvedPaletteColors;
  /** Formatted timestamp string for the event. */
  readonly timeLabel: string;
  /** Full Kubernetes resource hierarchy path. */
  readonly resourcePath: string;
  /** Primary card title. */
  readonly title: string;
  /** Secondary subtitle or method name. */
  readonly subtitle: string;
  /** Formatted Cloud Logging timestamp when available, or empty string. */
  readonly logTimestamp: string;
  /** Cloud Logging insertId for deduplication when available, or empty string. */
  readonly insertId: string;
  /** True when log query metadata fields should be rendered in the card header. */
  readonly hasQueryMetadata: boolean;
  /** Formatted diff or log body rows. */
  readonly lines: readonly ExcerptLineViewModel[];
  /** Optional human-readable analyst note explaining the card's significance. */
  readonly note: string;
  /** Action button text ('Open Diff' or 'Open Log'). */
  readonly actionLabel: string;
  /** Session jump target pointing to the backing timeline revision or log entry. */
  readonly jumpTarget: TimelineSessionJumpTarget;
}

/**
 * S-curve cubic Bezier SVG tether connecting a timeline pin drop stem to its deck card column.
 */
export interface ExcerptTetherViewModel {
  /** Unique scene identifier for this tether curve. */
  readonly id: string;
  /** Short badge token matching the parent excerpt. */
  readonly badge: string;
  /** Palette identifier referenced by this tether. */
  readonly paletteId: string;
  /** Resolved semantic stroke color for rendering the tether S-curve. */
  readonly colors: TimelineResolvedPaletteColors;
  /** SVG path data string tracing the cubic Bezier S-curve from pin to card anchor. */
  readonly svgPath: string;
  /** Stage-relative X coordinate where the tether begins below the drop stem in CSS pixels. */
  readonly pinX: number;
  /** Stage-relative X coordinate where the tether meets the deck card column in CSS pixels. */
  readonly cardAnchorX: number;
}

/**
 * ViewModel for the bottom synchronized evidence deck and its 32px S-curve tether channel.
 */
export interface ExcerptDeckViewModel {
  /** Selected grid layout preset based on card count and available stage width. */
  readonly gridPreset: ExcerptDeckGridPreset;
  /** Number of equal-width columns in the CSS grid. */
  readonly columns: number;
  /** Height of the S-curve tether strip connecting timeline pins to deck cards in CSS pixels. */
  readonly tetherStripHeightPx: number;
  /** Estimated total vertical height of the tether strip plus card grid in CSS pixels. */
  readonly estimatedHeightPx: number;
  /** Positioned cubic Bezier tethers linking timeline pins to deck columns. */
  readonly tethers: readonly ExcerptTetherViewModel[];
  /** Formatted evidence diff and log card ViewModels in display order. */
  readonly cards: readonly ExcerptCardViewModel[];
}

/**
 * Complete layout ViewModel for the timeline stage overlay and bottom evidence deck.
 */
export interface TimelineSceneViewModel {
  /** Fitted width of the left resource hierarchy index pane in CSS pixels. */
  readonly indexPaneWidthPx: number;
  /** Left X coordinate where the chart viewport begins in CSS pixels. */
  readonly chartLeftPx: number;
  /** Width of the right timeline chart viewport in CSS pixels. */
  readonly chartViewportWidthPx: number;
  /** Total horizontal stage width in CSS pixels. */
  readonly stageWidthPx: number;
  /** Height reserved above the timeline frame for staggered marker pills in CSS pixels. */
  readonly topBandHeightPx: number;
  /** Height of the timeline frame including ruler header and rows in CSS pixels. */
  readonly timelineFrameHeightPx: number;
  /** Combined height of the top marker band and timeline frame in CSS pixels. */
  readonly stageHeightPx: number;
  /** Total vertical capture height of the rendered scene in CSS pixels. */
  readonly estimatedTotalHeightPx: number;
  /** True when at least one visible overlay annotation is rendered. */
  readonly hasOverlayItems: boolean;
  /** Positioned vertical marker pills and stems. */
  readonly markers: readonly MarkerItemViewModel[];
  /** Positioned time-range span bands and header chips. */
  readonly spans: readonly SpanItemViewModel[];
  /** Positioned focal subtree highlight bands and left-gutter role badges. */
  readonly rowHighlights: readonly RowHighlightBandViewModel[];
  /** Positioned translucent masks over contiguous unlisted background rows. */
  readonly dimmedRowMasks: readonly DimmedRowMaskViewModel[];
  /** Positioned causal connectors and horizontal duration calipers. */
  readonly connectors: readonly ConnectorItemViewModel[];
  /** Positioned single-line inline callout cards and leader lines. */
  readonly callouts: readonly CalloutItemViewModel[];
  /** Positioned timeline pins and vertical drop stems linked to evidence deck cards. */
  readonly excerptPins: readonly ExcerptPinViewModel[];
  /** Bottom synchronized evidence deck grid and S-curve tether channel. */
  readonly deck: ExcerptDeckViewModel;
}

/**
 * Clamped horizontal bounds of the timeline stage and chart viewport in CSS pixels.
 */
export interface SceneHorizontalBounds {
  /** Clamped width of the left resource index pane in CSS pixels. */
  readonly indexPaneWidthPx: number;
  /** Left X coordinate where the timeline chart viewport starts in CSS pixels. */
  readonly chartLeftPx: number;
  /** Width of the timeline chart viewport in CSS pixels. */
  readonly chartViewportWidthPx: number;
  /** Total horizontal stage width in CSS pixels. */
  readonly stageWidthPx: number;
}

/**
 * Visual markers and time-range spans that fall within the visible chart viewport.
 */
export interface VisibleTimelineAnnotations {
  /** Markers whose timestamp intersects the visible viewport time range. */
  readonly markers: readonly TimelineResolvedMarker[];
  /** Spans whose time range intersects the visible viewport time range. */
  readonly spans: readonly TimelineResolvedSpan[];
}

/**
 * Vertical bounds and center coordinate of a rendered timeline row within the stage.
 */
export interface TimelineRowStageBounds {
  /** Zero-based vertical display index of the timeline row. */
  readonly index: number;
  /** Domain identifier of the timeline row. */
  readonly id: number;
  /** Top Y coordinate of the row in stage CSS pixels. */
  readonly topPx: number;
  /** Bottom Y coordinate of the row in stage CSS pixels. */
  readonly bottomPx: number;
  /** Vertical center Y coordinate of the row in stage CSS pixels. */
  readonly centerPx: number;
  /** Vertical height of the row in stage CSS pixels. */
  readonly heightPx: number;
}

/**
 * Vertical stage dimensions and positioned row bounds computed from row metrics.
 */
export interface TimelineStageRowLayout {
  /** Height reserved above the timeline frame for marker pills in CSS pixels. */
  readonly topBandHeightPx: number;
  /** Height of the timeline frame including ruler header and rows in CSS pixels. */
  readonly timelineFrameHeightPx: number;
  /** Combined height of the top marker band and timeline frame in CSS pixels. */
  readonly stageHeightPx: number;
  /** Ordered list of positioned row stage bounds. */
  readonly rowBoundsList: readonly TimelineRowStageBounds[];
  /** Map of timeline domain ID to positioned row stage bounds. */
  readonly rowBoundsById: ReadonlyMap<number, TimelineRowStageBounds>;
}

/**
 * Intermediate horizontal placement and lane index for a marker pill before stem resolution.
 */
export interface PreliminaryMarkerPlacement {
  /** Source resolved marker payload. */
  readonly marker: TimelineResolvedMarker;
  /** Zero-based vertical marker band lane index. */
  readonly laneIndex: number;
  /** Left X coordinate of the marker pill in stage CSS pixels. */
  readonly pillLeftPx: number;
  /** Horizontal width of the marker pill in CSS pixels. */
  readonly pillWidthPx: number;
  /** Horizontal X coordinate of the vertical marker stem in stage CSS pixels. */
  readonly stemX: number;
}

/**
 * Result of packing marker pills into non-overlapping horizontal lanes.
 */
export interface PackedMarkerLanes {
  /** Total number of horizontal marker lanes required to avoid pill overlaps. */
  readonly laneCount: number;
  /** Positioned preliminary marker placements ordered by timestamp. */
  readonly items: readonly PreliminaryMarkerPlacement[];
}

/**
 * Assembled row highlight bands and contiguous dimmed background row masks.
 */
export interface RowHighlightBuildResult {
  /** Positioned row highlight bands. */
  readonly highlights: readonly RowHighlightBandViewModel[];
  /** Positioned dimmed row masks over unlisted rows. */
  readonly dimmedMasks: readonly DimmedRowMaskViewModel[];
}

/**
 * Assembled timeline excerpt pins and bottom evidence deck ViewModel.
 */
export interface ExcerptPinsAndDeckResult {
  /** Positioned excerpt pins on the timeline. */
  readonly pins: readonly ExcerptPinViewModel[];
  /** Synchronized bottom evidence deck ViewModel. */
  readonly deck: ExcerptDeckViewModel;
}

/**
 * Selected deck grid preset and column count.
 */
export interface DeckGridPresetSelection {
  /** Selected grid preset. */
  readonly preset: ExcerptDeckGridPreset;
  /** Number of grid columns. */
  readonly columns: number;
}

/**
 * Vertical stem segment in stage coordinates used for obstacle avoidance during label placement.
 */
export interface VerticalStemSegment {
  /** Horizontal X coordinate of the vertical stem in stage CSS pixels. */
  readonly x: number;
  /** Top Y coordinate of the vertical stem in stage CSS pixels. */
  readonly topPx: number;
  /** Bottom Y coordinate of the vertical stem in stage CSS pixels. */
  readonly bottomPx: number;
}

/**
 * Local horizontal interval relative to a span's left edge that is free of intersecting vertical stems.
 */
export interface LocalSpanInterval {
  /** Left boundary of the stem-free interval in local span pixels. */
  readonly start: number;
  /** Right boundary of the stem-free interval in local span pixels. */
  readonly end: number;
}

/**
 * Vertical placement and row-scoping geometry for a timeline span.
 */
export interface SpanVerticalBounds {
  /** Top Y coordinate of the span band in stage CSS pixels. */
  readonly topPx: number;
  /** Bottom Y coordinate of the span band in stage CSS pixels. */
  readonly bottomPx: number;
  /** Vertical height of the span band in stage CSS pixels. */
  readonly heightPx: number;
  /** True when the span is constrained to specific timeline rows rather than full chart height. */
  readonly isRowScoped: boolean;
}

/**
 * Horizontal placement, width, and stem coincidence flags for a timeline span band.
 */
export interface SpanBoxGeometry {
  /** Left X coordinate of the span band in stage CSS pixels. */
  readonly leftPx: number;
  /** Horizontal width of the span band in CSS pixels. */
  readonly widthPx: number;
  /** True when the span is wide enough to display decorative corner brackets. */
  readonly showCornerBrackets: boolean;
  /** True when a vertical stem coincides with the span's left boundary. */
  readonly coincidentLeft: boolean;
  /** True when a vertical stem coincides with the span's right boundary. */
  readonly coincidentRight: boolean;
}

/**
 * Header label text and positioned chip coordinates for a timeline span band.
 */
export interface SpanHeaderChipLayout {
  /** Formatted header label including duration suffix when present. */
  readonly headerText: string;
  /** Horizontal offset of the header chip relative to span left in pixels. */
  readonly labelLeftPx: number;
  /** Left X coordinate of the header chip in stage CSS pixels. */
  readonly chipStageLeftPx: number;
  /** Top Y coordinate of the header chip in stage CSS pixels. */
  readonly chipStageTopPx: number;
  /** Maximum allowable width of the header chip in CSS pixels before truncation. */
  readonly chipMaxWidthPx: number;
}

/**
 * Resolves semantic palette colors for an annotation by palette ID with optional fallback.
 */
export type PaletteColorResolver = (
  paletteId: string,
  fallback?: TimelineResolvedPaletteColors,
) => TimelineResolvedPaletteColors;

/**
 * Projects a timestamp in epoch milliseconds to a horizontal stage coordinate in CSS pixels.
 */
export type StageTimeProjector = (timeMs: number) => number;
