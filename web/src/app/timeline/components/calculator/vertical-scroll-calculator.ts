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
import { StyleStoreLike } from 'src/app/store/domain/style-store';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { BASE_ROW_HEIGHT } from '../style-model';

/**
 * Enum for layer of a timeline.
 * Timelines are hierarchical structure, it has k8s specific names on each depths.
 */
export enum TimelineLayer {
  APIVersion = 0,
  Kind = 1,
  Namespace = 2,
  Name = 3,
  Subresource = 4,
}

/**
 * VerticalScrollCalculator calculates vertical virtual scrolling for timelines.
 * It handles cases where each timeline has a different height (e.g., depending on its style)
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
   * @param marginTimelineCount Number of timelines to render outside the visible area as a buffer. Defaults to 10.
   * @param styleStore Style store to resolve overridden timeline heights.
   */
  constructor(
    private readonly timelines: readonly ReadonlyDomainElement<Timeline>[],
    private readonly marginTimelineCount = 10,
    private readonly styleStore: StyleStoreLike,
  ) {
    this.accumulatedHeights = new Array<number>(this.timelines.length);
    let height = 0;
    for (let i = 0; i < this.timelines.length; i++) {
      this.accumulatedHeights[i] = height;
      const timeline = this.timelines[i];
      const timelineType = this.resolveTimelineType(timeline);
      const timelineHeight = timelineType.height * BASE_ROW_HEIGHT;
      height += timelineHeight;
      this.maxTimelineHeight = Math.max(this.maxTimelineHeight, timelineHeight);
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
  ): ReadonlyDomainElement<Timeline>[] {
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
   * @returns Array of sticky Timelines
   */
  stickyTimelines(scrollY: number): ReadonlyDomainElement<Timeline>[] {
    if (this.accumulatedHeights.length === 0) {
      return [];
    }
    // Find the index of the first timeline that can be visible if no sticky timelines are used.
    const upperVisibleCandidateIndex = Math.max(
      0,
      bisectLeft(this.accumulatedHeights, scrollY, defaultNumberComparator) - 1,
    );

    let stickyHeaderOrigin = upperVisibleCandidateIndex;
    let stickyHeight = 0;
    for (let i = upperVisibleCandidateIndex; i < this.timelines.length; i++) {
      const timeline = this.timelines[i];
      const currentStickyHeight =
        this.calculateStickyHeaderHeightFrom(timeline);
      stickyHeaderOrigin = i;
      stickyHeight = currentStickyHeight;
      // The timeline is visible even if the sticky timeline is drawn from its parent
      if (
        stickyHeight + scrollY <
        this.accumulatedHeights[i] +
          this.resolveTimelineType(timeline).height * BASE_ROW_HEIGHT
      ) {
        break;
      }
    }
    // After determined the first visible timeline, go up until the top level timeline is reached.
    const stickyOriginTimeline = this.timelines[stickyHeaderOrigin];
    const result: ReadonlyDomainElement<Timeline>[] = [];
    for (
      let timeline: ReadonlyDomainElement<Timeline> | null =
        stickyOriginTimeline.parent;
      timeline !== null;
      timeline = timeline.parent
    ) {
      result.push(timeline);
    }
    return result.reverse();
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
  timelineIDToTimelineBottomOffset(timelineID: number): number {
    const timelineIndex = this.timelines.findIndex(
      (timeline) => timeline.id === timelineID,
    );
    if (timelineIndex === -1) {
      return 0;
    }
    const timelineType = this.resolveTimelineType(
      this.timelines[timelineIndex],
    );
    const timelineHeight = timelineType.height * BASE_ROW_HEIGHT;
    return this.accumulatedHeights[timelineIndex] + timelineHeight;
  }

  /**
   * Calculates the top Y coordinate of a specific timeline identified by its ID.
   * Useful for scrolling to a specific timeline or determining its position in the full list.
   *
   * @param timelineID The unique identifier of the timeline
   * @returns The top Y coordinate of the timeline (px), or 0 if not found.
   */
  timelineIDToTimelineTopOffset(timelineID: number): number {
    const timelineIndex = this.timelines.findIndex(
      (timeline) => timeline.id === timelineID,
    );
    if (timelineIndex === -1) {
      return 0;
    }
    return this.accumulatedHeights[timelineIndex];
  }

  private calculateStickyHeaderHeightFrom(
    timeline: ReadonlyDomainElement<Timeline>,
  ): number {
    let result = 0;
    for (
      let t = timeline.childrenCount === 0 ? timeline.parent : timeline; // If the timeline is a leaf node, the sticky header should start from the parent.
      t !== null;
      t = t.parent
    ) {
      result += this.resolveTimelineType(t).height * BASE_ROW_HEIGHT;
    }
    return result;
  }

  /**
   * Resolves the latest timeline type.
   * Timeline type styles may be updated from the override dialog. This returns the latest one.
   */
  private resolveTimelineType(timeline: ReadonlyDomainElement<Timeline>) {
    return this.styleStore?.getTimelineType(timeline.type.id) ?? timeline.type;
  }
}
