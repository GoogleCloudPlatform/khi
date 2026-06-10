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

import { Environment } from '@marcbachmann/cel-js';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { Timeline } from 'src/app/store/domain/timeline';
import { Log } from 'src/app/store/domain/log';
import {
  CELLog,
  CELTimeline,
  toCelLog,
  toCelTimeline,
  CelFilterUpdateResult,
} from 'src/app/store/domain/filter/cel-types';
import {
  matchLogField,
  matchTimelinePath,
  matchTimelineRevisionBodyField,
} from 'src/app/store/domain/filter/cel-functions';

const SEVERITY_LEVELS = {
  UNKNOWN: 0n,
  INFO: 1n,
  WARNING: 2n,
  ERROR: 3n,
  FATAL: 4n,
};

/**
 * Manages the CEL Environment, compiles expressions, and evaluates raw Timelines.
 */
export class CELTimelineFilterEnvironment {
  private readonly environment: Environment;
  private evaluator?: (ctx: CELTimeline) => boolean;
  private currentTimeline?: CELTimeline;

  /**
   * Initializes the CEL Environment with unlistedVariablesAreDyn enabled to support top-level property resolution dynamically.
   */
  constructor() {
    this.environment = new Environment({
      unlistedVariablesAreDyn: false,
    })
      .registerType({
        name: 'Log',
        schema: {
          logType: 'string',
          severity: 'int',
          summary: 'string',
          body: 'map',
          bodyYAML: 'string',
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
          body: 'map',
        },
      })
      .registerType({
        name: 'Timeline',
        schema: {
          name: 'string',
          timelineType: 'string',
          path: 'map',
          events: 'list<Event>',
          revisions: 'list<Revision>',
        },
      })
      .registerVariable('t', 'Timeline')
      .registerVariable('name', 'string')
      .registerVariable('timelineType', 'string')
      .registerVariable('path', 'map')
      .registerVariable('events', 'list<Event>')
      .registerVariable('revisions', 'list<Revision>')
      .registerVariable('UNKNOWN', 'int')
      .registerVariable('INFO', 'int')
      .registerVariable('WARNING', 'int')
      .registerVariable('ERROR', 'int')
      .registerVariable('FATAL', 'int')
      // Global function registrations bound to the instance state
      .registerFunction('match(string, string): bool', (k, v) =>
        matchTimelinePath(this.currentTimeline, k, v),
      )
      .registerFunction('match(string, list): bool', (k, v) =>
        matchTimelinePath(this.currentTimeline, k, v),
      )
      .registerFunction('match(string): bool', (v) =>
        matchTimelinePath(this.currentTimeline, '*', v),
      )
      .registerFunction('match(list): bool', (v) =>
        matchTimelinePath(this.currentTimeline, '*', v),
      )
      .registerFunction('M(string, string): bool', (k, v) =>
        matchTimelinePath(this.currentTimeline, k, v),
      )
      .registerFunction('M(string, list): bool', (k, v) =>
        matchTimelinePath(this.currentTimeline, k, v),
      )
      .registerFunction('M(string): bool', (v) =>
        matchTimelinePath(this.currentTimeline, '*', v),
      )
      .registerFunction('M(list): bool', (v) =>
        matchTimelinePath(this.currentTimeline, '*', v),
      )
      .registerFunction('revision_body(string, string): bool', (k, v) =>
        matchTimelineRevisionBodyField(this.currentTimeline, k, v),
      )
      .registerFunction('revision_body(string, list): bool', (k, v) =>
        matchTimelineRevisionBodyField(this.currentTimeline, k, v),
      )
      .registerFunction('RB(string, string): bool', (k, v) =>
        matchTimelineRevisionBodyField(this.currentTimeline, k, v),
      )
      .registerFunction('RB(string, list): bool', (k, v) =>
        matchTimelineRevisionBodyField(this.currentTimeline, k, v),
      )
      .registerFunction('revision_body(string): bool', (v) =>
        matchTimelineRevisionBodyField(this.currentTimeline, '*', v),
      )
      .registerFunction('revision_body(list): bool', (v) =>
        matchTimelineRevisionBodyField(this.currentTimeline, '*', v),
      )
      .registerFunction('RB(string): bool', (v) =>
        matchTimelineRevisionBodyField(this.currentTimeline, '*', v),
      )
      .registerFunction('RB(list): bool', (v) =>
        matchTimelineRevisionBodyField(this.currentTimeline, '*', v),
      );
  }

