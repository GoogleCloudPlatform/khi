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
import { LogType, Severity, StyleProvider } from 'src/app/store/domain/style';
import {
  BufferLayoutBuilder,
  nextCapacity,
} from 'src/app/store/domain/buffer-util';
import {
  ReadonlyDomainElement,
  allocateBuffer,
} from 'src/app/store/domain/types';

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
  readonly bodyStructId?: number;
}

interface LogStoreLayout {
  readonly idsOffset: number;
  readonly logTypeIdsOffset: number;
  readonly severityIdsOffset: number;
  readonly summaryStringIdsOffset: number;
  readonly bodyStructIdsOffset: number;
  readonly timestampsOffset: number;
  readonly totalBytes: number;
}

/**
 * Store for managing and retrieving logs efficiently.
 */
export class LogStore {
  private ids!: Uint32Array;
  private timestamps!: BigUint64Array;
  private logTypeIds!: Uint32Array;
  private severityIds!: Uint32Array;
  private summaryStringIds!: Uint32Array;
  private bodyStructIds!: Uint32Array;

  private idToIndex: (number | undefined)[] = [];

  private logCount = 0;
  private previousTimestamp = 0n;

  private constructor(
    private readonly internPool: InternPoolStore,
    private readonly styleStore: StyleProvider,
  ) {}

  /**
   * Creates an empty LogStore instance.
   *
   * @param internPool The intern pool store.
   * @param styleStore The style provider.
   * @param initialCapacity The initial capacity for log metadata.
   * @returns A new empty LogStore instance.
   */
  public static create(
    internPool: InternPoolStore,
    styleStore: StyleProvider,
    initialCapacity = 1024,
  ): LogStore {
    const store = new LogStore(internPool, styleStore);
    if (initialCapacity > 0) {
      store.ensureCapacity(initialCapacity);
    }
    return store;
  }

  /**
   * Appends multiple logs to the store.
   * Logs must be in non-decreasing timestamp order.
   *
   * @param logs An iterable of LogDTO objects.
   */
  public addLogs(logs: Iterable<LogDTO>): void {
    if (Array.isArray(logs)) {
      this.ensureCapacity(this.logCount + logs.length);
    }
    for (const log of logs) {
      this.addLog(log);
    }
  }

  private calculateOffsets(capacity: number): LogStoreLayout {
    const builder = new BufferLayoutBuilder();
    return {
      idsOffset: builder.addField(capacity, 4),
      logTypeIdsOffset: builder.addField(capacity, 4),
      severityIdsOffset: builder.addField(capacity, 4),
      summaryStringIdsOffset: builder.addField(capacity, 4),
      bodyStructIdsOffset: builder.addField(capacity, 4),
      timestampsOffset: builder.addField(capacity, 8, 8),
      totalBytes: builder.totalBytes,
    };
  }

  private reallocate(newCap: number): void {
    const layout = this.calculateOffsets(newCap);
    const newMetadataBuffer = allocateBuffer(layout.totalBytes);

    const newIds = new Uint32Array(newMetadataBuffer, layout.idsOffset, newCap);
    const newLogTypeIds = new Uint32Array(
      newMetadataBuffer,
      layout.logTypeIdsOffset,
      newCap,
    );
    const newSeverityIds = new Uint32Array(
      newMetadataBuffer,
      layout.severityIdsOffset,
      newCap,
    );
    const newSummaryStringIds = new Uint32Array(
      newMetadataBuffer,
      layout.summaryStringIdsOffset,
      newCap,
    );
    const newBodyStructIds = new Uint32Array(
      newMetadataBuffer,
      layout.bodyStructIdsOffset,
      newCap,
    );
    const newTimestamps = new BigUint64Array(
      newMetadataBuffer,
      layout.timestampsOffset,
      newCap,
    );

    if (this.logCount > 0 && this.ids) {
      newIds.set(this.ids.subarray(0, this.logCount));
      newLogTypeIds.set(this.logTypeIds.subarray(0, this.logCount));
      newSeverityIds.set(this.severityIds.subarray(0, this.logCount));
      newSummaryStringIds.set(this.summaryStringIds.subarray(0, this.logCount));
      newBodyStructIds.set(this.bodyStructIds.subarray(0, this.logCount));
      newTimestamps.set(this.timestamps.subarray(0, this.logCount));
    }

    this.ids = newIds;
    this.logTypeIds = newLogTypeIds;
    this.severityIds = newSeverityIds;
    this.summaryStringIds = newSummaryStringIds;
    this.bodyStructIds = newBodyStructIds;
    this.timestamps = newTimestamps;
  }

  /**
   * Ensures the metadata buffer has at least minCapacity slots.
   *
   * @param minCapacity The required minimum capacity.
   */
  private ensureCapacity(minCapacity: number): void {
    const currentCap = this.ids ? this.ids.length : 0;
    if (minCapacity > currentCap) {
      this.reallocate(nextCapacity(currentCap, minCapacity));
    }
  }

  /**
   * Shrinks metadataBuffer to the minimal required size based on the current log count.
   */
  public shrinkToFit(): void {
    const currentCap = this.ids ? this.ids.length : 0;
    if (this.logCount < currentCap) {
      this.reallocate(this.logCount);
    }
  }

  /**
   * Appends a single log to the store.
   * Logs must be added in non-decreasing timestamp order.
   *
   * @param log The raw LogDTO to add.
   */
  public addLog(log: LogDTO): void {
    if (this.logCount > 0 && log.ts < this.previousTimestamp) {
      throw new Error(
        `Logs are not sorted by timestamp at index ${this.logCount}: timestamp ${log.ts} < ${this.previousTimestamp}`,
      );
    }
    this.previousTimestamp = log.ts;

    this.ensureCapacity(this.logCount + 1);

    const index = this.logCount;
    this.ids[index] = log.id;
    this.timestamps[index] = log.ts;
    this.logTypeIds[index] = log.logTypeId;
    this.severityIds[index] = log.severityTypeId;
    this.summaryStringIds[index] = log.summaryStringId;
    this.bodyStructIds[index] = log.bodyStructId ?? 0;

    this.idToIndex[log.id] = index;
    this.logCount++;
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
    return this.logCount;
  }

  /**
   * Returns an iterator for all logs in the store.
   */
  public *logs(): IterableIterator<ReadonlyDomainElement<Log>> {
    for (let i = 0; i < this.logCount; i++) {
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
   * Gets the interned struct ID of a log body, or 0 if not stored as a struct.
   * @note Intended solely for internal retrieval inside the {@link Log} domain adapter.
   */
  public _getBodyStructId(id: number): number {
    const index = this.getIndex(id);
    return this.bodyStructIds[index] ?? 0;
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
