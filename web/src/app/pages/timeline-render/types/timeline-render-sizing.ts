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

import { InjectionToken } from '@angular/core';
import {
  TimelineExcerptKind,
  TimelineResolvedExcerptCard,
  TimelineResolvedRowHighlight,
  TimelineSceneOverlay,
} from 'src/app/common/schema/timeline-render-types';
import { StyleStoreLike } from 'src/app/store/domain/style-store';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { BASE_ROW_HEIGHT } from 'src/app/timeline/components/style-model';
import { clampNumber } from 'src/app/utils/math-util';

/**
 * Height of the sticky timestamp ruler header inside TimelineFrameComponent in CSS pixels.
 */
export const TIMELINE_HEADER_HEIGHT_PX = 60;

/**
 * Width of the resizer gutter between the left index pane and chart canvas in CSS pixels.
 */
export const TIMELINE_GUTTER_WIDTH_PX = 8;

/**
 * Minimum width of the left resource index pane in CSS pixels so the corner indicator never clips.
 */
export const MIN_INDEX_PANE_WIDTH_PX = 300;

/**
 * Maximum width of the left resource index pane in CSS pixels to prevent pathological labels from dominating the frame.
 */
export const MAX_INDEX_PANE_WIDTH_PX = 1200;

/**
 * Minimum width of the timeline chart viewport in CSS pixels so timestamp ticks remain well spaced.
 */
export const MIN_CHART_VIEWPORT_WIDTH_PX = 1280;

/**
 * Maximum total stage width in CSS pixels so Retina 2x rendering stays within the 8192px WebGL backing store limit.
 */
export const MAX_STAGE_WIDTH_PX = 4096;

/**
 * Horizontal overdraw margin per side in CSS pixels for static timeline renders so chart and ruler canvases stay under 8192px at 2x DPR.
 */
export const TIMELINE_RENDER_HORIZONTAL_OVERDRAW_PX = 120;

/**
 * Horizontal inset from the right edge of the index pane to a row highlight role badge in CSS pixels.
 */
export const ROLE_BADGE_RIGHT_INSET_PX = 6;

/**
 * Minimum width of a single evidence deck card in CSS pixels so badges, paths, and actions remain legible.
 */
export const MIN_DECK_CARD_WIDTH_PX = 380;

/**
 * Maximum width allocated to a single evidence deck card in CSS pixels so pathological lines wrap instead of stretching the stage.
 */
export const MAX_DECK_CARD_WIDTH_PX = 1000;

/**
 * Horizontal and bottom padding around the evidence deck CSS grid in CSS pixels.
 */
export const DECK_GRID_PADDING_PX = 12;

/**
 * Horizontal gap between adjacent card columns in the evidence deck grid in CSS pixels.
 */
export const DECK_GRID_GAP_PX = 12;

/**
 * Vertical gap between wrapped card rows in the evidence deck grid in CSS pixels so routed tethers remain clearly visible.
 */
export const DECK_GRID_ROW_GAP_PX = 24;

/**
 * Maximum number of evidence excerpt cards rendered in the bottom deck.
 */
export const MAX_EXCERPT_DECK_CARDS = 6;

/**
 * Layout preset identifier for the bottom evidence deck CSS grid.
 */
export enum ExcerptDeckGridPreset {
  /** Empty deck when no evidence cards are rendered. */
  EMPTY = 'empty',
  /** Single full-width column layout for 1 card. */
  COLS_1 = 'cols-1',
  /** Two-column side-by-side layout for 2 cards. */
  COLS_2 = 'cols-2',
  /** Three-column side-by-side layout for 3 cards. */
  COLS_3 = 'cols-3',
  /** Four-column single-row layout when 4 cards fit without wrapping inside the stage limit. */
  COLS_4 = 'cols-4',
  /** Two-by-two wrapped grid layout when 4 wide cards exceed a single horizontal row. */
  GRID_2X2 = 'grid-2x2',
  /** Three-by-two grid layout for 5 or 6 cards. */
  GRID_3X2 = 'grid-3x2',
}

/**
 * Clamped horizontal stage, index pane, gutter boundary, and chart viewport widths in CSS pixels.
 */
