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

import { Injectable, signal } from '@angular/core';
import { Subject } from 'rxjs';
import {
  LogTimelineFilter,
  LogTimelineFilterContext,
} from 'src/app/store/domain/filter/types';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { CelFilterUpdateResult } from 'src/app/store/domain/filter/cel-types';
import {
  CELTimelineFilterEnvironment,
  CELLogFilterEnvironment,
} from 'src/app/store/domain/filter/cel-env';

/**
 * Filters timelines based on the configured CEL expression.
 */
@Injectable({ providedIn: 'root' })
export class CelTimelineFilter implements LogTimelineFilter {
  readonly displayName = 'Timeline CEL filter';
  readonly priority = 10;
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
    const passedTimelineIds = new Set<number>();
    let count = 0;
    const total = context.timelineIds.size;
    for (const id of context.timelineIds) {
      if (signal?.aborted) {
        throw new DOMException('Aborted', 'AbortError');
      }
      const t = timelineStore.getTimeline(id);
      if (this.celEnv.evaluate(t)) {
        passedTimelineIds.add(id);
      }
      count++;
      if (count % 500 === 0 || count === total) {
        onProgress?.(count, total);
        await new Promise((resolve) => setTimeout(resolve, 0));
      }
    }
    return {
      timelineIds: passedTimelineIds,
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
    const passedLogIds = new Set<number>();
    let count = 0;
    let total = 0;
    for (const tId of context.timelineIds) {
      const timeline = timelineStore.getTimeline(tId);
      total += timeline.events.length + timeline.revisions.length;
    }

    for (const tId of context.timelineIds) {
      if (signal?.aborted) {
        throw new DOMException('Aborted', 'AbortError');
      }
      const timeline = timelineStore.getTimeline(tId);
      for (const e of timeline.events) {
        if (signal?.aborted) {
          throw new DOMException('Aborted', 'AbortError');
        }
        const l = timelineStore.logStore.getLog(e.log.id);
        if (!passedLogIds.has(e.log.id) && this.celEnv.evaluate(l)) {
          passedLogIds.add(e.log.id);
        }
        count++;
        if (count % 1000 === 0 || count === total) {
          onProgress?.(count, total);
          await new Promise((resolve) => setTimeout(resolve, 0));
        }
      }
      for (const r of timeline.revisions) {
        if (signal?.aborted) {
          throw new DOMException('Aborted', 'AbortError');
        }
        const l = timelineStore.logStore.getLog(r.log.id);
        if (!passedLogIds.has(r.log.id) && this.celEnv.evaluate(l)) {
          passedLogIds.add(r.log.id);
        }
        count++;
        if (count % 1000 === 0 || count === total) {
          onProgress?.(count, total);
          await new Promise((resolve) => setTimeout(resolve, 0));
        }
      }
    }
    return {
      timelineIds: context.timelineIds,
      logIds: passedLogIds,
    };
  }
}
