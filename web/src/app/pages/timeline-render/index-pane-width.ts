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

/**
 * A timeline index row whose label width is estimated.
 */
export interface IndexPaneRow {
  /** The resource name shown as the main label. */
  readonly name: string;
  /** The timeline type label shown as a chip after the name. */
  readonly layerName: string;
  /** The hierarchy depth of the row (1 for the root layer). */
  readonly layer: number;
}

/**
 * Measures the rendered width of text in pixels for a CSS font shorthand.
 */
export type TextMeasurer = (text: string, font: string) => number;

// The constants below mirror timeline-index.component.scss. Font sizes use the upper bound of their clamp() so the estimate never truncates.
const MAIN_NAME_FONT_SIZE_PX = 14;
const LAYER_CHIP_FONT_SIZE_PX = 10;
const LAYER_CHIP_LETTER_SPACING_PX = 0.5;
const LAYER_CHIP_HORIZONTAL_PADDING_PX = 10;
const LAYER_CHIP_MARGIN_RIGHT_PX = 2;
const NAME_TO_CHIP_SPACING_PX = 8;
const ICON_SIZE_PX = 18;
const GRID_GAP_PX = 4;
const BORDER_LEFT_PX = 5;
const BASE_PADDING_LEFT_PX = 4;
const INDENT_UNIT_PX = 4;
const PADDING_RIGHT_PX = 8;

// Space taken by everything in a row except the name and chip text: border, padding, the toggle and icon columns, and the gaps between the four grid columns.
const ROW_CHROME_PX =
  BORDER_LEFT_PX +
  BASE_PADDING_LEFT_PX +
  ICON_SIZE_PX * 2 +
  GRID_GAP_PX * 3 +
  NAME_TO_CHIP_SPACING_PX +
  LAYER_CHIP_HORIZONTAL_PADDING_PX +
  LAYER_CHIP_MARGIN_RIGHT_PX +
  PADDING_RIGHT_PX;

/**
 * Computes the index pane width needed to show every row label without truncation.
 *
 * @param rows - The rows rendered in the index pane.
 * @param measureText - Measures text width for a CSS font shorthand.
 * @param fontFamily - The CSS font family used by the index pane.
 * @param minWidthPx - The minimum pane width, used when all labels fit.
 * @param maxWidthPx - The maximum pane width, so the chart keeps enough space.
 * @returns The pane width in whole pixels, clamped to [minWidthPx, maxWidthPx].
 */
export function computeIndexPaneWidth(
  rows: readonly IndexPaneRow[],
  measureText: TextMeasurer,
  fontFamily: string,
  minWidthPx: number,
  maxWidthPx: number,
): number {
  const nameFont = `400 ${MAIN_NAME_FONT_SIZE_PX}px ${fontFamily}`;
  const chipFont = `500 ${LAYER_CHIP_FONT_SIZE_PX}px ${fontFamily}`;
  let required = 0;
  for (const row of rows) {
    const indent = Math.max(0, row.layer - 1) * INDENT_UNIT_PX;
    const nameWidth = measureText(row.name, nameFont);
    const chipWidth =
      measureText(row.layerName, chipFont) +
      row.layerName.length * LAYER_CHIP_LETTER_SPACING_PX;
    required = Math.max(
      required,
      ROW_CHROME_PX + indent + nameWidth + chipWidth,
    );
  }
  return Math.min(maxWidthPx, Math.max(minWidthPx, Math.ceil(required)));
}

/**
 * Creates a TextMeasurer backed by a 2D canvas context.
 */
export function createCanvasTextMeasurer(): TextMeasurer {
  const context = document.createElement('canvas').getContext('2d');
  return (text, font) => {
    if (!context) {
      return 0;
    }
    context.font = font;
    return context.measureText(text).width;
  };
}
