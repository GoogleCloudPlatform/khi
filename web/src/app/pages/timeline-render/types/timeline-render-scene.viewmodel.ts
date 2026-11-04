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
import { StyleStoreLike } from 'src/app/store/domain/style-store';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { BASE_ROW_HEIGHT } from 'src/app/timeline/components/style-model';

/**
 * Height of the sticky timestamp ruler header inside TimelineFrameComponent in pixels.
 */
export const TIMELINE_HEADER_HEIGHT_PX = 60;

/**
 * Width of the left resource hierarchy index pane inside TimelineFrameComponent in pixels.
 */
export const TIMELINE_INDEX_PANE_WIDTH_PX = 300;

/**
 * Width of the resizer gutter between the index pane and chart canvas in pixels.
 */
export const TIMELINE_GUTTER_WIDTH_PX = 8;

/**
 * Left X offset where the timeline chart area begins in stage coordinates.
 */
export const TIMELINE_CHART_LEFT_PX =
  TIMELINE_INDEX_PANE_WIDTH_PX + TIMELINE_GUTTER_WIDTH_PX;

const MARKER_LANE_STEP_PX = 30;
const MARKER_PILL_HEIGHT_PX = 24;
const MARKER_BAND_VERTICAL_PADDING_PX = 12;
const DECK_TETHER_STRIP_HEIGHT_PX = 32;
const DECK_GRID_PADDING_PX = 12;
const DECK_GRID_GAP_PX = 12;
const MAX_EXCERPT_DECK_CARDS = 6;

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
  readonly stageWidthPx: number;
  readonly viewportLeftTimeMs: number;
  readonly pixelsPerMs: number;
  readonly rows: readonly TimelineRowMetric[];
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
 * Positioned focal subtree highlight band and left-gutter role badge for row highlights.
 */
export interface RowHighlightBandViewModel {
  readonly id: string;
  readonly roleLabel: string;
  readonly paletteId: string;
  readonly colors: TimelineResolvedPaletteColors;
  readonly topPx: number;
  readonly heightPx: number;
  readonly badgeRightX: number;
}

/**
 * Positioned translucent mask over contiguous background rows when dimUnlistedRows is active.
 */
export interface DimmedRowMaskViewModel {
  readonly id: string;
  readonly topPx: number;
  readonly heightPx: number;
}

/**
 * Positioned SVG path, polygon arrowhead, and midpoint label chip for timeline connectors.
 */
export interface ConnectorItemViewModel {
  readonly id: string;
  readonly badge: string;
  readonly label: string;
  readonly style: TimelineConnectorStyle;
  readonly paletteId: string;
  readonly colors: TimelineResolvedPaletteColors;
  readonly svgPath: string;
  readonly hasArrowHead: boolean;
  readonly arrowHeadPoints: string;
  readonly labelCenterX: number;
  readonly labelCenterY: number;
}

/**
 * Positioned compact single-line inline callout card and leader line for timeline callouts.
 */
export interface CalloutItemViewModel {
  readonly id: string;
  readonly label: string;
  readonly dashed: boolean;
  readonly paletteId: string;
  readonly colors: TimelineResolvedPaletteColors;
  readonly anchorX: number;
  readonly anchorY: number;
  readonly leaderPath: string;
  readonly cardLeftPx: number;
  readonly cardTopPx: number;
  readonly cardWidthPx: number;
  readonly cardHeightPx: number;
  readonly jumpTarget: TimelineSessionJumpTarget | null;
}

/**
 * Positioned badge pin and vertical drop stem on the timeline canvas for evidence excerpts.
 */
export interface ExcerptPinViewModel {
  readonly id: string;
  readonly badge: string;
  readonly kind: TimelineExcerptKind;
  readonly paletteId: string;
  readonly colors: TimelineResolvedPaletteColors;
  readonly pinX: number;
  readonly pinY: number;
  readonly stemTopPx: number;
  readonly stemHeightPx: number;
  readonly jumpTarget: TimelineSessionJumpTarget;
}

/**
 * Formatted display row inside an evidence deck card with normalized highlight parts.
 */
export interface ExcerptLineViewModel {
  readonly index: number;
  readonly kind: TimelineExcerptLineKind;
  readonly gutterOld: string;
  readonly gutterNew: string;
  readonly prefixSymbol: string;
  readonly text: string;
  readonly parts: readonly TimelineExcerptLinePart[];
}

/**
 * ViewModel for a single bottom evidence deck diff or log card.
 */
