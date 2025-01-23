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

import {
  CdkVirtualScrollViewport,
  FixedSizeVirtualScrollStrategy,
  VIRTUAL_SCROLL_STRATEGY,
} from '@angular/cdk/scrolling';
import {
  AfterViewInit,
  Component,
  ElementRef,
  OnDestroy,
  OnInit,
  ViewChild,
} from '@angular/core';
import {
  BehaviorSubject,
  Observable,
  ReplaySubject,
  Subject,
  combineLatest,
  delay,
  filter,
  map,
  shareReplay,
  take,
  takeUntil,
  withLatestFrom,
} from 'rxjs';
import { InspectionDataStoreService } from '../services/inspection-data-store.service';
import { SelectionManagerService } from '../services/selection-manager.service';
import { ObservableCSSClassBinder } from '../utils/observable-css-class-binder';
import { LogEntry } from '../store/log';
import { TimelineEntry } from '../store/timeline';
import { monitorElementHeight } from '../utils/observable-util';

class LogViewScrollingStrategy extends FixedSizeVirtualScrollStrategy {
  constructor() {
    super(23.33, 500, 1000);
  }
}

interface LogViewSelectionMoveCommand {
  direction: 'next' | 'prev';
}

@Component({
  selector: 'khi-log-view',
  templateUrl: './log-view.component.html',
  styleUrls: ['./log-view.component.sass'],
  providers: [
    { provide: VIRTUAL_SCROLL_STRATEGY, useClass: LogViewScrollingStrategy },
  ],
})
export class LogViewComponent implements OnInit, AfterViewInit, OnDestroy {
  /**
   * The minimal size of log list.
   */
  private static MINIMUM_LOG_LIST_SIZE = 100;

  /**
   * The default size of log list.
   */
  private static DEFAULT_LOG_LIST_SIZE = 400;

  /**
   * The minimal size of log body view.
   */
  private static MINIMUM_LOG_BODY_VIEW_SIZE = 300;

  /**
   * An observable emits a value on destroying this component. This is used for ubsubscribe subscribers on destroying component.
   */
  private destoroyed = new Subject<void>();

  /**
   * Reference to the root container element of this component.
   */
  @ViewChild('container')
  private readonly container!: ElementRef<HTMLDivElement>;

  /**
   * Reference to CdkVirtualScrollViewport to show the list of logs.
   */
  @ViewChild(CdkVirtualScrollViewport)
  private readonly viewPort!: CdkVirtualScrollViewport;

  filterByTimeline = new BehaviorSubject(true);

  logViewSelectionMoveCommand = new Subject<LogViewSelectionMoveCommand>();

  includeTimelineChildren =
    this.selectionManager.timelineSelectionShouldIncludeChildren;

  selectedLog = this.selectionManager.selectedLog;
  shownLogs: Observable<LogEntry[]> = combineLatest([
    this.inspectionDataStore.filteredLogs,
    this.filterByTimeline,
    this.selectionManager.selectedTimelinesWithChildren,
  ]).pipe(
    map(([filteredLogs, filterByTimeline, timelines]) => {
      if (!filterByTimeline || timelines.length === 0) return filteredLogs; // show all of the filtered logs when `filterByTimeline` option is disabled.
      return this.filterLogsWithTimelines(filteredLogs, timelines);
    }),
    shareReplay(1),
  );
  allLogsCount = this.inspectionDataStore.allLogs.pipe(
    map((logs) => logs.length),
  );
  shownLogsCount = this.shownLogs.pipe(map((logs) => logs.length));

  logBodyViewHeight: ReplaySubject<number> = new ReplaySubject(1);

  highlightLogBinder = new ObservableCSSClassBinder(
    'highlight',
    'log-view-log-index',
    this.selectionManager.highlightLogIndices,
    new Set(),
  );
  selectedLogBinder = new ObservableCSSClassBinder(
    'selected',
    'log-view-log-index',
    this.selectionManager.selectedLogIndex.pipe(
      map((index) => new Set(index === -1 ? [] : [index])),
    ),
    new Set(),
  );

  disableScrollForNext = false;

  constructor(
    private inspectionDataStore: InspectionDataStoreService,
    private selectionManager: SelectionManagerService,
  ) {
    this.logBodyViewHeight.next(LogViewComponent.MINIMUM_LOG_LIST_SIZE); // initial value of the log view size.
    this.logViewSelectionMoveCommand
      .pipe(
        takeUntil(this.destoroyed),
        withLatestFrom(this.selectionManager.selectedLogIndex, this.shownLogs),
      )
      .subscribe(([command, index, logs]) => {
        if (index < 0) return;
        const currentSelected = this.searchArrayIndexOfLog(logs, index);
        if (index < 0) return;
        const direction = command.direction === 'prev' ? -1 : 1;
        const nextSelected = Math.max(
          0,
          Math.min(logs.length - 1, currentSelected + direction),
        );
        this.selectionManager.changeSelectionByLog(logs[nextSelected]);
      });
  }
  ngOnDestroy(): void {
    this.destoroyed.next();
  }

