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

import { Injectable, signal, computed } from '@angular/core';
import { Environment } from '@marcbachmann/cel-js';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import {
  LogTimelineFilter,
  LogTimelineFilterContext,
} from 'src/app/store/domain/filter/types';
import { TimelineStore } from 'src/app/store/domain/timeline-store';

const SEVERITY_LEVELS = {
  UNKNOWN: 0n,
  INFO: 1n,
  WARNING: 2n,
  ERROR: 3n,
  FATAL: 4n,
};

function mapSeverityToNumber(label: string | undefined): bigint {
  if (!label) return SEVERITY_LEVELS.UNKNOWN;
  const upper = label.toUpperCase();
  if (upper.includes('FATAL')) return SEVERITY_LEVELS.FATAL;
  if (upper.includes('ERROR')) return SEVERITY_LEVELS.ERROR;
  if (upper.includes('WARN')) return SEVERITY_LEVELS.WARNING;
  if (upper.includes('INFO')) return SEVERITY_LEVELS.INFO;
  return SEVERITY_LEVELS.UNKNOWN;
}

/**
 * Represents a simplified representation of a Log optimized for CEL evaluation.
 */
export interface CELLog {
  readonly type: string;
  readonly severity: bigint;
  readonly summary: string;
}

/**
 * Represents a simplified representation of an Event optimized for CEL evaluation.
 */
export interface CELEvent {
  readonly log: CELLog;
}

/**
 * Represents a simplified representation of a Revision optimized for CEL evaluation.
 */
export interface CELRevision {
  readonly log: CELLog;
  readonly changedTime: bigint;
  readonly principal: string;
  readonly verb: string;
  readonly state: string;
}

/**
 * Represents a simplified, plain object representation of a Timeline optimized for CEL evaluation.
 */
export interface CELTimeline {
  readonly name: string;
  readonly type: string;
  readonly events: readonly CELEvent[];
  readonly revisions: readonly CELRevision[];
}

/**
 * Function type representing the evaluation of a timeline context against a CEL expression.
 */
export type TimelineEvaluator = (context: {
  readonly t: CELTimeline;
}) => boolean;

/**
 * Function type representing the evaluation of a log context against a CEL expression.
 */
export type LogEvaluator = (context: { readonly l: CELLog }) => boolean;

/**
 * Represents the result of updating a CEL filter expression.
 */
export interface CelFilterUpdateResult {
  readonly success: boolean;
  readonly error?: Error;
}

/**
 * Filters timelines based on the configured CEL expression.
 */
@Injectable({ providedIn: 'root' })
export class CelTimelineFilter implements LogTimelineFilter {
  private readonly environment: Environment;
  private readonly celExpr = signal<string>('');
  private readonly evaluator = computed<TimelineEvaluator | undefined>(() => {
    const val = this.celExpr();
    if (!val || val.trim() === '') return undefined;
    const checkRes = this.environment.check(val);
    if (!checkRes.valid) return undefined;
    const parsed = this.environment.parse(val);
    return (ctx) => Boolean(parsed({ ...ctx, ...SEVERITY_LEVELS }));
  });

  /**
   * Initializes a new instance of the CEL timeline filter by setting up the CEL evaluation environment.
   */
  constructor() {
    this.environment = new Environment({
      unlistedVariablesAreDyn: false,
    })
      .registerType({
        name: 'Log',
        schema: {
          type: 'string',
          severity: 'int',
          summary: 'string',
        },
      })
      .registerType({
        name: 'Event',
        schema: {
          log: 'Log',
        },
      })
      .registerType({
        name: 'Revision',
        schema: {
          log: 'Log',
          changedTime: 'int',
          principal: 'string',
          verb: 'string',
          state: 'string',
        },
      })
      .registerType({
        name: 'Timeline',
        schema: {
          name: 'string',
          type: 'string',
          events: 'list',
          revisions: 'list',
        },
      })
      .registerVariable('t', 'Timeline')
      .registerVariable('UNKNOWN', 'int')
      .registerVariable('INFO', 'int')
      .registerVariable('WARNING', 'int')
      .registerVariable('ERROR', 'int')
      .registerVariable('FATAL', 'int');
  }

  /**
   * Validates the given CEL timeline expression against the registered environment schemas.
   */
  public validate(celExpr: string): CelFilterUpdateResult {
    if (celExpr && celExpr.trim() !== '') {
      const checkRes = this.environment.check(celExpr);
      if (!checkRes.valid) {
        return {
          success: false,
          error:
            checkRes.error ??
            new Error('Timeline expression validation failed'),
        };
      }
    }
    return { success: true };
  }

