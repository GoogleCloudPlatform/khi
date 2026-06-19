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

import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { Timeline, Event, Revision } from 'src/app/store/domain/timeline';
import { Log } from 'src/app/store/domain/log';
import { mapSeverityToNumber } from 'src/app/store/domain/filter/cel-functions';

/**
 * Represents a simplified representation of a Log optimized for CEL evaluation.
 */
export interface CELLog {
  readonly logType: string;
  readonly severity: bigint;
  readonly summary: string;
  readonly body: Record<string, unknown>;
  readonly bodyYAML: string;
  readonly UNKNOWN: bigint;
  readonly INFO: bigint;
  readonly WARNING: bigint;
  readonly ERROR: bigint;
  readonly FATAL: bigint;
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
  readonly body: Record<string, unknown>;
  readonly bodyYAML: string;
}

/**
 * Represents a simplified representation of a Timeline optimized for CEL evaluation.
 */
export interface CELTimeline {
  readonly name: string;
  readonly timelineType: string;
  readonly path: Record<string, string>;
  readonly events: readonly CELEvent[];
  readonly revisions: readonly CELRevision[];
  readonly UNKNOWN: bigint;
  readonly INFO: bigint;
  readonly WARNING: bigint;
  readonly ERROR: bigint;
  readonly FATAL: bigint;
}

/**
 * Function type representing the evaluation of a timeline context against a CEL expression.
 */
export type TimelineEvaluator = (context: CELTimeline) => boolean;

/**
 * Function type representing the evaluation of a log context against a CEL expression.
 */
export type LogEvaluator = (context: CELLog) => boolean;

/**
 * Represents the result of compiling or updating a CEL expression/filter.
 */
export interface CelFilterUpdateResult {
  readonly success: boolean;
  readonly error?: Error;
}

/**
 * Converts a Log domain element to a simplified CELLog representation.
 */
export function toCelLog(log: ReadonlyDomainElement<Log>): CELLog {
  return {
    get logType(): string {
      return log.logType?.label ?? '';
    },
    get severity(): bigint {
      return mapSeverityToNumber(log.severity?.label);
    },
    get summary(): string {
      return log.summary ?? '';
    },
    get body(): Record<string, unknown> {
      return (log.body ?? {}) as Record<string, unknown>;
    },
    get bodyYAML(): string {
      return log.bodyYAML ?? '';
    },
    UNKNOWN: 0n,
    INFO: 1n,
    WARNING: 2n,
    ERROR: 3n,
    FATAL: 4n,
  };
}

/**
 * Converts an Event domain element to a simplified CELEvent representation.
 */
export function toCelEvent(event: ReadonlyDomainElement<Event>): CELEvent {
  return {
    get log(): CELLog {
      return toCelLog(event.log);
    },
  };
}

/**
 * Converts a Revision domain element to a simplified CELRevision representation.
 */
export function toCelRevision(
  revision: ReadonlyDomainElement<Revision>,
): CELRevision {
  return {
    get log(): CELLog {
      return toCelLog(revision.log);
    },
    get changedTime(): bigint {
      return revision.changedTime;
    },
    get principal(): string {
      return revision.principal;
    },
    get verb(): string {
      return revision.verb.label;
    },
    get state(): string {
      return revision.state.label;
    },
    get body(): Record<string, unknown> {
      return (revision.body ?? {}) as Record<string, unknown>;
    },
    get bodyYAML(): string {
      return revision.bodyYAML ?? '';
    },
  };
}

/**
 * Converts a Timeline domain element to a simplified CELTimeline representation.
 */
export function toCelTimeline(
  timeline: ReadonlyDomainElement<Timeline>,
): CELTimeline {
  return {
    get name(): string {
      return timeline.name;
    },
    get timelineType(): string {
      return timeline.type.label.toLowerCase();
    },
    get path(): Record<string, string> {
      const pathMap: Record<string, string> = {};
      if (timeline.path) {
        for (const node of timeline.path) {
          pathMap[node.type.label.toLowerCase()] = node.label;
        }
      }
      return pathMap;
    },
    get events(): readonly CELEvent[] {
      return timeline.events.map((e) => toCelEvent(e));
    },
    get revisions(): readonly CELRevision[] {
      return timeline.revisions.map((r) => toCelRevision(r));
    },
    UNKNOWN: 0n,
    INFO: 1n,
    WARNING: 2n,
    ERROR: 3n,
    FATAL: 4n,
  };
}
