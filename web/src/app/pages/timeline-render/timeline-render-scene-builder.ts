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

import { unreachable } from 'src/app/common/misc-util';
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
import {
  CalloutItemViewModel,
  ConnectorItemViewModel,
  DimmedRowMaskViewModel,
  ExcerptCardViewModel,
  ExcerptDeckViewModel,
  ExcerptLineViewModel,
  ExcerptPinViewModel,
  ExcerptPinsAndDeckResult,
  ExcerptTetherViewModel,
  LocalSpanInterval,
  MarkerItemViewModel,
  PackedMarkerLanes,
  PaletteColorResolver,
  PreliminaryMarkerPlacement,
  RowHighlightBandViewModel,
  RowHighlightBuildResult,
  SceneHorizontalBounds,
  SpanBoxGeometry,
  SpanHeaderChipLayout,
  SpanItemViewModel,
  SpanVerticalBounds,
  StageTimeProjector,
  TimelineRowMetric,
  TimelineRowStageBounds,
  TimelineSceneLayoutInput,
  TimelineSceneViewModel,
  TimelineSessionJumpTarget,
  TimelineStageRowLayout,
  VerticalStemSegment,
  VisibleTimelineAnnotations,
} from 'src/app/pages/timeline-render/types/timeline-render-scene.viewmodel';
import {
  DECK_GRID_GAP_PX,
  DECK_GRID_PADDING_PX,
  ExcerptDeckGridPreset,
  ExcerptDeckLayoutPlan,
  MAX_EXCERPT_DECK_CARDS,
  ROLE_BADGE_RIGHT_INSET_PX,
  TIMELINE_HEADER_HEIGHT_PX,
  TextWidthMeasurer,
  clampTimelineFrameHeightPx,
  createCanvasTextWidthMeasurer,
  estimateExcerptCardHeightPx,
  getExcerptActionLabel,
  planExcerptDeckLayout,
  resolveHorizontalStageGeometry,
  resolveTimelineRowHeightPx,
} from 'src/app/pages/timeline-render/types/timeline-render-sizing';
import { StyleStoreLike } from 'src/app/store/domain/style-store';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { clampNumber } from 'src/app/utils/math-util';

const MARKER_LANE_STEP_PX = 30;
const MARKER_PILL_HEIGHT_PX = 24;
const MARKER_BAND_VERTICAL_PADDING_PX = 12;
const DECK_TETHER_STRIP_HEIGHT_PX = 32;
const VIEWPORT_TIME_EPSILON_MS = 1e-3;
const defaultTextWidthMeasurer = createCanvasTextWidthMeasurer();

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
 * Extracts row height metrics in pixels from rendered timeline domain elements.
 */
export function extractTimelineRowMetrics(
  timelines: readonly ReadonlyDomainElement<Timeline>[],
  styleStore: StyleStoreLike,
): TimelineRowMetric[] {
  return timelines.map((timeline) => ({
    id: timeline.id,
    heightPx: resolveTimelineRowHeightPx(timeline, styleStore),
  }));
}

/**
 * Resolves clamped horizontal stage, index pane, gutter boundary, and chart viewport widths in CSS pixels.
 */
