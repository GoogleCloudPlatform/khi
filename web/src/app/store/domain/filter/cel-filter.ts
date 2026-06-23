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

import { Injectable, signal, inject } from '@angular/core';
import { Subject } from 'rxjs';
import {
  CancellationError,
  LogTimelineFilter,
  LogTimelineFilterContext,
} from 'src/app/store/domain/filter/types';
import { Timeline } from 'src/app/store/domain/timeline';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { CelFilterUpdateResult } from 'src/app/store/domain/filter/cel-types';
import {
  CELTimelineFilterEnvironment,
  CELLogFilterEnvironment,
} from 'src/app/store/domain/filter/cel-env';
import { SearchWorkerManager } from 'src/app/services/search-worker-manager.service';

/**
 * Filters timelines based on the configured CEL expression.
 */
@Injectable({ providedIn: 'root' })
export class CelTimelineFilter implements LogTimelineFilter {
  readonly displayName = 'Timeline CEL filter';
  readonly priority = 10;
  private readonly searchWorkerManager = inject(SearchWorkerManager);
  private readonly celEnv = new CELTimelineFilterEnvironment();
  private readonly _celExpr = signal<string>('');
  private readonly _onChanged = new Subject<void>();

  /**
   * Emits whenever the filter's internal configurations or state changes.
   */
  public readonly onChanged = this._onChanged.asObservable();

  /**
   * The currently active CEL timeline filter expression as a read-only signal.
   */
  public readonly celExpr = this._celExpr.asReadonly();

  /**
   * Validates the given CEL timeline expression against the registered environment schemas.
   */
  public validate(celExpr: string): CelFilterUpdateResult {
    const tempEnv = new CELTimelineFilterEnvironment();
    return tempEnv.compile(celExpr);
  }

  /**
   * Updates the evaluator with a new CEL expression, validating its syntax beforehand.
   */
  public updateFilter(celExpr: string): CelFilterUpdateResult {
    const compileRes = this.celEnv.compile(celExpr);
    if (!compileRes.success) {
      this._celExpr.set('');

      return compileRes;
    }
    this._celExpr.set(celExpr);
    this._onChanged.next();
    return { success: true };
  }

  /**
   * Evaluates each timeline in the context against the CEL expression and retains only matching timeline IDs.
   */
  public async process(
    context: LogTimelineFilterContext,
    timelineStore: TimelineStore,
    signal?: AbortSignal,
    onProgress?: (current: number, total: number) => void,
  ): Promise<LogTimelineFilterContext> {
    if (this.celExpr() === '') {
      return context;
    }
    if (signal?.aborted) {
      throw new CancellationError();
    }

    onProgress?.(0, context.timelineIds.size);

    const passedTimelineIds = await this.searchWorkerManager.searchTimelines(
      this.celExpr(),
      onProgress,
    );

    if (signal?.aborted) {
      throw new CancellationError();
    }

    const filteredIds = new Set<number>();
    for (const id of context.timelineIds) {
      if (passedTimelineIds.has(id)) {
        filteredIds.add(id);
      }
    }

    return {
      timelineIds: filteredIds,
      logIds: context.logIds,
    };
  }
}

/**
 * Filters timelines based on the configured CEL exclusion expression, removing matched timelines and their descendants.
 */
@Injectable({ providedIn: 'root' })
export class CelTimelineExclusionFilter implements LogTimelineFilter {
  readonly displayName = 'Timeline CEL exclusion filter';
  readonly priority = 25;
  private readonly searchWorkerManager = inject(SearchWorkerManager);
  private readonly celEnv = new CELTimelineFilterEnvironment();
  private readonly _celExpr = signal<string>('');
  private readonly _onChanged = new Subject<void>();

  /**
   * Emits whenever the filter's internal configurations or state changes.
   */
  public readonly onChanged = this._onChanged.asObservable();

  /**
   * The currently active CEL timeline exclusion filter expression as a read-only signal.
   */
  public readonly celExpr = this._celExpr.asReadonly();

  /**
   * Validates the given CEL timeline expression against the registered environment schemas.
   */
  public validate(celExpr: string): CelFilterUpdateResult {
    const tempEnv = new CELTimelineFilterEnvironment();
    return tempEnv.compile(celExpr);
  }

