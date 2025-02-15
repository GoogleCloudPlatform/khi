/**
 * Copyright 2024 Google LLC
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

import { KHILogAnnotation } from '../common/schema/khi-file-types';
import {
  LogType,
  LogTypeMetadata,
  Severity,
  SeverityMetadata,
} from '../generated';
import { ResourceTimeline } from './timeline';
import { TextReference } from '../common/loader/interface';
import { ToTextReferenceFromKHIFileBinary } from '../common/loader/reference-type';

export class LogEntry {
  /**
   * Set of timelines relate to this log.
   */
  public relatedTimelines: Set<ResourceTimeline> = new Set();

  public logTypeLabel = LogTypeMetadata[this.logType].label;

  public logSeverityLabel = SeverityMetadata[this.severity].label;

  constructor(
    public readonly logIndex: number,
    public readonly insertId: string,
    public readonly logType: LogType,
    public readonly severity: Severity,
    public readonly time: number,
    public readonly summary: string,
    public readonly body: TextReference,
    public readonly annotations: KHILogAnnotation[],
  ) {}

  public static clone(entry: LogEntry): LogEntry {
    return new LogEntry(
      entry.logIndex,
      entry.insertId,
      entry.logType,
      entry.severity,
      entry.time,
      entry.summary,
      entry.body,
      entry.annotations,
    );
  }
}

/**
 * NullLog is just a placeholder for log reference when the resource status is not inferred from any logs.
 */
export const NullLog = new LogEntry(
  -1,
  '',
  LogType.LogTypeUnknown,
  Severity.SeverityUnknown,
  0,
  '',
  ToTextReferenceFromKHIFileBinary({ offset: 0, len: 0, buffer: 0 }),
  [],
);
