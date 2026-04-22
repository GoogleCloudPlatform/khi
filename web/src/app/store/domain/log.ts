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

import { LogType, Severity } from 'src/app/store/domain/style';
import { LogStore } from 'src/app/store/domain/log-store';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { BigIntTimeUtil } from 'src/app/utils/bigint-time-util';

/**
 * Lazy adapter for a Log entry.
 * Evaluates fields on demand and ensures deep immutability.
 */
export class Log {
  private _body?: ReadonlyDomainElement<Record<string, unknown>> | null;

  constructor(
    public readonly id: number,
    private readonly store: LogStore,
  ) {}

  /**
   * Gets the chronological index of this log in the store.
   */
  get logIndex(): number {
    return this.store.getIndex(this.id);
  }

  /**
   * Gets the timestamp of the log entry.
   */
  get timestamp(): bigint {
    return this.store._getTimestamp(this.id);
  }

  /**
   * Gets the timestamp (in milliseconds) of the log entry.
   * @deprecated Use {@link timestamp} instead, which returns the timestamp in nanoseconds.
   */
  get legacyTimestampMs(): number {
    return BigIntTimeUtil.NsToNumberMs(this.timestamp);
  }

  /**
   * Gets the human-readable summary of the log.
   */
  get summary(): string {
    return this.store._getSummary(this.id);
  }

  /**
   * Gets the associated log source category.
   */
  get logType(): ReadonlyDomainElement<LogType> {
    return this.store._getLogType(this.id);
  }

  /**
   * Gets the severity priority of the log.
   */
  get severity(): ReadonlyDomainElement<Severity> {
    return this.store._getSeverity(this.id);
  }

  /**
   * Gets the structured log attributes decoded from Intern pool data stores.
   */
  get body(): ReadonlyDomainElement<Record<string, unknown>> | null {
    if (this._body === undefined) {
      this._body = this.store._decodeBody(this.id) as ReadonlyDomainElement<
        Record<string, unknown>
      > | null;
    }
    return this._body;
  }
}
