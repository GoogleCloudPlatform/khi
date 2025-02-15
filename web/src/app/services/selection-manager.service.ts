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

import { Injectable } from '@angular/core';
import {
  BehaviorSubject,
  combineLatest,
  combineLatestWith,
  filter,
  map,
  shareReplay,
  startWith,
  Subject,
  take,
  withLatestFrom,
} from 'rxjs';
import { InspectionDataStoreService } from './inspection-data-store.service';
import { LogEntry, NullLog } from '../store/log';
import { ResourceTimeline } from '../store/timeline';
import { ResourceRevision } from '../store/revision';
import { ResourceEvent } from '../store/event';
type LogSelectionQuery = LogEntry | number | null;
type LogHighlightQuery = LogSelectionQuery | number[];

/**
 * SelectionManager provides selected/highlighted list of logs, timelines, revisions or events from the received user interaction.
 */
@Injectable({ providedIn: 'root' })
export class SelectionManagerService {
  private logSelectionQuery: Subject<LogSelectionQuery> = new Subject();

  /**
   * Return a selected LogEntity.
   */
  public selectedLog = this.inspectionDataStore.allLogs.pipe(
    combineLatestWith(this.logSelectionQuery),
    map(([logs, query]) => this._filterSelectedLog(logs, query)),
    startWith(null),
    shareReplay({
      bufferSize: 1,
      refCount: true,
    }),
  );

  private logHighlightQuery: Subject<LogHighlightQuery> = new Subject();
  /**
   * The list of logs highlighted.
   */
  public highlightedLogs = this.inspectionDataStore.allLogs.pipe(
    combineLatestWith(this.logHighlightQuery),
    map(([logs, query]) => this._filterHighlightedLogs(logs, query)),
    startWith([]),
    shareReplay({
      bufferSize: 1,
      refCount: true,
    }),
  );

  public selectedLogIndex = this.selectedLog.pipe(
    map((l) => (l !== null ? l.logIndex : -1)),
    shareReplay({
      bufferSize: 1,
      refCount: true,
    }),
  );

  public highlightLogIndices = this.highlightedLogs.pipe(
    map((logs) => new Set(logs.map((log) => log.logIndex))),
    shareReplay({
      bufferSize: 1,
      refCount: true,
    }),
  );

  private highlightedTimelineSubject: Subject<ResourceTimeline | null> =
    new Subject();

  public highlightedTimeline = this.highlightedTimelineSubject.pipe(
    startWith(null),
    shareReplay({
      bufferSize: 1,
      refCount: true,
    }),
  );

  /**
   * Emits highlighted revisions only on curent timeline.
   */
  public highlightedRevisionsOnCurrentTimeline = combineLatest([
    this.highlightedTimeline,
    this.highlightLogIndices,
  ]).pipe(
    map(([timeline, logIndices]) => {
      const result: ResourceRevision[] = [];
      if (timeline === null) return result;
      for (const revision of timeline.revisions) {
        if (logIndices.has(revision.logIndex)) {
          result.push(revision);
        }
      }
      return result;
    }),
    startWith<ResourceRevision[]>([]),
    shareReplay({
      bufferSize: 1,
      refCount: true,
    }),
  );

  private selectedTimelineSubject: Subject<ResourceTimeline | null> =
    new Subject();

  public selectedTimeline = this.selectedTimelineSubject.pipe(
    startWith(null),
    shareReplay({
      bufferSize: 1,
      refCount: true,
    }),
  );

  private selectedRevisionSubject: Subject<ResourceRevision | null> =
    new Subject();

  public selectedRevision = this.selectedRevisionSubject.pipe(
    startWith(null),
    shareReplay({
      bufferSize: 1,
      refCount: true,
    }),
  );
  public previousOfSelectedRevision = this.selectedRevision.pipe(
    startWith(null),
    combineLatestWith(this.selectedTimeline),
    map(([revision, timeline]) => {
      if (revision === null) return null;
      if (timeline === null) return null;
      const revisionIndex = timeline.revisions.indexOf(revision);
      return revisionIndex > 0 ? timeline.revisions[revisionIndex - 1] : null;
    }),
  );
  public timelineSelectionShouldIncludeChildren = new BehaviorSubject(true);