export interface ExcerptCardViewModel {
  readonly id: string;
  readonly gridArea: string;
  readonly badge: string;
  readonly kind: TimelineExcerptKind;
  readonly paletteId: string;
  readonly colors: TimelineResolvedPaletteColors;
  readonly timeLabel: string;
  readonly resourcePath: string;
  readonly title: string;
  readonly subtitle: string;
  readonly logTimestamp: string;
  readonly insertId: string;
  readonly hasQueryMetadata: boolean;
  readonly lines: readonly ExcerptLineViewModel[];
  readonly note: string;
  readonly actionLabel: string;
  readonly jumpTarget: TimelineSessionJumpTarget;
}

/**
 * S-curve cubic Bezier SVG tether connecting a timeline pin drop stem to its deck card column.
 */
export interface ExcerptTetherViewModel {
  readonly id: string;
  readonly badge: string;
  readonly paletteId: string;
  readonly colors: TimelineResolvedPaletteColors;
  readonly svgPath: string;
  readonly pinX: number;
  readonly cardAnchorX: number;
}

/**
 * ViewModel for the bottom synchronized evidence deck and its 32px S-curve tether channel.
 */
export interface ExcerptDeckViewModel {
  readonly gridPreset: ExcerptDeckGridPreset;
  readonly columns: number;
  readonly tetherStripHeightPx: number;
  readonly estimatedHeightPx: number;
  readonly tethers: readonly ExcerptTetherViewModel[];
  readonly cards: readonly ExcerptCardViewModel[];
}

/**
 * Complete layout ViewModel for the timeline stage overlay and bottom evidence deck.
 */
export interface TimelineSceneViewModel {
  readonly stageWidthPx: number;
  readonly topBandHeightPx: number;
  readonly timelineFrameHeightPx: number;
  readonly stageHeightPx: number;
  readonly estimatedTotalHeightPx: number;
  readonly hasOverlayItems: boolean;
  readonly markers: readonly MarkerItemViewModel[];
  readonly spans: readonly SpanItemViewModel[];
  readonly rowHighlights: readonly RowHighlightBandViewModel[];
  readonly dimmedRowMasks: readonly DimmedRowMaskViewModel[];
  readonly connectors: readonly ConnectorItemViewModel[];
  readonly callouts: readonly CalloutItemViewModel[];
  readonly excerptPins: readonly ExcerptPinViewModel[];
  readonly deck: ExcerptDeckViewModel;
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

interface RowHighlightBuildResult {
  readonly highlights: readonly RowHighlightBandViewModel[];
  readonly dimmedMasks: readonly DimmedRowMaskViewModel[];
}

interface ExcerptPinsAndDeckResult {
  readonly pins: readonly ExcerptPinViewModel[];
  readonly deck: ExcerptDeckViewModel;
}

interface DeckGridPresetSelection {
  readonly preset: ExcerptDeckGridPreset;
  readonly columns: number;
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
 * Computes the complete geometry and layout ViewModel for all timeline annotations and deck cards.
 */
export function buildTimelineSceneViewModel(
  input: TimelineSceneLayoutInput,
): TimelineSceneViewModel {
  const stageWidthPx = Math.max(
    TIMELINE_CHART_LEFT_PX + 100,
    input.stageWidthPx,
  );
  const chartWidthPx = stageWidthPx - TIMELINE_CHART_LEFT_PX;
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
      chartWidthPx,
    );
    return Math.round((TIMELINE_CHART_LEFT_PX + clampedX) * 10) / 10;
  };