export function resolveSceneHorizontalBounds(
  rawIndexPaneWidthPx: number,
  requiredDeckWidthPx = 0,
): SceneHorizontalBounds {
  return resolveHorizontalStageGeometry(
    rawIndexPaneWidthPx,
    requiredDeckWidthPx,
  );
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
  const minVisibleTimeMs = viewportLeftTimeMs - VIEWPORT_TIME_EPSILON_MS;
  const maxVisibleTimeMs = viewportRightTimeMs + VIEWPORT_TIME_EPSILON_MS;
  const markers = (overlay?.markers ?? []).filter(
    (m) => m.timeMs >= minVisibleTimeMs && m.timeMs <= maxVisibleTimeMs,
  );
  const spans = (overlay?.spans ?? []).filter(
    (s) =>
      Math.max(s.startTimeMs, s.endTimeMs) >= minVisibleTimeMs &&
      Math.min(s.startTimeMs, s.endTimeMs) <= maxVisibleTimeMs,
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
  const timelineFrameHeightPx = clampTimelineFrameHeightPx(rowsTotalHeightPx);
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
    const rowBounds =
      item.marker.timelineId !== undefined
        ? rowBoundsById.get(item.marker.timelineId)
        : undefined;

    let stemHeightPx: number;
    let hasTargetDot = false;
    let dotY = 0;

    if (item.marker.style === TimelineMarkerStyle.PIN) {
      if (rowBounds) {
        dotY = rowBounds.centerPx;
        stemHeightPx = Math.max(0, dotY - stemTopPx);
        hasTargetDot = true;
      } else {
        stemHeightPx = Math.max(0, stageHeightPx - stemTopPx);
      }
    } else {
      stemHeightPx = Math.max(0, stageHeightPx - stemTopPx);
    }

    return {
      id: item.marker.id,
      badge: item.marker.badge,
      label: item.marker.label,
      timeLabel: item.marker.timeLabel,
      style: item.marker.style,
      paletteId: item.marker.paletteId,
      colors: resolveColors(item.marker.paletteId),
      pillLeftPx: item.pillLeftPx,
      pillTopPx,
      pillWidthPx: item.pillWidthPx,
      pillHeightPx: MARKER_PILL_HEIGHT_PX,
      stemX: item.stemX,
      stemTopPx,
      stemHeightPx,
      hasTargetDot,
      dotY,
      jumpTarget: buildJumpTarget(
        item.marker.timelineId,
        item.marker.revisionIndex,
        item.marker.logId,
      ),
    };
  });
}

/**
 * Collects vertical stem segments from positioned markers for span label obstacle avoidance.
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
 * Collects non-zero vertical drop stem segments from excerpt pin ViewModels for span chip obstacle avoidance.
 */
export function collectExcerptPinVerticalStems(
  pins: readonly ExcerptPinViewModel[],
): VerticalStemSegment[] {
  const stems: VerticalStemSegment[] = [];
  for (const pin of pins) {
    if (pin.stemHeightPx > 0) {
      stems.push({
        x: pin.pinX,
        topPx: pin.stemTopPx,
        bottomPx: pin.stemTopPx + pin.stemHeightPx,
      });
    }
  }
  return stems;
}

/**
 * Computes top and bottom vertical boundaries and row-scoping flag for a timeline span.
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
 * Computes the complete geometry and layout ViewModel for all timeline annotations and deck cards.
 */
