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
  LogTimelineFilter,
  LogTimelineFilterContext,
} from 'src/app/store/domain/filter/types';

/**
 * Provides a modular, filtered view of timelines and logs from a TimelineStore by applying a prioritized pipeline of LogTimelineFilters.
 * The view automatically re-evaluates whenever filters are added/removed or when any filter's internal signals change.
 */
export class TimelineView {
  private readonly filters = signal<Set<LogTimelineFilter>>(new Set());

  private readonly context = computed<LogTimelineFilterContext>(() => {
    const sortedFilters = Array.from(this.filters()).sort(
      (a, b) => a.priority() - b.priority(),
    );

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

    for (const filter of sortedFilters) {
      ctx = filter.process(ctx, this.store);
    }
    return ctx;
  });

  /**
   * Emits the final list of timelines that successfully passed the pipeline evaluation.
   */
  public readonly filteredTimelines = computed<
    ReadonlyDomainElement<Timeline>[]
  >(() => {
    const ctx = this.context();
    const allTimelines = this.store.timelines;
    return allTimelines.filter((t) => ctx.timelineIds.has(t.id));
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
  constructor(private readonly store: TimelineStore) {}

  /**
   * Registers a new filter step into the processing pipeline.
   */
  public addFilter(filter: LogTimelineFilter): void {
    this.filters.update((f) => {
      f.add(filter);
      return new Set(f);
    });
  }

  /**
   * Removes a previously registered filter step from the processing pipeline.
   */
  public removeFilter(filter: LogTimelineFilter): void {
    this.filters.update((f) => {
      f.delete(filter);
      return new Set(f);
    });
  }

  /**
   * Clears all registered filters from the processing pipeline.
   */
  public clearFilters(): void {
    this.filters.set(new Set());
  }
}
