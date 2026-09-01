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
import { LogStore } from 'src/app/store/domain/log-store';
import { IdBitset } from 'src/app/store/domain/filter/id-bitset';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { bisectLeft } from 'src/app/common/misc-util';

interface TimelineItemWithStructId {
  readonly timestamp: bigint;
  readonly structId: number;
}

interface SurroundingCandidate {
  readonly logIndex: number;
  readonly structIds: readonly number[];
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
   * Prefetches the struct IDs of logs surrounding the selected log.
   *
   * @param log The currently selected log.
   */
  public prefetchSurroundingLogs(log: ReadonlyDomainElement<Log>): void {
    const data = this.inspectionDataStore.inspectionData();
    const timelineView = this.inspectionDataStore.timelineView();
    if (!data || !timelineView) {
      return;
    }

    const logStore = data.logStore;
    const filteredLogIds = timelineView.filteredLogIds();
    const selectedTimelines =
      this.selectionManager.selectedTimelinesWithChildren();
    const targetLogIndex = log.logIndex;
    const structIds = new Set<number>();

    // Include the selected log's struct ID if available
    if (log.structId > 0) {
      structIds.add(log.structId);
    }

    const radius = StructYamlPrefetchService.PREFETCH_SURROUNDING_LOGS_RADIUS;

    if (selectedTimelines.length === 0) {
      // Pass 1: Timeline filter is OFF. Search directly within LogStore around targetLogIndex.
      this.collectSurroundingLogStoreStructIds(
        logStore,
        targetLogIndex,
        filteredLogIds,
        radius,
        structIds,
      );
    } else {
      // Pass 2: Timeline filter is ON.
      // Collect surrounding items from revisions and events for all selected timelines.
      const backwardCandidates: SurroundingCandidate[] = [];
      const forwardCandidates: SurroundingCandidate[] = [];

      for (const timeline of selectedTimelines) {
        this.collectSurroundingTimelineItemCandidates(
          timeline.revisions,
          targetLogIndex,
          filteredLogIds,
          (rev) => [rev.structId, logStore.getBodyStructId(rev.logId)],
          backwardCandidates,
          forwardCandidates,
        );
        this.collectSurroundingTimelineItemCandidates(
          timeline.events,
          targetLogIndex,
          filteredLogIds,
          (evt) => [logStore.getBodyStructId(evt.logId)],
          backwardCandidates,
          forwardCandidates,
        );
      }

      // Add nearest backward candidates (descending logIndex order)
      this.addTopCandidates(
        backwardCandidates,
        (a, b) => b.logIndex - a.logIndex,
        radius,
        structIds,
      );

      // Add nearest forward candidates (ascending logIndex order)
      this.addTopCandidates(
        forwardCandidates,
        (a, b) => a.logIndex - b.logIndex,
        radius,
        structIds,
      );
    }

    if (structIds.size > 0) {
      this.workbenchClient.prefetchStructYAMLs(Array.from(structIds));
    }
  }

  /**
   * Scans logStore directly to collect surrounding struct IDs when timeline filter is inactive.
   *
   * @param logStore The domain log store.
   * @param targetLogIndex The chronological log index of the selected log.
   * @param filteredLogIds The bitset of filtered log IDs.
   * @param radius The maximum number of items to scan backwards and forwards.
   * @param structIds Output set of collected struct IDs.
   */
  private collectSurroundingLogStoreStructIds(
    logStore: LogStore,
    targetLogIndex: number,
    filteredLogIds: IdBitset,
    radius: number,
    structIds: Set<number>,
  ): void {
    let backwardCount = 0;
    for (let i = targetLogIndex - 1; i >= 0 && backwardCount < radius; i--) {
      const id = logStore.getLogIdByIndex(i);
      if (filteredLogIds.has(id)) {
        backwardCount++;
        const structId = logStore.getBodyStructId(id);
        if (structId > 0) {
          structIds.add(structId);
        }
      }
    }

    let forwardCount = 0;
    for (
      let i = targetLogIndex + 1;
      i < logStore.count && forwardCount < radius;
      i++
    ) {
      const id = logStore.getLogIdByIndex(i);
      if (filteredLogIds.has(id)) {
        forwardCount++;
        const structId = logStore.getBodyStructId(id);
        if (structId > 0) {
          structIds.add(structId);
        }
      }
    }
  }

  /**
   * Collects surrounding candidate struct IDs from an ordered array of timeline items (revisions or events).
   *
   * @param items The chronologically ordered array of timeline items.
   * @param targetLogIndex The logIndex of the currently selected log.
   * @param filteredLogIds The bitset of filtered log IDs.
   * @param getStructIds A function returning an array of candidate struct IDs for an item.
   * @param backwardCandidates Output array for items before targetLogIndex.
   * @param forwardCandidates Output array for items at or after targetLogIndex.
   */
  private collectSurroundingTimelineItemCandidates<
    T extends { readonly logIndex: number; readonly logId: number },
  >(
    items: readonly T[],
    targetLogIndex: number,
    filteredLogIds: IdBitset,
    getStructIds: (item: T) => number[],
    backwardCandidates: SurroundingCandidate[],
    forwardCandidates: SurroundingCandidate[],
  ): void {
    if (items.length === 0) {
      return;
    }
    const idx = bisectLeft(
      items,
      targetLogIndex,
      (item, target) => item.logIndex - target,
    );
    const radius = StructYamlPrefetchService.PREFETCH_SURROUNDING_LOGS_RADIUS;

    // Backward items before targetLogIndex
    const startBackward = Math.max(0, idx - radius);
    for (let i = idx - 1; i >= startBackward; i--) {
      const item = items[i];
      if (filteredLogIds.has(item.logId)) {
        const candidateStructIds = getStructIds(item).filter((id) => id > 0);
        if (candidateStructIds.length > 0) {
          backwardCandidates.push({
            logIndex: item.logIndex,
            structIds: candidateStructIds,
          });
        }
      }
    }

    // Forward items at or after targetLogIndex
    const endForward = Math.min(items.length, idx + radius);
    for (let i = idx; i < endForward; i++) {
      const item = items[i];
      if (filteredLogIds.has(item.logId)) {
        const candidateStructIds = getStructIds(item).filter((id) => id > 0);
        if (candidateStructIds.length > 0) {
          forwardCandidates.push({
            logIndex: item.logIndex,
            structIds: candidateStructIds,
          });
        }
      }
    }
  }

  /**
   * Sorts candidate items, selects up to the limit, and adds their struct IDs to the output set.
   *
   * @param candidates The list of collected candidates.
   * @param compareFn Comparison function for ordering.
   * @param limit Maximum number of items to select.
   * @param structIds Output set of struct IDs.
   */
  private addTopCandidates(
    candidates: SurroundingCandidate[],
    compareFn: (a: SurroundingCandidate, b: SurroundingCandidate) => number,
    limit: number,
    structIds: Set<number>,
  ): void {
    candidates.sort(compareFn);
    const top = candidates.slice(0, limit);
    for (const item of top) {
      for (const id of item.structIds) {
        structIds.add(id);
      }
    }
  }
}
