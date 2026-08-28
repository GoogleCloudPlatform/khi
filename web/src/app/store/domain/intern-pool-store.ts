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

import {
  BufferLayoutBuilder,
  nextCapacity,
} from 'src/app/store/domain/buffer-util';
import { allocateBuffer } from 'src/app/store/domain/types';

/**
 * Represents an entry in the interned string pool.
 */
export interface StringEntryDTO {
  /**
   * The unique ID of the interned string.
   */
  readonly id: number;
  /**
   * The actual string value.
   */
  readonly value: string;
}

interface InternPoolStoreLayout {
  readonly bufferIndicesOffset: number;
  readonly offsetsOffset: number;
  readonly lengthsOffset: number;
  readonly totalBytes: number;
}

/**
 * Manages the interned strings used in inspection data using ArrayBuffers.
 */
export class InternPoolStore {
  /**
   * Allocated buffers storing the encoded string data.
   */
  private readonly buffers: Uint8Array[] = [];

  /**
   * Tracks the buffer index for each string ID (1-based index, 0 represents uninitialized).
   */
  private bufferIndices: Uint16Array;

  /**
   * Tracks the byte offset inside the buffer for each string ID.
   */
  private offsets: Uint32Array;

  /**
   * Tracks the byte length of the encoded string for each string ID.
   */
  private lengths: Uint32Array;

  /**
   * The single ArrayBuffer holding all metadata arrays.
   */
  private metadataBuffer: ArrayBuffer;

  /**
   * The index of the buffer currently being written to.
   */
  private currentBufferIndex = -1;

  /**
   * The current write offset in the active buffer.
   */
  private currentOffset = 0;

  private readonly encoder = new TextEncoder();
  private readonly decoder = new TextDecoder();

  private static readonly INITIAL_CAPACITY = 1024;

  private _maxStringId = 0;
  private stringCount = 0;

  // Private constructor
  private constructor(
    initialCapacity: number,
    private readonly maxBufferSize: number,
  ) {
    const layout = this.calculateOffsets(initialCapacity);
    this.metadataBuffer = allocateBuffer(layout.totalBytes);
    this.bufferIndices = new Uint16Array(
      this.metadataBuffer,
      layout.bufferIndicesOffset,
      initialCapacity,
    );
    this.offsets = new Uint32Array(
      this.metadataBuffer,
      layout.offsetsOffset,
      initialCapacity,
    );
    this.lengths = new Uint32Array(
      this.metadataBuffer,
      layout.lengthsOffset,
      initialCapacity,
    );
  }

  /**
   * Creates an empty InternPoolStore instance.
   *
   * @param initialCapacity The initial capacity of the metadata buffer.
   * @param maxBufferSize The maximum capacity of each buffer segment in bytes.
   * @returns A new empty InternPoolStore instance.
   */
  public static create(
    initialCapacity: number = InternPoolStore.INITIAL_CAPACITY,
    maxBufferSize: number = 100 * 1024 * 1024,
  ): InternPoolStore {
    return new InternPoolStore(initialCapacity, maxBufferSize);
  }

  /**
   * Returns the count of unique strings stored in the pool.
   */
  public get count(): number {
    return this.stringCount;
  }

  /**
   * Returns the highest string ID encountered.
   */
  public get maxStringId(): number {
    return this._maxStringId;
  }

  /**
   * Adds a single string entry to the pool.
   *
   * @param entry An object containing id and value.
   */
  public addString(entry: StringEntryDTO): void {
    const { id, value } = entry;
    const encoded = this.encoder.encode(value);
    this.ensureCapacity(id + 1);

    if (
      this.currentBufferIndex === -1 ||
      this.maxBufferSize - this.currentOffset < encoded.length
    ) {
      const newSize = Math.max(this.maxBufferSize, encoded.length);
      const buffer = allocateBuffer(newSize);
      this.buffers.push(new Uint8Array(buffer));
      this.currentBufferIndex = this.buffers.length - 1;
      this.currentOffset = 0;
    }

    const activeBuffer = this.buffers[this.currentBufferIndex];
    activeBuffer.set(encoded, this.currentOffset);

    if (this.bufferIndices[id] === 0) {
      this.stringCount++;
    }

    this.bufferIndices[id] = this.currentBufferIndex + 1;
    this.offsets[id] = this.currentOffset;
    this.lengths[id] = encoded.length;

    this.currentOffset += encoded.length;
    if (id > this._maxStringId) {
      this._maxStringId = id;
    }
  }

