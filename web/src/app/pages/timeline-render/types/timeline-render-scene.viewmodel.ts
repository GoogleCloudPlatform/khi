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
  TimelineMarkerStyle,
  TimelineResolvedMarker,
  TimelineResolvedPaletteColors,
  TimelineResolvedSpan,
  TimelineSceneOverlay,
  TimelineSpanPattern,
} from 'src/app/common/schema/timeline-render-types';

/**
 * Target resource timeline, revision, or log entry to focus when opening a full KHI session.
 */
export interface TimelineSessionJumpTarget {
  /**
   * Target timeline domain ID when navigating to a specific resource row.
   */
  readonly timelineId?: number;
  /**
   * Target revision index within the resource timeline when focusing a state diff.
   */
  readonly revisionIndex?: number;
  /**
   * Target log entry ID when focusing a specific log line in KHI.
   */
  readonly logId?: number;
}

/**
 * Minimal height metric for a single rendered timeline row.
 */
export interface TimelineRowMetric {
  /**
   * Unique numeric timeline domain ID for the rendered row.
   */
  readonly id: number;
  /**
   * Rendered vertical row height in CSS pixels.
   */
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
  /** Vertical length of the marker stem in CSS pixels. */
  readonly stemHeightPx: number;
  /** True when the marker terminates with a target dot on a matched timeline row. */
  readonly hasTargetDot: boolean;
  /** Center Y coordinate of the target row dot in stage CSS pixels. */
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
  /** Formatted header chip text combining the span label and optional duration. */
  readonly headerText: string;
  /** Fill pattern applied to the span background band. */
  readonly pattern: TimelineSpanPattern;
  /** Palette identifier referenced by this span. */
  readonly paletteId: string;
  /** Resolved semantic foreground and surface colors for rendering the span. */
  readonly colors: TimelineResolvedPaletteColors;
  /** True when the span is vertically restricted to specific timeline rows. */
  readonly isRowScoped: boolean;
  /** True when corner brackets are rendered to frame a wide enough span band. */
  readonly showCornerBrackets: boolean;
  /** True when the left span boundary coincides with a vertical marker or excerpt stem. */
  readonly coincidentLeft: boolean;
  /** True when the right span boundary coincides with a vertical marker or excerpt stem. */
  readonly coincidentRight: boolean;
  /** Local horizontal offset of the header chip from the left edge of the span in CSS pixels. */
  readonly labelLeftPx: number;
  /** Stage-relative left X coordinate of the header chip in CSS pixels. */
  readonly chipStageLeftPx: number;
  /** Stage-relative top Y coordinate of the header chip in CSS pixels. */
  readonly chipStageTopPx: number;
  /** Maximum horizontal width available to the header chip before truncation in CSS pixels. */
  readonly chipMaxWidthPx: number;
  /** Stage-relative left X coordinate of the span band in CSS pixels. */
  readonly leftPx: number;
  /** Stage-relative top Y coordinate of the span band in CSS pixels. */
  readonly topPx: number;
  /** Horizontal width of the span band in CSS pixels. */
  readonly widthPx: number;
  /** Vertical height of the span band in CSS pixels. */
  readonly heightPx: number;
}

/**
 * Layout ViewModel for the timeline stage overlay.
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