  /**
   * Compiles the CEL expression and keeps it as internal state.
   *
   * @param celExpr - The CEL expression string to compile
   * @returns success/error result
   */
  public compile(celExpr: string): CelFilterUpdateResult {
    if (!celExpr || celExpr.trim() === '') {
      this.evaluator = undefined;
      return { success: true };
    }

    const checkRes = this.environment.check(celExpr);
    if (!checkRes.valid) {
      this.evaluator = undefined;
      return {
        success: false,
        error:
          checkRes.error ?? new Error('Timeline expression validation failed'),
      };
    }

    try {
      const parsed = this.environment.parse(celExpr);
      this.evaluator = (ctx) =>
        Boolean(parsed({ t: ctx, ...ctx, ...SEVERITY_LEVELS }));
      return { success: true };
    } catch (err) {
      this.evaluator = undefined;
      return { success: false, error: err as Error };
    }
  }

  /**
   * Evaluates a raw Timeline element against the compiled CEL expression.
   *
   * @param timeline - Raw Timeline element
   * @returns True if the timeline passes the filter
   */
  public evaluate(timeline: ReadonlyDomainElement<Timeline>): boolean {
    if (!this.evaluator) {
      return true; // Pass-through if no active expression
    }

    const celTimeline = toCelTimeline(timeline);
    this.currentTimeline = celTimeline;
    try {
      return this.evaluator(celTimeline);
    } finally {
      this.currentTimeline = undefined;
    }
  }
}

/**
 * Manages the CEL Environment, compiles expressions, and evaluates raw Logs.
 */
export class CELLogFilterEnvironment {
  private readonly environment: Environment;
  private evaluator?: (ctx: CELLog) => boolean;
  private currentLog?: CELLog;

  /**
   * Initializes the CEL Environment with unlistedVariablesAreDyn enabled to support top-level property resolution dynamically.
   */
  constructor() {
    this.environment = new Environment({
      unlistedVariablesAreDyn: false,
    })
      .registerType({
        name: 'Log',
        schema: {
          logType: 'string',
          severity: 'int',
          summary: 'string',
          body: 'map',
          bodyYAML: 'string',
        },
      })
      .registerVariable('l', 'Log')
      .registerVariable('logType', 'string')
      .registerVariable('severity', 'int')
      .registerVariable('summary', 'string')
      .registerVariable('body', 'map')
      .registerVariable('bodyYAML', 'string')
      .registerVariable('UNKNOWN', 'int')
      .registerVariable('INFO', 'int')
      .registerVariable('WARNING', 'int')
      .registerVariable('ERROR', 'int')
      .registerVariable('FATAL', 'int')
      .registerFunction('body(string, string): bool', (k, v) =>
        matchLogField(this.currentLog, k, v),
      )
      .registerFunction('body(string, list): bool', (k, v) =>
        matchLogField(this.currentLog, k, v),
      )
      .registerFunction('body(string): bool', (v) =>
        matchLogField(this.currentLog, '*', v),
      )
      .registerFunction('body(list): bool', (v) =>
        matchLogField(this.currentLog, '*', v),
      )
      .registerFunction('B(string, string): bool', (k, v) =>
        matchLogField(this.currentLog, k, v),
      )
      .registerFunction('B(string, list): bool', (k, v) =>
        matchLogField(this.currentLog, k, v),
      )
      .registerFunction('B(string): bool', (v) =>
        matchLogField(this.currentLog, '*', v),
      )
      .registerFunction('B(list): bool', (v) =>
        matchLogField(this.currentLog, '*', v),
      );
  }

  /**
   * Compiles the CEL expression and keeps it as internal state.
   *
   * @param celExpr - The CEL expression string to compile
   * @returns success/error result
   */
  public compile(celExpr: string): CelFilterUpdateResult {
    if (!celExpr || celExpr.trim() === '') {
      this.evaluator = undefined;
      return { success: true };
    }

    const checkRes = this.environment.check(celExpr);
    if (!checkRes.valid) {
      this.evaluator = undefined;
      return {
        success: false,
        error: checkRes.error ?? new Error('Log expression validation failed'),
      };
    }

    try {
      const parsed = this.environment.parse(celExpr);
      this.evaluator = (ctx) =>
        Boolean(parsed({ l: ctx, ...ctx, ...SEVERITY_LEVELS }));
      return { success: true };
    } catch (err) {
      this.evaluator = undefined;
      return { success: false, error: err as Error };
    }
  }

  /**
   * Evaluates a raw Log element against the compiled CEL expression.
   *
   * @param log - Raw Log element
   * @returns True if the log passes the filter
   */
  public evaluate(log: ReadonlyDomainElement<Log>): boolean {
    if (!this.evaluator) {
      return true; // Pass-through if no active expression
    }

    const celLog = toCelLog(log);
    this.currentLog = celLog;
    try {
      return this.evaluator(celLog);
    } finally {
      this.currentLog = undefined;
    }
  }
}
