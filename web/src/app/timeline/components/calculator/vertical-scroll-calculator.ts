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
  bisectLeft,
  bisectRight,
  defaultNumberComparator,
} from 'src/app/common/misc-util';
import { ResourceTimeline, TimelineLayer } from 'src/app/store/timeline';
import { TimelineChartStyle } from '../style-model';

/**
 * VerticalScrollCalculator calculates vertical virtual scrolling for timelines.
 * It handles cases where each timeline has a different height (e.g., depending on the layer)
 * and efficiently calculates the index or offset of timelines corresponding to a specified scroll position.
 */
export class VerticalScrollCalculator {
  /**
   * The total height of all timelines.
   */
  public readonly totalHeight: number;

  /**
   * An array holding the starting Y coordinate (accumulated height) of each timeline.
   * The value at index i represents the top Y coordinate of the i-th timeline.
   * The length of the array is equal to the number of timelines.
   */
  private readonly accumulatedHeights: number[];

  /**
   * The maximum height of a timeline.
   */
  private maxTimelineHeight: number = 0;

  /**
   * @param timelines List of resource timelines to be displayed.
   * @param style Timeline chart style containing height definitions for each layer.
   * @param marginTimelineCount Number of timelines to render outside the visible area as a buffer. Defaults to 10.
   */
  constructor(
    private readonly timelines: ResourceTimeline[],
    private readonly style: TimelineChartStyle,
    private readonly marginTimelineCount = 10,
  ) {
    this.accumulatedHeights = new Array<number>(this.timelines.length);
    let height = 0;
    for (let i = 0; i < this.timelines.length; i++) {
      this.accumulatedHeights[i] = height;
      height += this.style.heightsByLayer[this.timelines[i].layer];
      this.maxTimelineHeight = Math.max(
        this.maxTimelineHeight,
        this.style.heightsByLayer[this.timelines[i].layer],
      );
    }
    this.totalHeight = height;
  }

  /**
   * Returns the offset corresponding to the top edge of the draw area based on the current scroll position.
   * This is used as the starting Y coordinate (e.g., translateY value) when rendering content in virtual scrolling.
   *
   * @param scrollY Current vertical scroll position (px)
   * @returns The starting Y coordinate of the first visible timeline
   */
  topDrawAreaOffset(scrollY: number): number {
    if (this.accumulatedHeights.length === 0) {
      return 0;
    }
    if (
      scrollY >= this.accumulatedHeights[this.accumulatedHeights.length - 1]
    ) {
      return this.accumulatedHeights[this.accumulatedHeights.length - 1];
    }
    const timelineIndexAtleastVisible = bisectRight(
      this.accumulatedHeights,
      scrollY,
      defaultNumberComparator,
    );
    // bisectRight returns the first index where scrollY < value,
    // so the index before that is the start position of the timeline containing scrollY (or above it).
    return this.accumulatedHeights[
      Math.max(0, timelineIndexAtleastVisible - 1 - this.marginTimelineCount)
    ];
  }

  /**
   * Returns the list of timelines to be rendered based on the current scroll position and visible height.
   * It may return surrounding timelines that are outside the screen to prevent flickering during scrolling.
   *
   * @param scrollY Current vertical scroll position (px)
   * @param visibleHeight Height of the visible area (px)
   * @returns Array of timelines to be rendered
   */
  timelinesInDrawArea(
    scrollY: number,
    visibleHeight: number,
  ): ResourceTimeline[] {
    if (this.accumulatedHeights.length === 0) {
      return [];
    }
    const timelineIndexAtleastVisible = bisectRight(
      this.accumulatedHeights,
      scrollY,
      defaultNumberComparator,
    );
    const timelineIndexAtmostVisible = bisectRight(
      this.accumulatedHeights,
      scrollY + visibleHeight,
      defaultNumberComparator,
    );

    // Slice with a slightly wider range
    return this.timelines.slice(
      Math.max(0, timelineIndexAtleastVisible - 1 - this.marginTimelineCount),
      Math.min(
        timelineIndexAtmostVisible + this.marginTimelineCount,
        this.timelines.length,
      ),
    );
  }

