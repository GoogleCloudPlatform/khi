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

import { signal } from '@angular/core';
import { Subject } from 'rxjs';
import {
  CancellationError,
  isCancellationError,
  LogTimelineFilter,
  LogTimelineFilterContext,
} from 'src/app/store/domain/filter/types';
import { IdBitset } from 'src/app/store/domain/filter/id-bitset';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';

/**
 * Filter parameters interface for updating BackendFilter.
 */
export interface BackendFilterParams {
  readonly timelineQuery?: string;
  readonly timelineExclusionQuery?: string;
  readonly logQuery?: string;
  readonly excludeNoLogs?: boolean;
}

/**
 * BackendFilter delegates CEL timeline queries, log queries, hierarchy expansion,
 * and log-matching filters to the backend Go Workbench service and caches results.
 */
export class BackendFilter implements LogTimelineFilter {
  /** The display name of the filter. */
  public readonly displayName = 'Backend filter';
  /** The priority value determining execution order. Priority 10 ensures it runs before local filters. */
  public readonly priority = 10;

  private readonly _onChanged = new Subject<void>();
  /** Emits whenever any filter query parameter signal changes. */
  public readonly onChanged = this._onChanged.asObservable();

  /** Active CEL expression for including timelines and expanding hierarchies. */
  public readonly timelineQuery = signal<string>('');
  /** Active CEL expression for excluding timelines. */
  public readonly timelineExclusionQuery = signal<string>('');
  /** Active CEL expression for filtering logs. */
  public readonly logQuery = signal<string>('');
  /** Whether to hide timelines that have no matching logs. */
  public readonly excludeNoLogs = signal<boolean>(false);

  private lastCacheKey: string | null = null;
  private lastResultContext: LogTimelineFilterContext | null = null;

  /**
   * Initializes a new instance of BackendFilter.
   *
   * @param workbenchClient - Optional client for communicating with the backend WorkbenchService.
   */
  constructor(private readonly workbenchClient?: WorkbenchClientService) {}

  /**
   * Updates multiple filter query parameters in a single call and triggers onChanged if changed.
   *
   * @param params - The filter parameters to update.
   */
  public updateFilterParams(params: BackendFilterParams): void {
    let changed = false;
    if (
      params.timelineQuery !== undefined &&
      params.timelineQuery !== this.timelineQuery()
    ) {
      this.timelineQuery.set(params.timelineQuery);
      changed = true;
    }
    if (
      params.timelineExclusionQuery !== undefined &&
      params.timelineExclusionQuery !== this.timelineExclusionQuery()
    ) {
      this.timelineExclusionQuery.set(params.timelineExclusionQuery);
      changed = true;
    }
    if (params.logQuery !== undefined && params.logQuery !== this.logQuery()) {
      this.logQuery.set(params.logQuery);
      changed = true;
    }
    if (
      params.excludeNoLogs !== undefined &&
      params.excludeNoLogs !== this.excludeNoLogs()
    ) {
      this.excludeNoLogs.set(params.excludeNoLogs);
      changed = true;
    }
    if (changed) {
      this._onChanged.next();
    }
  }

  /**
   * Notifies subscribers that filter signals were updated.
   */
  public notifyChanged(): void {
    this._onChanged.next();
  }

  /**
   * Invalidates the backend response cache, forcing the next evaluation to query the backend.
   */
  public invalidateCache(): void {
    this.lastCacheKey = null;
    this.lastResultContext = null;
  }

  /**
   * Executes the backend filtering pipeline or returns the cached result if parameters haven't changed.
   *
   * @param context - The incoming filter context.
   * @param timelineStore - The loaded timeline store.
   * @param signal - Optional AbortSignal to cancel backend RPC.
   * @param onProgress - Progress reporting callback.
   * @returns The updated filter context containing matched timeline IDs and log IDs.
   */
  public async process(
    context: LogTimelineFilterContext,
    timelineStore: TimelineStore,
    signal?: AbortSignal,
    onProgress?: (current: number, total: number) => void,
  ): Promise<LogTimelineFilterContext> {
    if (!this.workbenchClient || !this.workbenchClient.isWorkbenchActive()) {
      return context;
    }

    const currentKey = `${this.timelineQuery()}###${this.timelineExclusionQuery()}###${this.logQuery()}###${this.excludeNoLogs()}`;
    if (this.lastCacheKey === currentKey && this.lastResultContext) {
      return this.lastResultContext;
    }

    if (signal?.aborted) {
      throw new CancellationError();
    }

    try {
      const res = await this.workbenchClient.filterTimeline(
        {
          timelineQuery: this.timelineQuery(),
          timelineExclusionQuery: this.timelineExclusionQuery(),
          logQuery: this.logQuery(),
          excludeNoLogs: this.excludeNoLogs(),
        },
        (_stageName, current, total) => {
          onProgress?.(current, total);
        },
        signal,
      );

      this.lastCacheKey = currentKey;
      const resultContext: LogTimelineFilterContext = {
        timelineIds: IdBitset.fromSparseBitset(
          res.timelineMode,
          res.timelineBitset,
          timelineStore.timelines.length,
        ),
        logIds: IdBitset.fromSparseBitset(
          res.logMode,
          res.logBitset,
          timelineStore.logStore.count,
        ),
      };
      this.lastResultContext = resultContext;
      return resultContext;
    } catch (err) {
      if (signal?.aborted || isCancellationError(err)) {
        throw new CancellationError();
      }
      throw err;
    }
  }
}
