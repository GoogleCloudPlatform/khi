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

import { Log } from 'src/app/store/domain/log';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
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
  /** The total number of logs. */
  public readonly allLogsCount = input.required<number>();
  /** The list of filtered log entries. */
  public readonly filteredLogs = input.required<ReadonlyDomainElement<Log>[]>();
  /** The index of the currently selected log. */
  public readonly selectedLogIndex = input.required<number>();
  /** The set of indices of highlighted logs. */
  public readonly highlightLogIndices = input.required<Set<number>>();
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

  protected readonly shownLogs = computed(() => {
    const logs = this.filteredLogs();
    const filterByTimeline = this.filterByTimeline();
    const timelines = this.selectedTimelinesWithChildren();

    if (!filterByTimeline || !timelines || timelines.length === 0) return logs;
    return this.filterLogsWithTimelines(logs, timelines);
  });

  protected readonly shownLogsCount = computed(() => this.shownLogs().length);

  private disableScrollForNext = false;

  constructor() {
    effect(() => {
      const viewport = this.viewPort();

      const logs = this.shownLogs();
      const selectedIndex = this.selectedLogIndex();
      this.selectedTimelinesWithChildren();

      if (selectedIndex === -1) return;

      if (!this.disableScrollForNext) {
        const arrayIndex = this.searchArrayIndexOfLog(logs, selectedIndex);
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

  protected selectLog(logEntry: ReadonlyDomainElement<Log>) {
    this.disableScrollForNext = true;
    this.logSelected.emit(logEntry);
  }

  protected onLogHover(logEntry: ReadonlyDomainElement<Log>) {
    this.logHovered.emit(logEntry);
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

    const logs = this.shownLogs();
    if (logs.length === 0) return;

    // Prevent the default browser scrolling behavior when navigating the log list.
    event.preventDefault();

    const selectedIndex = this.selectedLogIndex();
    const arrayIndex = this.searchArrayIndexOfLog(logs, selectedIndex);

    let nextArrayIndex = -1;
    if (event.key === 'ArrowUp') {
      if (arrayIndex === -1) {
        nextArrayIndex = logs.length - 1;
      } else if (arrayIndex > 0) {
        nextArrayIndex = arrayIndex - 1;
      }
    } else if (event.key === 'ArrowDown') {
      if (arrayIndex === -1) {
        nextArrayIndex = 0;
      } else if (arrayIndex < logs.length - 1) {
        nextArrayIndex = arrayIndex + 1;
      }
    }

    if (nextArrayIndex >= 0 && nextArrayIndex < logs.length) {
      // Direct emission without setting disableScrollForNext ensures that the view
      // automatically scrolls to the newly selected log.
      this.logSelected.emit(logs[nextArrayIndex]);
    }
  }

  private filterLogsWithTimelines(
    logs: ReadonlyDomainElement<Log>[],
    timelines: ReadonlyDomainElement<Timeline>[],
  ): ReadonlyDomainElement<Log>[] {
    const logIndices = new Set<number>();
    for (const timeline of timelines) {
      for (const revision of timeline.revisions) {
        logIndices.add(revision.logIndex);
      }
      for (const event of timeline.events) {
        logIndices.add(event.logIndex);
      }
    }
    const result: ReadonlyDomainElement<Log>[] = [];
    for (const log of logs) {
      if (logIndices.has(log.logIndex)) {
        result.push(log);
      }
    }
    return result;
  }

  private searchArrayIndexOfLog(
    logs: ReadonlyDomainElement<Log>[],
    logIndex: number,
  ): number {
    const idx = bisectLeft(logs, logIndex, (l, t) => l.logIndex - t);
    return idx < logs.length && logs[idx].logIndex === logIndex ? idx : -1;
  }
}
