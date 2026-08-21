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
  FilterResultMode,
  SparseBitset,
} from 'src/app/generated/api/v1/workbench_pb';

/**
 * High-performance bitset implementation for tracking positive integer entity IDs.
 * Utilizes a contiguous 32-bit typed array (`Uint32Array`) to perform sub-millisecond bitwise operations
 * and eliminate JavaScript object allocation overhead during timeline and log filtering.
 */
export class IdBitset {
  private words: Uint32Array;
  private count = 0;

  /**
   * Initializes a new instance of IdBitset with capacity for the specified maximum ID.
   *
   * @param maxId - The highest anticipated entity ID to preallocate words for.
   */
  constructor(maxId = 1024) {
    const wordCount = Math.ceil((Math.max(0, maxId) + 1) / 32);
    this.words = new Uint32Array(wordCount);
  }

  /**
   * Checks whether the specified entity ID is present in the bitset.
   *
   * @param id - The numeric entity ID to check.
   * @returns True if the bit corresponding to the ID is set; otherwise false.
   */
  public has(id: number): boolean {
    if (id < 0) return false;
    const wordIdx = id >>> 5;
    if (wordIdx >= this.words.length) return false;
    return (this.words[wordIdx] & (1 << (id & 31))) !== 0;
  }

  /**
   * Adds the specified entity ID to the bitset.
   *
   * @param id - The numeric entity ID to add.
   */
  public add(id: number): void {
    if (id < 0) return;
    const wordIdx = id >>> 5;
    if (wordIdx >= this.words.length) {
      this.grow(id);
    }
    const mask = 1 << (id & 31);
    if ((this.words[wordIdx] & mask) === 0) {
      this.words[wordIdx] |= mask;
      this.count++;
    }
  }

  /**
   * Removes the specified entity ID from the bitset.
   *
   * @param id - The numeric entity ID to remove.
   * @returns True if the entity ID was present and removed; otherwise false.
   */
  public delete(id: number): boolean {
    if (id < 0) return false;
    const wordIdx = id >>> 5;
    if (wordIdx >= this.words.length) return false;
    const mask = 1 << (id & 31);
    if ((this.words[wordIdx] & mask) !== 0) {
      this.words[wordIdx] &= ~mask;
      this.count--;
      return true;
    }
    return false;
  }

  /**
   * Clears all set bits in the bitset.
   */
  public clear(): void {
    this.words.fill(0);
    this.count = 0;
  }

  /**
   * Gets the total count of active set entity IDs in the bitset.
   */
  public get size(): number {
    return this.count;
  }

  /**
   * Creates an exact shallow copy of the bitset and its internal words array.
   *
   * @returns A newly cloned IdBitset instance.
   */
  public clone(): IdBitset {
    const cloned = new IdBitset(0);
    cloned.words = new Uint32Array(this.words);
    cloned.count = this.count;
    return cloned;
  }

  /**
   * Yields an iterator over all numeric IDs currently set in the bitset in ascending order.
   *
   * @returns An iterator yielding each set ID.
   */
  public *values(): IterableIterator<number> {
    for (let w = 0; w < this.words.length; w++) {
      const word = this.words[w];
      if (word === 0) continue;
      const base = w * 32;
      for (let b = 0; b < 32; b++) {
        if ((word & (1 << b)) !== 0) {
          yield base + b;
        }
      }
    }
  }

  /**
   * Default iterator implementation.
   */
  public [Symbol.iterator](): IterableIterator<number> {
    return this.values();
  }

  private grow(targetId: number): void {
    const requiredWords = Math.ceil((targetId + 1) / 32);
    const newWordsCount = Math.max(requiredWords, this.words.length * 2);
    const newWords = new Uint32Array(newWordsCount);
    newWords.set(this.words);
    this.words = newWords;
  }

