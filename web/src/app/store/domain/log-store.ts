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

import { Log } from 'src/app/store/domain/log';
import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { StyleStore } from 'src/app/store/domain/style-store';
import { InternedStructDecoder } from 'src/app/store/domain/struct-decoder';
import { InternedStructSchema } from 'src/app/generated/khifile/shared_pb';
import { LogType, Severity } from 'src/app/store/domain/style';
import { ReadonlyDomainElement, Undefinable } from 'src/app/store/domain/types';
import { fromBinary } from '@bufbuild/protobuf';

/**
 * Raw Log object interface from the assembler.
 *
 * This type is used because domain layer stores must not receive proto type
 * directly to decouple file version difference with domain stores.
 */
export interface LogDTO {
  readonly id: number;
  readonly ts: bigint;
  readonly logTypeId: number;
  readonly severityTypeId: number;
  readonly summaryStringId: number;
  readonly body?: Uint8Array;
}

/**
 * Store for managing and retrieving logs efficiently.
 */
export class LogStore {
  private ids = new Uint32Array(0);
  private timestamps = new BigUint64Array(0);
  private logTypeIds = new Uint32Array(0);
  private severityIds = new Uint32Array(0);
  private summaryStringIds = new Uint32Array(0);

  private bodies: Undefinable<Uint8Array>[] = [];
  private decodedBodyCache: WeakRef<
    ReadonlyDomainElement<Record<string, unknown>>
  >[] = [];
  private idToIndex: Undefinable<number>[] = [];
  private readonly decoder: InternedStructDecoder;

  constructor(
    private readonly internPool: InternPoolStore,
    private readonly styleStore: StyleStore,
  ) {
    this.decoder = new InternedStructDecoder(this.internPool);
  }

  /**
   * Initializes the store with the raw logs.
   * Assumes logs are already sorted by timestamp.
   * @param logs The iterable of raw logs.
   * @param count The total number of logs.
   */
  public initialize(logs: Iterable<LogDTO>, count: number): void {
    this.ids = new Uint32Array(count);
    this.timestamps = new BigUint64Array(count);
    this.logTypeIds = new Uint32Array(count);
    this.severityIds = new Uint32Array(count);
    this.summaryStringIds = new Uint32Array(count);

    this.bodies = new Array<Undefinable<Uint8Array>>(count);
    this.decodedBodyCache = new Array<
      WeakRef<ReadonlyDomainElement<Record<string, unknown>>>
    >(count);
    this.idToIndex = [];

    let index = 0;
    let prevTs = 0n;
    for (const log of logs) {
      if (index > 0 && log.ts < prevTs) {
        throw new Error(
          `Logs are not sorted by timestamp at index ${index}: timestamp ${log.ts} < ${prevTs}`,
        );
      }
      prevTs = log.ts;

      this.ids[index] = log.id;
      this.timestamps[index] = log.ts;
      this.logTypeIds[index] = log.logTypeId;
      this.severityIds[index] = log.severityTypeId;
      this.summaryStringIds[index] = log.summaryStringId;
      this.bodies[index] = log.body;
      this.idToIndex[log.id] = index;
      index++;
    }
  }

  /**
   * Gets a log entry adapter by its ID.
   * @param id The ID of the log.
   * @returns The log entry adapter.
   */
  public getLog(id: number): ReadonlyDomainElement<Log> {
    const index = this.idToIndex[id];
    if (index === undefined) {
      throw new Error(`Log ID ${id} not found`);
    }

    return new Log(id, this);
  }

  /**
   * Gets the total number of logs.
   */
  public get count(): number {
    return this.ids.length;
  }

  /**
   * Returns an iterator for all logs in the store.
   */
  public *logs(): IterableIterator<ReadonlyDomainElement<Log>> {
    for (let i = 0; i < this.ids.length; i++) {
      yield new Log(this.ids[i], this);
    }
  }

  // --- Internal getters for Log adapter ---

  /**
   * Gets the timestamp of a log by its ID.
   * @note Intended solely for internal retrieval inside the {@link Log} domain adapter.
   */
  public _getTimestamp(id: number): bigint {
    return this.timestamps[this.getIndex(id)];
  }

  /**
   * Gets the summary value of a log by its ID.
   * @note Intended solely for internal retrieval inside the {@link Log} domain adapter.
   */
  public _getSummary(id: number): string {
    return this.internPool.getString(this.summaryStringIds[this.getIndex(id)]);
  }

  /**
   * Gets the log type metadata of a log by its ID.
   * @note Intended solely for internal retrieval inside the {@link Log} domain adapter.
   */
  public _getLogType(id: number): ReadonlyDomainElement<LogType> {
    return this.styleStore.getLogType(this.logTypeIds[this.getIndex(id)]);
  }

  /**
   * Gets the severity metadata of a log by its ID.
   * @note Intended solely for internal retrieval inside the {@link Log} domain adapter.
   */
  public _getSeverity(id: number): ReadonlyDomainElement<Severity> {
    return this.styleStore.getSeverity(this.severityIds[this.getIndex(id)]);
  }

  /**
   * Decodes the nested properties of a log by its ID.
   * @note Intended solely for internal retrieval inside the {@link Log} domain adapter.
   */
  public _decodeBody(
    id: number,
  ): ReadonlyDomainElement<Record<string, unknown>> | null {
    const index = this.getIndex(id);
    const cached = this.decodedBodyCache[index]?.deref();
    if (cached) {
      return cached;
    }

    const body = this.bodies[index];
    if (!body) return null;
    const struct = fromBinary(InternedStructSchema, body);
    const decoded = this.decoder.decode(struct);
    this.decodedBodyCache[index] = new WeakRef(decoded);
    return decoded;
  }

  /**
   * Gets the index of a log in the store by its ID.
   * @param id The ID of the log.
   * @returns The index of the log.
   */
  public getIndex(id: number): number {
    const index = this.idToIndex[id];
    if (index === undefined) {
      throw new Error(`Log ID ${id} not found`);
    }
    return index;
  }
}
