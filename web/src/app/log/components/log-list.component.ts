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

import { LogEntry } from '../../store/log';
import { ResourceTimeline } from '../../store/timeline';
import { LogViewLogLineComponent } from './log-view-log-line.component';
import { IconToggleButtonComponent } from '../../shared/components/icon-toggle-button/icon-toggle-button.component';
import { bisectLeft } from '../../common/misc-util';

class LogListScrollingStrategy extends FixedSizeVirtualScrollStrategy {
  constructor() {
    super(14.5, 500, 1000);
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
  public readonly allLogsCount = input.required<number>();
  public readonly filteredLogs = input.required<LogEntry[]>();
  public readonly selectedLogIndex = input.required<number>();
  public readonly highlightLogIndices = input.required<Set<number>>();
  public readonly selectedTimelinesWithChildren =
    input.required<ResourceTimeline[]>();

  public readonly filterByTimeline = model<boolean>(true);
  public readonly includeTimelineChildren = model<boolean>(true);

  public readonly logSelected = output<LogEntry>();
  public readonly logHovered = output<LogEntry>();

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

  protected selectLog(logEntry: LogEntry) {
    this.disableScrollForNext = true;
    this.logSelected.emit(logEntry);
  }

  protected onLogHover(logEntry: LogEntry) {
    this.logHovered.emit(logEntry);
  }

  private filterLogsWithTimelines(
    logs: LogEntry[],
    timelines: ResourceTimeline[],
  ): LogEntry[] {
    const logIndices = new Set<number>();
    for (const timeline of timelines) {
      for (const revision of timeline.revisions) {
        logIndices.add(revision.logIndex);
      }
      for (const event of timeline.events) {
        logIndices.add(event.logIndex);
      }
    }
    const result: LogEntry[] = [];
    for (const log of logs) {
      if (logIndices.has(log.logIndex)) {
        result.push(log);
      }
    }
    return result;
  }

  private searchArrayIndexOfLog(logs: LogEntry[], logIndex: number): number {
    const idx = bisectLeft(logs, logIndex, (l, t) => l.logIndex - t);
    return idx < logs.length && logs[idx].logIndex === logIndex ? idx : -1;
  }
}