  /**
   * Adds multiple strings to the pool.
   *
   * @param strings An iterable of objects containing id and value.
   */
  public addStrings(strings: Iterable<StringEntryDTO>): void {
    if (Array.isArray(strings)) {
      let maxId = this._maxStringId;
      for (const entry of strings) {
        if (entry.id > maxId) {
          maxId = entry.id;
        }
      }
      this.ensureCapacity(maxId + 1);
    }
    for (const entry of strings) {
      this.addString(entry);
    }
  }

  private reallocate(newCapacity: number): void {
    const layout = this.calculateOffsets(newCapacity);
    const newMetadataBuffer = allocateBuffer(layout.totalBytes);
    const newBufferIndices = new Uint16Array(
      newMetadataBuffer,
      layout.bufferIndicesOffset,
      newCapacity,
    );
    const newOffsets = new Uint32Array(
      newMetadataBuffer,
      layout.offsetsOffset,
      newCapacity,
    );
    const newLengths = new Uint32Array(
      newMetadataBuffer,
      layout.lengthsOffset,
      newCapacity,
    );

    const copyLen = Math.min(this.bufferIndices.length, newCapacity);
    if (copyLen > 0) {
      newBufferIndices.set(this.bufferIndices.subarray(0, copyLen));
      newOffsets.set(this.offsets.subarray(0, copyLen));
      newLengths.set(this.lengths.subarray(0, copyLen));
    }

    this.metadataBuffer = newMetadataBuffer;
    this.bufferIndices = newBufferIndices;
    this.offsets = newOffsets;
    this.lengths = newLengths;
  }

  /**
   * Shrinks metadataBuffer to the minimal required size based on the current count.
   */
  public shrinkToFit(): void {
    const neededCapacity = this._maxStringId + 1;
    if (neededCapacity < this.bufferIndices.length && neededCapacity > 0) {
      this.reallocate(neededCapacity);
    }
  }

  /**
   * Retrieves a string value by its ID from the pool.
   *
   * @param id The ID of the string.
   * @returns The string value.
   * @throws Error if the ID is not found in the pool.
   */
  public getString(id: number): string {
    if (id < 0 || id >= this.bufferIndices.length) {
      throw new Error(`String ID ${id} not found in pool`);
    }

    const bufferIndexPlusOne = this.bufferIndices[id];
    if (bufferIndexPlusOne === 0) {
      throw new Error(`String ID ${id} not found in pool`);
    }

    const bufferIndex = bufferIndexPlusOne - 1;
    const offset = this.offsets[id];
    const length = this.lengths[id];

    const buffer = this.buffers[bufferIndex];
    const bytes = buffer.subarray(offset, offset + length);
    return this.decoder.decode(bytes);
  }

  /**
   * Ensures metadata buffer has at least minCapacity slots.
   *
   * @param minCapacity The required minimum capacity.
   */
  private ensureCapacity(minCapacity: number): void {
    if (minCapacity <= this.bufferIndices.length) {
      return;
    }

    const newCapacity = nextCapacity(this.bufferIndices.length, minCapacity);
    this.reallocate(newCapacity);
  }

  private calculateOffsets(capacity: number): InternPoolStoreLayout {
    const builder = new BufferLayoutBuilder();
    return {
      bufferIndicesOffset: builder.addField(capacity, 2),
      offsetsOffset: builder.addField(capacity, 4, 4),
      lengthsOffset: builder.addField(capacity, 4, 4),
      totalBytes: builder.totalBytes,
    };
  }
}
