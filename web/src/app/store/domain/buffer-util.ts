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

import { align } from 'src/app/common/memory-util';
import { allocateBuffer } from 'src/app/store/domain/types';

/**
 * Calculates the next capacity using exponential growth (doubling).
 *
 * @param currentCapacity The current buffer capacity in elements.
 * @param minCapacity The minimum required capacity in elements.
 * @param initialCapacity The fallback initial capacity if currentCapacity is zero.
 * @returns The new capacity greater than or equal to minCapacity.
 */
export function nextCapacity(
  currentCapacity: number,
  minCapacity: number,
  initialCapacity = 1024,
): number {
  if (currentCapacity <= 0) {
    return Math.max(initialCapacity, minCapacity);
  }
  let cap = currentCapacity * 2;
  while (cap < minCapacity) {
    cap *= 2;
  }
  return cap;
}

/**
 * Helper to compute sequential byte offsets with alignment in compound ArrayBuffers.
 */
export class BufferLayoutBuilder {
  private currentOffset = 0;

  /**
   * Adds an aligned field slot with the specified element count and byte size.
   * @param capacity The number of elements in the field view.
   * @param bytesPerElement The byte size of each element.
   * @param alignment The required byte boundary alignment (defaults to bytesPerElement).
   * @returns The starting byte offset for the field view.
   */
  public addField(
    capacity: number,
    bytesPerElement: number,
    alignment: number = bytesPerElement,
  ): number {
    this.currentOffset = align(this.currentOffset, alignment);
    const start = this.currentOffset;
    this.currentOffset += capacity * bytesPerElement;
    return start;
  }

  /**
   * Gets the total byte length required for all fields added so far.
   */
  public get totalBytes(): number {
    return this.currentOffset;
  }
}

/**
 * Allocates a new ArrayBuffer of the requested byte size and copies existing content into it.
 *
 * @param oldBuffer The previous buffer containing existing data.
 * @param newByteLength The total byte length of the new buffer.
 * @param bytesToCopy The number of valid bytes to copy from the previous buffer. Defaults to all bytes that fit.
 * @returns The newly allocated and populated ArrayBuffer.
 */
export function reallocateBuffer(
  oldBuffer: ArrayBuffer,
  newByteLength: number,
  bytesToCopy: number = Math.min(oldBuffer.byteLength, newByteLength),
): ArrayBuffer {
  const newBuffer = allocateBuffer(newByteLength);
  const copyLength = Math.min(bytesToCopy, newByteLength);
  if (copyLength > 0 && oldBuffer.byteLength > 0) {
    new Uint8Array(newBuffer).set(new Uint8Array(oldBuffer, 0, copyLength));
  }
  return newBuffer;
}

/**
 * Reallocates a Uint32Array to a new element length, copying elements up to copyLength.
 *
 * @param oldArray The current Uint32Array.
 * @param newLength The new element capacity.
 * @param copyLength The number of active elements to copy from oldArray.
 * @returns The newly allocated Uint32Array.
 */
export function reallocateUint32Array(
  oldArray: Uint32Array,
  newLength: number,
  copyLength: number = Math.min(oldArray.length, newLength),
): Uint32Array {
  const newArray = new Uint32Array(newLength);
  if (copyLength > 0) {
    newArray.set(oldArray.subarray(0, copyLength));
  }
  return newArray;
}
