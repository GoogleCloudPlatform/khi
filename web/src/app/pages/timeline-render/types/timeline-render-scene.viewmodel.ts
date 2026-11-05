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
import {
  MAX_INDEX_PANE_WIDTH_PX,
  MAX_STAGE_WIDTH_PX,
  MIN_CHART_VIEWPORT_WIDTH_PX,
  MIN_INDEX_PANE_WIDTH_PX,
  TIMELINE_GUTTER_WIDTH_PX,
  TIMELINE_HEADER_HEIGHT_PX,
} from 'src/app/pages/timeline-render/types/timeline-render-sizing';
import { StyleStoreLike } from 'src/app/store/domain/style-store';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { BASE_ROW_HEIGHT } from 'src/app/timeline/components/style-model';

const MARKER_LANE_STEP_PX = 30;
const MARKER_PILL_HEIGHT_PX = 24;
const MARKER_BAND_VERTICAL_PADDING_PX = 12;

/**
 * Target resource timeline, revision, or log entry to focus when opening a full KHI session.
 */
export interface TimelineSessionJumpTarget {
  readonly timelineId?: number;
  readonly revisionIndex?: number;
  readonly logId?: number;
}

/**
 * Minimal height metric for a single rendered timeline row.
 */
export interface TimelineRowMetric {
  readonly id: number;
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
 * Built-in fallback palette colors when a scene item uses the default blue palette.
 */
export const DEFAULT_TIMELINE_PALETTE_COLORS: TimelineResolvedPaletteColors = {
  id: 'default',
  main: '#1976d2',
  onMain: '#fafafa',
  soft: 'rgba(25, 118, 210, 0.13)',
  surface: '#e8f1fb',
};

/**
 * Built-in fallback palette colors when an external reference marker omits paletteId.
 */
export const DEFAULT_REFERENCE_PALETTE_COLORS: TimelineResolvedPaletteColors = {
  id: 'default-reference',
  main: '#5e35b1',
  onMain: '#fafafa',
  soft: 'rgba(94, 53, 177, 0.13)',
  surface: '#efebf7',
};

/**
 * Positioned vertical marker pill, stem, and optional target dot for timeline markers.
 */
export interface MarkerItemViewModel {
  readonly id: string;
  readonly badge: string;
  readonly label: string;
  readonly timeLabel: string;
  readonly style: TimelineMarkerStyle;
  readonly paletteId: string;
  readonly colors: TimelineResolvedPaletteColors;
  readonly pillLeftPx: number;
  readonly pillTopPx: number;
  readonly pillWidthPx: number;
  readonly pillHeightPx: number;
  readonly stemX: number;
  readonly stemTopPx: number;
  readonly stemHeightPx: number;
  readonly hasTargetDot: boolean;
  readonly dotY: number;
  readonly jumpTarget: TimelineSessionJumpTarget | null;
}

/**
 * Positioned full-height or row-scoped time-range band for timeline spans.
 */
export interface SpanItemViewModel {
  readonly id: string;
  readonly headerText: string;
  readonly pattern: TimelineSpanPattern;
  readonly paletteId: string;
  readonly colors: TimelineResolvedPaletteColors;
  readonly isRowScoped: boolean;
  readonly showCornerBrackets: boolean;
  readonly coincidentLeft: boolean;
  readonly coincidentRight: boolean;
  readonly labelLeftPx: number;
  readonly chipStageLeftPx: number;
  readonly chipStageTopPx: number;
  readonly chipMaxWidthPx: number;
  readonly leftPx: number;
  readonly topPx: number;
  readonly widthPx: number;
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

/**
 * Extracts row height metrics in pixels from rendered timeline domain elements.
 */
export function extractTimelineRowMetrics(
  timelines: readonly ReadonlyDomainElement<Timeline>[],
  styleStore: StyleStoreLike,
): TimelineRowMetric[] {
  return timelines.map((timeline) => {
    let timelineType = timeline.type;
    try {
      timelineType = styleStore.getTimelineType(timeline.type.id);
    } catch {
      timelineType = timeline.type;
    }
    return {
      id: timeline.id,
      heightPx: timelineType.height * BASE_ROW_HEIGHT,
    };
  });
}

/**
 * Resolves clamped horizontal stage, index pane, gutter boundary, and chart viewport widths in CSS pixels.
 */
export function resolveSceneHorizontalBounds(
  rawIndexPaneWidthPx: number,
): SceneHorizontalBounds {
  const indexPaneWidthPx = clampNumber(
    Number.isFinite(rawIndexPaneWidthPx)
      ? Math.ceil(rawIndexPaneWidthPx)
      : MIN_INDEX_PANE_WIDTH_PX,
    MIN_INDEX_PANE_WIDTH_PX,
    MAX_INDEX_PANE_WIDTH_PX,
  );
  const chartLeftPx = indexPaneWidthPx + TIMELINE_GUTTER_WIDTH_PX;
  const timelineMinWidthPx = chartLeftPx + MIN_CHART_VIEWPORT_WIDTH_PX;
  const stageWidthPx = Math.min(timelineMinWidthPx, MAX_STAGE_WIDTH_PX);
  const chartViewportWidthPx = Math.max(0, stageWidthPx - chartLeftPx);

  return {
    indexPaneWidthPx,
    chartLeftPx,
    chartViewportWidthPx,
    stageWidthPx,
  };
}

/**
 * Creates a coordinate projector mapping epoch millisecond timestamps to clamped horizontal stage pixels.
 */
export function createStageTimeProjector(
  viewportLeftTimeMs: number,
  chartLeftPx: number,
  chartViewportWidthPx: number,
  pixelsPerMs: number,
): StageTimeProjector {
  const safePixelsPerMs =
    Number.isFinite(pixelsPerMs) && pixelsPerMs > 0 ? pixelsPerMs : 0;
  return (timeMs: number): number => {
    const rawX = (timeMs - viewportLeftTimeMs) * safePixelsPerMs;
    const clampedX = clampNumber(
      Number.isFinite(rawX) ? rawX : 0,
      0,
      chartViewportWidthPx,
    );
    return Math.round((chartLeftPx + clampedX) * 10) / 10;
  };
}

/**
 * Filters overlay markers and spans to those intersecting the visible chart viewport time range.
 */
export function selectVisibleAnnotations(
  overlay: TimelineSceneOverlay | null,
  viewportLeftTimeMs: number,
  chartViewportWidthPx: number,
  pixelsPerMs: number,
): VisibleTimelineAnnotations {
  const safePixelsPerMs =
    Number.isFinite(pixelsPerMs) && pixelsPerMs > 0 ? pixelsPerMs : 0;
  if (safePixelsPerMs <= 0) {
    return {
      markers: overlay?.markers ?? [],
      spans: overlay?.spans ?? [],
    };
  }
  const viewportRightTimeMs =
    viewportLeftTimeMs + chartViewportWidthPx / Math.max(safePixelsPerMs, 1e-9);
  const markers = (overlay?.markers ?? []).filter(
    (m) => m.timeMs >= viewportLeftTimeMs && m.timeMs <= viewportRightTimeMs,
  );
  const spans = (overlay?.spans ?? []).filter(
    (s) =>
      Math.max(s.startTimeMs, s.endTimeMs) >= viewportLeftTimeMs &&
      Math.min(s.startTimeMs, s.endTimeMs) <= viewportRightTimeMs,
  );
  return { markers, spans };
}

/**
 * Packs markers into non-overlapping horizontal lanes to prevent pill collisions above the timeline frame.
 */
export function packMarkerLanes(
  markers: readonly TimelineResolvedMarker[],
  stageWidthPx: number,
  toStageX: StageTimeProjector,
): PackedMarkerLanes {
  if (markers.length === 0) {
    return { laneCount: 0, items: [] };
  }
  const sorted = [...markers].sort((a, b) => a.timeMs - b.timeMs);
  const laneRightEdges: number[] = [];
  const items: PreliminaryMarkerPlacement[] = [];

  for (const marker of sorted) {
    const stemX = toStageX(marker.timeMs);
    const textLen =
      marker.badge.length + marker.label.length + marker.timeLabel.length + 3;
    const pillWidthPx = clampNumber(Math.round(28 + textLen * 6.6), 96, 360);
    const pillLeftPx = Math.round(
      clampNumber(stemX - pillWidthPx / 2, 8, stageWidthPx - pillWidthPx - 8),
    );
    const pillRightPx = pillLeftPx + pillWidthPx;

    let assignedLane = -1;
    for (let lane = 0; lane < laneRightEdges.length; lane++) {
      if (laneRightEdges[lane] + 8 <= pillLeftPx) {
        assignedLane = lane;
        laneRightEdges[lane] = pillRightPx;
        break;
      }
    }
    if (assignedLane < 0) {
      assignedLane = laneRightEdges.length;
      laneRightEdges.push(pillRightPx);
    }

    items.push({
      marker,
      laneIndex: assignedLane,
      pillLeftPx,
      pillWidthPx,
      stemX,
    });
  }

  return {
    laneCount: laneRightEdges.length,
    items,
  };
}

/**
 * Computes vertical stage dimensions and individual row boundaries from row height metrics.
 */
export function computeStageRowLayout(
  rows: readonly TimelineRowMetric[],
  markerLaneCount: number,
): TimelineStageRowLayout {
  const topBandHeightPx =
    markerLaneCount > 0
      ? markerLaneCount * MARKER_LANE_STEP_PX + MARKER_BAND_VERTICAL_PADDING_PX
      : 0;

  let rowsTotalHeightPx = 0;
  for (const row of rows) {
    rowsTotalHeightPx += row.heightPx;
  }
  const timelineFrameHeightPx = Math.max(
    TIMELINE_HEADER_HEIGHT_PX + BASE_ROW_HEIGHT,
    Math.ceil(TIMELINE_HEADER_HEIGHT_PX + rowsTotalHeightPx),
  );
  const stageHeightPx = topBandHeightPx + timelineFrameHeightPx;

  const rowBoundsList: TimelineRowStageBounds[] = [];
  const rowBoundsById = new Map<number, TimelineRowStageBounds>();
  let cursorY = topBandHeightPx + TIMELINE_HEADER_HEIGHT_PX;
  for (let i = 0; i < rows.length; i++) {
    const row = rows[i];
    const bounds: TimelineRowStageBounds = {
      index: i,
      id: row.id,
      topPx: cursorY,
      bottomPx: cursorY + row.heightPx,
      centerPx: cursorY + row.heightPx / 2,
      heightPx: row.heightPx,
    };
    rowBoundsList.push(bounds);
    rowBoundsById.set(row.id, bounds);
    cursorY += row.heightPx;
  }

  return {
    topBandHeightPx,
    timelineFrameHeightPx,
    stageHeightPx,
    rowBoundsList,
    rowBoundsById,
  };
}

/**
 * Creates a color resolver lookup function from an optional overlay palette definition.
 */
export function createPaletteColorResolver(
  palette: readonly TimelineResolvedPaletteColors[] | undefined,
): PaletteColorResolver {
  const paletteById = new Map<string, TimelineResolvedPaletteColors>();
  for (const entry of palette ?? []) {
    paletteById.set(entry.id, entry);
  }
  return (
    paletteId: string,
    fallback: TimelineResolvedPaletteColors = DEFAULT_TIMELINE_PALETTE_COLORS,
  ): TimelineResolvedPaletteColors => {
    return paletteById.get(paletteId) ?? fallback;
  };
}

/**
 * Builds positioned marker ViewModels including pill coordinates, drop stem height, and target dots.
 */
export function buildMarkerViewModels(
  placements: readonly PreliminaryMarkerPlacement[],
  stageHeightPx: number,
  rowBoundsById: ReadonlyMap<number, TimelineRowStageBounds>,
  resolveColors: PaletteColorResolver,
): MarkerItemViewModel[] {
  return placements.map((item): MarkerItemViewModel => {
    const pillTopPx = 6 + item.laneIndex * MARKER_LANE_STEP_PX;
    const stemTopPx = pillTopPx + MARKER_PILL_HEIGHT_PX;
    const targetRow =
      item.marker.timelineId !== undefined
        ? rowBoundsById.get(item.marker.timelineId)
        : undefined;
    const hasTargetDot =
      item.marker.style === TimelineMarkerStyle.PIN && targetRow !== undefined;
    const stemEndPx =
      hasTargetDot && targetRow ? targetRow.centerPx : stageHeightPx;
    const fallbackColors =
      item.marker.style === TimelineMarkerStyle.REFERENCE ||
      item.marker.paletteId === DEFAULT_REFERENCE_PALETTE_COLORS.id
        ? DEFAULT_REFERENCE_PALETTE_COLORS
        : DEFAULT_TIMELINE_PALETTE_COLORS;
    return {
      id: item.marker.id,
      badge: item.marker.badge,
      label: item.marker.label,
      timeLabel: item.marker.timeLabel,
      style: item.marker.style,
      paletteId: item.marker.paletteId,
      colors: resolveColors(item.marker.paletteId, fallbackColors),
      pillLeftPx: item.pillLeftPx,
      pillTopPx,
      pillWidthPx: item.pillWidthPx,
      pillHeightPx: MARKER_PILL_HEIGHT_PX,
      stemX: item.stemX,
      stemTopPx,
      stemHeightPx: Math.max(0, stemEndPx - stemTopPx),
      hasTargetDot,
      dotY: hasTargetDot && targetRow ? targetRow.centerPx : stemEndPx,
      jumpTarget: buildJumpTarget(
        item.marker.timelineId,
        item.marker.revisionIndex,
        item.marker.logId,
      ),
    };
  });
}

/**
 * Collects non-zero vertical drop stem segments from marker ViewModels for span chip obstacle avoidance.
 */
export function collectMarkerVerticalStems(
  markers: readonly MarkerItemViewModel[],
): VerticalStemSegment[] {
  const stems: VerticalStemSegment[] = [];
  for (const marker of markers) {
    if (marker.stemHeightPx > 0) {
      stems.push({
        x: marker.stemX,
        topPx: marker.stemTopPx,
        bottomPx: marker.stemTopPx + marker.stemHeightPx,
      });
    }
  }
  return stems;
}

/**
 * Resolves vertical boundaries and row-scoped insets for a timeline span band.
 */
export function resolveSpanVerticalBounds(
  span: TimelineResolvedSpan,
  chartTopPx: number,
  stageHeightPx: number,
  rowBoundsById: ReadonlyMap<number, TimelineRowStageBounds>,
): SpanVerticalBounds | null {
  if (!span.timelineIds || span.timelineIds.length === 0) {
    return {
      topPx: chartTopPx,
      bottomPx: stageHeightPx,
      heightPx: Math.max(16, stageHeightPx - chartTopPx),
      isRowScoped: false,
    };
  }

  let minTop = Number.POSITIVE_INFINITY;
  let maxBottom = Number.NEGATIVE_INFINITY;
  for (const tid of span.timelineIds) {
    const bounds = rowBoundsById.get(tid);
    if (bounds) {
      minTop = Math.min(minTop, bounds.topPx);
      maxBottom = Math.max(maxBottom, bounds.bottomPx);
    }
  }
  if (!Number.isFinite(minTop) || !Number.isFinite(maxBottom)) {
    return null;
  }
  const rawSpanHeight = maxBottom - minTop;
  const verticalInsetPx = rawSpanHeight >= 20 ? 2 : 0;
  const topPx = minTop + verticalInsetPx;
  const bottomPx = maxBottom - verticalInsetPx;
  return {
    topPx,
    bottomPx,
    heightPx: Math.max(16, bottomPx - topPx),
    isRowScoped: true,
  };
}

/**
 * Computes horizontal boundaries, clamped width, and boundary stem coincidence for a span band.
 */
export function resolveSpanBoxGeometry(
  span: TimelineResolvedSpan,
  chartLeftPx: number,
  stageWidthPx: number,
  vertical: SpanVerticalBounds,
  toStageX: StageTimeProjector,
  verticalStems: readonly VerticalStemSegment[],
): SpanBoxGeometry {
  const x1 = toStageX(span.startTimeMs);
  const x2 = toStageX(span.endTimeMs);
  const widthPx = Math.max(4, Math.abs(x2 - x1));
  const leftPx = clampNumber(
    Math.min(x1, x2),
    chartLeftPx,
    stageWidthPx - widthPx,
  );
  const showCornerBrackets = widthPx >= 20;
  const rightEdgePx = leftPx + widthPx;
  const coincidentLeft =
    widthPx >= 24 &&
    verticalStems.some(
      (stem) =>
        stem.bottomPx >= vertical.topPx &&
        stem.topPx <= vertical.bottomPx &&
        Math.abs(stem.x - leftPx) <= 3,
    );
  const coincidentRight =
    widthPx >= 24 &&
    verticalStems.some(
      (stem) =>
        stem.bottomPx >= vertical.topPx &&
        stem.topPx <= vertical.bottomPx &&
        Math.abs(stem.x - rightEdgePx) <= 3,
    );

  return {
    leftPx,
    widthPx,
    showCornerBrackets,
    coincidentLeft,
    coincidentRight,
  };
}

/**
 * Finds local horizontal sub-intervals within a span header band that avoid intersecting vertical stems.
 */
export function findStemFreeSpanIntervals(
  verticalStems: readonly VerticalStemSegment[],
  leftPx: number,
  topPx: number,
  headerBottomPx: number,
  minLocalLeft: number,
  maxLocalRight: number,
): LocalSpanInterval[] {
  const interiorStemXs = verticalStems
    .filter(
      (stem) =>
        stem.bottomPx >= topPx &&
        stem.topPx <= headerBottomPx &&
        stem.x - leftPx > minLocalLeft - 4 &&
        stem.x - leftPx < maxLocalRight + 4,
    )
    .map((stem) => stem.x - leftPx)
    .sort((a, b) => a - b);

  const candidateIntervals: LocalSpanInterval[] = [];
  let cursor = minLocalLeft;
  for (const sx of interiorStemXs) {
    const gapEnd = sx - 6;
    if (gapEnd > cursor) {
      candidateIntervals.push({ start: cursor, end: gapEnd });
    }
    cursor = Math.max(cursor, sx + 6);
  }
  if (maxLocalRight > cursor) {
    candidateIntervals.push({ start: cursor, end: maxLocalRight });
  }

  return candidateIntervals;
}

/**
 * Computes header text formatting and obstacle-avoiding chip placement for a timeline span.
 */
export function layoutSpanHeaderChip(
  span: TimelineResolvedSpan,
  box: SpanBoxGeometry,
  vertical: SpanVerticalBounds,
  verticalStems: readonly VerticalStemSegment[],
): SpanHeaderChipLayout {
  const headerText = span.durationLabel
    ? `${span.label} (${span.durationLabel})`
    : span.label;
  const headerBottomPx = vertical.topPx + Math.min(vertical.heightPx, 26);
  const hasLeftHeaderStem =
    box.widthPx >= 48 &&
    (box.coincidentLeft ||
      verticalStems.some(
        (stem) =>
          stem.bottomPx >= vertical.topPx &&
          stem.topPx <= headerBottomPx &&
          stem.x >= box.leftPx - 3 &&
          stem.x <= box.leftPx + 14,
      ));
  const minLocalLeft = hasLeftHeaderStem ? 14 : 6;
  const rightPaddingPx = box.widthPx >= 48 && box.coincidentRight ? 10 : 6;
  const maxLocalRight = Math.max(minLocalLeft, box.widthPx - rightPaddingPx);
  // Estimates horizontal chip width for 11px label typography plus border and padding.
  const estimatedChipWidthPx = Math.round(headerText.length * 5.35 + 14);

  const candidateIntervals = findStemFreeSpanIntervals(
    verticalStems,
    box.leftPx,
    vertical.topPx,
    headerBottomPx,
    minLocalLeft,
    maxLocalRight,
  );

  let labelLeftPx = minLocalLeft;
  if (box.widthPx >= 48 && candidateIntervals.length > 0) {
    const fullFit = candidateIntervals.find(
      (seg) => seg.end - seg.start >= estimatedChipWidthPx,
    );
    if (fullFit) {
      labelLeftPx = Math.round(fullFit.start);
    } else {
      const unclippedStarts = candidateIntervals.filter(
        (seg) => maxLocalRight - seg.start >= estimatedChipWidthPx,
      );
      if (unclippedStarts.length > 0) {
        labelLeftPx = Math.round(
          unclippedStarts[unclippedStarts.length - 1].start,
        );
      }
    }
  }

  const chipStageLeftPx = Math.round((box.leftPx + labelLeftPx) * 10) / 10;
  const chipStageTopPx =
    vertical.topPx +
    Math.min(4, Math.max(1, Math.floor((vertical.heightPx - 18) / 2)));
  const chipMaxWidthPx = Math.max(12, Math.round(maxLocalRight - labelLeftPx));

  return {
    headerText,
    labelLeftPx,
    chipStageLeftPx,
    chipStageTopPx,
    chipMaxWidthPx,
  };
}

/**
 * Builds positioned time-range span ViewModels including vertical bounds, box geometry, and header chips.
 */
export function buildSpanViewModels(
  spans: readonly TimelineResolvedSpan[],
  chartLeftPx: number,
  stageWidthPx: number,
  topBandHeightPx: number,
  stageHeightPx: number,
  rowBoundsById: ReadonlyMap<number, TimelineRowStageBounds>,
  toStageX: StageTimeProjector,
  resolveColors: PaletteColorResolver,
  verticalStems: readonly VerticalStemSegment[],
): SpanItemViewModel[] {
  const chartTopPx = topBandHeightPx + TIMELINE_HEADER_HEIGHT_PX;
  const result: SpanItemViewModel[] = [];

  for (const span of spans) {
    const vertical = resolveSpanVerticalBounds(
      span,
      chartTopPx,
      stageHeightPx,
      rowBoundsById,
    );
    if (!vertical) {
      continue;
    }

    const box = resolveSpanBoxGeometry(
      span,
      chartLeftPx,
      stageWidthPx,
      vertical,
      toStageX,
      verticalStems,
    );

    const chip = layoutSpanHeaderChip(span, box, vertical, verticalStems);

    result.push({
      id: span.id,
      headerText: chip.headerText,
      pattern: span.pattern,
      paletteId: span.paletteId,
      colors: resolveColors(span.paletteId),
      isRowScoped: vertical.isRowScoped,
      showCornerBrackets: box.showCornerBrackets,
      coincidentLeft: box.coincidentLeft,
      coincidentRight: box.coincidentRight,
      labelLeftPx: chip.labelLeftPx,
      chipStageLeftPx: chip.chipStageLeftPx,
      chipStageTopPx: chip.chipStageTopPx,
      chipMaxWidthPx: chip.chipMaxWidthPx,
      leftPx: box.leftPx,
      topPx: vertical.topPx,
      widthPx: box.widthPx,
      heightPx: vertical.heightPx,
    });
  }

  return result;
}

/**
 * Computes the geometry and layout ViewModel for timeline markers and spans.
 */
export function buildTimelineSceneViewModel(
  input: TimelineSceneLayoutInput,
): TimelineSceneViewModel {
  const { indexPaneWidthPx, chartLeftPx, chartViewportWidthPx, stageWidthPx } =
    resolveSceneHorizontalBounds(input.indexPaneWidthPx);
  const toStageX = createStageTimeProjector(
    input.viewportLeftTimeMs,
    chartLeftPx,
    chartViewportWidthPx,
    input.pixelsPerMs,
  );
  const { markers: visibleMarkers, spans: visibleSpans } =
    selectVisibleAnnotations(
      input.overlay,
      input.viewportLeftTimeMs,
      chartViewportWidthPx,
      input.pixelsPerMs,
    );
  const markerPlacements = packMarkerLanes(
    visibleMarkers,
    stageWidthPx,
    toStageX,
  );
  const {
    topBandHeightPx,
    timelineFrameHeightPx,
    stageHeightPx,
    rowBoundsById,
  } = computeStageRowLayout(input.rows, markerPlacements.laneCount);
  const resolveColors = createPaletteColorResolver(input.overlay?.palette);

  const markers = buildMarkerViewModels(
    markerPlacements.items,
    stageHeightPx,
    rowBoundsById,
    resolveColors,
  );
  const verticalStems = collectMarkerVerticalStems(markers);
  const spans = buildSpanViewModels(
    visibleSpans,
    chartLeftPx,
    stageWidthPx,
    topBandHeightPx,
    stageHeightPx,
    rowBoundsById,
    toStageX,
    resolveColors,
    verticalStems,
  );

  const hasOverlayItems = markers.length > 0 || spans.length > 0;

  return {
    indexPaneWidthPx,
    chartLeftPx,
    chartViewportWidthPx,
    stageWidthPx,
    topBandHeightPx,
    timelineFrameHeightPx,
    stageHeightPx,
    estimatedTotalHeightPx: stageHeightPx,
    hasOverlayItems,
    markers,
    spans,
  };
}

function buildJumpTarget(
  timelineId?: number,
  revisionIndex?: number,
  logId?: number,
): TimelineSessionJumpTarget | null {
  if (
    timelineId === undefined &&
    revisionIndex === undefined &&
    logId === undefined
  ) {
    return null;
  }
  return {
    timelineId,
    revisionIndex,
    logId,
  };
}

function clampNumber(value: number, min: number, max: number): number {
  if (max < min) {
    return min;
  }
  return Math.max(min, Math.min(max, value));
}
