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
import { InternedStructDecoder } from 'src/app/store/domain/struct-decoder';
import { InternedStructSchema } from 'src/app/generated/khifile/shared_pb';
import { LogType, Severity, StyleProvider } from 'src/app/store/domain/style';
import {
  ReadonlyDomainElement,
  allocateBuffer,
  isSharedBuffer,
} from 'src/app/store/domain/types';
import { fromBinary } from '@bufbuild/protobuf';

/**
 * Align the offset to the specified byte alignment.
 */
function align(offset: number, alignment: number): number {
  return Math.ceil(offset / alignment) * alignment;
}

/**
 * Represents the shared memory structure of the log store.
 */
export interface LogStoreSharedData {
  readonly metadataSab: SharedArrayBuffer | ArrayBuffer;
  readonly bodyBufferSabs: readonly (SharedArrayBuffer | ArrayBuffer)[];
  readonly count: number;
  readonly idToIndex: readonly (number | undefined)[];
}

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
  private readonly readOnly: boolean;

  private metadataSab!: SharedArrayBuffer | ArrayBuffer;
  private ids!: Uint32Array;
  private timestamps!: BigUint64Array;
  private logTypeIds!: Uint32Array;
  private severityIds!: Uint32Array;
  private summaryStringIds!: Uint32Array;

  // Packed body metadata
  private bodyBufferIndices!: Uint16Array;
  private bodyOffsets!: Uint32Array;
  private bodyLengths!: Uint32Array;

  private readonly bodyBufferSabs: (SharedArrayBuffer | ArrayBuffer)[] = [];
  private readonly bodyBuffers: Uint8Array[] = [];

  private currentBufferIndex = -1;
  private currentOffset = 0;

  private readonly decodedBodyCache: WeakRef<
    ReadonlyDomainElement<Record<string, unknown>>
  >[] = [];
  private idToIndex: (number | undefined)[] = [];
  private readonly decoder: InternedStructDecoder;

  private constructor(
    private readonly internPool: InternPoolStore,
    private readonly styleStore: StyleProvider,
    private readonly maxBufferSize: number,
    readOnly: boolean,
    initialData: number | LogStoreSharedData,
  ) {
    this.readOnly = readOnly;
    this.decoder = new InternedStructDecoder(this.internPool);

    if (typeof initialData === 'number') {
      const initialCapacity = initialData;
      this.allocateMetadata(initialCapacity);
    } else {
      const sharedData = initialData;
      this.metadataSab = sharedData.metadataSab;
      this.bodyBufferSabs = Array.from(sharedData.bodyBufferSabs);
      this.bodyBuffers = this.bodyBufferSabs.map((sab) => new Uint8Array(sab));
      this.idToIndex = Array.from(sharedData.idToIndex);
      this.mapMetadataViews(sharedData.count);
    }
  }

  /**
   * Creates a new writable LogStore instance.
   */
  public static create(
    internPool: InternPoolStore,
    styleStore: StyleProvider,
    maxBufferSize: number = 100 * 1024 * 1024,
  ): LogStore {
    return new LogStore(internPool, styleStore, maxBufferSize, false, 1024);
  }

  /**
   * Reconstructs a read-only LogStore instance from shared memory data.
   */
  public static fromSharedData(
    internPool: InternPoolStore,
    styleStore: StyleProvider,
    sharedData: LogStoreSharedData,
    maxBufferSize: number = 100 * 1024 * 1024,
  ): LogStore {
    return new LogStore(
      internPool,
      styleStore,
      maxBufferSize,
      true,
      sharedData,
    );
  }

  private allocateMetadata(capacity: number): void {
    let currentOffset = 0;
    const idsOffset = currentOffset;
    currentOffset += capacity * 4;

    const logTypeIdsOffset = currentOffset;
    currentOffset += capacity * 4;

    const severityIdsOffset = currentOffset;
    currentOffset += capacity * 4;

    const summaryStringIdsOffset = currentOffset;
    currentOffset += capacity * 4;

    const timestampsOffset = align(currentOffset, 8);
    currentOffset = timestampsOffset + capacity * 8;

    const bodyBufferIndicesOffset = align(currentOffset, 2);
    currentOffset = bodyBufferIndicesOffset + capacity * 2;

    const bodyOffsetsOffset = align(currentOffset, 4);
    currentOffset = bodyOffsetsOffset + capacity * 4;

    const bodyLengthsOffset = align(currentOffset, 4);
    currentOffset = bodyLengthsOffset + capacity * 4;

    const totalBytes = currentOffset;
    this.metadataSab = allocateBuffer(totalBytes);

    this.ids = new Uint32Array(this.metadataSab, idsOffset, capacity);
    this.logTypeIds = new Uint32Array(
      this.metadataSab,
      logTypeIdsOffset,
      capacity,
    );
    this.severityIds = new Uint32Array(
      this.metadataSab,
      severityIdsOffset,
      capacity,
    );
    this.summaryStringIds = new Uint32Array(
      this.metadataSab,
      summaryStringIdsOffset,
      capacity,
    );
    this.timestamps = new BigUint64Array(
      this.metadataSab,
      timestampsOffset,
      capacity,
    );
    this.bodyBufferIndices = new Uint16Array(
      this.metadataSab,
      bodyBufferIndicesOffset,
      capacity,
    );
    this.bodyOffsets = new Uint32Array(
      this.metadataSab,
      bodyOffsetsOffset,
      capacity,
    );
    this.bodyLengths = new Uint32Array(
      this.metadataSab,
      bodyLengthsOffset,
      capacity,
    );
  }

  private mapMetadataViews(capacity: number): void {
    let currentOffset = 0;
    const idsOffset = currentOffset;
    currentOffset += capacity * 4;

    const logTypeIdsOffset = currentOffset;
    currentOffset += capacity * 4;

    const severityIdsOffset = currentOffset;
    currentOffset += capacity * 4;

    const summaryStringIdsOffset = currentOffset;
    currentOffset += capacity * 4;

    const timestampsOffset = align(currentOffset, 8);
    currentOffset = timestampsOffset + capacity * 8;

    const bodyBufferIndicesOffset = align(currentOffset, 2);
    currentOffset = bodyBufferIndicesOffset + capacity * 2;

    const bodyOffsetsOffset = align(currentOffset, 4);
    currentOffset = bodyOffsetsOffset + capacity * 4;

    const bodyLengthsOffset = align(currentOffset, 4);
    currentOffset = bodyLengthsOffset + capacity * 4;

    this.ids = new Uint32Array(this.metadataSab, idsOffset, capacity);
    this.logTypeIds = new Uint32Array(
      this.metadataSab,
      logTypeIdsOffset,
      capacity,
    );
    this.severityIds = new Uint32Array(
      this.metadataSab,
      severityIdsOffset,
      capacity,
    );
    this.summaryStringIds = new Uint32Array(
      this.metadataSab,
      summaryStringIdsOffset,
      capacity,
    );
    this.timestamps = new BigUint64Array(
      this.metadataSab,
      timestampsOffset,
      capacity,
    );
    this.bodyBufferIndices = new Uint16Array(
      this.metadataSab,
      bodyBufferIndicesOffset,
      capacity,
    );
    this.bodyOffsets = new Uint32Array(
      this.metadataSab,
      bodyOffsetsOffset,
      capacity,
    );
    this.bodyLengths = new Uint32Array(
      this.metadataSab,
      bodyLengthsOffset,
      capacity,
    );
  }

  /**
   * Initializes the store with the raw logs.
   * Assumes logs are already sorted by timestamp.
   * @param logs The iterable of raw logs.
   * @param count The total number of logs.
   */
  public initialize(logs: Iterable<LogDTO>, count: number): void {
    if (this.readOnly) {
      throw new Error('Cannot write to a shared read-only LogStore');
    }

    this.allocateMetadata(count);

    this.decodedBodyCache.length = count;
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

      if (log.body !== undefined && log.body.length > 0) {
        this.addBody(index, log.body);
      } else {
        this.bodyBufferIndices[index] = 0;
        this.bodyOffsets[index] = 0;
        this.bodyLengths[index] = 0;
      }

      this.idToIndex[log.id] = index;
      index++;
    }
  }

  private addBody(index: number, bodyBytes: Uint8Array): void {
    const length = bodyBytes.length;

    if (length > this.maxBufferSize) {
      const sab = allocateBuffer(length);
      const buf = new Uint8Array(sab);
      buf.set(bodyBytes);
      this.bodyBufferSabs.push(sab);
      this.bodyBuffers.push(buf);

      const newBufIdx = this.bodyBuffers.length - 1;
      this.bodyBufferIndices[index] = newBufIdx + 1;
      this.bodyOffsets[index] = 0;
      this.bodyLengths[index] = length;
      return;
    }

    if (
      this.currentBufferIndex === -1 ||
      this.currentOffset + length > this.maxBufferSize
    ) {
      const sab = allocateBuffer(this.maxBufferSize);
      const buf = new Uint8Array(sab);
      this.bodyBufferSabs.push(sab);
      this.bodyBuffers.push(buf);
      this.currentBufferIndex = this.bodyBuffers.length - 1;
      this.currentOffset = 0;
    }

    const currentBuf = this.bodyBuffers[this.currentBufferIndex];
    currentBuf.set(bodyBytes, this.currentOffset);

    this.bodyBufferIndices[index] = this.currentBufferIndex + 1;
    this.bodyOffsets[index] = this.currentOffset;
    this.bodyLengths[index] = length;

    this.currentOffset += length;
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

    const bufIdx = this.bodyBufferIndices[index];
    if (bufIdx === 0) return null;

    const offset = this.bodyOffsets[index];
    const length = this.bodyLengths[index];

    const buffer = this.bodyBuffers[bufIdx - 1];
    const bytes = buffer.subarray(offset, offset + length);
    let decodeTarget = bytes;
    if (isSharedBuffer(bytes.buffer)) {
      const nonSharedBytes = new Uint8Array(length);
      nonSharedBytes.set(bytes);
      decodeTarget = nonSharedBytes;
    }

    const struct = fromBinary(InternedStructSchema, decodeTarget);
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

  /**
   * Returns the shared memory representation of this LogStore.
   */
  public getSharedData(): LogStoreSharedData {
    return {
      metadataSab: this.metadataSab,
      bodyBufferSabs: this.bodyBufferSabs,
      count: this.ids.length,
      idToIndex: this.idToIndex,
    };
  }
}
