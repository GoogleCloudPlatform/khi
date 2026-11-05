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
      ctx.font = font;
      return ctx.measureText(text).width;
    }
    const fontSizeMatch = /(\d+(?:\.\d+)?)px/.exec(font);
    const fontSizePx = fontSizeMatch ? Number(fontSizeMatch[1]) : 12;
    return text.length * fontSizePx * FALLBACK_CHAR_WIDTH_RATIO;
  };
}

/**
 * Calculates the fitted left index pane width in CSS pixels to hold all rendered timeline rows without truncation.
 */
export function calculateIndexPaneWidthPx(
  timelines: readonly ReadonlyDomainElement<Timeline>[],
  styleStore: StyleStoreLike,
  measureText: TextWidthMeasurer,
): number {
  let requiredWidthPx = MIN_INDEX_PANE_WIDTH_PX;

  for (const timeline of timelines) {
    const rowHeightPx = resolveRowHeightPx(timeline, styleStore);
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

    const rowRequiredWidthPx = Math.ceil(
      INDEX_ROW_LEFT_BORDER_PX +
        leftPaddingPx +
        iconSizePx * 3 +
        INDEX_ROW_GRID_GAP_PX * INDEX_ROW_GRID_COLUMN_GAPS +
        nameWidthPx +
        chipBlockWidthPx +
        INDEX_ROW_RIGHT_PADDING_PX +
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
 * Calculates the total timeline frame height in CSS pixels from the ruler header and rendered rows.
 */
export function calculateTimelineFrameHeightPx(
  timelines: readonly ReadonlyDomainElement<Timeline>[],
  styleStore: StyleStoreLike,
): number {
  let rowsHeightPx = 0;
  for (const timeline of timelines) {
    rowsHeightPx += resolveRowHeightPx(timeline, styleStore);
  }
  return Math.max(
    TIMELINE_HEADER_HEIGHT_PX + BASE_ROW_HEIGHT,
    Math.ceil(TIMELINE_HEADER_HEIGHT_PX + rowsHeightPx),
  );
}

/**
 * Calculates fitted CSS pixel dimensions for the timeline stage so index labels expand stage width without shrinking the chart.
 */
export function calculateTimelineStageDimensions(
  timelines: readonly ReadonlyDomainElement<Timeline>[],
  styleStore: StyleStoreLike,
  measureText: TextWidthMeasurer,
): TimelineStageDimensions {
  const indexPaneWidthPx = calculateIndexPaneWidthPx(
    timelines,
    styleStore,
    measureText,
  );
  const timelineMinWidthPx =
    indexPaneWidthPx + TIMELINE_GUTTER_WIDTH_PX + MIN_CHART_VIEWPORT_WIDTH_PX;
  const stageWidthPx = Math.min(timelineMinWidthPx, MAX_STAGE_WIDTH_PX);
  const chartViewportWidthPx = Math.max(
    0,
    stageWidthPx - indexPaneWidthPx - TIMELINE_GUTTER_WIDTH_PX,
  );
  const timelineFrameHeightPx = calculateTimelineFrameHeightPx(
    timelines,
    styleStore,
  );

  return {
    indexPaneWidthPx,
    chartViewportWidthPx,
    stageWidthPx,
    timelineFrameHeightPx,
  };
}

function resolveRowHeightPx(
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
