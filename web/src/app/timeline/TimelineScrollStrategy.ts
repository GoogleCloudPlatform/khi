/**
 * Copyright 2024 Google LLC
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

import { Inject, Injectable } from '@angular/core';
import {
  BehaviorSubject,
  Observable,
  ReplaySubject,
  Subject,
  combineLatest,
  distinctUntilChanged,
  filter,
  map,
  withLatestFrom,
} from 'rxjs';
import { TIMELINE_ITEM_HEIGHTS } from './canvas/types';
import { ResourceTimeline, TimelineLayer } from '../store/timeline';
import {
  DEFAULT_TIMELINE_FILTER,
  TimelineFilter,
} from '../services/timeline-filter.service';

/**
 * Set of properties needed for scrolling behaviors and computed values of TimelineEntry.
 */
export interface PerRowScrollingProperty {
  height: number;
  offset: number;
  source: ResourceTimeline | null;
}

@Injectable()
export class TimelinesScrollStrategy {
  /**
   * Extra margin at the bottom in pixels.
   * This is needed not to stick the bottom element at the bottom of screen.
   */
  private static BOTTOM_OFFSET = 100;

  /**
   * Extra height to be rendered not to show the white area when user scrolls fast.
   */
  private static EXTRA_BOTTOM_RENDER_HEIGHT = 0;

  /**
   * Scrolling the timeline to this offset from top edge of the viewport.
   */
  private static SCROLL_OFFSET_FROM_TOP = 300;

  /**
   * Viewport automatically scrolls to a selected timeline when the timeline is not in the visible area.
   * Timelines around the edge(near start/end indices of visible range) should be regarded as invisible area.
   * This is the count of indices from these edge regarded as invisible area.
   */
  private static VERTICAL_SCROLL_ENABLING_IN_VISIBLE_RANGE_PADDING_INDICES_FROM_EDGE = 3;

  viewport!: HTMLDivElement;

  /**
   * Size of the viewport(scrollable area) in pixel.
   */
  viewportHeight: Subject<number> = new Subject();

  private viewportSizeObserver: ResizeObserver = new ResizeObserver(() => {
    if (!this.viewport) return;
    const rect = this.viewport.getBoundingClientRect();
    this.viewportHeight.next(rect.height);
  });

  private viewportScrollOffsetRaw: Subject<number> = new ReplaySubject(1);
  /**
   * Amount of scroll in the viewport.
   */
  viewportScrollOffset = this.viewportScrollOffsetRaw.pipe(
    distinctUntilChanged(),
  );

  /**
   * Pre-computed values needed for scrolling behavior.
   */
  perRowScrollingProperties: BehaviorSubject<PerRowScrollingProperty[]> =
    new BehaviorSubject<PerRowScrollingProperty[]>([]);

  /**
   * Range of item indices in the visible area.
   */
  visibleItemRange = combineLatest([
    this.viewportHeight,
    this.viewportScrollOffset,
    this.timelineFilter.filteredTimeline,
    this.perRowScrollingProperties,
  ]).pipe(
    map(([height, offset, timelines, rows]) => ({
      start: this.offsetToFirstVisibleIndex(rows, offset),
      end: Math.min(
        timelines.length,
        this.offsetToFirstVisibleIndex(
          rows,
          offset + height + TimelinesScrollStrategy.EXTRA_BOTTOM_RENDER_HEIGHT,
        ) + 1,
      ),
    })),
  );

  /**
   * Height in pixels of visible items. This could be smaller than the viewport when the data height is smaller than viewport.
   */
  visibleItemHeight: Observable<number> = combineLatest([
    this.visibleItemRange,
    this.perRowScrollingProperties,
  ]).pipe(
    filter(([{ end }, perRowProperties]) => end < perRowProperties.length),
    map(
      ([{ start, end }, perRowProperties]) =>
        perRowProperties[end].offset -
        perRowProperties[start].offset +
        perRowProperties[end].height,
    ),
  );

  /**
   * First visible item index in the timeline array.
   * Overriding VirtualScrollStrategy.
   */
  scrolledIndexChange: Observable<number> = this.visibleItemRange.pipe(
    distinctUntilChanged((p, c) => p.start === c.start),
    map((r) => r.start),
  );

  /**
   * Offset to the first visible item in pxiels.
   */
  offsetToRenderedContent: Observable<number> = combineLatest([
    this.scrolledIndexChange,
    this.perRowScrollingProperties,
  ]).pipe(map(([first, rows]) => this.getOffsetForIndex(rows, first)));

  /**
   * Offset to the top of the first item from top of the visible area in pixels.
   * This amount should be negative always.
   */
  offsetToFirstItemFromVisibleArea: Observable<number> = combineLatest([
    this.viewportScrollOffset,
    this.offsetToRenderedContent,
  ]).pipe(
    map(
      ([scrollOffset, offsetToRenderedContent]) =>
        offsetToRenderedContent - scrollOffset,
    ),
  );
  /**
   * Total height of timelines in pxiels.
   */
  totalScrollableHeight: Observable<number> =
    this.perRowScrollingProperties.pipe(
      map((timelines) =>
        timelines.length > 0
          ? timelines[timelines.length - 1].offset +
            timelines[timelines.length - 1].height +
            TimelinesScrollStrategy.BOTTOM_OFFSET
          : 0,
      ),
    );