  /**
   * Updates the evaluator with a new CEL expression, validating its syntax beforehand.
   */
  public updateFilter(celExpr: string): CelFilterUpdateResult {
    const compileRes = this.celEnv.compile(celExpr);
    if (!compileRes.success) {
      this._celExpr.set('');
      return compileRes;
    }
    this._celExpr.set(celExpr);
    this._onChanged.next();
    return { success: true };
  }

  /**
   * Evaluates each timeline in the context against the CEL exclusion expression and removes matching timeline IDs and their descendants.
   */
  public async process(
    context: LogTimelineFilterContext,
    timelineStore: TimelineStore,
    signal?: AbortSignal,
    onProgress?: (current: number, total: number) => void,
  ): Promise<LogTimelineFilterContext> {
    if (this.celExpr() === '') {
      return context;
    }
    if (signal?.aborted) {
      throw new CancellationError();
    }

    onProgress?.(0, context.timelineIds.size);

    const excludedTimelineIds = await this.searchWorkerManager.searchTimelines(
      this.celExpr(),
      onProgress,
    );

    if (signal?.aborted) {
      throw new CancellationError();
    }

    const filteredIds = new Set<number>();
    const isTimelineAndAncestorsValid = (
      t: ReadonlyDomainElement<Timeline>,
    ): boolean => {
      let current: ReadonlyDomainElement<Timeline> | null | undefined = t;
      while (current) {
        if (excludedTimelineIds.has(current.id)) {
          return false;
        }
        current = current.parent;
      }
      return true;
    };

    for (const id of context.timelineIds) {
      const t = timelineStore.getTimeline(id);
      if (t && isTimelineAndAncestorsValid(t)) {
        filteredIds.add(id);
      }
    }

    return {
      timelineIds: filteredIds,
      logIds: context.logIds,
    };
  }
}

/**
 * Filters logs based on the configured CEL expression.
 */
@Injectable({ providedIn: 'root' })
export class CelLogFilter implements LogTimelineFilter {
  readonly displayName = 'Log CEL Filter';
  readonly priority = 30;
  private readonly searchWorkerManager = inject(SearchWorkerManager);
  private readonly celEnv = new CELLogFilterEnvironment();
  private readonly _celExpr = signal<string>('');
  private readonly _onChanged = new Subject<void>();

  /**
   * Emits whenever the filter's internal configurations or state changes.
   */
  public readonly onChanged = this._onChanged.asObservable();

  /**
   * The currently active CEL log filter expression as a read-only signal.
   */
  public readonly celExpr = this._celExpr.asReadonly();

  /**
   * Validates the given CEL log expression against the registered environment schemas.
   */
  public validate(celExpr: string): CelFilterUpdateResult {
    const tempEnv = new CELLogFilterEnvironment();
    return tempEnv.compile(celExpr);
  }

  /**
   * Updates the evaluator with a new CEL expression, validating its syntax beforehand.
   */
  public updateFilter(celExpr: string): CelFilterUpdateResult {
    const compileRes = this.celEnv.compile(celExpr);
    if (!compileRes.success) {
      this._celExpr.set('');
      this._onChanged.next();
      return compileRes;
    }
    this._celExpr.set(celExpr);
    this._onChanged.next();
    return { success: true };
  }

  /**
   * Evaluates each log in the context against the CEL expression and retains only matching log IDs.
   */
  public async process(
    context: LogTimelineFilterContext,
    timelineStore: TimelineStore,
    signal?: AbortSignal,
    onProgress?: (current: number, total: number) => void,
  ): Promise<LogTimelineFilterContext> {
    if (this.celExpr() === '') {
      return context;
    }
    if (signal?.aborted) {
      throw new CancellationError();
    }
    const passedLogIds = await this.searchWorkerManager.searchLogs(
      this.celExpr(),
      Array.from(context.timelineIds),
      onProgress,
    );

    if (signal?.aborted) {
      throw new CancellationError();
    }

    return {
      timelineIds: context.timelineIds,
      logIds: passedLogIds,
    };
  }
}
