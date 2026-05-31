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
  private readonly celEnv = new CELTimelineFilterEnvironment();
  private readonly _celExpr = signal<string>('');
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
    return { success: true };
  }

  /**
   * Returns the priority value determining the execution order of this filter.
   */
  priority(): number {
    return 10;
  }

  /**
   * Evaluates each timeline in the context against the CEL expression and retains only matching timeline IDs.
   */
  process(
    context: LogTimelineFilterContext,
    timelineStore: TimelineStore,
  ): LogTimelineFilterContext {
    if (this.celExpr() === '') {
      return context;
    }
    const passedTimelineIds = new Set<number>();
    for (const id of context.timelineIds) {
      const t = timelineStore.getTimeline(id);
      if (this.celEnv.evaluate(t)) {
        passedTimelineIds.add(id);
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
  private readonly celEnv = new CELLogFilterEnvironment();
  private readonly _celExpr = signal<string>('');
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
      return compileRes;
    }
    this._celExpr.set(celExpr);
    return { success: true };
  }

  /**
   * Returns the priority value determining the execution order of this filter.
   */
  priority(): number {
    return 30;
  }

  /**
   * Evaluates each log in the context against the CEL expression and retains only matching log IDs.
   */
  process(
    context: LogTimelineFilterContext,
    timelineStore: TimelineStore,
  ): LogTimelineFilterContext {
    if (this.celExpr() === '') {
      return context;
    }
    const passedLogIds = new Set<number>();
    for (const tId of context.timelineIds) {
      const timeline = timelineStore.getTimeline(tId);
      for (const e of timeline.events) {
        const l = timelineStore.logStore.getLog(e.log.id);
        if (!passedLogIds.has(e.log.id) && this.celEnv.evaluate(l)) {
          passedLogIds.add(e.log.id);
        }
      }
      for (const r of timeline.revisions) {
        const l = timelineStore.logStore.getLog(r.log.id);
        if (!passedLogIds.has(r.log.id) && this.celEnv.evaluate(l)) {
          passedLogIds.add(r.log.id);
        }
      }
    }
    return {
      timelineIds: context.timelineIds,
      logIds: passedLogIds,
    };
  }
}