export function buildTimelineSceneViewModel(
  input: TimelineSceneLayoutInput,
): TimelineSceneViewModel {
  const measureText = input.measureText ?? defaultTextWidthMeasurer;
  const deckPlan =
    input.deckPlan ??
    planExcerptDeckLayout(input.overlay?.excerpts ?? [], measureText);
  const { indexPaneWidthPx, chartLeftPx, chartViewportWidthPx, stageWidthPx } =
    resolveSceneHorizontalBounds(
      input.indexPaneWidthPx,
      deckPlan.requiredDeckWidthPx,
    );
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
    rowBoundsList,
    rowBoundsById,
  } = computeStageRowLayout(input.rows, markerPlacements.laneCount);
  const resolveColors = createPaletteColorResolver(input.overlay?.palette);

  const markers = buildMarkerViewModels(
    markerPlacements.items,
    stageHeightPx,
    rowBoundsById,
    resolveColors,
  );

  const excerptResult = buildExcerptPinsAndDeck(
    input.overlay?.excerpts ?? [],
    deckPlan,
    stageWidthPx,
    topBandHeightPx,
    stageHeightPx,
    rowBoundsById,
    toStageX,
    resolveColors,
    measureText,
  );

  const verticalStems: VerticalStemSegment[] = [
    ...collectMarkerVerticalStems(markers),
    ...collectExcerptPinVerticalStems(excerptResult.pins),
  ];

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

  const rowHighlightResult = buildRowHighlightViewModels(
    input.overlay?.rowHighlights ?? [],
    Boolean(input.overlay?.dimUnlistedRows),
    indexPaneWidthPx,
    rowBoundsList,
    rowBoundsById,
    resolveColors,
  );

  const connectors = buildConnectorViewModels(
    input.overlay?.connectors ?? [],
    topBandHeightPx,
    rowBoundsById,
    toStageX,
    resolveColors,
  );

  const callouts = buildCalloutViewModels(
    input.overlay?.callouts ?? [],
    chartLeftPx,
    stageWidthPx,
    topBandHeightPx,
    stageHeightPx,
    rowBoundsById,
    toStageX,
    resolveColors,
  );

  const hasOverlayItems =
    markers.length > 0 ||
    spans.length > 0 ||
    rowHighlightResult.highlights.length > 0 ||
    rowHighlightResult.dimmedMasks.length > 0 ||
    connectors.length > 0 ||
    callouts.length > 0 ||
    excerptResult.pins.length > 0;

  return {
    indexPaneWidthPx,
    chartLeftPx,
    chartViewportWidthPx,
    stageWidthPx,
    topBandHeightPx,
    timelineFrameHeightPx,
    stageHeightPx,
    estimatedTotalHeightPx:
      stageHeightPx + excerptResult.deck.estimatedHeightPx,
    hasOverlayItems,
    markers,
    spans,
    rowHighlights: rowHighlightResult.highlights,
    dimmedRowMasks: rowHighlightResult.dimmedMasks,
    connectors,
    callouts,
    excerptPins: excerptResult.pins,
    deck: excerptResult.deck,
  };
}

function buildRowHighlightViewModels(
  rowHighlights: readonly TimelineResolvedRowHighlight[],
  dimUnlistedRows: boolean,
  indexPaneWidthPx: number,
  rowBoundsList: readonly TimelineRowStageBounds[],
  rowBoundsById: ReadonlyMap<number, TimelineRowStageBounds>,
  resolveColors: (paletteId: string) => TimelineResolvedPaletteColors,
): RowHighlightBuildResult {
  const highlights: RowHighlightBandViewModel[] = [];
  const highlightedRowIndices = new Set<number>();

  for (const rh of rowHighlights) {
    const matchedRows: TimelineRowStageBounds[] = [];
    const seenIndices = new Set<number>();
    for (const tid of rh.timelineIds) {
      const bounds = rowBoundsById.get(tid);
      if (bounds && !seenIndices.has(bounds.index)) {
        seenIndices.add(bounds.index);
        matchedRows.push(bounds);
        highlightedRowIndices.add(bounds.index);
      }
    }
    if (matchedRows.length === 0) {
      continue;
    }
    matchedRows.sort((a, b) => a.index - b.index);

    let runStart = matchedRows[0];
    let runEnd = matchedRows[0];
    let runCount = 0;
    const colors = resolveColors(rh.paletteId);

    const flushRun = (): void => {
      highlights.push({
        id: `${rh.id}-run-${runCount}`,
        roleLabel: runCount === 0 ? (rh.roleLabel?.trim() ?? '') : '',
        paletteId: rh.paletteId,
        colors,
        topPx: runStart.topPx,
        heightPx: runEnd.bottomPx - runStart.topPx,
        badgeRightX: indexPaneWidthPx - ROLE_BADGE_RIGHT_INSET_PX,
      });
      runCount++;
    };

    for (let i = 1; i < matchedRows.length; i++) {
      const curr = matchedRows[i];
      if (curr.index === runEnd.index + 1) {
        runEnd = curr;
      } else if (curr.index > runEnd.index + 1) {
        flushRun();
        runStart = curr;
        runEnd = curr;
      }
    }
    flushRun();
  }

  const dimmedMasks: DimmedRowMaskViewModel[] = [];
  if (dimUnlistedRows && highlightedRowIndices.size > 0) {
    let maskStart: TimelineRowStageBounds | null = null;
    let maskEnd: TimelineRowStageBounds | null = null;
    let maskCount = 0;

    for (const bounds of rowBoundsList) {
      if (!highlightedRowIndices.has(bounds.index)) {
        if (!maskStart) {
          maskStart = bounds;
        }
        maskEnd = bounds;
      } else if (maskStart && maskEnd) {
        dimmedMasks.push({
          id: `dim-mask-${maskCount++}`,
          topPx: maskStart.topPx,
          heightPx: maskEnd.bottomPx - maskStart.topPx,
        });
        maskStart = null;
        maskEnd = null;
      }
    }
    if (maskStart && maskEnd) {
      dimmedMasks.push({
        id: `dim-mask-${maskCount}`,
        topPx: maskStart.topPx,
        heightPx: maskEnd.bottomPx - maskStart.topPx,
      });
    }
  }

  return { highlights, dimmedMasks };
}

