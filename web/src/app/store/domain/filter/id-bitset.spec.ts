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

import { IdBitset } from 'src/app/store/domain/filter/id-bitset';
import { FilterResultMode } from 'src/app/generated/api/v1/workbench_pb';
import { SparseBitsetSchema } from 'src/app/generated/api/v1/sparse_bitset_pb';
import { create } from '@bufbuild/protobuf';

describe('IdBitset', () => {
  it('should create empty bitset with correct initial state', () => {
    const bitset = IdBitset.createEmpty();
    expect(bitset.size).toBe(0);
    expect(bitset.has(0)).toBeFalse();
    expect(bitset.has(100)).toBeFalse();
    expect(Array.from(bitset.values())).toEqual([]);
  });

  it('should add IDs and check presence with has()', () => {
    const bitset = new IdBitset();
    bitset.add(1);
    bitset.add(31);
    bitset.add(32);
    bitset.add(64);
    bitset.add(1000);

    expect(bitset.size).toBe(5);
    expect(bitset.has(1)).toBeTrue();
    expect(bitset.has(31)).toBeTrue();
    expect(bitset.has(32)).toBeTrue();
    expect(bitset.has(64)).toBeTrue();
    expect(bitset.has(1000)).toBeTrue();

    expect(bitset.has(0)).toBeFalse();
    expect(bitset.has(2)).toBeFalse();
    expect(bitset.has(33)).toBeFalse();
    expect(bitset.has(999)).toBeFalse();

    // Adding duplicate ID should not increase size
    bitset.add(1);
    expect(bitset.size).toBe(5);
  });

  it('should delete IDs correctly', () => {
    const bitset = IdBitset.fromAll([10, 20, 30, 40]);
    expect(bitset.size).toBe(4);

    expect(bitset.delete(20)).toBeTrue();
    expect(bitset.has(20)).toBeFalse();
    expect(bitset.size).toBe(3);

    // Deleting non-existent ID
    expect(bitset.delete(999)).toBeFalse();
    expect(bitset.size).toBe(3);
  });

  it('should clone correctly and remain isolated from mutations', () => {
    const original = IdBitset.fromAll([1, 2, 3]);
    const cloned = original.clone();

    expect(cloned.size).toBe(3);
    expect(cloned.has(1)).toBeTrue();

    cloned.add(4);
    expect(cloned.has(4)).toBeTrue();
    expect(original.has(4)).toBeFalse();

    cloned.delete(1);
    expect(cloned.has(1)).toBeFalse();
    expect(original.has(1)).toBeTrue();
  });

  it('should iterate over values in ascending order', () => {
    const bitset = IdBitset.fromAll([100, 5, 32, 1, 64]);
    const values = Array.from(bitset.values());
    expect(values).toEqual([1, 5, 32, 64, 100]);
  });

  it('should create sequential bitset from 1 to totalCount with fromSequential', () => {
    const testCounts = [0, 1, 30, 31, 32, 33, 63, 64, 65, 1000];
    for (const count of testCounts) {
      const bitset = IdBitset.fromSequential(count);
      expect(bitset.size).toBe(count);
      expect(bitset.has(0)).toBeFalse();
      for (let id = 1; id <= count; id++) {
        expect(bitset.has(id)).toBeTrue();
      }
      expect(bitset.has(count + 1)).toBeFalse();

      const values = Array.from(bitset.values());
      expect(values.length).toBe(count);
      if (count > 0) {
        expect(values[0]).toBe(1);
        expect(values[values.length - 1]).toBe(count);
      }
    }
  });

  describe('fromSparseBitset', () => {
    const totalCount = 64;

    it('should decode INCLUDE mode correctly', () => {
      // INCLUDE IDs: 1, 32 (block 0 bit 1 -> 0x2, block 1 bit 0 -> 0x1)
      const sparse = create(SparseBitsetSchema, {
        indices: [0, 1],
        masks: [0x2, 0x1],
      });

      const bitset = IdBitset.fromSparseBitset(
        FilterResultMode.INCLUDE,
        sparse,
        totalCount,
      );

      expect(bitset.size).toBe(2);
      expect(bitset.has(1)).toBeTrue();
      expect(bitset.has(32)).toBeTrue();
      expect(bitset.has(2)).toBeFalse();
      expect(bitset.has(33)).toBeFalse();
      expect(bitset.has(64)).toBeFalse();
    });

    it('should decode EXCLUDE mode correctly', () => {
      // EXCLUDE IDs: 2, 33 (block 0 bit 2 -> 0x4, block 1 bit 1 -> 0x2)
      const sparse = create(SparseBitsetSchema, {
        indices: [0, 1],
        masks: [0x4, 0x2],
      });

      const bitset = IdBitset.fromSparseBitset(
        FilterResultMode.EXCLUDE,
        sparse,
        totalCount,
      );

      expect(bitset.has(1)).toBeTrue();
      expect(bitset.has(2)).toBeFalse(); // excluded
      expect(bitset.has(3)).toBeTrue();
      expect(bitset.has(4)).toBeTrue();
      expect(bitset.has(5)).toBeTrue();
      expect(bitset.has(32)).toBeTrue();
      expect(bitset.has(33)).toBeFalse(); // excluded
      expect(bitset.has(64)).toBeTrue();
      expect(bitset.size).toBe(62);
    });

    it('should handle undefined or empty sparse bitset in INCLUDE mode as empty', () => {
      const bitset = IdBitset.fromSparseBitset(
        FilterResultMode.INCLUDE,
        undefined,
        totalCount,
      );
      expect(bitset.size).toBe(0);
    });

    it('should handle undefined or empty sparse bitset in EXCLUDE mode as all included', () => {
      const bitset = IdBitset.fromSparseBitset(
        FilterResultMode.EXCLUDE,
        undefined,
        totalCount,
      );
      expect(bitset.size).toBe(totalCount);
      for (let id = 1; id <= totalCount; id++) {
        expect(bitset.has(id)).toBeTrue();
      }
    });
  });

  describe('toSparseBitset', () => {
    it('should encode empty bitset into empty SparseBitset', () => {
      const bitset = new IdBitset();
      const sparse = bitset.toSparseBitset();
      expect(sparse.indices).toEqual([]);
      expect(sparse.masks).toEqual([]);
    });

    it('should encode populated bitset into correct SparseBitset chunks', () => {
      const bitset = new IdBitset();
      bitset.add(1);
      bitset.add(32);
      bitset.add(65);

      const sparse = bitset.toSparseBitset();
      expect(sparse.indices).toEqual([0, 1, 2]);
      expect(sparse.masks).toEqual([1 << 1, 1 << 0, 1 << 1]);
    });
  });
});