  /**
   * Updates the evaluator with a new CEL expression, validating its syntax beforehand.
   */
  public updateFilter(celExpr: string): CelFilterUpdateResult {
    const checkRes = this.validate(celExpr);
    if (!checkRes.success) {
      this.celExpr.set('');
      return checkRes;
    }
    this.celExpr.set(celExpr);
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
    const evalFn = this.evaluator();
    if (!evalFn) {
      return context;
    }
    const passedTimelineIds = new Set<number>();
    for (const id of context.timelineIds) {
      const t = timelineStore.getTimeline(id);
      if (evalFn({ t: this.toCelTimeline(t) })) {
        passedTimelineIds.add(id);
      }
    }
    return {
      timelineIds: passedTimelineIds,
      logIds: context.logIds,
    };
  }

  private toCelTimeline(
    timeline: ReadonlyDomainElement<Timeline>,
  ): CELTimeline {
    return {
      name: timeline.name,
      type: timeline.type.label,
      events: timeline.events.map((e) => ({
        log: this.toCelLog(e.log),
      })),
      revisions: timeline.revisions.map((r) => ({
        log: this.toCelLog(r.log),
        changedTime: r.changedTime,
        principal: r.principal,
        verb: r.verb.label,
        state: r.state.label,
      })),
    };
  }

  private toCelLog(log: {
    readonly summary?: string;
    readonly logType?: { readonly label: string };
    readonly severity?: { readonly label: string };
  }): CELLog {
    return {
      type: log.logType?.label ?? '',
      severity: mapSeverityToNumber(log.severity?.label),
      summary: log.summary ?? '',
    };
  }
}

/**
 * Filters logs based on the configured CEL expression.
 */
@Injectable({ providedIn: 'root' })
export class CelLogFilter implements LogTimelineFilter {
  private readonly environment: Environment;
  private readonly celExpr = signal<string>('');
  private readonly evaluator = computed<LogEvaluator | undefined>(() => {
    const val = this.celExpr();
    if (!val || val.trim() === '') return undefined;
    const checkRes = this.environment.check(val);
    if (!checkRes.valid) return undefined;
    const parsed = this.environment.parse(val);
    return (ctx) => Boolean(parsed({ ...ctx, ...SEVERITY_LEVELS }));
  });

  /**
   * Initializes a new instance of the CEL log filter by setting up the CEL evaluation environment.
   */
  constructor() {
    this.environment = new Environment({
      unlistedVariablesAreDyn: false,
    })
      .registerType({
        name: 'Log',
        schema: {
          type: 'string',
          severity: 'int',
          summary: 'string',
        },
      })
      .registerVariable('l', 'Log')
      .registerVariable('UNKNOWN', 'int')
      .registerVariable('INFO', 'int')
      .registerVariable('WARNING', 'int')
      .registerVariable('ERROR', 'int')
      .registerVariable('FATAL', 'int');
  }

  /**
   * Validates the given CEL log expression against the registered environment schemas.
   */
  public validate(celExpr: string): CelFilterUpdateResult {
    if (celExpr && celExpr.trim() !== '') {
      const checkRes = this.environment.check(celExpr);
      if (!checkRes.valid) {
        return {
          success: false,
          error:
            checkRes.error ?? new Error('Log expression validation failed'),
        };
      }
    }
    return { success: true };
  }

  /**
   * Updates the evaluator with a new CEL expression, validating its syntax beforehand.
   */
  public updateFilter(celExpr: string): CelFilterUpdateResult {
    const checkRes = this.validate(celExpr);
    if (!checkRes.success) {
      this.celExpr.set('');
      return checkRes;
    }
    this.celExpr.set(celExpr);
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
    const evalFn = this.evaluator();
    if (!evalFn) {
      return context;
    }
    const passedLogIds = new Set<number>();
    for (const tId of context.timelineIds) {
      const timeline = timelineStore.getTimeline(tId);
      for (const e of timeline.events) {
        const l = timelineStore.logStore.getLog(e.log.id);
        if (!passedLogIds.has(e.log.id) && evalFn({ l: this.toCelLog(l) })) {
          passedLogIds.add(e.log.id);
        }
      }
      for (const r of timeline.revisions) {
        const l = timelineStore.logStore.getLog(r.log.id);
        if (!passedLogIds.has(r.log.id) && evalFn({ l: this.toCelLog(l) })) {
          passedLogIds.add(r.log.id);
        }
      }
    }
    return {
      timelineIds: context.timelineIds,
      logIds: passedLogIds,
    };
  }

  private toCelLog(log: {
    readonly summary?: string;
    readonly logType?: { readonly label: string };
    readonly severity?: { readonly label: string };
  }): CELLog {
    return {
      type: log.logType?.label ?? '',
      severity: mapSeverityToNumber(log.severity?.label),
      summary: log.summary ?? '',
    };
  }
}
