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

import { Injectable, effect, inject } from '@angular/core';
import { SelectionManager } from 'src/app/services/selection-manager.service';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';
import { Timeline, Revision } from 'src/app/store/domain/timeline';
import { Log } from 'src/app/store/domain/log';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { bisectLeft } from 'src/app/common/misc-util';

interface TimelineItemWithStructId {
  readonly timestamp: bigint;
  readonly structId: number;
}

/**
 * Service that orchestrates prefetching of Struct YAMLs in the background.
 *
 * Listens to selection changes in {@link SelectionManager} and delegates batch
 * prefetch requests to {@link WorkbenchClientService}.
 */
@Injectable({ providedIn: 'root' })
export class StructYamlPrefetchService {
  /**
   * Maximum number of struct IDs to prefetch when a timeline is selected.
   */
  public static readonly PREFETCH_TIMELINE_LIMIT = 50;

  /**
   * Radius of revisions before and after to prefetch when a revision is selected.
   */
  public static readonly PREFETCH_SURROUNDING_REVISIONS_RADIUS = 20;

  /**
   * Radius of logs before and after to prefetch when a log is selected.
   */
  public static readonly PREFETCH_SURROUNDING_LOGS_RADIUS = 20;

  private readonly selectionManager = inject(SelectionManager);
  private readonly inspectionDataStore = inject(InspectionDataStore);
  private readonly workbenchClient = inject(WorkbenchClientService);

  constructor() {
    this.watchSelection(this.selectionManager.selectedTimeline, (timeline) => {
      this.prefetchTimeline(timeline);
    });
    this.watchSelection(this.selectionManager.selectedRevision, (revision) => {
      this.prefetchSurroundingRevisions(revision);
    });
    this.watchSelection(this.selectionManager.selectedLog, (log) => {
      this.prefetchSurroundingLogs(log);
    });
  }

  /**
   * Watches a domain element selection signal and executes a callback when the selection changes to a new element.
   *
   * @param signal The selection signal to watch.
   * @param onSelect The callback to invoke when a new non-null element is selected.
   */
  private watchSelection<T extends { readonly id: number }>(
    signal: () => ReadonlyDomainElement<T> | null,
    onSelect: (item: ReadonlyDomainElement<T>) => void,
  ): void {
    let lastId: number | null = null;
    effect(() => {
      const item = signal();
      if (item === null) {
        lastId = null;
      } else if (item.id !== lastId) {
        lastId = item.id;
        onSelect(item);
      }
    });
  }

  /**
   * Computes the clamped index range [start, end) around a central index with a given radius.
   *
   * @param center The center index.
   * @param radius The radius before and after the center.
   * @param length The total length of the array.
   * @returns A tuple of [startIndex, endIndex].
   */
  private getSurroundingRange(
    center: number,
    radius: number,
    length: number,
  ): [number, number] {
    return [
      Math.max(0, center - radius),
      Math.min(length, center + radius + 1),
    ];
  }

  /**
   * Prefetches the initial batch of struct IDs from a selected timeline in chronological order.
   *
   * @param timeline The timeline to prefetch struct IDs for.
   */
  public prefetchTimeline(timeline: ReadonlyDomainElement<Timeline>): void {
    const items: TimelineItemWithStructId[] = [];

    for (const revision of timeline.revisions) {
      if (revision.structId > 0) {
        items.push({
          timestamp: revision.changedTime,
          structId: revision.structId,
        });
      }
      const log = revision.log;
      if (log && log.structId > 0) {
        items.push({
          timestamp: log.timestamp,
          structId: log.structId,
        });
      }
    }

    for (const event of timeline.events) {
      const log = event.log;
      if (log && log.structId > 0) {
        items.push({
          timestamp: event.timestamp,
          structId: log.structId,
        });
      }
    }

    if (items.length === 0) {
      return;
    }

    items.sort((a, b) =>
      a.timestamp < b.timestamp ? -1 : a.timestamp > b.timestamp ? 1 : 0,
    );

    const structIds: number[] = [];
    const seen = new Set<number>();
    for (const item of items) {
      if (!seen.has(item.structId)) {
        seen.add(item.structId);
        structIds.push(item.structId);
        if (
          structIds.length >= StructYamlPrefetchService.PREFETCH_TIMELINE_LIMIT
        ) {
          break;
        }
      }
    }

    this.workbenchClient.prefetchStructYAMLs(structIds);
  }

  /**
   * Prefetches the struct IDs of revisions surrounding the selected revision on the same timeline.
   *
   * @param revision The currently selected revision.
   */
  public prefetchSurroundingRevisions(
    revision: ReadonlyDomainElement<Revision>,
  ): void {
    const timeline = revision.timeline;
    if (!timeline) {
      return;
    }
    const revisions = timeline.revisions;
    if (revisions.length === 0) {
      return;
    }

    const [startIndex, endIndex] = this.getSurroundingRange(
      revision.index,
      StructYamlPrefetchService.PREFETCH_SURROUNDING_REVISIONS_RADIUS,
      revisions.length,
    );

    const structIds = new Set<number>();
    for (let i = startIndex; i < endIndex; i++) {
      const rev = revisions[i];
      if (rev.structId > 0) {
        structIds.add(rev.structId);
      }
      const log = rev.log;
      if (log && log.structId > 0) {
        structIds.add(log.structId);
      }
    }

    if (structIds.size > 0) {
      this.workbenchClient.prefetchStructYAMLs(Array.from(structIds));
    }
  }

  /**
   * Prefetches the struct IDs of logs surrounding the selected log in the filtered logs list.
   *
   * @param log The currently selected log.
   */
  public prefetchSurroundingLogs(log: ReadonlyDomainElement<Log>): void {
    const filteredLogs =
      this.inspectionDataStore.timelineView()?.filteredLogs() ?? [];
    if (filteredLogs.length === 0) {
      return;
    }

    const targetLogIndex = log.logIndex;
    const arrayIndex = bisectLeft(
      filteredLogs,
      targetLogIndex,
      (item, target) => item.logIndex - target,
    );

    const matchIndex =
      arrayIndex < filteredLogs.length &&
      filteredLogs[arrayIndex].logIndex === targetLogIndex
        ? arrayIndex
        : -1;

    if (matchIndex === -1) {
      return;
    }

    const [startIndex, endIndex] = this.getSurroundingRange(
      matchIndex,
      StructYamlPrefetchService.PREFETCH_SURROUNDING_LOGS_RADIUS,
      filteredLogs.length,
    );

    const structIds = new Set<number>();
    for (let i = startIndex; i < endIndex; i++) {
      const structId = filteredLogs[i].structId;
      if (structId > 0) {
        structIds.add(structId);
      }
    }

    if (structIds.size > 0) {
      this.workbenchClient.prefetchStructYAMLs(Array.from(structIds));
    }
  }
}