  /**
   * Returns an array of TimelineEntry selected or children of selection in recursive.
   */
  public selectedTimelinesWithChildren = this.selectedTimeline.pipe(
    combineLatestWith(this.timelineSelectionShouldIncludeChildren),
    map(([selectedTimeline, shouldIncludeChild]) => {
      if (!selectedTimeline) return [];
      if (!shouldIncludeChild) return [selectedTimeline];
      return [selectedTimeline, ...selectedTimeline.getAllChildrenRecursive()];
    }),
    shareReplay({
      bufferSize: 1,
      refCount: true,
    }),
  );
  /**
   * Array of children TimelineEntry of the selected timelines.
   * This will be always an empty array when timelineSelectionShouldIncludeChildren is false.
   */
  public highlightedChildrenOfSelectedTimeline = this.selectedTimeline.pipe(
    combineLatestWith(
      this.timelineSelectionShouldIncludeChildren,
      this.inspectionDataStore.allTimelines,
    ),
    map(([selectedTimeline, includeChildren, allTimelines]) => {
      if (!includeChildren) return [];
      if (selectedTimeline === null) return [];
      for (const timeline of allTimelines) {
        if (timeline === selectedTimeline) {
          return timeline.getAllChildrenRecursive();
        }
      }
      return [];
    }),
    map((timelines) => new Set(timelines)),
    shareReplay({
      bufferSize: 1,
      refCount: true,
    }),
  );
  constructor(private inspectionDataStore: InspectionDataStoreService) {
    // Change selection status when current timeline selection was changed.
    this.selectedTimeline
      .pipe(
        filter((timeline): timeline is ResourceTimeline => !!timeline),
        withLatestFrom(this.selectedLog, this.selectedRevision),
      )
      .subscribe(([timeline, log, revision]) => {
        // Check if current selected log is included in the timeline
        if (log) {
          let included = false;
          for (const revision of timeline.revisions) {
            if (revision.logIndex === log.logIndex) {
              included = true;
              break;
            }
          }
          if (!included) {
            for (const event of timeline.events) {
              if (event.logIndex === log.logIndex) {
                included = true;
                break;
              }
            }
          }
          if (!included) {
            // current selected log is not included in the newly selected timeline. Deselect the log.
            this.changeSelectionByLog(-1);
          }
        }
        if (revision) {
          for (const r of timeline.revisions) {
            if (r === revision) {
              return;
            }
          }
          this.selectedRevisionSubject.next(null);
        }
      });
  }

  public onSelectTimeline(timeline: ResourceTimeline | string | null) {
    if (typeof timeline === 'string') {
      this.inspectionDataStore.inspectionData
        .pipe(
          take(1),
          filter((inspectionData) => !!inspectionData),
        )
        .subscribe((inspectionData) => {
          const timelineEntry =
            inspectionData!.getTimelineByResourcePath(timeline);
          if (timelineEntry) {
            this.selectedTimelineSubject.next(timelineEntry);
          } else {
            console.warn(timeline);
          }
        });
      return;
    }
    this.selectedTimelineSubject.next(timeline);
  }

  public onHighlightTimeline(timeline: ResourceTimeline | string | null) {
    if (typeof timeline === 'string') {
      this.inspectionDataStore.inspectionData
        .pipe(
          take(1),
          filter((inspectionData) => !!inspectionData),
        )
        .subscribe((inspectionData) => {
          const timelineEntry =
            inspectionData!.getTimelineByResourcePath(timeline);
          if (timelineEntry) {
            this.highlightedTimelineSubject.next(timelineEntry);
          } else {
            console.warn(timeline);
          }
        });
      return;
    }
    this.highlightedTimelineSubject.next(timeline);
  }

  /**
   * Notify specified log element or log array to be highlighted.
   * Highlight target can be multiple when multiple events or revisions placed nearby and hover mouse on them.
   */
  public onHighlightLog(log: LogHighlightQuery) {
    this.logHighlightQuery.next(log);
  }