function buildConnectorViewModels(
  connectors: readonly TimelineResolvedConnector[],
  topBandHeightPx: number,
  rowBoundsById: ReadonlyMap<number, TimelineRowStageBounds>,
  toStageX: (timeMs: number) => number,
  resolveColors: (paletteId: string) => TimelineResolvedPaletteColors,
): ConnectorItemViewModel[] {
  const defaultY = topBandHeightPx + TIMELINE_HEADER_HEIGHT_PX + 24;
  const result: ConnectorItemViewModel[] = [];

  for (const conn of connectors) {
    if (
      (conn.fromTimelineId !== undefined &&
        !rowBoundsById.has(conn.fromTimelineId)) ||
      (conn.toTimelineId !== undefined && !rowBoundsById.has(conn.toTimelineId))
    ) {
      continue;
    }

    const fromX = toStageX(conn.fromTimeMs);
    const toX = toStageX(conn.toTimeMs);
    const fromY =
      conn.fromTimelineId !== undefined
        ? (rowBoundsById.get(conn.fromTimelineId)?.centerPx ?? defaultY)
        : defaultY;
    const toY =
      conn.toTimelineId !== undefined
        ? (rowBoundsById.get(conn.toTimelineId)?.centerPx ?? fromY)
        : fromY;
    const midX = (fromX + toX) / 2;

    let svgPath: string;
    let hasArrowHead = false;
    let arrowHeadPoints = '';
    let labelCenterY = (fromY + toY) / 2;

    if (conn.style === TimelineConnectorStyle.CALIPER) {
      const caliperY = Math.max(
        topBandHeightPx + TIMELINE_HEADER_HEIGHT_PX + 10,
        Math.min(fromY, toY) - 12,
      );
      svgPath = `M ${fromX} ${caliperY - 5} L ${fromX} ${caliperY + 5} M ${toX} ${caliperY - 5} L ${toX} ${caliperY + 5} M ${fromX} ${caliperY} L ${toX} ${caliperY}`;
      labelCenterY = caliperY - 12;
    } else {
      svgPath =
        conn.style === TimelineConnectorStyle.STEP
          ? `M ${fromX} ${fromY} L ${midX} ${fromY} L ${midX} ${toY} L ${toX} ${toY}`
          : `M ${fromX} ${fromY} C ${midX} ${fromY}, ${midX} ${toY}, ${toX} ${toY}`;

      if (fromX !== toX || fromY !== toY) {
        // Evaluates cubic Bezier chord direction near t = 0.85 so wide S-curves and vertical curves orient smoothly.
        const angleRad =
          conn.style === TimelineConnectorStyle.STEP
            ? midX !== toX
              ? Math.atan2(0, toX - midX)
              : Math.atan2(toY - fromY, 0)
            : Math.atan2((toY - fromY) * 0.312, toX - fromX);
        const baseCenterX = toX - Math.cos(angleRad) * 8;
        const baseCenterY = toY - Math.sin(angleRad) * 8;
        const perpX = -Math.sin(angleRad) * 4.5;
        const perpY = Math.cos(angleRad) * 4.5;
        const p1X = Math.round((baseCenterX + perpX) * 10) / 10;
        const p1Y = Math.round((baseCenterY + perpY) * 10) / 10;
        const p2X = Math.round((baseCenterX - perpX) * 10) / 10;
        const p2Y = Math.round((baseCenterY - perpY) * 10) / 10;
        hasArrowHead = true;
        arrowHeadPoints = `${toX},${toY} ${p1X},${p1Y} ${p2X},${p2Y}`;
      }
    }

    result.push({
      id: conn.id,
      badge: conn.badge ?? '',
      label: conn.label ?? '',
      style: conn.style,
      paletteId: conn.paletteId,
      colors: resolveColors(conn.paletteId),
      svgPath,
      hasArrowHead,
      arrowHeadPoints,
      labelCenterX: midX,
      labelCenterY,
    });
  }

  return result;
}