  /**
   * Returns the sticky timelines (e.g. Kind, Namespace, Name) that should be pinned to the top of the view
   * at the current scroll position.
   *
   * @param scrollY Current vertical scroll position (px)
   * @returns Array of sticky ResourceTimelines
   */
  stickyTimelines(scrollY: number): ResourceTimeline[] {
    if (this.accumulatedHeights.length === 0) {
      return [];
    }
    let maxStickyLayer = TimelineLayer.Name;
    let currIndex = 0;
    // At scroll position 0, simply returns the timelines up to the maximum sticky layer to ensure initial shadow consistency.
    // While an empty array visually behaves almost the same, having actual items prevents the bottom shadow from disappearing initially.
    if (scrollY === 0) {
      const result = [];
      for (let i = 0; i < this.timelines.length; i++) {
        result.push(this.timelines[i]);
        if (this.timelines[i].layer === maxStickyLayer) {
          break;
        }
      }
      return result;
    }
    // Looks ahead by the total size of the candidate sticky header. If the resulting timeline layer matches the candidate layer, it would overlap (e.g. a Pod row sticking over another Pod row).
    // In such cases, shrinks the maximum sticky layer candidate by one to fallback to an upper layer timeline as the base context.
    for (; maxStickyLayer > TimelineLayer.APIVersion; maxStickyLayer--) {
      let stickyHeaderSize = 0;
      for (let l = TimelineLayer.APIVersion; l <= maxStickyLayer; l++) {
        stickyHeaderSize += this.style.heightsByLayer[l];
      }
      let i = bisectLeft(
        this.accumulatedHeights,
        scrollY + stickyHeaderSize,
        defaultNumberComparator,
      );
      i = Math.min(Math.max(0, i - 1), this.timelines.length - 1);
      if (this.timelines[i].layer > maxStickyLayer) {
        currIndex = i;
        break;
      }
    }

    // Retrieves the ancestor timelines for each target sticky layer from the established base index.
    const result: ResourceTimeline[] = [];
    for (let l = maxStickyLayer; l >= TimelineLayer.Kind; l--) {
      for (let j = currIndex; j >= 0; j--) {
        if (this.timelines[j].layer === l) {
          result.push(this.timelines[j]);
          currIndex = j;
          break;
        }
        if (this.timelines[j].layer < l) {
          break;
        }
      }
    }
    return result.filter((timeline) => timeline !== null).reverse();
  }

  /**
   * Calculates the total height related to the virtual scrolling rendering area.
   * This includes the viewport height plus a safety margin to prevent showing the white background
   * when scrolling fast.
   *
   * @param viewportHeight The height of the visible part of the scroll container (px)
   * @returns Total efficient render height (px)
   */
  totalRenderHeight(viewportHeight: number): number {
    // To prevent resizing, use the largest possible margin of the timeline.
    return (
      viewportHeight + this.marginTimelineCount * 2 * this.maxTimelineHeight
    );
  }

  /**
   * Calculates the bottom Y coordinate of a specific timeline identified by its ID.
   * Useful for scrolling to a specific timeline or determining its position in the full list.
   *
   * @param timelineID The unique identifier of the timeline
   * @returns The bottom Y coordinate of the timeline (px), or 0 if not found.
   */
  timelineIDToTimelineBottomOffset(timelineID: string): number {
    const timelineIndex = this.timelines.findIndex(
      (timeline) => timeline.timelineId === timelineID,
    );
    if (timelineIndex === -1) {
      return 0;
    }
    return (
      this.accumulatedHeights[timelineIndex] +
      this.style.heightsByLayer[this.timelines[timelineIndex].layer]
    );
  }

  /**
   * Calculates the top Y coordinate of a specific timeline identified by its ID.
   * Useful for scrolling to a specific timeline or determining its position in the full list.
   *
   * @param timelineID The unique identifier of the timeline
   * @returns The top Y coordinate of the timeline (px), or 0 if not found.
   */
  timelineIDToTimelineTopOffset(timelineID: string): number {
    const timelineIndex = this.timelines.findIndex(
      (timeline) => timeline.timelineId === timelineID,
    );
    if (timelineIndex === -1) {
      return 0;
    }
    return this.accumulatedHeights[timelineIndex];
  }
}
