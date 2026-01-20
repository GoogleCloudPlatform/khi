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
 * HorizontalScrollCalculator calculates horizontal virtual scrolling for the timeline.
 * It manages the conversion between time (ms) and pixel coordinates, taking into account
 * zoom levels (pixels per millisecond) and extra buffer offsets.
 */
export class HorizontalScrollCalculator {
  /**
   * @param minTimeMs The minimum timestamp (start time) of the log data.
   * @param maxTimeMs The maximum timestamp (end time) of the log data.
   * @param extraOffsetWidthInPx Extra width in pixels to render outside the viewport for buffering. Defaults to 300.
   */
  constructor(
    readonly minTimeMs: number,
    readonly maxTimeMs: number,
    readonly extraOffsetWidthInPx: number = 300,
  ) {}

  /**
   * The total width of the scrollable area in pixels.
   *
   * @param pixelsPerMs The current zoom level in pixels per millisecond.
   * @returns Total width in pixels.
   */
  totalWidth(pixelsPerMs: number): number {
    const timeSpan =
      this.maxScrollableTimeMS(pixelsPerMs) -
      this.minScrollableTimeMS(pixelsPerMs);
    return timeSpan * pixelsPerMs;
  }

  /**
   * Calculates the total width to be rendered, including the visible viewport and buffers.
   *
   * @param viewportWidth The width of the visible part of the scroll container (px).
   * @returns Total efficient render width (px).
   */
  totalRenderWidth(viewportWidth: number): number {
    return viewportWidth + this.extraOffsetWidthInPx * 2;
  }

  /**
   * Calculates the starting time (ms) for the content to be rendered based on the viewport's position.
   * It aligns the start time to the nearest tick interval.
   *
   * @param viewportLeftTimeMS The time at the left edge of the visible viewport.
   * @param tickTimeMS The current tick interval in milliseconds.
   * @param pixelsPerMs The current zoom level in pixels per millisecond.
   * @returns The aligned start time (ms) for rendering.
   */
  leftDrawAreaTimeMS(
    viewportLeftTimeMS: number,
    tickTimeMS: number,
    pixelsPerMs: number,
  ): number {
    return (
      Math.floor(
        (viewportLeftTimeMS - this.extraOffsetTimeMS(pixelsPerMs)) / tickTimeMS,
      ) * tickTimeMS
    );
  }

  /**
   * Calculates the CSS `left` offset for the rendered content area.
   * This aligns the content with the virtual scroll position.
   *
   * @param viewportLeftTimeMS The time at the left edge of the visible viewport.
   * @param tickTimeMS The current tick interval in milliseconds.
   * @param pixelsPerMs The current zoom level in pixels per millisecond.
   * @returns The pixel offset for the left edge of the rendered content.
   */
  leftDrawAreaOffset(
    viewportLeftTimeMS: number,
    tickTimeMS: number,
    pixelsPerMs: number,
  ): number {
    const vpLeftToDrawAreaLeftInPx =
      (viewportLeftTimeMS -
        this.leftDrawAreaTimeMS(viewportLeftTimeMS, tickTimeMS, pixelsPerMs)) *
      pixelsPerMs;
    return (
      (viewportLeftTimeMS - this.minScrollableTimeMS(pixelsPerMs)) *
        pixelsPerMs -
      vpLeftToDrawAreaLeftInPx
    );
  }

  /**
   * Calculates the minimum zoom level (pixels per ms) required to fit the entire timeline in the viewport.
   *
   * @param viewportWidth The width of the viewport (px).
   * @returns The minimum pixels per millisecond.
   */
  minPixelPerMs(viewportWidth: number): number {
    const logTimeMS = this.maxTimeMs - this.minTimeMs;
    return viewportWidth / logTimeMS;
  }

  /**
   * Returns the maximum allowed zoom level.
   *
   * @returns The maximum pixels per millisecond.
   */
  maxPixelPerMs(): number {
    return 10;
  }

  /**
   * Calculates the minimum scrollable time (start of the scrollable area), including the buffer.
   *
   * @param pixelsPerMs The current zoom level in pixels per millisecond.
   * @returns The minimum scrollable time in milliseconds.
   */
  minScrollableTimeMS(pixelsPerMs: number): number {
    return this.minTimeMs - this.extraOffsetTimeMS(pixelsPerMs);
  }

  /**
   * Calculates the maximum scrollable time (end of the scrollable area), including the buffer.
   *
   * @param pixelsPerMs The current zoom level in pixels per millisecond.
   * @returns The maximum scrollable time in milliseconds.
   */
  maxScrollableTimeMS(pixelsPerMs: number): number {
    return this.maxTimeMs + this.extraOffsetTimeMS(pixelsPerMs);
  }

  /**
   * Converts a timestamp to its left offset in pixels relative to the start of the scrollable area.
   *
   * @param time The timestamp to convert (ms).
   * @param pixelsPerMs The current zoom level in pixels per millisecond.
   * @returns The left offset in pixels.
   */
  timeMSToOffsetLeft(time: number, pixelsPerMs: number): number {
    return (time - this.minScrollableTimeMS(pixelsPerMs)) * pixelsPerMs;
  }

  /**
   * Calculates the maximum possible horizontal scroll position (scrollLeft).
   *
   * @param pixelsPerMs The current zoom level in pixels per millisecond.
   * @param viewportWidth The width of the viewport (px).
   * @returns The maximum scrollLeft value (px).
   */
  maxScrollLeft(pixelsPerMs: number, viewportWidth: number): number {
    return (
      (this.maxTimeMs -
        (viewportWidth - this.extraOffsetWidthInPx) / pixelsPerMs -
        this.minTimeMs) *
      pixelsPerMs
    );
  }

  /**
   * Calculates the time duration corresponding to the extra buffer offset.
   *
   * @param pixelsPerMs The current zoom level in pixels per millisecond.
   * @returns The buffer duration in milliseconds.
   */
  extraOffsetTimeMS(pixelsPerMs: number): number {
    return this.extraOffsetWidthInPx / pixelsPerMs;
  }

  /**
   * Converts the horizontal scroll position (scrollLeft) to the corresponding time at the left edge.
   *
   * @param scrollX The horizontal scroll position (px).
   * @param pixelsPerMs The current zoom level in pixels per millisecond.
   * @returns The time at the left edge of the visible area (ms).
   */
  scrollToViewportLeftTime(scrollX: number, pixelsPerMs: number): number {
    return scrollX / pixelsPerMs + this.minScrollableTimeMS(pixelsPerMs);
  }

  /**
   * Calculates the scroll position keeping mouse position time fixed after a zoom operation.
   *
   * @param currentPixelsPerMs The current zoom level in pixels per millisecond.
   * @param newPixelsPerMs The new zoom level in pixels per millisecond.
   * @param viewportRelativeMousePosition The relative position of the mouse within the viewport.
   * @param currentScrollLeft The current scroll position.
   * @returns The new scroll position after the zoom operation.
   */
  calculateZoomScrollLeft(
    currentPixelsPerMs: number,
    newPixelsPerMs: number,
    viewportRelativeMousePosition: number,
    currentScrollLeft: number,
  ): number {
    const cMinSc = this.minScrollableTimeMS(currentPixelsPerMs);
    const nMinSc = this.minScrollableTimeMS(newPixelsPerMs);
    return (
      newPixelsPerMs * (cMinSc - nMinSc) +
      (newPixelsPerMs / currentPixelsPerMs) *
        (currentScrollLeft + viewportRelativeMousePosition) -
      viewportRelativeMousePosition
    );
  }
}
