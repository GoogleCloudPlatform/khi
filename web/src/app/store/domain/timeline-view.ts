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

import { signal, computed } from '@angular/core';
import { Timeline } from 'src/app/store/domain/timeline';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import {
  isCancellationError,
  LogTimelineFilter,
  LogTimelineFilterContext,
} from 'src/app/store/domain/filter/types';
import { IdBitset } from 'src/app/store/domain/filter/id-bitset';
import { Subscription } from 'rxjs';
import { CollapseTimelineFilter } from 'src/app/store/domain/filter/collapse-filter';
import { BackendFilter } from 'src/app/store/domain/filter/backend-filter';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';
import { SparseBitset } from 'src/app/generated/api/v1/sparse_bitset_pb';

/**
 * Holds the progress information of a specific filter step.
 */
export interface FilteringProgressInfo {
  /** The name of the filter class currently being executed. */
  filterName: string;
  /** The number of evaluated items in the current step. */
  current: number;
  /** The total number of items to evaluate in the current step. */
  total: number;
}

/**
 * Provides a modular, filtered view of timelines and logs from a TimelineStore by applying a prioritized pipeline of LogTimelineFilters.
 * The view automatically re-evaluates whenever filters are added/removed or when any filter's internal signals change.
 */
export class TimelineView {
  private readonly filters = new Set<LogTimelineFilter>();
  private readonly subscriptions = new Map<LogTimelineFilter, Subscription>();
  private readonly collapseFilter = new CollapseTimelineFilter();
  public readonly backendFilter: BackendFilter;

  /** Active CEL expression for including timelines and expanding hierarchies. */
  public readonly timelineQuery;
  /** Active CEL expression for excluding timelines. */
  public readonly timelineExclusionQuery;
  /** Active CEL expression for filtering logs. */
  public readonly logQuery;
  /** Whether to hide timelines that have no matching logs. */
  public readonly excludeNoLogs;

  private readonly _context = signal<LogTimelineFilterContext>({
    timelineIds: IdBitset.createEmpty(),
    logIds: IdBitset.createEmpty(),
  });
  private readonly _isFiltering = signal<boolean>(false);
  private readonly _progress = signal<FilteringProgressInfo | null>(null);
  private readonly _collapsedTimelineIds = signal<ReadonlySet<number>>(
    new Set(),
  );
  private activeAbortController?: AbortController;

  /**
   * Exposes the set of currently collapsed timeline IDs.
   */
  public readonly collapsedTimelineIds =
    this._collapsedTimelineIds.asReadonly();

  /**
   * Exposes whether a filtering task is currently executing.
   */
  public readonly isFiltering = this._isFiltering.asReadonly();

  /**
   * Exposes the current concrete filtering progress.
   */
  public readonly progress = this._progress.asReadonly();

  private readonly context = this._context.asReadonly();

  /**
   * Emits the final list of timelines that successfully passed the pipeline evaluation.
   */
  public readonly filteredTimelines = computed<
    ReadonlyDomainElement<Timeline>[]
  >(() => {
    const ctx = this.context();
    return this.store.timelines.filter((t) => ctx.timelineIds.has(t.id));
  });

  /**
   * Emits the SparseBitset encoding the currently filtered timeline IDs.
   */
  public readonly filteredTimelineBitset = computed<SparseBitset>(() => {
    const ctx = this.context();
    return ctx.timelineIds.toSparseBitset();
  });

  /**
   * Emits the bitset of timeline IDs that successfully passed the pipeline evaluation.
   */
  public readonly filteredTimelineIds = computed<IdBitset>(() => {
    return this.context().timelineIds;
  });

  /**
   * Emits the bitset of log IDs that successfully passed the pipeline evaluation.
   */
  public readonly filteredLogIds = computed<IdBitset>(() => {
    return this.context().logIds;
  });

  /**
   * Initializes a new instance of the TimelineView utilizing the target timeline store and optional WorkbenchClientService.
   */
  constructor(
    private readonly store: TimelineStore,
    private readonly workbenchClient?: WorkbenchClientService,
  ) {
    this.backendFilter = new BackendFilter(this.workbenchClient);
    this.timelineQuery = this.backendFilter.timelineQuery;
    this.timelineExclusionQuery = this.backendFilter.timelineExclusionQuery;
    this.logQuery = this.backendFilter.logQuery;
    this.excludeNoLogs = this.backendFilter.excludeNoLogs;

    // Initialize context with all timelines/logs initially
    this._context.set({
      timelineIds: IdBitset.fromSequential(this.store.timelines.length),
      logIds: IdBitset.fromSequential(this.store.logStore.count),
    });

    this.addFilter(this.backendFilter);
    this.addFilter(this.collapseFilter);
  }

  /**
   * Expands all collapsed timelines.
   */
  public expandAllTimelines(): void {
    this.collapseFilter.expandAll();
    this._collapsedTimelineIds.set(this.collapseFilter.collapsedTimelineIds);
  }