  /**
   * The timeline where the viewport needs to scroll to.
   */
  scrollToTimelineVerticallyCommand: Subject<ResourceTimeline> = new Subject();

  constructor(
    @Inject(DEFAULT_TIMELINE_FILTER) private timelineFilter: TimelineFilter,
  ) {
    // Calculate properties needed for scrolling behavior from updated timeline array.
    this.timelineFilter.filteredTimeline
      .pipe(
        map((timelines) => {
          const result: PerRowScrollingProperty[] = [];
          let lastOffset = 0;
          for (const timeline of timelines) {
            result.push({
              height: TIMELINE_ITEM_HEIGHTS[timeline.layer],
              offset: lastOffset,
              source: timeline,
            });
            lastOffset += result[result.length - 1].height;
          }
          result.push({
            height: 0,
            offset: lastOffset,
            source: null,
          });
          return result;
        }),
      )
      .subscribe(this.perRowScrollingProperties);

    // Scroll viewport in response to `scrollToIndexCommand` observable.
    this.scrollToTimelineVerticallyCommand
      .pipe(
        withLatestFrom(
          this.timelineFilter.filteredTimeline,
          this.perRowScrollingProperties,
          this.visibleItemRange,
        ),
      )
      .subscribe(([timeline, timelines, rows, range]) => {
        const index = timelines.indexOf(timeline);
        if (
          index !== -1 &&
          (range.start +
            TimelinesScrollStrategy.VERTICAL_SCROLL_ENABLING_IN_VISIBLE_RANGE_PADDING_INDICES_FROM_EDGE >=
            index ||
            range.end -
              TimelinesScrollStrategy.VERTICAL_SCROLL_ENABLING_IN_VISIBLE_RANGE_PADDING_INDICES_FROM_EDGE <=
              index)
        ) {
          const offset = this.getOffsetForIndex(rows, index);
          this.viewport.scroll({
            top: offset - TimelinesScrollStrategy.SCROLL_OFFSET_FROM_TOP,
            behavior: 'smooth',
          });
        }
      });
  }

  visibleTimelines: Observable<ResourceTimeline[]> = combineLatest([
    this.visibleItemRange,
    this.timelineFilter.filteredTimeline,
  ]).pipe(
    map(([range, timelines]) => {
      return timelines.slice(range.start, range.end);
    }),
  );

  /**
   * List of timelines should be shown as the sticky header.
   */
  stickyTimelines: Observable<ResourceTimeline[]> = combineLatest([
    this.visibleItemRange,
    this.timelineFilter.filteredTimeline,
  ]).pipe(
    filter(([, timelines]) => timelines.length > 0),
    map(([range, timelines]) => {
      let currentSeekIndex = range.start;
      if (timelines.length <= currentSeekIndex) {
        return [];
      }
      let stickyNamespace = timelines[currentSeekIndex];
      while (stickyNamespace.layer !== TimelineLayer.Namespace) {
        if (stickyNamespace.layer === TimelineLayer.Subresource) {
          stickyNamespace = stickyNamespace.parent!.parent!;
          break;
        }
        if (stickyNamespace.layer === TimelineLayer.Name) {
          stickyNamespace = stickyNamespace.parent!;
          break;
        }
        currentSeekIndex--;
        if (currentSeekIndex === -1) {
          break;
        }
        stickyNamespace = timelines[currentSeekIndex];
      }
      if (stickyNamespace.layer !== TimelineLayer.Namespace) {
        return [];
      }
      return [stickyNamespace.parent!, stickyNamespace];
    }),
  );

  attach(viewport: HTMLDivElement): void {
    this.viewport = viewport;
    this.viewportSizeObserver.observe(this.viewport);
    const checkScrollAmount = () => {
      this.viewportScrollOffsetRaw.next(this.viewport.scrollTop);
      requestAnimationFrame(checkScrollAmount);
    };
    checkScrollAmount();
  }

  scrollToTimeline(timeline: ResourceTimeline): void {
    this.scrollToTimelineVerticallyCommand.next(timeline);
  }

  private getOffsetForIndex(
    currentPerRowScrollingProperties: PerRowScrollingProperty[],
    index: number,
  ): number {
    return index < currentPerRowScrollingProperties.length
      ? currentPerRowScrollingProperties[index].offset
      : 0;
  }

  private offsetToFirstVisibleIndex(
    currentPerRowScrollingProperties: PerRowScrollingProperty[],
    offset: number,
  ): number {
    let atLeastOverOffset = 0;
    let atLeastUnderOffset = currentPerRowScrollingProperties.length;
    while (atLeastUnderOffset - atLeastOverOffset > 1) {
      //binary search
      const mid = (atLeastOverOffset + atLeastUnderOffset) >> 1;
      if (currentPerRowScrollingProperties[mid].offset <= offset) {
        atLeastOverOffset = mid;
      } else {
        atLeastUnderOffset = mid;
      }
    }
    return atLeastOverOffset;
  }
}
