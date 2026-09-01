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
  CdkVirtualScrollViewport,
  FixedSizeVirtualScrollStrategy,
  ScrollingModule,
  VIRTUAL_SCROLL_STRATEGY,
} from '@angular/cdk/scrolling';
import {
  Component,
  input,
  model,
  output,
  computed,
  effect,
  viewChild,
} from '@angular/core';
import { CommonModule } from '@angular/common';

import { LogStore } from 'src/app/store/domain/log-store';
import { Log } from 'src/app/store/domain/log';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { IdBitset } from 'src/app/store/domain/filter/id-bitset';
import { LogViewLogLineComponent } from './log-view-log-line.component';
import { IconToggleButtonComponent } from '../../shared/components/icon-toggle-button/icon-toggle-button.component';
import { bisectLeft } from '../../common/misc-util';

class LogListScrollingStrategy extends FixedSizeVirtualScrollStrategy {
  constructor() {
    // heght:12px + border-bottom: 0.2px
    super(12.2, 500, 1000);
  }
}

@Component({
  selector: 'khi-log-list',
  templateUrl: './log-list.component.html',
  styleUrls: ['./log-list.component.scss'],
  imports: [
    CommonModule,
    ScrollingModule,
    IconToggleButtonComponent,
    CdkVirtualScrollViewport,
    LogViewLogLineComponent,
  ],
  providers: [
    { provide: VIRTUAL_SCROLL_STRATEGY, useClass: LogListScrollingStrategy },
  ],
})
export class LogListComponent {
  /** The store containing all logs. */
  public readonly logStore = input<LogStore>();
  /** The bitset of filtered log IDs. */
  public readonly filteredLogIds = input.required<IdBitset>();
  /** The index of the currently selected log. */
  public readonly selectedLogIndex = input.required<number>();
  /** The set of indices of highlighted logs. */
  public readonly highlightedLogIndices = input.required<Set<number>>();
  /** The list of selected timelines including their children. */
  public readonly selectedTimelinesWithChildren =
    input.required<ReadonlyDomainElement<Timeline>[]>();

  /** Whether to filter logs by selected timelines. */
  public readonly filterByTimeline = model<boolean>(true);
  /** Whether to include child timelines in the filter. */
  public readonly includeTimelineChildren = model<boolean>(true);

  /** Emits when a log entry is selected. */
  public readonly logSelected = output<ReadonlyDomainElement<Log>>();
  /** Emits when a log entry is hovered. */
  public readonly logHovered = output<ReadonlyDomainElement<Log>>();

  private readonly viewPort = viewChild(CdkVirtualScrollViewport);

  protected readonly allLogsCount = computed(() => this.logStore()?.count ?? 0);

  protected readonly visibleLogIds = computed(() => {
    const logStore = this.logStore();
    if (!logStore) {
      return [];
    }

    const filterByTimeline = this.filterByTimeline();
    const filterLogIds = this.filteredLogIds();

    if (filterByTimeline) {
      const timelines = this.selectedTimelinesWithChildren();
      if (timelines && timelines.length > 0) {
        const seenLogIds = new Set<number>();
        const matchedLogIds: number[] = [];

        for (const timeline of timelines) {
          for (const revision of timeline.revisions) {
            const logId = revision.logId;
            if (!seenLogIds.has(logId) && filterLogIds.has(logId)) {
              seenLogIds.add(logId);
              matchedLogIds.push(logId);
            }
          }
          for (const event of timeline.events) {
            const logId = event.logId;
            if (!seenLogIds.has(logId) && filterLogIds.has(logId)) {
              seenLogIds.add(logId);
              matchedLogIds.push(logId);
            }
          }
        }
        matchedLogIds.sort(
          (a, b) => logStore.getIndex(a) - logStore.getIndex(b),
        );
        return matchedLogIds;
      }
    }

    const totalCount = logStore.count;
    const result: number[] = [];
    for (let i = 0; i < totalCount; i++) {
      const logId = logStore.getLogIdByIndex(i);
      if (filterLogIds.has(logId)) {
        result.push(logId);
      }
    }
    return result;
  });

  protected readonly visibleLogsCount = computed(
    () => this.visibleLogIds().length,
  );

  private disableScrollForNext = false;

  constructor() {
    effect(() => {
      const viewport = this.viewPort();

      const logIds = this.visibleLogIds();
      const selectedIndex = this.selectedLogIndex();
      this.selectedTimelinesWithChildren();

      if (selectedIndex === -1) return;

      if (!this.disableScrollForNext) {
        const arrayIndex = this.searchArrayIndexOfLog(logIds, selectedIndex);
        if (arrayIndex >= 0 && viewport) {
          // The child virtual scroll viewport might not have received the list of updated logs yet.
          // Wait a frame to ensure the viewport has the correct list of logs.
          requestAnimationFrame(() => {
            viewport.scrollToIndex(arrayIndex, 'smooth');
          });
        }
      }
      this.disableScrollForNext = false;
    });
  }

  protected selectLog(logId: number) {
    this.disableScrollForNext = true;
    const store = this.logStore();
    if (store) {
      this.logSelected.emit(store.getLog(logId));
    }
  }

  protected onLogHover(logId: number) {
    const store = this.logStore();
    if (store) {
      this.logHovered.emit(store.getLog(logId));
    }
  }

  /**
   * Handles keyboard navigation (ArrowUp/ArrowDown) on the log list container to allow
   * selecting and scrolling through logs using keyboard controls.
   */
  protected onKeyDown(event: KeyboardEvent) {
    if (event.key !== 'ArrowUp' && event.key !== 'ArrowDown') {
      return;
    }
    if (event.altKey || event.ctrlKey || event.metaKey || event.shiftKey) {
      return;
    }

    const logIds = this.visibleLogIds();
    const store = this.logStore();
    if (logIds.length === 0 || !store) return;

    // Prevent the default browser scrolling behavior when navigating the log list.
    event.preventDefault();

    const selectedIndex = this.selectedLogIndex();
    const arrayIndex = this.searchArrayIndexOfLog(logIds, selectedIndex);

    let nextArrayIndex = -1;
    if (event.key === 'ArrowUp') {
      if (arrayIndex === -1) {
        nextArrayIndex = logIds.length - 1;
      } else if (arrayIndex > 0) {
        nextArrayIndex = arrayIndex - 1;
      }
    } else if (event.key === 'ArrowDown') {
      if (arrayIndex === -1) {
        nextArrayIndex = 0;
      } else if (arrayIndex < logIds.length - 1) {
        nextArrayIndex = arrayIndex + 1;
      }
    }

    if (nextArrayIndex >= 0 && nextArrayIndex < logIds.length) {
      // Direct emission without setting disableScrollForNext ensures that the view
      // automatically scrolls to the newly selected log.
      this.logSelected.emit(store.getLog(logIds[nextArrayIndex]));
    }
  }

  private searchArrayIndexOfLog(
    logIds: readonly number[],
    logIndex: number,
  ): number {
    const store = this.logStore();
    if (!store) return -1;
    const idx = bisectLeft(logIds, logIndex, (id, t) => store.getIndex(id) - t);
    return idx < logIds.length && store.getIndex(logIds[idx]) === logIndex
      ? idx
      : -1;
  }
}
