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

import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import {
  StructStore,
  StructValueKind,
} from 'src/app/store/domain/struct-store';

describe('StructStore', () => {
  let internPool: InternPoolStore;
  let structStore: StructStore;

  beforeEach(() => {
    internPool = InternPoolStore.create();
    structStore = StructStore.create(internPool);
  });

  it('decodes simple struct value variants correctly', () => {
    internPool.addStrings([
      { id: 1, value: 'a' },
      { id: 2, value: 'b' },
      { id: 3, value: 'c' },
      { id: 4, value: 'd' },
      { id: 5, value: 'e' },
      { id: 6, value: 'f' },
      { id: 100, value: 'resolved_value' },
    ]);

    structStore.addFieldPathSet({
      id: 10,
      fieldPathStringIds: [1, 2, 3, 4, 5, 6],
    });

    structStore.addStruct({
      id: 1,
      fieldPathSetId: 10,
      values: [
        { kind: StructValueKind.Null },
        { kind: StructValueKind.Int64, value: 123n },
        { kind: StructValueKind.Double, value: 3.14 },
        { kind: StructValueKind.Bool, value: true },
        { kind: StructValueKind.String, stringId: 100 },
        {
          kind: StructValueKind.Timestamp,
          timestampNs: 1_000_000_000_500n,
        },
      ],
    });

    expect(structStore.count).toBe(1);

    const result = structStore.getStruct(1);
    expect(result).toEqual({
      a: null,
      b: 123,
      c: 3.14,
      d: true,
      e: 'resolved_value',
      f: 1_000_000_000_500n,
    });
  });

  it('decodes flattened keys with \\0 separators into nested objects', () => {
    internPool.addStrings([
      { id: 1, value: 'metadata\0name' },
      { id: 2, value: 'metadata\0labels\0env' },
      { id: 10, value: 'frontend-pod' },
      { id: 11, value: 'production' },
    ]);

    structStore.addFieldPathSet({
      id: 20,
      fieldPathStringIds: [1, 2],
    });

    structStore.addStruct({
      id: 5,
      fieldPathSetId: 20,
      values: [
        { kind: StructValueKind.String, stringId: 10 },
        { kind: StructValueKind.String, stringId: 11 },
      ],
    });

    const result = structStore.getStruct(5);
    expect(result).toEqual({
      metadata: {
        name: 'frontend-pod',
        labels: {
          env: 'production',
        },
      },
    });
  });

  it('decodes recursive nested structs and list value variants', () => {
    internPool.addStrings([
      { id: 1, value: 'nested' },
      { id: 2, value: 'elements' },
      { id: 3, value: 'key' },
      { id: 10, value: 'leaf' },
    ]);

    structStore.addFieldPathSet({
      id: 30,
      fieldPathStringIds: [1, 2],
    });
    structStore.addFieldPathSet({
      id: 31,
      fieldPathStringIds: [3],
    });

    // Nested struct (id: 100)
    structStore.addStruct({
      id: 100,
      fieldPathSetId: 31,
      values: [{ kind: StructValueKind.String, stringId: 10 }],
    });

    // Parent struct (id: 200)
    structStore.addStruct({
      id: 200,
      fieldPathSetId: 30,
      values: [
        { kind: StructValueKind.StructId, structId: 100 },
        {
          kind: StructValueKind.List,
          values: [
            { kind: StructValueKind.Int64, value: 10n },
            { kind: StructValueKind.Bool, value: false },
          ],
        },
      ],
    });

    const result = structStore.getStruct(200);
    expect(result).toEqual({
      nested: {
        key: 'leaf',
      },
      elements: [10, false],
    });
  });

  it('returns null for nonexistent or uninitialized struct IDs', () => {
    expect(structStore.getStruct(0)).toBeNull();
    expect(structStore.getStruct(999)).toBeNull();
    expect(structStore.getStruct(-1)).toBeNull();
  });

  it('dynamically expands buffer and shrinks to fit correctly', () => {
    const smallStore = StructStore.create(internPool, 2);
    internPool.addStrings([{ id: 1, value: 'field' }]);

    smallStore.addFieldPathSet({
      id: 1,
      fieldPathStringIds: [1],
    });

    for (let i = 1; i <= 20; i++) {
      smallStore.addStruct({
        id: i,
        fieldPathSetId: 1,
        values: [{ kind: StructValueKind.Int64, value: BigInt(i * 10) }],
      });
    }

    expect(smallStore.count).toBe(20);
    smallStore.shrinkToFit();

    expect(smallStore.count).toBe(20);
    expect(smallStore.getStruct(1)).toEqual({ field: 10 });
    expect(smallStore.getStruct(20)).toEqual({ field: 200 });
  });

  it('preserves int64 as bigint when value exceeds MAX_SAFE_INTEGER', () => {
    internPool.addStrings([
      { id: 1, value: 'safe' },
      { id: 2, value: 'unsafe' },
      { id: 3, value: 'negative_unsafe' },
    ]);
    structStore.addFieldPathSet({
      id: 1,
      fieldPathStringIds: [1, 2, 3],
    });

    const safeValue = BigInt(Number.MAX_SAFE_INTEGER);
    const unsafeValue = BigInt(Number.MAX_SAFE_INTEGER) + 100n;
    const negativeUnsafe = BigInt(Number.MIN_SAFE_INTEGER) - 100n;

    structStore.addStruct({
      id: 1,
      fieldPathSetId: 1,
      values: [
        { kind: StructValueKind.Int64, value: safeValue },
        { kind: StructValueKind.Int64, value: unsafeValue },
        { kind: StructValueKind.Int64, value: negativeUnsafe },
      ],
    });

    const result = structStore.getStruct(1);
    expect(result).toEqual({
      safe: Number.MAX_SAFE_INTEGER,
      unsafe: unsafeValue,
      negative_unsafe: negativeUnsafe,
    });
  });

  it('handles payload buffer expansion beyond initial 64KB', () => {
    internPool.addStrings([{ id: 1, value: 'num' }]);
    structStore.addFieldPathSet({
      id: 1,
      fieldPathStringIds: [1],
    });

    // Each struct with a double adds tag (1) + alignment (7) + double (8) = 16 bytes.
    // 5000 structs = ~80KB, which exceeds initial 64KB (65536 bytes) payloadBuffer.
    for (let i = 1; i <= 5000; i++) {
      structStore.addStruct({
        id: i,
        fieldPathSetId: 1,
        values: [{ kind: StructValueKind.Double, value: i * 0.5 }],
      });
    }

    expect(structStore.count).toBe(5000);
    expect(structStore.getStruct(1)).toEqual({ num: 0.5 });
    expect(structStore.getStruct(2500)).toEqual({ num: 1250 });
    expect(structStore.getStruct(5000)).toEqual({ num: 2500 });

    structStore.shrinkToFit();
    // Idempotent
    structStore.shrinkToFit();
    expect(structStore.count).toBe(5000);
    expect(structStore.getStruct(5000)).toEqual({ num: 2500 });
  });

  it('handles empty struct with no values', () => {
    structStore.addFieldPathSet({
      id: 1,
      fieldPathStringIds: [],
    });
    structStore.addStruct({
      id: 1,
      fieldPathSetId: 1,
      values: [],
    });

    expect(structStore.count).toBe(1);
    expect(structStore.getStruct(1)).toEqual({});
  });

  it('handles fieldPathsBuffer dynamic expansion beyond initial capacity', () => {
    const stringIds: number[] = [];
    for (let i = 1; i <= 1500; i++) {
      internPool.addString({ id: i, value: `key_${i}` });
      stringIds.push(i);
    }

    // 1500 fields exceeds INITIAL_CAPACITY of 1024
    structStore.addFieldPathSet({
      id: 1,
      fieldPathStringIds: stringIds,
    });

    const values = stringIds.map((_, idx) => ({
      kind: StructValueKind.Int64 as const,
      value: BigInt(idx),
    }));

    structStore.addStruct({
      id: 1,
      fieldPathSetId: 1,
      values,
    });

    structStore.shrinkToFit();

    const result = structStore.getStruct(1);
    expect(result).not.toBeNull();
    expect(result!['key_1']).toBe(0);
    expect(result!['key_1500']).toBe(1499);
  });

  it('handles empty lists and nested lists in struct values', () => {
    internPool.addStrings([
      { id: 1, value: 'emptyList' },
      { id: 2, value: 'nestedList' },
    ]);

    structStore.addFieldPathSet({
      id: 1,
      fieldPathStringIds: [1, 2],
    });

    structStore.addStruct({
      id: 1,
      fieldPathSetId: 1,
      values: [
        { kind: StructValueKind.List, values: [] },
        {
          kind: StructValueKind.List,
          values: [
            {
              kind: StructValueKind.List,
              values: [{ kind: StructValueKind.String, stringId: 1 }],
            },
          ],
        },
      ],
    });

    const result = structStore.getStruct(1);
    expect(result).toEqual({
      emptyList: [],
      nestedList: [['emptyList']],
    });
  });

  it('safely ignores field paths attempting prototype pollution or containing empty segments', () => {
    internPool.addStrings([
      { id: 1, value: '__proto__\0polluted' },
      { id: 2, value: 'constructor\0prototype\0polluted' },
      { id: 3, value: 'safe\0prototype\0polluted' },
      { id: 4, value: 'valid\0key' },
      { id: 5, value: 'valid\0' }, // trailing empty segment
      { id: 6, value: '\0empty_leading' }, // leading empty segment
      { id: 10, value: 'attack_value' },
      { id: 11, value: 'legit_value' },
    ]);

    structStore.addFieldPathSet({
      id: 1,
      fieldPathStringIds: [1, 2, 3, 4, 5, 6],
    });

    structStore.addStruct({
      id: 1,
      fieldPathSetId: 1,
      values: [
        { kind: StructValueKind.String, stringId: 10 },
        { kind: StructValueKind.String, stringId: 10 },
        { kind: StructValueKind.String, stringId: 10 },
        { kind: StructValueKind.String, stringId: 11 },
        { kind: StructValueKind.String, stringId: 10 },
        { kind: StructValueKind.String, stringId: 10 },
      ],
    });

    const result = structStore.getStruct(1);
    expect(result).toEqual({
      valid: {
        key: 'legit_value',
      },
    });

    // Ensure global Object.prototype was not polluted
    expect(
      (Object.prototype as Record<string, unknown>)['polluted'],
    ).toBeUndefined();
  });
});