export interface HorizontalStageGeometry {
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
 * Planned column geometry and minimum horizontal stage width for the bottom evidence deck.
 */
export interface ExcerptDeckLayoutPlan {
  /** Selected CSS grid preset for the visible evidence deck cards. */
  readonly preset: ExcerptDeckGridPreset;
  /** Number of horizontal columns in the evidence deck grid, or 0 when empty. */
  readonly columns: number;
  /** Minimum horizontal stage width in CSS pixels required to fit the deck columns cleanly. */
  readonly requiredDeckWidthPx: number;
}

const INDEX_ROW_LEFT_BORDER_PX = 5;
const INDEX_ROW_BASE_LEFT_PADDING_PX = 4;
const INDEX_ROW_INDENT_PER_LAYER_PX = 4;
const INDEX_ROW_RIGHT_PADDING_PX = 8;
const INDEX_ROW_GRID_GAP_PX = 4;
const INDEX_ROW_GRID_COLUMN_GAPS = 3;
const INDEX_ICON_MIN_PX = 12;
const INDEX_ICON_MAX_PX = 18;
const INDEX_ICON_HEIGHT_RATIO = 0.64;
const INDEX_NAME_FONT_MIN_PX = 10;
const INDEX_NAME_FONT_MAX_PX = 14;
const INDEX_NAME_FONT_HEIGHT_RATIO = 0.52;
const INDEX_NAME_FONT_WEIGHT = 400;
const INDEX_CHIP_FONT_MIN_PX = 8;
const INDEX_CHIP_FONT_MAX_PX = 10;
const INDEX_CHIP_FONT_HEIGHT_RATIO = 0.4;
const INDEX_CHIP_FONT_WEIGHT = 500;
const INDEX_CHIP_MIN_CONTENT_WIDTH_PX = 24;
const INDEX_CHIP_HORIZONTAL_CHROME_PX = 12;
const INDEX_CHIP_LETTER_SPACING_PX = 0.5;
const INDEX_NAME_TO_CHIP_GAP_PX = 8;
const INDEX_SUBPIXEL_SAFETY_PX = 4;
const ROLE_BADGE_FONT = '500 10px Roboto, sans-serif';
const ROLE_BADGE_LETTER_SPACING_PX = 0.5;
const ROLE_BADGE_HORIZONTAL_CHROME_PX = 14;
const ROLE_BADGE_GAP_FROM_LABEL_PX = 8;
const DECK_HEADER_BADGE_FONT = '500 10px Roboto, sans-serif';
const DECK_HEADER_KIND_FONT = '500 9px Roboto, sans-serif';
const DECK_HEADER_PATH_FONT = '400 11px "Google Sans Code", monospace';
const DECK_HEADER_TIME_FONT = '400 10px "Google Sans Code", monospace';
const DECK_HEADER_ACTION_FONT = '500 10px Roboto, sans-serif';
const DECK_TITLE_FONT = '500 11px Roboto, sans-serif';
const DECK_SUBTITLE_FONT = '400 11px Roboto, sans-serif';
const DECK_META_FONT = '400 10px "Google Sans Code", monospace';
const DECK_CODE_FONT = '400 11px "Google Sans Code", monospace';
const DECK_NOTE_FONT = '400 11px Roboto, sans-serif';
const DECK_CARD_HORIZONTAL_CHROME_PX = 26;
const DECK_HEADER_GAPS_AND_PILLS_CHROME_PX = 64;
const DECK_DIFF_LINE_CHROME_PX = 106;
const DECK_LOG_LINE_CHROME_PX = 44;
const DECK_NOTE_TAG_AND_GAP_PX = 46;
const DECK_CARD_SUBPIXEL_SAFETY_PX = 4;
const FALLBACK_CHAR_WIDTH_RATIO = 0.6;

/**
 * Measures rendered single-line text width in CSS pixels for a CSS font shorthand specification.
 */
export type TextWidthMeasurer = (text: string, font: string) => number;

/**
 * Injection token supplying the TextWidthMeasurer used to fit timeline render stages to content.
 */
export const TIMELINE_TEXT_WIDTH_MEASURER =
  new InjectionToken<TextWidthMeasurer>('TIMELINE_TEXT_WIDTH_MEASURER', {
    providedIn: 'root',
    factory: createCanvasTextWidthMeasurer,
  });

/**
 * Fitted CSS pixel dimensions for the rendered timeline stage and its internal panes.
 */
export interface TimelineStageDimensions {
  /** Fitted width of the left resource hierarchy index pane in CSS pixels. */
  readonly indexPaneWidthPx: number;
  /** Width of the right timeline chart viewport in CSS pixels. */
  readonly chartViewportWidthPx: number;
  /** Total horizontal stage width in CSS pixels across index pane, gutter, and chart viewport. */
  readonly stageWidthPx: number;
  /** Total vertical height of the timeline frame including ruler header and rows in CSS pixels. */
  readonly timelineFrameHeightPx: number;
}

/**
 * Creates a TextWidthMeasurer backed by an offscreen HTMLCanvasElement 2D context.
 */
export function createCanvasTextWidthMeasurer(): TextWidthMeasurer {
  let ctx: CanvasRenderingContext2D | null | undefined;
  let currentFont = '';
  return (text: string, font: string): number => {
    if (text.length === 0) {
      return 0;
    }
    if (ctx === undefined) {
      ctx =
        typeof document !== 'undefined'
          ? document.createElement('canvas').getContext('2d')
          : null;
    }
    if (ctx) {
      if (currentFont !== font) {
        ctx.font = font;
        currentFont = font;
      }
      return ctx.measureText(text).width;
    }
    const fontSizeMatch = /(\d+(?:\.\d+)?)px/.exec(font);
    const fontSizePx = fontSizeMatch ? Number(fontSizeMatch[1]) : 12;
    return text.length * fontSizePx * FALLBACK_CHAR_WIDTH_RATIO;
  };
}

/**
 * Calculates the fitted left index pane width in CSS pixels to hold rendered timeline rows and primary role badges without truncation.
 */
export function calculateIndexPaneWidthPx(
  timelines: readonly ReadonlyDomainElement<Timeline>[],
  styleStore: StyleStoreLike,
  measureText: TextWidthMeasurer,
  rowHighlights?: readonly TimelineResolvedRowHighlight[],
): number {
  let requiredWidthPx = MIN_INDEX_PANE_WIDTH_PX;
  const roleBadgeWidthByRowIndex = collectPrimaryRoleBadgeWidthsByRowIndex(
    timelines,
    rowHighlights,
    measureText,
  );

  for (let i = 0; i < timelines.length; i++) {
    const timeline = timelines[i];
    const rowHeightPx = resolveTimelineRowHeightPx(timeline, styleStore);
    const leftPaddingPx = Math.max(
      0,
      INDEX_ROW_BASE_LEFT_PADDING_PX +
        (timeline.layer - 1) * INDEX_ROW_INDENT_PER_LAYER_PX,
    );
    const iconSizePx = clampNumber(
      rowHeightPx * INDEX_ICON_HEIGHT_RATIO,
      INDEX_ICON_MIN_PX,
      INDEX_ICON_MAX_PX,
    );
    const nameFontPx = clampNumber(
      rowHeightPx * INDEX_NAME_FONT_HEIGHT_RATIO,
      INDEX_NAME_FONT_MIN_PX,
      INDEX_NAME_FONT_MAX_PX,
    );
    const nameWidthPx = measureText(
      timeline.name,
      `${INDEX_NAME_FONT_WEIGHT} ${nameFontPx}px Roboto, sans-serif`,
    );

    let chipBlockWidthPx = 0;
    const chipLabel = timeline.type.label;
    if (chipLabel) {
      const chipFontPx = clampNumber(
        rowHeightPx * INDEX_CHIP_FONT_HEIGHT_RATIO,
        INDEX_CHIP_FONT_MIN_PX,
        INDEX_CHIP_FONT_MAX_PX,
      );
      const chipTextWidthPx =
        measureText(
          chipLabel,
          `${INDEX_CHIP_FONT_WEIGHT} ${chipFontPx}px Roboto, sans-serif`,
        ) +
        chipLabel.length * INDEX_CHIP_LETTER_SPACING_PX;
      chipBlockWidthPx =
        INDEX_NAME_TO_CHIP_GAP_PX +
        Math.max(INDEX_CHIP_MIN_CONTENT_WIDTH_PX, chipTextWidthPx) +
        INDEX_CHIP_HORIZONTAL_CHROME_PX;
    }

    const roleBadgeWidthPx = roleBadgeWidthByRowIndex.get(i) ?? 0;
    const rightEdgeAllowancePx =
      roleBadgeWidthPx > 0
        ? ROLE_BADGE_GAP_FROM_LABEL_PX +
          roleBadgeWidthPx +
          ROLE_BADGE_RIGHT_INSET_PX
        : INDEX_ROW_RIGHT_PADDING_PX;

    const rowRequiredWidthPx = Math.ceil(
      INDEX_ROW_LEFT_BORDER_PX +
        leftPaddingPx +
        iconSizePx * 3 +
        INDEX_ROW_GRID_GAP_PX * INDEX_ROW_GRID_COLUMN_GAPS +
        nameWidthPx +
        chipBlockWidthPx +
        rightEdgeAllowancePx +
        INDEX_SUBPIXEL_SAFETY_PX,
    );
    requiredWidthPx = Math.max(requiredWidthPx, rowRequiredWidthPx);
  }

  return clampNumber(
    requiredWidthPx,
    MIN_INDEX_PANE_WIDTH_PX,
    MAX_INDEX_PANE_WIDTH_PX,
  );
}

/**
 * Returns the interactive jump button label for a diff or log evidence deck card.
 */
export function getExcerptActionLabel(kind: TimelineExcerptKind): string {
  return kind === TimelineExcerptKind.DIFF ? 'Open Diff' : 'Open Log';
}

/**
 * Measures the unclamped single-line horizontal width of an evidence excerpt card in CSS pixels.
 */
export function measureExcerptCardUnclampedWidthPx(
  card: TimelineResolvedExcerptCard,
  measureText: TextWidthMeasurer,
): number {
  const isDiff = card.kind === TimelineExcerptKind.DIFF;
  const kindTag = isDiff ? 'DIFF' : 'LOG';
  const actionLabel = getExcerptActionLabel(card.kind);

  const headerTopRowWidthPx =
    measureText(card.badge, DECK_HEADER_BADGE_FONT) +
    measureText(kindTag, DECK_HEADER_KIND_FONT) +
    measureText(card.resourcePath, DECK_HEADER_PATH_FONT) +
    measureText(card.timeLabel, DECK_HEADER_TIME_FONT) +
    measureText(actionLabel, DECK_HEADER_ACTION_FONT) +
    DECK_HEADER_GAPS_AND_PILLS_CHROME_PX +
    DECK_CARD_HORIZONTAL_CHROME_PX;

  const subtitleExtraPx = card.subtitle
    ? 8 + measureText(card.subtitle, DECK_SUBTITLE_FONT)
    : 0;
  const titleRowWidthPx =
    measureText(card.title, DECK_TITLE_FONT) +
    subtitleExtraPx +
    DECK_CARD_HORIZONTAL_CHROME_PX;

  let maxRowWidthPx = Math.max(headerTopRowWidthPx, titleRowWidthPx);

  const metaSegments: string[] = [];
  if (card.logTimestamp) {
    metaSegments.push(`timestamp=${card.logTimestamp}`);
  }
  if (card.insertId) {
    metaSegments.push(`insertId=${card.insertId}`);
  }
  if (metaSegments.length > 0) {
    const metaRowWidthPx =
      measureText(metaSegments.join('  '), DECK_META_FONT) +
      DECK_CARD_HORIZONTAL_CHROME_PX;
    maxRowWidthPx = Math.max(maxRowWidthPx, metaRowWidthPx);
  }

  const lineChromePx = isDiff
    ? DECK_DIFF_LINE_CHROME_PX
    : DECK_LOG_LINE_CHROME_PX;
  for (const line of card.lines) {
    const lineRowWidthPx =
      measureText(line.text, DECK_CODE_FONT) + lineChromePx;
    maxRowWidthPx = Math.max(maxRowWidthPx, lineRowWidthPx);
  }

  if (card.note) {
    const noteRowWidthPx =
      DECK_NOTE_TAG_AND_GAP_PX +
      measureText(card.note, DECK_NOTE_FONT) +
      DECK_CARD_HORIZONTAL_CHROME_PX;
    maxRowWidthPx = Math.max(maxRowWidthPx, noteRowWidthPx);
  }

  return Math.max(
    MIN_DECK_CARD_WIDTH_PX,
    Math.ceil(maxRowWidthPx + DECK_CARD_SUBPIXEL_SAFETY_PX),
  );
}

/**
 * Plans the bottom evidence deck grid preset, column count, and required horizontal stage width in CSS pixels.
 */
export function planExcerptDeckLayout(
  excerpts: readonly TimelineResolvedExcerptCard[],
  measureText: TextWidthMeasurer,
): ExcerptDeckLayoutPlan {
  const visible = excerpts.slice(0, MAX_EXCERPT_DECK_CARDS);
  if (visible.length === 0) {
    return {
      preset: ExcerptDeckGridPreset.EMPTY,
      columns: 0,
      requiredDeckWidthPx: 0,
    };
  }

  let rawMaxCardWidthPx = MIN_DECK_CARD_WIDTH_PX;
  for (const card of visible) {
    rawMaxCardWidthPx = Math.max(
      rawMaxCardWidthPx,
      measureExcerptCardUnclampedWidthPx(card, measureText),
    );
  }
  const clampedMaxCardWidthPx = Math.min(
    rawMaxCardWidthPx,
    MAX_DECK_CARD_WIDTH_PX,
  );

  let preset: ExcerptDeckGridPreset;
  let columns: number;
  switch (visible.length) {
    case 1:
      preset = ExcerptDeckGridPreset.COLS_1;
      columns = 1;
      break;
    case 2:
      preset = ExcerptDeckGridPreset.COLS_2;
      columns = 2;
      break;
    case 3:
      preset = ExcerptDeckGridPreset.COLS_3;
      columns = 3;
      break;
    case 4: {
      // Uses a single 4-column row only when unclamped cards fit without wrapping and stay inside the 4096px @2x stage limit.
      const fourColUnclampedWidthPx =
        DECK_GRID_PADDING_PX * 2 + 4 * rawMaxCardWidthPx + 3 * DECK_GRID_GAP_PX;
      if (
        rawMaxCardWidthPx <= MAX_DECK_CARD_WIDTH_PX &&
        fourColUnclampedWidthPx <= MAX_STAGE_WIDTH_PX
      ) {
        preset = ExcerptDeckGridPreset.COLS_4;
        columns = 4;
      } else {
        preset = ExcerptDeckGridPreset.GRID_2X2;
        columns = 2;
      }
      break;
    }
    default:
      preset = ExcerptDeckGridPreset.GRID_3X2;
      columns = 3;
      break;
  }

  const requiredDeckWidthPx = Math.ceil(
    DECK_GRID_PADDING_PX * 2 +
      columns * clampedMaxCardWidthPx +
      (columns - 1) * DECK_GRID_GAP_PX,
  );

  return {
    preset,
    columns,
    requiredDeckWidthPx,
  };
}

/**
 * Estimates the vertical height of an evidence excerpt card in CSS pixels, accounting for line wrapping at clamped column widths.
 */
export function estimateExcerptCardHeightPx(
  card: TimelineResolvedExcerptCard,
  colWidthPx: number,
  hasQueryMetadata: boolean,
  measureText: TextWidthMeasurer,
): number {
  const isDiff = card.kind === TimelineExcerptKind.DIFF;
  const lineChromePx = isDiff
    ? DECK_DIFF_LINE_CHROME_PX
    : DECK_LOG_LINE_CHROME_PX;
  const availableLineTextWidthPx = Math.max(80, colWidthPx - lineChromePx);

  let visualLineCount = 0;
  for (const line of card.lines) {
    const lineTextWidthPx = measureText(line.text, DECK_CODE_FONT);
    visualLineCount += Math.max(
      1,
      Math.ceil(lineTextWidthPx / availableLineTextWidthPx),
    );
  }
  visualLineCount = Math.max(1, visualLineCount);

  let noteHeightPx = 0;
  if (card.note) {
    const availableNoteTextWidthPx = Math.max(
      80,
      colWidthPx - DECK_NOTE_TAG_AND_GAP_PX - DECK_CARD_HORIZONTAL_CHROME_PX,
    );
    const noteRows = Math.max(
      1,
      Math.ceil(
        measureText(card.note, DECK_NOTE_FONT) / availableNoteTextWidthPx,
      ),
    );
    noteHeightPx = 12 + noteRows * 18;
  }

  return (
    52 + (hasQueryMetadata ? 20 : 0) + visualLineCount * 20 + noteHeightPx + 16
  );
}

/**
 * Resolves clamped horizontal stage, index pane, gutter, and chart viewport widths in CSS pixels.
 */
export function resolveHorizontalStageGeometry(
  rawIndexPaneWidthPx: number,
  requiredDeckWidthPx: number,
): HorizontalStageGeometry {
  const indexPaneWidthPx = clampNumber(
    Number.isFinite(rawIndexPaneWidthPx)
      ? Math.ceil(rawIndexPaneWidthPx)
      : MIN_INDEX_PANE_WIDTH_PX,
    MIN_INDEX_PANE_WIDTH_PX,
    MAX_INDEX_PANE_WIDTH_PX,
  );
  const chartLeftPx = indexPaneWidthPx + TIMELINE_GUTTER_WIDTH_PX;
  const timelineMinWidthPx = chartLeftPx + MIN_CHART_VIEWPORT_WIDTH_PX;
  const stageWidthPx = Math.min(
    Math.max(timelineMinWidthPx, requiredDeckWidthPx),
    MAX_STAGE_WIDTH_PX,
  );
  const chartViewportWidthPx = Math.max(0, stageWidthPx - chartLeftPx);

  return {
    indexPaneWidthPx,
    chartLeftPx,
    chartViewportWidthPx,
    stageWidthPx,
  };
}

/**
 * Clamps total timeline frame height in CSS pixels so at least one row remains visible beneath the ruler header.
 */
export function clampTimelineFrameHeightPx(rowsTotalHeightPx: number): number {
  return Math.max(
    TIMELINE_HEADER_HEIGHT_PX + BASE_ROW_HEIGHT,
    Math.ceil(TIMELINE_HEADER_HEIGHT_PX + rowsTotalHeightPx),
  );
}

/**
 * Calculates the total timeline frame height in CSS pixels from the ruler header and rendered rows.
 */
export function calculateTimelineFrameHeightPx(
  timelines: readonly ReadonlyDomainElement<Timeline>[],
  styleStore: StyleStoreLike,
): number {
  let rowsHeightPx = 0;
  for (const timeline of timelines) {
    rowsHeightPx += resolveTimelineRowHeightPx(timeline, styleStore);
  }
  return clampTimelineFrameHeightPx(rowsHeightPx);
}

/**
 * Calculates fitted CSS pixel dimensions for the timeline stage so index labels, role badges, and deck cards expand stage width horizontally.
 */
export function calculateTimelineStageDimensions(
  timelines: readonly ReadonlyDomainElement<Timeline>[],
  styleStore: StyleStoreLike,
  measureText: TextWidthMeasurer,
  overlay?: TimelineSceneOverlay | null,
): TimelineStageDimensions {
  const rawIndexPaneWidthPx = calculateIndexPaneWidthPx(
    timelines,
    styleStore,
    measureText,
    overlay?.rowHighlights,
  );
  const deckPlan = planExcerptDeckLayout(overlay?.excerpts ?? [], measureText);
  const horizontal = resolveHorizontalStageGeometry(
    rawIndexPaneWidthPx,
    deckPlan.requiredDeckWidthPx,
  );
  const timelineFrameHeightPx = calculateTimelineFrameHeightPx(
    timelines,
    styleStore,
  );

  return {
    indexPaneWidthPx: horizontal.indexPaneWidthPx,
    chartViewportWidthPx: horizontal.chartViewportWidthPx,
    stageWidthPx: horizontal.stageWidthPx,
    timelineFrameHeightPx,
  };
}

function collectPrimaryRoleBadgeWidthsByRowIndex(
  timelines: readonly ReadonlyDomainElement<Timeline>[],
  rowHighlights: readonly TimelineResolvedRowHighlight[] | undefined,
  measureText: TextWidthMeasurer,
): ReadonlyMap<number, number> {
  const result = new Map<number, number>();
  if (!rowHighlights || rowHighlights.length === 0 || timelines.length === 0) {
    return result;
  }

  const rowIndexByTimelineId = new Map<number, number>();
  for (let i = 0; i < timelines.length; i++) {
    rowIndexByTimelineId.set(timelines[i].id, i);
  }

  for (const rh of rowHighlights) {
    const roleLabel = rh.roleLabel?.trim() ?? '';
    if (!roleLabel) {
      continue;
    }
    let minRowIndex = Number.POSITIVE_INFINITY;
    for (const timelineId of rh.timelineIds) {
      const idx = rowIndexByTimelineId.get(timelineId);
      if (idx !== undefined && idx < minRowIndex) {
        minRowIndex = idx;
      }
    }
    if (!Number.isFinite(minRowIndex)) {
      continue;
    }
    const pillWidthPx = Math.ceil(
      measureText(roleLabel, ROLE_BADGE_FONT) +
        roleLabel.length * ROLE_BADGE_LETTER_SPACING_PX +
        ROLE_BADGE_HORIZONTAL_CHROME_PX,
    );
    result.set(
      minRowIndex,
      Math.max(result.get(minRowIndex) ?? 0, pillWidthPx),
    );
  }

  return result;
}

/**
 * Resolves the rendered height of a single timeline row in CSS pixels, falling back to timeline.type on missing overrides.
 */
export function resolveTimelineRowHeightPx(
  timeline: ReadonlyDomainElement<Timeline>,
  styleStore: StyleStoreLike,
): number {
  let timelineType = timeline.type;
  try {
    timelineType =
      styleStore.getTimelineType(timeline.type.id) ?? timeline.type;
  } catch {
    timelineType = timeline.type;
  }
  return timelineType.height * BASE_ROW_HEIGHT;
}