function buildCalloutViewModels(
  callouts: readonly TimelineResolvedCallout[],
  chartLeftPx: number,
  stageWidthPx: number,
  topBandHeightPx: number,
  stageHeightPx: number,
  rowBoundsById: ReadonlyMap<number, TimelineRowStageBounds>,
  toStageX: (timeMs: number) => number,
  resolveColors: (paletteId: string) => TimelineResolvedPaletteColors,
): CalloutItemViewModel[] {
  const chartTopPx = topBandHeightPx + TIMELINE_HEADER_HEIGHT_PX;
  const cardHeightPx = 24;
  const leaderGapPx = 18;
  const minTop = chartTopPx + 4;
  const maxTop = stageHeightPx - cardHeightPx - 4;
  const result: CalloutItemViewModel[] = [];

  for (const callout of callouts) {
    if (
      callout.timelineId !== undefined &&
      !rowBoundsById.has(callout.timelineId)
    ) {
      continue;
    }

    const anchorX = toStageX(callout.timeMs);
    const anchorY =
      callout.timelineId !== undefined
        ? (rowBoundsById.get(callout.timelineId)?.centerPx ?? chartTopPx + 24)
        : chartTopPx + 24;
    const cardWidthPx = clampNumber(
      Math.round(22 + callout.label.length * 6.6),
      88,
      320,
    );
    const minLeft = chartLeftPx + 6;
    const maxLeft = stageWidthPx - cardWidthPx - 8;

    const resolvedPlacement: Exclude<
      TimelineCalloutPlacement,
      TimelineCalloutPlacement.AUTO
    > =
      callout.placement !== TimelineCalloutPlacement.AUTO
        ? callout.placement
        : anchorX + leaderGapPx + cardWidthPx <= stageWidthPx - 10
          ? TimelineCalloutPlacement.RIGHT
          : anchorX - leaderGapPx - cardWidthPx >= chartLeftPx + 10
            ? TimelineCalloutPlacement.LEFT
            : TimelineCalloutPlacement.ABOVE;

    let cardLeftPx: number;
    let cardTopPx: number;
    let attachX: number;
    let attachY: number;

    switch (resolvedPlacement) {
      case TimelineCalloutPlacement.LEFT:
      case TimelineCalloutPlacement.RIGHT: {
        const desiredLeft =
          resolvedPlacement === TimelineCalloutPlacement.LEFT
            ? anchorX - leaderGapPx - cardWidthPx
            : anchorX + leaderGapPx;
        cardLeftPx = Math.round(clampNumber(desiredLeft, minLeft, maxLeft));
        cardTopPx = Math.round(
          clampNumber(anchorY - cardHeightPx / 2, minTop, maxTop),
        );
        attachX = clampNumber(anchorX, cardLeftPx, cardLeftPx + cardWidthPx);
        attachY = cardTopPx + cardHeightPx / 2;
        break;
      }
      case TimelineCalloutPlacement.ABOVE:
      case TimelineCalloutPlacement.BELOW: {
        const desiredTop =
          resolvedPlacement === TimelineCalloutPlacement.ABOVE
            ? anchorY - leaderGapPx - cardHeightPx
            : anchorY + leaderGapPx;
        cardLeftPx = Math.round(
          clampNumber(anchorX - cardWidthPx / 2, minLeft, maxLeft),
        );
        cardTopPx = Math.round(clampNumber(desiredTop, minTop, maxTop));
        attachX = clampNumber(
          anchorX,
          cardLeftPx + 8,
          cardLeftPx + cardWidthPx - 8,
        );
        attachY =
          resolvedPlacement === TimelineCalloutPlacement.ABOVE
            ? cardTopPx + cardHeightPx
            : cardTopPx;
        break;
      }
      default:
        return unreachable(resolvedPlacement);
    }

    result.push({
      id: callout.id,
      label: callout.label,
      dashed: Boolean(callout.dashed),
      paletteId: callout.paletteId,
      colors: resolveColors(callout.paletteId),
      anchorX,
      anchorY,
      leaderPath: `M ${anchorX} ${anchorY} L ${attachX} ${attachY}`,
      cardLeftPx,
      cardTopPx,
      cardWidthPx,
      cardHeightPx,
      jumpTarget: buildJumpTarget(
        callout.timelineId,
        callout.revisionIndex,
        callout.logId,
      ),
    });
  }

  return result;
}