  /**
   * Select a log.
   * Calling this function changes other resource selections (Timeline, Event or Revision) in response to the log selection change.
   * @param log the selected log.
   */
  public changeSelectionByLog(log: LogEntry | number) {
    this.changeSelectionByLogInternal(log, false);
  }

  /**
   * Select an event.
   * Calling this function changes other resource selections (Timeline, Log or Revision) in response to the event selection change.
   * @param timeline the timeline containing selected event
   * @param event selected event
   */
  public changeSelectionByEvent(
    timeline: ResourceTimeline,
    event: ResourceEvent,
  ) {
    this.changeSelectionByEventInternal(timeline, event, false);
  }

  /**
   * Select a revision.
   * Calling this function changes other resource selections (Timeline, Log or Event) in response to the revision selection change.
   * @param timeline the timeline containing selected revision
   * @param revision selected revision
   */
  public changeSelectionByRevision(
    timeline: ResourceTimeline,
    revision: ResourceRevision,
  ) {
    this.changeSelectionByRevisionInternal(timeline, revision, false);
  }

  private changeSelectionByLogInternal(
    log: LogEntry | number,
    ignoreResourceSelect: boolean,
  ) {
    this.logSelectionQuery.next(log);
    if (!ignoreResourceSelect) {
      combineLatest([this.selectedLog, this.selectedTimelinesWithChildren])
        .pipe(take(1))
        .subscribe(([log, timelines]) => {
          if (!log) return;
          // Find if the new log selection isn't inside of the timeline selection
          if (
            !timelines.find(
              (timeline) =>
                timeline.events.find((e) => e.logIndex === log.logIndex) ||
                timeline.revisions.find((r) => r.logIndex === log.logIndex),
            )
          ) {
            for (const timeline of log.relatedTimelines) {
              // Pick a related timeline of this log
              const relatedRevision = timeline.revisions.find(
                (r) => r.logIndex === log.logIndex,
              );
              if (relatedRevision) {
                this.changeSelectionByRevisionInternal(
                  timeline,
                  relatedRevision,
                  true,
                );
              }
              const relatedEvent = timeline.events.find(
                (r) => r.logIndex === log.logIndex,
              );
              if (relatedEvent) {
                this.changeSelectionByEventInternal(
                  timeline,
                  relatedEvent,
                  true,
                );
              }
              return;
            }
          }
        });
    }
  }

  private changeSelectionByEventInternal(
    timeline: ResourceTimeline,
    event: ResourceEvent,
    ignoreLogSelect: boolean,
  ) {
    this.selectedTimelineSubject.next(timeline);

    if (!ignoreLogSelect)
      this.changeSelectionByLogInternal(event.logIndex, true);
  }

  private changeSelectionByRevisionInternal(
    timeline: ResourceTimeline,
    revision: ResourceRevision,
    ignoreLogSelect: boolean,
  ) {
    this.selectedRevisionSubject.next(revision);
    this.selectedTimelineSubject.next(timeline);

    if (!ignoreLogSelect)
      this.changeSelectionByLogInternal(revision.logIndex, true);
  }

  private _filterSelectedLog(
    logs: LogEntry[],
    query: LogSelectionQuery,
  ): LogEntry | null {
    return (
      this._filterLogsWithIndexSet(
        this._highlightQueryToIndexSet(query),
        logs,
      )[0] ?? null
    );
  }

  private _filterHighlightedLogs(
    logs: LogEntry[],
    query: LogHighlightQuery,
  ): LogEntry[] {
    return this._filterLogsWithIndexSet(
      this._highlightQueryToIndexSet(query),
      logs,
    );
  }

  private _filterLogsWithIndexSet(
    indexSet: Set<number>,
    logs: LogEntry[],
  ): LogEntry[] {
    const result: LogEntry[] = [];
    for (const index of indexSet) {
      if (index >= 0 && index < logs.length) result.push(logs[index]);
      else if (index === -1) result.push(NullLog);
    }
    return result;
  }

  private _highlightQueryToIndexSet(query: LogHighlightQuery): Set<number> {
    if (query === null) return new Set();
    if (Array.isArray(query)) return new Set(query);
    if (typeof query === 'number') return new Set([query]);
    return new Set([query.logIndex]);
  }
}
