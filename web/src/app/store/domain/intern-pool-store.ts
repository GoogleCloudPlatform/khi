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

import { allocateBuffer } from 'src/app/store/domain/types';

/**
 * Represents the shared memory structure of the intern pool.
 * This can be transferred to a WebWorker via postMessage.
 */
export interface InternPoolSharedData {
  readonly bufferSabs: readonly ArrayBuffer[];
  readonly metadataSab: ArrayBuffer;
  readonly capacity: number;
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
   * The ArrayBuffers backing the encoded string buffers.
   */
  private readonly bufferSabs: ArrayBuffer[] = [];

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
  private metadataSab: ArrayBuffer;

  /**
   * Whether this store is read-only (for worker-side decoding).
   */
  private readonly readOnly: boolean;

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

  // Private constructor
  private constructor(
    private readonly maxBufferSize: number,
    readOnly: boolean,
    initialCapacityOrSharedData: number | InternPoolSharedData,
  ) {
    this.readOnly = readOnly;
    if (typeof initialCapacityOrSharedData === 'number') {
      const initialCapacity = initialCapacityOrSharedData;
      this.metadataSab = allocateBuffer(initialCapacity * 10);
      this.bufferIndices = new Uint16Array(
        this.metadataSab,
        0,
        initialCapacity,
      );
      this.offsets = new Uint32Array(
        this.metadataSab,
        initialCapacity * 2,
        initialCapacity,
      );
      this.lengths = new Uint32Array(
        this.metadataSab,
        initialCapacity * 6,
        initialCapacity,
      );
    } else {
      const sharedData = initialCapacityOrSharedData;
      this.bufferSabs = Array.from(sharedData.bufferSabs);
      this.buffers = this.bufferSabs.map((sab) => new Uint8Array(sab));
      this.metadataSab = sharedData.metadataSab;
      this.bufferIndices = new Uint16Array(
        this.metadataSab,
        0,
        sharedData.capacity,
      );
      this.offsets = new Uint32Array(
        this.metadataSab,
        sharedData.capacity * 2,
        sharedData.capacity,
      );
      this.lengths = new Uint32Array(
        this.metadataSab,
        sharedData.capacity * 6,
        sharedData.capacity,
      );
      this.currentBufferIndex = this.buffers.length - 1;
      this.currentOffset = 0;
    }
  }

  /**
   * Creates a new writable InternPoolStore instance.
   * @param maxBufferSize The maximum capacity of each buffer segment in bytes.
   */
  public static create(
    maxBufferSize: number = 100 * 1024 * 1024,
  ): InternPoolStore {
    return new InternPoolStore(maxBufferSize, false, 1024);
  }

  /**
   * Reconstructs a read-only InternPoolStore instance from shared memory data.
   * @param sharedData The ArrayBuffers and capacity metadata.
   * @param maxBufferSize The maximum capacity of each buffer segment in bytes.
   */
  public static fromSharedData(
    sharedData: InternPoolSharedData,
    maxBufferSize: number = 100 * 1024 * 1024,
  ): InternPoolStore {
    return new InternPoolStore(maxBufferSize, true, sharedData);
  }

  /**
   * Adds multiple strings to the pool.
   * @param strings An iterable of objects containing id and value.
   */
  public addStrings(strings: Iterable<StringEntryDTO>): void {
    if (this.readOnly) {
      throw new Error('Cannot write to a shared read-only InternPoolStore');
    }
    for (const { id, value } of strings) {
      const encoded = this.encoder.encode(value);
      this.ensureCapacity(id + 1);

      if (
        this.currentBufferIndex === -1 ||
        this.maxBufferSize - this.currentOffset < encoded.length
      ) {
        const newSize = Math.max(this.maxBufferSize, encoded.length);
        const sab = allocateBuffer(newSize);
        this.bufferSabs.push(sab);
        this.buffers.push(new Uint8Array(sab));
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

  /**
   * Transfers shared memory structure of this store.
   * @returns The ArrayBuffers and metadata.
   */
  public getSharedData(): InternPoolSharedData {
    return {
      bufferSabs: this.bufferSabs,
      metadataSab: this.metadataSab,
      capacity: this.bufferIndices.length,
    };
  }

  /**
   * Ensures the metadata TypedArrays can hold up to the given capacity by reallocating ArrayBuffers.
   * @param minCapacity The required minimum capacity.
   */
  private ensureCapacity(minCapacity: number): void {
    if (this.readOnly) {
      throw new Error('Cannot resize a shared read-only InternPoolStore');
    }
    if (minCapacity <= this.bufferIndices.length) {
      return;
    }

    let newCapacity = this.bufferIndices.length * 2;
    while (newCapacity < minCapacity) {
      newCapacity *= 2;
    }

    const newMetadataSab = allocateBuffer(newCapacity * 10);
    const newBufferIndices = new Uint16Array(newMetadataSab, 0, newCapacity);
    const newOffsets = new Uint32Array(
      newMetadataSab,
      newCapacity * 2,
      newCapacity,
    );
    const newLengths = new Uint32Array(
      newMetadataSab,
      newCapacity * 6,
      newCapacity,
    );

    newBufferIndices.set(this.bufferIndices);
    newOffsets.set(this.offsets);
    newLengths.set(this.lengths);

    this.metadataSab = newMetadataSab;
    this.bufferIndices = newBufferIndices;
    this.offsets = newOffsets;
    this.lengths = newLengths;
  }
}
