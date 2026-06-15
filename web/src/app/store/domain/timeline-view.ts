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
import { Log } from 'src/app/store/domain/log';
import { Timeline } from 'src/app/store/domain/timeline';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import {
  CancellationError,
  LogTimelineFilter,
  LogTimelineFilterContext,
} from 'src/app/store/domain/filter/types';
import { Subscription } from 'rxjs';

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

  private readonly _context = signal<LogTimelineFilterContext>({
    timelineIds: new Set(),
    logIds: new Set(),
  });
  private readonly _isFiltering = signal<boolean>(false);
  private readonly _progress = signal<FilteringProgressInfo | null>(null);
  private activeAbortController?: AbortController;

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
   * Emits the final list of logs that successfully passed the pipeline evaluation.
   */
  public readonly filteredLogs = computed<ReadonlyDomainElement<Log>[]>(() => {
    const ctx = this.context();
    const logs: ReadonlyDomainElement<Log>[] = [];
    for (const id of ctx.logIds) {
      logs.push(this.store.logStore.getLog(id));
    }
    logs.sort((a, b) => {
      const tsA = a.timestamp ?? 0n;
      const tsB = b.timestamp ?? 0n;
      return tsA < tsB ? -1 : tsA > tsB ? 1 : 0;
    });
    return logs;
  });

  /**
   * Emits the set of log IDs that successfully passed the pipeline evaluation.
   */
  public readonly filteredLogIds = computed<Set<number>>(() => {
    return this.context().logIds;
  });

  /**
   * Initializes a new instance of the TimelineView utilizing the target timeline store.
   */
  constructor(private readonly store: TimelineStore) {
    // Initialize context with all timelines/logs initially
    const allTimelines = this.store.timelines;
    const allTimelineIds = new Set(allTimelines.map((t) => t.id));
    const allLogIds = new Set<number>();
    for (const l of this.store.logStore.logs()) {
      allLogIds.add(l.id);
    }
    this._context.set({
      timelineIds: allTimelineIds,
      logIds: allLogIds,
    });
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

    const allTimelines = this.store.timelines;
    const allTimelineIds = new Set(allTimelines.map((t) => t.id));
    const allLogIds = new Set<number>();
    for (const l of this.store.logStore.logs()) {
      allLogIds.add(l.id);
    }

    let ctx: LogTimelineFilterContext = {
      timelineIds: allTimelineIds,
      logIds: allLogIds,
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
      if (err instanceof CancellationError) {
        return;
      }
      console.warn('Error during async filtering pipeline:', err);
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