  const markerPlacements = packMarkerLanes(
    overlay?.markers ?? [],
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

  const rowBoundsList: RowStageBounds[] = [];
  const rowBoundsById = new Map<number, RowStageBounds>();
  let cursorY = topBandHeightPx + TIMELINE_HEADER_HEIGHT_PX;
  for (let i = 0; i < input.rows.length; i++) {
    const row = input.rows[i];
    const bounds: RowStageBounds = {
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

  const excerptResult = buildExcerptPinsAndDeck(
    overlay?.excerpts ?? [],
    stageWidthPx,
    topBandHeightPx,
    stageHeightPx,
    rowBoundsById,
    toStageX,
    resolveColors,
  );

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
  for (const pin of excerptResult.pins) {
    if (pin.stemHeightPx > 0) {
      verticalStems.push({
        x: pin.pinX,
        topPx: pin.stemTopPx,
        bottomPx: pin.stemTopPx + pin.stemHeightPx,
      });
    }
  }

  const spans = buildSpanViewModels(
    overlay?.spans ?? [],
    stageWidthPx,
    topBandHeightPx,
    stageHeightPx,
    rowBoundsById,
    toStageX,
    resolveColors,
    verticalStems,
  );

  const rowHighlightResult = buildRowHighlightViewModels(
    overlay?.rowHighlights ?? [],
    Boolean(overlay?.dimUnlistedRows),
    rowBoundsList,
    rowBoundsById,
    resolveColors,
  );

  const connectors = buildConnectorViewModels(
    overlay?.connectors ?? [],
    topBandHeightPx,
    rowBoundsById,
    toStageX,
    resolveColors,
  );

  const callouts = buildCalloutViewModels(
    overlay?.callouts ?? [],
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
      TIMELINE_CHART_LEFT_PX,
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

function buildRowHighlightViewModels(
  rowHighlights: readonly TimelineResolvedRowHighlight[],
  dimUnlistedRows: boolean,
  rowBoundsList: readonly RowStageBounds[],
  rowBoundsById: ReadonlyMap<number, RowStageBounds>,
  resolveColors: (paletteId: string) => TimelineResolvedPaletteColors,
): RowHighlightBuildResult {
  const highlights: RowHighlightBandViewModel[] = [];
  const highlightedRowIndices = new Set<number>();

  for (const rh of rowHighlights) {
    const matchedRows: RowStageBounds[] = [];
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
        roleLabel: runCount === 0 ? (rh.roleLabel ?? '') : '',
        paletteId: rh.paletteId,
        colors,
        topPx: runStart.topPx,
        heightPx: runEnd.bottomPx - runStart.topPx,
        badgeRightX: TIMELINE_INDEX_PANE_WIDTH_PX - 6,
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
    let maskStart: RowStageBounds | null = null;
    let maskEnd: RowStageBounds | null = null;
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
  rowBoundsById: ReadonlyMap<number, RowStageBounds>,
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
  stageWidthPx: number,
  topBandHeightPx: number,
  stageHeightPx: number,
  rowBoundsById: ReadonlyMap<number, RowStageBounds>,
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
    const minLeft = TIMELINE_CHART_LEFT_PX + 6;
    const maxLeft = stageWidthPx - cardWidthPx - 8;

    const resolvedPlacement: Exclude<
      TimelineCalloutPlacement,
      TimelineCalloutPlacement.AUTO
    > =
      callout.placement !== TimelineCalloutPlacement.AUTO
        ? callout.placement
        : anchorX + leaderGapPx + cardWidthPx <= stageWidthPx - 10
          ? TimelineCalloutPlacement.RIGHT
          : anchorX - leaderGapPx - cardWidthPx >= TIMELINE_CHART_LEFT_PX + 10
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
  stageWidthPx: number,
  topBandHeightPx: number,
  stageHeightPx: number,
  rowBoundsById: ReadonlyMap<number, RowStageBounds>,
  toStageX: (timeMs: number) => number,
  resolveColors: (paletteId: string) => TimelineResolvedPaletteColors,
): ExcerptPinsAndDeckResult {
  const visibleExcerpts = excerpts.slice(0, MAX_EXCERPT_DECK_CARDS);
  if (visibleExcerpts.length === 0) {
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

  const gridConfig = selectDeckGridPreset(visibleExcerpts.length, stageWidthPx);
  const columns = gridConfig.columns;
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
      actionLabel:
        excerpt.kind === TimelineExcerptKind.DIFF ? 'Open Diff' : 'Open Log',
      jumpTarget,
    });

    const gridRowIndex = Math.floor(i / columns);
    const cardEstimatedHeightPx =
      52 +
      (hasQueryMetadata ? 20 : 0) +
      Math.max(1, formattedLines.length) * 20 +
      (excerpt.note ? 30 : 0) +
      16;
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
      gridPreset: gridConfig.preset,
      columns,
      tetherStripHeightPx: DECK_TETHER_STRIP_HEIGHT_PX,
      estimatedHeightPx: DECK_TETHER_STRIP_HEIGHT_PX + cardsGridHeightPx,
      tethers,
      cards,
    },
  };
}

function selectDeckGridPreset(
  cardCount: number,
  stageWidthPx: number,
): DeckGridPresetSelection {
  if (cardCount <= 1) {
    return { preset: ExcerptDeckGridPreset.COLS_1, columns: 1 };
  }
  if (cardCount === 2) {
    return { preset: ExcerptDeckGridPreset.COLS_2, columns: 2 };
  }
  if (cardCount === 3) {
    return { preset: ExcerptDeckGridPreset.COLS_3, columns: 3 };
  }
  if (cardCount === 4) {
    if (stageWidthPx >= 1440) {
      return { preset: ExcerptDeckGridPreset.COLS_4, columns: 4 };
    }
    return { preset: ExcerptDeckGridPreset.GRID_2X2, columns: 2 };
  }
  return { preset: ExcerptDeckGridPreset.GRID_3X2, columns: 3 };
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

function clampNumber(value: number, min: number, max: number): number {
  if (max < min) {
    return min;
  }
  return Math.max(min, Math.min(max, value));
}
