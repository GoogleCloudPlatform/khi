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

import { bisectRight } from 'src/app/common/misc-util';
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
    );
    const timelineIndexAtmostVisible = bisectRight(
      this.accumulatedHeights,
      scrollY + visibleHeight,
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
   * Returns the sticky timelines (e.g. Kind, Namespace) that should be pinned to the top of the view
   * at the current scroll position.
   *
   * @param scrollY Current vertical scroll position (px)
   * @returns Array of sticky ResourceTimelines
   */
  stickyTimelines(scrollY: number): ResourceTimeline[] {
    if (this.accumulatedHeights.length === 0) {
      return [];
    }
    const stickyHeaderSize =
      this.style.heightsByLayer[TimelineLayer.Kind] +
      this.style.heightsByLayer[TimelineLayer.Namespace];
    let i = bisectRight(this.accumulatedHeights, scrollY + stickyHeaderSize); // Starting from the timeline that is at least visible behind the sticky header
    i = Math.min(i, this.timelines.length - 1);
    let namespaceTimeline: ResourceTimeline | null = null;
    for (; i >= 0; i--) {
      if (this.timelines[i].layer === TimelineLayer.Namespace) {
        namespaceTimeline = this.timelines[i];
        break;
      }
    }
    let kindTimeline: ResourceTimeline | null = null;
    for (; i >= 0; i--) {
      if (this.timelines[i].layer === TimelineLayer.Kind) {
        kindTimeline = this.timelines[i];
        break;
      }
    }
    if (kindTimeline === null || namespaceTimeline === null) {
      return [];
    }
    return [kindTimeline, namespaceTimeline];
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
