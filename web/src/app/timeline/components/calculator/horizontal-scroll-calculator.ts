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

function floorDiv(a: bigint, b: bigint): bigint {
  const q = a / b;
  const r = a % b;
  return r !== 0n && a < 0n !== b < 0n ? q - 1n : q;
}

/**
 * HorizontalScrollCalculator calculates horizontal virtual scrolling for the timeline.
 * It manages the conversion between time (ns) and pixel coordinates, taking into account
 * zoom levels (pixels per nanosecond) and extra buffer offsets.
 */
export class HorizontalScrollCalculator {
  /**
   * @param minTimeNs The minimum timestamp (start time) of the log data in nanoseconds.
   * @param maxTimeNs The maximum timestamp (end time) of the log data in nanoseconds.
   * @param extraOffsetWidthInPx Extra width in pixels to render outside the viewport for buffering. Defaults to 300.
   */
  constructor(
    readonly minTimeNs: bigint,
    readonly maxTimeNs: bigint,
    readonly extraOffsetWidthInPx: number = 300,
  ) {}

  /**
   * The total width of the scrollable area in pixels.
   *
   * @param pixelsPerNs The current zoom level in pixels per nanosecond.
   * @returns Total width in pixels.
   */
  totalWidth(pixelsPerNs: number): number {
    const timeSpanNs =
      this.maxScrollableTimeNs(pixelsPerNs) -
      this.minScrollableTimeNs(pixelsPerNs);
    return Number(timeSpanNs) * pixelsPerNs;
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
   * Calculates the starting time (ns) for the content to be rendered based on the viewport's position.
   * It aligns the start time to the nearest tick interval.
   *
   * @param viewportLeftTimeNs The time at the left edge of the visible viewport (ns).
   * @param tickTimeNs The current tick interval in nanoseconds.
   * @param pixelsPerNs The current zoom level in pixels per nanosecond.
   * @returns The aligned start time (ns) for rendering.
   */
  leftDrawAreaTimeNs(
    viewportLeftTimeNs: bigint,
    tickTimeNs: bigint,
    pixelsPerNs: number,
  ): bigint {
    return (
      floorDiv(
        viewportLeftTimeNs - this.extraOffsetTimeNs(pixelsPerNs),
        tickTimeNs,
      ) * tickTimeNs
    );
  }

  /**
   * Calculates the CSS `left` offset for the rendered content area.
   * This aligns the content with the virtual scroll position.
   *
   * @param viewportLeftTimeNs The time at the left edge of the visible viewport (ns).
   * @param tickTimeNs The current tick interval in nanoseconds.
   * @param pixelsPerNs The current zoom level in pixels per nanosecond.
   * @returns The pixel offset for the left edge of the rendered content.
   */
  leftDrawAreaOffset(
    viewportLeftTimeNs: bigint,
    tickTimeNs: bigint,
    pixelsPerNs: number,
  ): number {
    const vpLeftToDrawAreaLeftInPx =
      Number(
        viewportLeftTimeNs -
          this.leftDrawAreaTimeNs(viewportLeftTimeNs, tickTimeNs, pixelsPerNs),
      ) * pixelsPerNs;
    return (
      Number(viewportLeftTimeNs - this.minScrollableTimeNs(pixelsPerNs)) *
        pixelsPerNs -
      vpLeftToDrawAreaLeftInPx
    );
  }

  /**
   * Calculates the minimum zoom level (pixels per ns) required to fit the entire timeline in the viewport.
   *
   * @param viewportWidth The width of the viewport (px).
   * @returns The minimum pixels per nanosecond.
   */
  minPixelPerNs(viewportWidth: number): number {
    const logTimeNs = Number(this.maxTimeNs - this.minTimeNs);
    return viewportWidth / logTimeNs;
  }

  /**
   * Returns the maximum allowed zoom level.
   *
   * @returns The maximum pixels per nanosecond.
   */
  maxPixelPerNs(): number {
    return 1e-5;
  }

  /**
   * Calculates the minimum scrollable time (start of the scrollable area), including the buffer.
   *
   * @param pixelsPerNs The current zoom level in pixels per nanosecond.
   * @returns The minimum scrollable time in nanoseconds.
   */
  minScrollableTimeNs(pixelsPerNs: number): bigint {
    return this.minTimeNs - this.extraOffsetTimeNs(pixelsPerNs);
  }

  /**
   * Calculates the maximum scrollable time (end of the scrollable area), including the buffer.
   *
   * @param pixelsPerNs The current zoom level in pixels per nanosecond.
   * @returns The maximum scrollable time in nanoseconds.
   */
  maxScrollableTimeNs(pixelsPerNs: number): bigint {
    return this.maxTimeNs + this.extraOffsetTimeNs(pixelsPerNs);
  }

  /**
   * Converts a timestamp to its left offset in pixels relative to the start of the scrollable area.
   *
   * @param timeNs The timestamp to convert (ns).
   * @param pixelsPerNs The current zoom level in pixels per nanosecond.
   * @returns The left offset in pixels.
   */
  timeNsToOffsetLeft(timeNs: bigint, pixelsPerNs: number): number {
    return Number(timeNs - this.minScrollableTimeNs(pixelsPerNs)) * pixelsPerNs;
  }

  /**
   * Calculates the maximum possible horizontal scroll position (scrollLeft).
   *
   * @param pixelsPerNs The current zoom level in pixels per nanosecond.
   * @param viewportWidth The width of the viewport (px).
   * @returns The maximum scrollLeft value (px).
   */
  maxScrollLeft(pixelsPerNs: number, viewportWidth: number): number {
    return (
      Number(this.maxTimeNs - this.minTimeNs) * pixelsPerNs -
      (viewportWidth - this.extraOffsetWidthInPx)
    );
  }

  /**
   * Calculates the time duration corresponding to the extra buffer offset.
   *
   * @param pixelsPerNs The current zoom level in pixels per nanosecond.
   * @returns The buffer duration in nanoseconds.
   */
  extraOffsetTimeNs(pixelsPerNs: number): bigint {
    return BigInt(Math.round(this.extraOffsetWidthInPx / pixelsPerNs));
  }

  /**
   * Converts the horizontal scroll position (scrollLeft) to the corresponding time at the left edge.
   *
   * @param scrollX The horizontal scroll position (px).
   * @param pixelsPerNs The current zoom level in pixels per nanosecond.
   * @returns The time at the left edge of the visible area (ns).
   */
  scrollToViewportLeftTimeNs(scrollX: number, pixelsPerNs: number): bigint {
    return (
      BigInt(Math.round(scrollX / pixelsPerNs)) +
      this.minScrollableTimeNs(pixelsPerNs)
    );
  }

  /**
   * Calculates the scroll position keeping mouse position time fixed after a zoom operation.
   *
   * @param currentPixelsPerNs The current zoom level in pixels per nanosecond.
   * @param newPixelsPerNs The new zoom level in pixels per nanosecond.
   * @param viewportRelativeMousePosition The relative position of the mouse within the viewport.
   * @param currentScrollLeft The current scroll position.
   * @returns The new scroll position after the zoom operation.
   */
  calculateZoomScrollLeft(
    currentPixelsPerNs: number,
    newPixelsPerNs: number,
    viewportRelativeMousePosition: number,
    currentScrollLeft: number,
  ): number {
    const cMinSc = this.minScrollableTimeNs(currentPixelsPerNs);
    const nMinSc = this.minScrollableTimeNs(newPixelsPerNs);
    return (
      newPixelsPerNs * Number(cMinSc - nMinSc) +
      (newPixelsPerNs / currentPixelsPerNs) *
        (currentScrollLeft + viewportRelativeMousePosition) -
      viewportRelativeMousePosition
    );
  }
}
