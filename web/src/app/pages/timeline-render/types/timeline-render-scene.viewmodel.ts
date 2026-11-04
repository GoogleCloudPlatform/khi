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

interface RowStageBounds {
  readonly index: number;
  readonly id: number;
  readonly topPx: number;
  readonly bottomPx: number;
  readonly centerPx: number;
  readonly heightPx: number;
}

interface PreliminaryMarkerPlacement {
  readonly marker: TimelineResolvedMarker;
  readonly laneIndex: number;
  readonly pillLeftPx: number;
  readonly pillWidthPx: number;
  readonly stemX: number;
}

interface PackedMarkerLanes {
  readonly laneCount: number;
  readonly items: readonly PreliminaryMarkerPlacement[];
}

interface VerticalStemSegment {
  readonly x: number;
  readonly topPx: number;
  readonly bottomPx: number;
}

interface LocalSpanInterval {
  readonly start: number;
  readonly end: number;
}

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
 * Computes the geometry and layout ViewModel for timeline markers and spans.
 */
export function buildTimelineSceneViewModel(
  input: TimelineSceneLayoutInput,
): TimelineSceneViewModel {
  const indexPaneWidthPx = clampNumber(
    Number.isFinite(input.indexPaneWidthPx)
      ? Math.ceil(input.indexPaneWidthPx)
      : MIN_INDEX_PANE_WIDTH_PX,
    MIN_INDEX_PANE_WIDTH_PX,
    MAX_INDEX_PANE_WIDTH_PX,
  );
  const chartLeftPx = indexPaneWidthPx + TIMELINE_GUTTER_WIDTH_PX;
  const timelineMinWidthPx = chartLeftPx + MIN_CHART_VIEWPORT_WIDTH_PX;
  const stageWidthPx = Math.min(timelineMinWidthPx, MAX_STAGE_WIDTH_PX);
  const chartViewportWidthPx = Math.max(0, stageWidthPx - chartLeftPx);
  const safePixelsPerMs =
    Number.isFinite(input.pixelsPerMs) && input.pixelsPerMs > 0
      ? input.pixelsPerMs
      : 0;
  const overlay = input.overlay;

  const toStageX = (timeMs: number): number => {
    const rawX = (timeMs - input.viewportLeftTimeMs) * safePixelsPerMs;
    const clampedX = clampNumber(
      Number.isFinite(rawX) ? rawX : 0,
      0,
      chartViewportWidthPx,
    );
    return Math.round((chartLeftPx + clampedX) * 10) / 10;
  };

  const viewportRightTimeMs =
    input.viewportLeftTimeMs +
    chartViewportWidthPx / Math.max(safePixelsPerMs, 1e-9);
  const visibleMarkers =
    safePixelsPerMs > 0
      ? (overlay?.markers ?? []).filter(
          (m) =>
            m.timeMs >= input.viewportLeftTimeMs &&
            m.timeMs <= viewportRightTimeMs,
        )
      : (overlay?.markers ?? []);
  const visibleSpans =
    safePixelsPerMs > 0
      ? (overlay?.spans ?? []).filter(
          (s) =>
            Math.max(s.startTimeMs, s.endTimeMs) >= input.viewportLeftTimeMs &&
            Math.min(s.startTimeMs, s.endTimeMs) <= viewportRightTimeMs,
        )
      : (overlay?.spans ?? []);

  const markerPlacements = packMarkerLanes(
    visibleMarkers,
    stageWidthPx,
    toStageX,
  );
  const topBandHeightPx =
    markerPlacements.laneCount > 0
      ? markerPlacements.laneCount * MARKER_LANE_STEP_PX +
        MARKER_BAND_VERTICAL_PADDING_PX
      : 0;

  let rowsTotalHeightPx = 0;
  for (const row of input.rows) {
    rowsTotalHeightPx += row.heightPx;
  }
  const timelineFrameHeightPx = Math.max(
    TIMELINE_HEADER_HEIGHT_PX + BASE_ROW_HEIGHT,
    Math.ceil(TIMELINE_HEADER_HEIGHT_PX + rowsTotalHeightPx),
  );
  const stageHeightPx = topBandHeightPx + timelineFrameHeightPx;

  const rowBoundsById = new Map<number, RowStageBounds>();
  let cursorY = topBandHeightPx + TIMELINE_HEADER_HEIGHT_PX;
  for (let i = 0; i < input.rows.length; i++) {
    const row = input.rows[i];
    rowBoundsById.set(row.id, {
      index: i,
      id: row.id,
      topPx: cursorY,
      bottomPx: cursorY + row.heightPx,
      centerPx: cursorY + row.heightPx / 2,
      heightPx: row.heightPx,
    });
    cursorY += row.heightPx;
  }

  const paletteById = new Map<string, TimelineResolvedPaletteColors>();
  for (const entry of overlay?.palette ?? []) {
    paletteById.set(entry.id, entry);
  }
  const resolveColors = (
    paletteId: string,
    fallback: TimelineResolvedPaletteColors = DEFAULT_TIMELINE_PALETTE_COLORS,
  ): TimelineResolvedPaletteColors => {
    return paletteById.get(paletteId) ?? fallback;
  };

  const markers = markerPlacements.items.map((item): MarkerItemViewModel => {
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

  const verticalStems: VerticalStemSegment[] = [];
  for (const marker of markers) {
    if (marker.stemHeightPx > 0) {
      verticalStems.push({
        x: marker.stemX,
        topPx: marker.stemTopPx,
        bottomPx: marker.stemTopPx + marker.stemHeightPx,
      });
    }
  }

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

function packMarkerLanes(
  markers: readonly TimelineResolvedMarker[],
  stageWidthPx: number,
  toStageX: (timeMs: number) => number,
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

function buildSpanViewModels(
  spans: readonly TimelineResolvedSpan[],
  chartLeftPx: number,
  stageWidthPx: number,
  topBandHeightPx: number,
  stageHeightPx: number,
  rowBoundsById: ReadonlyMap<number, RowStageBounds>,
  toStageX: (timeMs: number) => number,
  resolveColors: (paletteId: string) => TimelineResolvedPaletteColors,
  verticalStems: readonly VerticalStemSegment[],
): SpanItemViewModel[] {
  const chartTopPx = topBandHeightPx + TIMELINE_HEADER_HEIGHT_PX;
  const result: SpanItemViewModel[] = [];

  for (const span of spans) {
    let topPx = chartTopPx;
    let bottomPx = stageHeightPx;
    let isRowScoped = false;

    if (span.timelineIds && span.timelineIds.length > 0) {
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
        continue;
      }
      const rawSpanHeight = maxBottom - minTop;
      const verticalInsetPx = rawSpanHeight >= 20 ? 2 : 0;
      topPx = minTop + verticalInsetPx;
      bottomPx = maxBottom - verticalInsetPx;
      isRowScoped = true;
    }

    const x1 = toStageX(span.startTimeMs);
    const x2 = toStageX(span.endTimeMs);
    const widthPx = Math.max(4, Math.abs(x2 - x1));
    const leftPx = clampNumber(
      Math.min(x1, x2),
      chartLeftPx,
      stageWidthPx - widthPx,
    );

    const heightPx = Math.max(16, bottomPx - topPx);
    const showCornerBrackets = widthPx >= 20;
    const rightEdgePx = leftPx + widthPx;
    const coincidentLeft =
      widthPx >= 24 &&
      verticalStems.some(
        (stem) =>
          stem.bottomPx >= topPx &&
          stem.topPx <= bottomPx &&
          Math.abs(stem.x - leftPx) <= 3,
      );
    const coincidentRight =
      widthPx >= 24 &&
      verticalStems.some(
        (stem) =>
          stem.bottomPx >= topPx &&
          stem.topPx <= bottomPx &&
          Math.abs(stem.x - rightEdgePx) <= 3,
      );

    const headerText = span.durationLabel
      ? `${span.label} (${span.durationLabel})`
      : span.label;
    const headerBottomPx = topPx + Math.min(heightPx, 26);
    const hasLeftHeaderStem =
      widthPx >= 48 &&
      (coincidentLeft ||
        verticalStems.some(
          (stem) =>
            stem.bottomPx >= topPx &&
            stem.topPx <= headerBottomPx &&
            stem.x >= leftPx - 3 &&
            stem.x <= leftPx + 14,
        ));
    const minLocalLeft = hasLeftHeaderStem ? 14 : 6;
    const rightPaddingPx = widthPx >= 48 && coincidentRight ? 10 : 6;
    const maxLocalRight = Math.max(minLocalLeft, widthPx - rightPaddingPx);
    // Estimates horizontal chip width for 11px label typography plus border and padding.
    const estimatedChipWidthPx = Math.round(headerText.length * 5.35 + 14);

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

    let labelLeftPx = minLocalLeft;
    if (widthPx >= 48 && candidateIntervals.length > 0) {
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

    const chipStageLeftPx = Math.round((leftPx + labelLeftPx) * 10) / 10;
    const chipStageTopPx =
      topPx + Math.min(4, Math.max(1, Math.floor((heightPx - 18) / 2)));
    const chipMaxWidthPx = Math.max(
      12,
      Math.round(maxLocalRight - labelLeftPx),
    );

    result.push({
      id: span.id,
      headerText,
      pattern: span.pattern,
      paletteId: span.paletteId,
      colors: resolveColors(span.paletteId),
      isRowScoped,
      showCornerBrackets,
      coincidentLeft,
      coincidentRight,
      labelLeftPx,
      chipStageLeftPx,
      chipStageTopPx,
      chipMaxWidthPx,
      leftPx,
      topPx,
      widthPx,
      heightPx,
    });
  }

  return result;
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
