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

import { align } from 'src/app/common/memory-util';
import { allocateBuffer } from 'src/app/store/domain/types';

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

  // Private constructor
  private constructor(private readonly maxBufferSize: number) {
    const layout = this.calculateOffsets(InternPoolStore.INITIAL_CAPACITY);
    this.metadataBuffer = allocateBuffer(layout.totalBytes);
    this.bufferIndices = new Uint16Array(
      this.metadataBuffer,
      layout.bufferIndicesOffset,
      InternPoolStore.INITIAL_CAPACITY,
    );
    this.offsets = new Uint32Array(
      this.metadataBuffer,
      layout.offsetsOffset,
      InternPoolStore.INITIAL_CAPACITY,
    );
    this.lengths = new Uint32Array(
      this.metadataBuffer,
      layout.lengthsOffset,
      InternPoolStore.INITIAL_CAPACITY,
    );
  }

  /**
   * Initializes a new InternPoolStore instance with interned strings.
   * @param strings An iterable of objects containing id and value.
   * @param maxBufferSize The maximum capacity of each buffer segment in bytes.
   */
  public static initialize(
    strings: Iterable<StringEntryDTO> = [],
    maxBufferSize: number = 100 * 1024 * 1024,
  ): InternPoolStore {
    const store = new InternPoolStore(maxBufferSize);
    store.addStrings(strings);
    return store;
  }

  /**
   * Adds multiple strings to the pool.
   * @param strings An iterable of objects containing id and value.
   */
  public addStrings(strings: Iterable<StringEntryDTO>): void {
    for (const { id, value } of strings) {
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

      this.bufferIndices[id] = this.currentBufferIndex + 1;
      this.offsets[id] = this.currentOffset;
      this.lengths[id] = encoded.length;

      this.currentOffset += encoded.length;
    }
  }

  /**
   * Retrieves a string value by its ID from the pool.
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

  private ensureCapacity(minCapacity: number): void {
    if (minCapacity <= this.bufferIndices.length) {
      return;
    }

    let newCapacity = this.bufferIndices.length * 2;
    while (newCapacity < minCapacity) {
      newCapacity *= 2;
    }

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

    newBufferIndices.set(this.bufferIndices);
    newOffsets.set(this.offsets);
    newLengths.set(this.lengths);

    this.metadataBuffer = newMetadataBuffer;
    this.bufferIndices = newBufferIndices;
    this.offsets = newOffsets;
    this.lengths = newLengths;
  }

  private calculateOffsets(capacity: number): InternPoolStoreLayout {
    let currentOffset = 0;

    const bufferIndicesOffset = currentOffset;
    currentOffset += capacity * 2; // Uint16Array: 2 bytes per element

    const offsetsOffset = align(currentOffset, 4); // Uint32Array: 4-byte aligned
    currentOffset = offsetsOffset + capacity * 4;

    const lengthsOffset = align(currentOffset, 4); // Uint32Array: 4-byte aligned
    currentOffset = lengthsOffset + capacity * 4;

    return {
      bufferIndicesOffset,
      offsetsOffset,
      lengthsOffset,
      totalBytes: currentOffset,
    };
  }
}