  /**
   * Expands direct children timelines of a parent timeline.
   * @param parentTimeline - The parent timeline whose direct children will be expanded.
   */
  public expandChildren(parentTimeline: Timeline): void {
    this.collapseFilter.expandChildren(parentTimeline);
    this._collapsedTimelineIds.set(this.collapseFilter.collapsedTimelineIds);
  }

  /**
   * Collapses direct children timelines of a parent timeline.
   * @param parentTimeline - The parent timeline whose direct children will be collapsed.
   */
  public collapseChildren(parentTimeline: Timeline): void {
    this.collapseFilter.collapseChildren(parentTimeline);
    this._collapsedTimelineIds.set(this.collapseFilter.collapsedTimelineIds);
  }

  /**
   * Expands a parent timeline and all of its descendants recursively.
   * @param parentTimeline - The parent timeline to expand recursively.
   */
  public expandDescendants(parentTimeline: Timeline): void {
    this.collapseFilter.expandDescendants(parentTimeline);
    this._collapsedTimelineIds.set(this.collapseFilter.collapsedTimelineIds);
  }

  /**
   * Collapses a parent timeline and all of its descendants recursively.
   * @param parentTimeline - The parent timeline to collapse recursively.
   */
  public collapseDescendants(parentTimeline: Timeline): void {
    this.collapseFilter.collapseDescendants(parentTimeline);
    this._collapsedTimelineIds.set(this.collapseFilter.collapsedTimelineIds);
  }

  /**
   * Collapses a specific timeline.
   * @param timelineId - The ID of the timeline to collapse.
   */
  public collapseTimeline(timelineId: number): void {
    this.collapseFilter.collapseTimeline(timelineId);
    this._collapsedTimelineIds.set(this.collapseFilter.collapsedTimelineIds);
  }

  /**
   * Toggles the collapse state of a specific timeline.
   * @param timelineId - The ID of the timeline to toggle collapse.
   */
  public toggleTimelineCollapse(timelineId: number): void {
    this.collapseFilter.toggleTimelineCollapse(timelineId);
    this._collapsedTimelineIds.set(this.collapseFilter.collapsedTimelineIds);
  }

  private pipelineScheduled = false;

  private schedulePipeline(): void {
    if (this.pipelineScheduled) return;
    this.pipelineScheduled = true;
    Promise.resolve().then(() => {
      this.pipelineScheduled = false;
      this.runPipeline(Array.from(this.filters));
    });
  }

  private async runPipeline(filters: LogTimelineFilter[]): Promise<void> {
    this.activeAbortController?.abort();
    const abortController = new AbortController();
    this.activeAbortController = abortController;

    const sortedFilters = [...filters].sort((a, b) => a.priority - b.priority);

    this._isFiltering.set(true);

    let ctx: LogTimelineFilterContext = {
      timelineIds: IdBitset.fromSequential(this.store.timelines.length),
      logIds: IdBitset.fromSequential(this.store.logStore.count),
    };

    try {
      for (const filter of sortedFilters) {
        ctx = await filter.process(
          ctx,
          this.store,
          abortController.signal,
          (current, total) => {
            if (!abortController.signal.aborted) {
              this._progress.set({
                filterName: filter.displayName,
                current,
                total,
              });
            }
          },
        );
      }

      if (!abortController.signal.aborted) {
        this._context.set(ctx);
      }
    } catch (err) {
      if (abortController.signal.aborted || isCancellationError(err)) {
        return;
      }
      console.error('Error during async filtering pipeline:', err);
    } finally {
      if (this.activeAbortController === abortController) {
        this._isFiltering.set(false);
        this._progress.set(null);
      }
    }
  }

  /**
   * Registers a new filter step into the processing pipeline.
   */
  public addFilter(filter: LogTimelineFilter): void {
    if (this.filters.has(filter)) return;
    this.filters.add(filter);

    if (filter.onChanged) {
      const sub = filter.onChanged.subscribe(() => {
        this.schedulePipeline();
      });
      this.subscriptions.set(filter, sub);
    }
    this.schedulePipeline();
  }

  /**
   * Removes a previously registered filter step from the processing pipeline.
   */
  public removeFilter(filter: LogTimelineFilter): void {
    if (!this.filters.has(filter)) return;
    this.filters.delete(filter);

    const sub = this.subscriptions.get(filter);
    if (sub) {
      sub.unsubscribe();
      this.subscriptions.delete(filter);
    }
    this.schedulePipeline();
  }

  /**
   * Clears all registered filters from the processing pipeline.
   */
  public clearFilters(): void {
    for (const sub of this.subscriptions.values()) {
      sub.unsubscribe();
    }
    this.subscriptions.clear();
    this.filters.clear();
    this.schedulePipeline();
  }
}