  ngOnInit(): void {
    this.initScrollEventOnScroll();
  }

  ngAfterViewInit(): void {
    // delay(0) is to trigger the following subscription handlers to be triggered in the next change detection cycle.
    // This observable needs @ViewChild to be initialized to obtain, but ngAfterViewInit doesn't allow triggering change detection.
    const containerHeight = monitorElementHeight(
      this.container.nativeElement,
    ).pipe(delay(0));
    // Limit logBodyViewHeight by comparing with container height.
    combineLatest([containerHeight, this.logBodyViewHeight])
      .pipe(takeUntil(this.destoroyed))
      .subscribe(([containerHeight, logBodyViewHeight]) => {
        if (
          containerHeight <
          logBodyViewHeight + LogViewComponent.MINIMUM_LOG_LIST_SIZE
        ) {
          // Adjust the size to be container size - min size of log list to keep log list area.
          this.logBodyViewHeight.next(
            containerHeight - LogViewComponent.MINIMUM_LOG_LIST_SIZE,
          );
        } else {
          if (logBodyViewHeight < LogViewComponent.MINIMUM_LOG_BODY_VIEW_SIZE) {
            // give up adjusting size if minimum sizes can't fit in the container to prevent call this subscription recursively.
            if (
              containerHeight <
              LogViewComponent.MINIMUM_LOG_LIST_SIZE +
                LogViewComponent.MINIMUM_LOG_BODY_VIEW_SIZE
            ) {
              return;
            }
            this.logBodyViewHeight.next(
              LogViewComponent.MINIMUM_LOG_BODY_VIEW_SIZE,
            );
          }
        }
      });
    // update viewport size of the virtual scroll area for logs.
    this.logBodyViewHeight.pipe(takeUntil(this.destoroyed)).subscribe(() => {
      this.viewPort.checkViewportSize();
    });

    // set the default log body height.
    containerHeight.pipe(take(1)).subscribe((containerHeight) => {
      this.logBodyViewHeight.next(
        containerHeight - LogViewComponent.DEFAULT_LOG_LIST_SIZE,
      );
    });
  }

  _selectLog(logEntry: LogEntry) {
    this.disableScrollForNext = true;
    this.selectionManager.changeSelectionByLog(logEntry);
  }

  _onLogHover(logEntry: LogEntry) {
    this.selectionManager.onHighlightLog(logEntry);
  }

  _resizeStart() {
    window.addEventListener('mouseup', () => {
      window.removeEventListener('mousemove', this._resizeMove);
    });
    window.addEventListener('mousemove', this._resizeMove);
  }

  _resizeMove = (e: MouseEvent) => {
    this.logBodyViewHeight.pipe(take(1)).subscribe((currentSize) => {
      this.logBodyViewHeight.next(currentSize - e.movementY);
    });
  };

  _onScroll() {
    this.selectedLogBinder.invalidate();
    this.highlightLogBinder.invalidate();
  }

  /**
   * filterLogsWithTimelines returns a list of LogEntries being related to any of given list of timelines.
   */
  private filterLogsWithTimelines(
    logs: LogEntry[],
    timelines: TimelineEntry[],
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

  private initScrollEventOnScroll() {
    combineLatest([this.shownLogs, this.selectedLog])
      .pipe(
        filter(([, selected]) => selected !== null),
        delay(1),
      ) // Wait virtual scroll box to update the list before scroll
      .subscribe(([logs, selected]) => {
        if (!this.disableScrollForNext) {
          const arrayIndex = this.searchArrayIndexOfLog(
            logs,
            selected!.logIndex,
          );
          if (arrayIndex >= 0) {
            this.viewPort.scrollToIndex(arrayIndex, 'smooth');
          }
        }
        this.disableScrollForNext = false;
      });
  }

  public setIncludeTimelineChildren(include: boolean) {
    this.includeTimelineChildren.next(include);
  }

  /**
   * Search the array index of logEntry in given log array.
   * This method assume the log array is already sorted in logIndex order.
   * This will return -1 when the specified logIndex is not found in the list of the logs.
   */
  private searchArrayIndexOfLog(logs: LogEntry[], logIndex: number): number {
    if (logs.length === 0) return -1;
    if (logs[0].logIndex === logIndex) return 0;

    // binary search
    let less = 0;
    let moreOrEqual = logs.length - 1;
    while (Math.abs(less - moreOrEqual) > 1) {
      const middle = Math.floor((less + moreOrEqual) / 2);
      if (logs[middle].logIndex < logIndex) {
        less = middle;
      } else {
        moreOrEqual = middle;
      }
    }

    return logs[moreOrEqual].logIndex == logIndex ? moreOrEqual : -1;
  }

  public onKeyDown(keyEvent: KeyboardEvent) {
    if (keyEvent.key === 'ArrowDown') {
      this.logViewSelectionMoveCommand.next({
        direction: 'next',
      });
      keyEvent.preventDefault();
    }
    if (keyEvent.key === 'ArrowUp') {
      this.logViewSelectionMoveCommand.next({
        direction: 'prev',
      });
      keyEvent.preventDefault();
    }
  }
}