  /**
   * Creates an empty IdBitset with preallocated word capacity.
   *
   * @param maxId - Optional maximum ID to preallocate storage for.
   * @returns An empty IdBitset.
   */
  public static createEmpty(maxId = 1024): IdBitset {
    return new IdBitset(maxId);
  }

  /**
   * Creates an IdBitset containing all IDs provided in the iterable.
   *
   * @param allIds - Iterable of numeric IDs to include in the bitset.
   * @param maxId - Optional maximum ID to preallocate storage for.
   * @returns An IdBitset with all specified IDs set.
   */
  public static fromAll(
    allIds: Iterable<number>,
    maxId?: number,
  ): IdBitset {
    let calculatedMax = maxId ?? 0;
    if (maxId === undefined && Array.isArray(allIds)) {
      for (let i = 0; i < allIds.length; i++) {
        if (allIds[i] > calculatedMax) calculatedMax = allIds[i];
      }
    }
    const bitset = new IdBitset(calculatedMax);
    for (const id of allIds) {
      bitset.add(id);
    }
    return bitset;
  }

  /**
   * Creates an IdBitset containing all sequential 1-indexed IDs from 1 to totalCount.
   *
   * @param totalCount - The total number of sequential entity IDs (1 to totalCount).
   * @returns An IdBitset with all sequential IDs set.
   */
  public static fromSequential(totalCount: number): IdBitset {
    const count = totalCount ?? 0;
    const bitset = new IdBitset(count);
    for (let id = 1; id <= count; id++) {
      bitset.add(id);
    }
    return bitset;
  }

  /**
   * Creates an IdBitset by decoding a SparseBitset Protobuf payload in either INCLUDE or EXCLUDE mode.
   *
   * @param mode - The filter result mode specifying whether set bits represent matching or excluded items.
   * @param sparse - The SparseBitset Protobuf message containing block indices and masks.
   * @param totalCount - The total count of sequential 1-indexed entity IDs in the dataset.
   * @returns The decoded IdBitset.
   */
  public static fromSparseBitset(
    mode: FilterResultMode | undefined,
    sparse: SparseBitset | undefined,
    totalCount: number,
  ): IdBitset {
    const count = totalCount ?? 0;
    if (mode === FilterResultMode.EXCLUDE) {
      const bitset = IdBitset.fromSequential(count);
      if (!sparse || !sparse.indices || !sparse.masks) {
        return bitset;
      }
      const indices = sparse.indices;
      const masks = sparse.masks;
      const len = Math.min(indices.length, masks.length);
      for (let i = 0; i < len; i++) {
        const wordIdx = indices[i];
        const mask = masks[i];
        if (wordIdx < bitset.words.length) {
          const oldWord = bitset.words[wordIdx];
          const newWord = oldWord & ~mask;
          if (oldWord !== newWord) {
            // Count number of cleared bits to keep size accurate
            let clearedBits = oldWord ^ newWord;
            while (clearedBits !== 0) {
              bitset.count--;
              clearedBits &= clearedBits - 1;
            }
            bitset.words[wordIdx] = newWord;
          }
        }
      }
      return bitset;
    }

    // Default to INCLUDE mode
    const bitset = new IdBitset(count);
    if (!sparse || !sparse.indices || !sparse.masks) {
      return bitset;
    }
    const indices = sparse.indices;
    const masks = sparse.masks;
    const len = Math.min(indices.length, masks.length);
    for (let i = 0; i < len; i++) {
      const wordIdx = indices[i];
      const mask = masks[i];
      if (wordIdx >= bitset.words.length) {
        bitset.grow((wordIdx + 1) * 32);
      }
      const oldWord = bitset.words[wordIdx];
      const newWord = oldWord | mask;
      if (oldWord !== newWord) {
        let addedBits = oldWord ^ newWord;
        while (addedBits !== 0) {
          bitset.count++;
          addedBits &= addedBits - 1;
        }
        bitset.words[wordIdx] = newWord;
      }
    }
    return bitset;
  }
}