function buildExcerptPinsAndDeck(
  excerpts: readonly TimelineResolvedExcerptCard[],
  deckPlan: ExcerptDeckLayoutPlan,
  stageWidthPx: number,
  topBandHeightPx: number,
  stageHeightPx: number,
  rowBoundsById: ReadonlyMap<number, TimelineRowStageBounds>,
  toStageX: (timeMs: number) => number,
  resolveColors: (paletteId: string) => TimelineResolvedPaletteColors,
  measureText: TextWidthMeasurer,
): ExcerptPinsAndDeckResult {
  const visibleExcerpts = excerpts.slice(0, MAX_EXCERPT_DECK_CARDS);
  if (visibleExcerpts.length === 0 || deckPlan.columns === 0) {
    return {
      pins: [],
      deck: {
        gridPreset: ExcerptDeckGridPreset.EMPTY,
        columns: 0,
        tetherStripHeightPx: 0,
        estimatedHeightPx: 0,
        tethers: [],
        cards: [],
      },
    };
  }

  const columns = deckPlan.columns;
  const usableWidthPx =
    stageWidthPx - DECK_GRID_PADDING_PX * 2 - (columns - 1) * DECK_GRID_GAP_PX;
  const colWidthPx = usableWidthPx / columns;

  const pins: ExcerptPinViewModel[] = [];
  const tethers: ExcerptTetherViewModel[] = [];
  const cards: ExcerptCardViewModel[] = [];
  const cardRowHeights: number[] = [];

  const fallbackPinY = topBandHeightPx + TIMELINE_HEADER_HEIGHT_PX + 20;

  for (let i = 0; i < visibleExcerpts.length; i++) {
    const excerpt = visibleExcerpts[i];
    const colors = resolveColors(excerpt.paletteId);
    const pinX = toStageX(excerpt.timeMs);
    const pinY =
      rowBoundsById.get(excerpt.timelineId)?.centerPx ?? fallbackPinY;
    const jumpTarget: TimelineSessionJumpTarget = {
      timelineId: excerpt.timelineId,
      revisionIndex: excerpt.revisionIndex,
      logId: excerpt.logId,
    };

    const stemTopPx = Math.min(stageHeightPx, pinY + 10);
    pins.push({
      id: excerpt.id,
      badge: excerpt.badge,
      kind: excerpt.kind,
      paletteId: excerpt.paletteId,
      colors,
      pinX,
      pinY,
      stemTopPx,
      stemHeightPx: Math.max(0, stageHeightPx - stemTopPx),
      jumpTarget,
    });

    const colIndex = i % columns;
    const cardAnchorX =
      Math.round(
        (DECK_GRID_PADDING_PX +
          colIndex * (colWidthPx + DECK_GRID_GAP_PX) +
          colWidthPx / 2) *
          10,
      ) / 10;
    const ctrlY1 = Math.round(DECK_TETHER_STRIP_HEIGHT_PX * 0.56);
    const ctrlY2 = Math.round(DECK_TETHER_STRIP_HEIGHT_PX * 0.44);
    tethers.push({
      id: `tether-${excerpt.id}`,
      badge: excerpt.badge,
      paletteId: excerpt.paletteId,
      colors,
      svgPath: `M ${pinX} 0 C ${pinX} ${ctrlY1}, ${cardAnchorX} ${ctrlY2}, ${cardAnchorX} ${DECK_TETHER_STRIP_HEIGHT_PX}`,
      pinX,
      cardAnchorX,
    });

    const formattedLines = excerpt.lines.map((line, lineIdx) =>
      formatExcerptLine(line, lineIdx),
    );
    const logTimestamp = excerpt.logTimestamp ?? '';
    const insertId = excerpt.insertId ?? '';
    const hasQueryMetadata = logTimestamp !== '' || insertId !== '';
    cards.push({
      id: excerpt.id,
      gridArea: `card-${i}`,
      badge: excerpt.badge,
      kind: excerpt.kind,
      paletteId: excerpt.paletteId,
      colors,
      timeLabel: excerpt.timeLabel,
      resourcePath: excerpt.resourcePath,
      title: excerpt.title,
      subtitle: excerpt.subtitle,
      logTimestamp,
      insertId,
      hasQueryMetadata,
      lines: formattedLines,
      note: excerpt.note ?? '',
      actionLabel: getExcerptActionLabel(excerpt.kind),
      jumpTarget,
    });

    const gridRowIndex = Math.floor(i / columns);
    const cardEstimatedHeightPx = estimateExcerptCardHeightPx(
      excerpt,
      colWidthPx,
      hasQueryMetadata,
      measureText,
    );
    cardRowHeights[gridRowIndex] = Math.max(
      cardRowHeights[gridRowIndex] ?? 0,
      cardEstimatedHeightPx,
    );
  }

  let cardsGridHeightPx = DECK_GRID_PADDING_PX;
  for (let r = 0; r < cardRowHeights.length; r++) {
    cardsGridHeightPx += cardRowHeights[r];
    if (r > 0) {
      cardsGridHeightPx += DECK_GRID_GAP_PX;
    }
  }

  return {
    pins,
    deck: {
      gridPreset: deckPlan.preset,
      columns,
      tetherStripHeightPx: DECK_TETHER_STRIP_HEIGHT_PX,
      estimatedHeightPx: DECK_TETHER_STRIP_HEIGHT_PX + cardsGridHeightPx,
      tethers,
      cards,
    },
  };
}

function formatExcerptLine(
  line: TimelineExcerptLine,
  index: number,
): ExcerptLineViewModel {
  const parts: readonly TimelineExcerptLinePart[] =
    line.parts && line.parts.length > 0
      ? line.parts
      : [{ text: line.text, matched: false }];

  let prefixSymbol = '';
  if (line.kind === TimelineExcerptLineKind.ADDED) {
    prefixSymbol = '+';
  } else if (line.kind === TimelineExcerptLineKind.DELETED) {
    prefixSymbol = '-';
  } else if (line.kind === TimelineExcerptLineKind.CONTEXT) {
    prefixSymbol = ' ';
  }

  return {
    index,
    kind: line.kind,
    gutterOld:
      line.oldLineNum !== undefined && line.oldLineNum > 0
        ? String(line.oldLineNum)
        : '',
    gutterNew:
      line.newLineNum !== undefined && line.newLineNum > 0
        ? String(line.newLineNum)
        : '',
    prefixSymbol,
    text: line.text,
    parts,
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
