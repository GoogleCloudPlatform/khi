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

import { Delta } from 'jsondiffpatch';
import {
  buildMergeTree,
  shouldHighlightEntireValue,
  ValueType,
} from 'src/app/shared/components/yaml-viewer/diff-util';
import { DiffStatus } from 'src/app/shared/components/yaml-viewer/lcs';

describe('shouldHighlightEntireValue', () => {
  it('should return true if either value is null', () => {
    expect(shouldHighlightEntireValue(null, 'value')).toBeTrue();
    expect(shouldHighlightEntireValue('value', null)).toBeTrue();
    expect(shouldHighlightEntireValue(null, null)).toBeTrue();
  });

  it('should return true if either value is boolean', () => {
    expect(shouldHighlightEntireValue(true, 'value')).toBeTrue();
    expect(shouldHighlightEntireValue('value', false)).toBeTrue();
    expect(shouldHighlightEntireValue(true, false)).toBeTrue();
  });

  it('should return true if either value is a boolean-like string', () => {
    expect(shouldHighlightEntireValue('true', 'value')).toBeTrue();
    expect(shouldHighlightEntireValue('value', 'FALSE')).toBeTrue();
    expect(shouldHighlightEntireValue('yes', 'no')).toBeTrue();
  });

  it('should return false for other types or normal strings', () => {
    expect(shouldHighlightEntireValue('hello', 'world')).toBeFalse();
    expect(shouldHighlightEntireValue(123, 456)).toBeFalse();
    expect(shouldHighlightEntireValue({ a: 1 }, { b: 2 })).toBeFalse();
  });
});

describe('buildMergeTree', () => {
  describe('Scalars', () => {
    it('should handle identical scalars (Unchanged)', () => {
      const node = buildMergeTree(10, 10, undefined, '', 'key');
      expect(node).toEqual({
        key: 'key',
        path: 'key',
        status: DiffStatus.Unchanged,
        valueType: ValueType.Number,
        oldValue: 10,
        newValue: 10,
      });
    });

    it('should handle different scalars (Modified)', () => {
      const node = buildMergeTree(10, 20, undefined, '', 'key');
      expect(node).toEqual({
        key: 'key',
        path: 'key',
        status: DiffStatus.Modified,
        valueType: ValueType.Number,
        oldValue: 10,
        newValue: 20,
      });
    });

    it('should handle added scalar delta', () => {
      const delta = [10] as unknown as Delta;
      const node = buildMergeTree(undefined, 10, delta, '', 'key');
      expect(node).toEqual({
        key: 'key',
        path: 'key',
        status: DiffStatus.Added,
        valueType: ValueType.Number,
        newValue: 10,
      });
    });

    it('should handle modified scalar delta', () => {
      const delta = [10, 20] as unknown as Delta;
      const node = buildMergeTree(10, 20, delta, '', 'key');
      expect(node).toEqual({
        key: 'key',
        path: 'key',
        status: DiffStatus.Modified,
        valueType: ValueType.Number,
        oldValue: 10,
        newValue: 20,
      });
    });

    it('should handle deleted scalar delta', () => {
      const delta = [10, 0, 0] as unknown as Delta;
      const node = buildMergeTree(10, undefined, delta, '', 'key');
      expect(node).toEqual({
        key: 'key',
        path: 'key',
        status: DiffStatus.Deleted,
        valueType: ValueType.Number,
        oldValue: 10,
      });
    });

    it('should handle text diff delta for scalar values', () => {
      const delta = ['@@ -1,3 +1,3 @@\n-cat\n+cut\n', 0, 2] as unknown as Delta;
      const node = buildMergeTree('cat', 'cut', delta, '', 'key');
      expect(node).toEqual({
        key: 'key',
        path: 'key',
        status: DiffStatus.Modified,
        valueType: ValueType.String,
        oldValue: 'cat',
        newValue: 'cut',
      });
    });
  });

  describe('Objects', () => {
    it('should handle empty objects', () => {
      const node = buildMergeTree({}, {}, undefined, '', 'key');
      expect(node).toEqual({
        key: 'key',
        path: 'key',
        status: DiffStatus.Unchanged,
        valueType: ValueType.Object,
        oldValue: {},
        newValue: {},
        children: [],
      });
    });

    it('should handle identical objects', () => {
      const node = buildMergeTree({ a: 1 }, { a: 1 }, undefined, '', 'key');
      expect(node.status).toBe(DiffStatus.Unchanged);
      expect(node.children?.length).toBe(1);
      expect(node.children?.[0]).toEqual({
        key: 'a',
        path: 'key.a',
        status: DiffStatus.Unchanged,
        valueType: ValueType.Number,
        oldValue: 1,
        newValue: 1,
      });
    });

    it('should handle added key', () => {
      const delta = { b: [2] } as unknown as Delta;
      const node = buildMergeTree({ a: 1 }, { a: 1, b: 2 }, delta, '', 'key');
      expect(node.status).toBe(DiffStatus.Modified);
      expect(node.children).toEqual([
        {
          key: 'a',
          path: 'key.a',
          status: DiffStatus.Unchanged,
          valueType: ValueType.Number,
          oldValue: 1,
          newValue: 1,
        },
        {
          key: 'b',
          path: 'key.b',
          status: DiffStatus.Added,
          valueType: ValueType.Number,
          newValue: 2,
        },
      ]);
    });

    it('should handle deleted key', () => {
      const delta = { b: [2, 0, 0] } as unknown as Delta;
      const node = buildMergeTree({ a: 1, b: 2 }, { a: 1 }, delta, '', 'key');
      expect(node.status).toBe(DiffStatus.Modified);
      expect(node.children).toEqual([
        {
          key: 'a',
          path: 'key.a',
          status: DiffStatus.Unchanged,
          valueType: ValueType.Number,
          oldValue: 1,
          newValue: 1,
        },
        {
          key: 'b',
          path: 'key.b',
          status: DiffStatus.Deleted,
          valueType: ValueType.Number,
          oldValue: 2,
        },
      ]);
    });

    it('should handle empty/no-op delta on object', () => {
      const delta = {} as unknown as Delta;
      const node = buildMergeTree({ a: 1 }, { a: 1 }, delta, '', 'key');
      expect(node.status).toBe(DiffStatus.Unchanged);
    });

    it('should mark children as Added when parent is added (no delta)', () => {
      const node = buildMergeTree(undefined, { a: 1 }, undefined, '', 'key');
      expect(node).toEqual({
        key: 'key',
        path: 'key',
        status: DiffStatus.Unchanged,
        valueType: ValueType.Object,
        children: [
          {
            key: 'a',
            path: 'key.a',
            status: DiffStatus.Unchanged,
            valueType: ValueType.Number,
            newValue: 1,
            oldValue: undefined,
          },
        ],
      });
    });

    it('should mark children as Deleted when parent is deleted (no delta)', () => {
      const node = buildMergeTree({ a: 1 }, undefined, undefined, '', 'key');
      expect(node).toEqual({
        key: 'key',
        path: 'key',
        status: DiffStatus.Unchanged,
        valueType: ValueType.Object,
        children: [
          {
            key: 'a',
            path: 'key.a',
            status: DiffStatus.Unchanged,
            valueType: ValueType.Number,
            oldValue: 1,
            newValue: undefined,
          },
        ],
      });
    });
  });

  describe('Arrays', () => {
    it('should handle empty arrays', () => {
      const node = buildMergeTree([], [], undefined, '', 'key');
      expect(node).toEqual({
        key: 'key',
        path: 'key',
        status: DiffStatus.Unchanged,
        valueType: ValueType.Array,
        oldValue: [],
        newValue: [],
        children: [],
      });
    });

    it('should handle identical arrays', () => {
      const node = buildMergeTree([1, 2], [1, 2], undefined, '', 'key');
      expect(node.status).toBe(DiffStatus.Unchanged);
      expect(node.children?.length).toBe(2);
      expect(node.children?.[0].status).toBe(DiffStatus.Unchanged);
      expect(node.children?.[1].status).toBe(DiffStatus.Unchanged);
    });

    it('should handle added element', () => {
      const delta = { 1: [2], _t: 'a' } as unknown as Delta;
      const node = buildMergeTree([1], [1, 2], delta, '', 'key');
      expect(
        node.children?.map((c) => ({ key: c.key, status: c.status })),
      ).toEqual([
        { key: '[0]', status: DiffStatus.Unchanged },
        { key: '[1]', status: DiffStatus.Added },
      ]);
    });

    it('should handle deleted element', () => {
      const delta = { _1: [2, 0, 0], _t: 'a' } as unknown as Delta;
      const node = buildMergeTree([1, 2], [1], delta, '', 'key');
      expect(
        node.children?.map((c) => ({ key: c.key, status: c.status })),
      ).toEqual([
        { key: '[0]', status: DiffStatus.Unchanged },
        { key: '[1]', status: DiffStatus.Deleted },
      ]);
    });

    it('should handle moved elements (MovedIn / MovedOut)', () => {
      // Move index 0 to index 1
      const delta = {
        _0: ['', 1, 3],
        _t: 'a',
      } as unknown as Delta;
      const node = buildMergeTree([1, 2], [2, 1], delta, '', 'key');

      expect(node.children).toEqual([
        {
          key: '[0]',
          path: 'key.[0]',
          status: DiffStatus.MovedOut,
          valueType: ValueType.Number,
          oldValue: 1,
          newValue: 1,
          movedTo: '[1]',
        },
        {
          key: '[0]',
          path: 'key.[0]',
          status: DiffStatus.Unchanged,
          valueType: ValueType.Number,
          oldValue: 2,
          newValue: 2,
        },
        {
          key: '[1]',
          path: 'key.[1]',
          status: DiffStatus.MovedIn,
          valueType: ValueType.Number,
          oldValue: 1,
          newValue: 1,
          movedFrom: '[0]',
        },
      ]);
    });

    it('should pair by name hash', () => {
      const left = [{ name: 'foo', val: 1 }];
      const right = [{ name: 'foo', val: 2 }];
      const delta = {
        0: { val: [1, 2] },
        _t: 'a',
      } as unknown as Delta;
      const node = buildMergeTree(left, right, delta, '', 'key');
      expect(node.children?.[0].status).toBe(DiffStatus.Modified);
      expect(node.children?.[0].key).toBe('[0]');
    });
  });

  describe('Structural & Type Changes', () => {
    it('should handle scalar to object transition', () => {
      const delta = [42, { value: 42 }] as unknown as Delta;
      const node = buildMergeTree(42, { value: 42 }, delta, '', 'key');
      expect(node.status).toBe(DiffStatus.Modified);
      expect(node.valueType).toBe(ValueType.Object);
      expect(node.oldValueType).toBe(ValueType.Number);
      expect(node.oldValue).toBe(42);
      expect(node.children).toEqual([
        {
          key: 'value',
          path: 'key.value',
          status: DiffStatus.Added,
          valueType: ValueType.Number,
          oldValue: undefined,
          newValue: 42,
        },
      ]);
    });

    it('should handle object to scalar transition', () => {
      const delta = [{ value: 42 }, 42] as unknown as Delta;
      const node = buildMergeTree({ value: 42 }, 42, delta, '', 'key');
      expect(node.status).toBe(DiffStatus.Modified);
      expect(node.valueType).toBe(ValueType.Number);
      expect(node.oldValueType).toBe(ValueType.Object);
      expect(node.newValue).toBe(42);
      expect(node.children).toEqual([
        {
          key: 'value',
          path: 'key.value',
          status: DiffStatus.Deleted,
          valueType: ValueType.Number,
          oldValue: 42,
          newValue: undefined,
        },
      ]);
    });

    it('should handle null transitions', () => {
      const delta = [null, { a: 1 }] as unknown as Delta;
      const node = buildMergeTree(null, { a: 1 }, delta, '', 'key');
      expect(node.status).toBe(DiffStatus.Modified);
      expect(node.valueType).toBe(ValueType.Object);
      expect(node.oldValueType).toBe(ValueType.Null);
      expect(node.oldValue).toBeNull();
      expect(node.children).toEqual([
        {
          key: 'a',
          path: 'key.a',
          status: DiffStatus.Added,
          valueType: ValueType.Number,
          oldValue: undefined,
          newValue: 1,
        },
      ]);
    });
  });

  describe('Deep Added/Deleted Trees', () => {
    it('should set status for deep added objects', () => {
      const delta = [{ a: { b: 1 } }] as unknown as Delta;
      const node = buildMergeTree(undefined, { a: { b: 1 } }, delta, '', 'key');
      expect(node.status).toBe(DiffStatus.Added);
      expect(node.children?.[0].status).toBe(DiffStatus.Unchanged);
      expect(node.children?.[0].children?.[0].status).toBe(
        DiffStatus.Unchanged,
      );
    });
  });

  describe('Array Edge Cases', () => {
    it('should handle duplicate primitives without incorrect moves', () => {
      // Left: [1, 1], Right: [1]
      const delta = { _1: [1, 0, 0], _t: 'a' } as unknown as Delta;
      const node = buildMergeTree([1, 1], [1], delta, '', 'key');
      expect(node.children?.map((c) => c.status)).toEqual([
        DiffStatus.Unchanged,
        DiffStatus.Deleted,
      ]);
    });

    it('should fallback to index-based pairing for objects without name/type', () => {
      const left = [{ x: 1 }];
      const right = [{ x: 2 }];
      const delta = {
        0: { x: [1, 2] },
        _t: 'a',
      } as unknown as Delta;
      const node = buildMergeTree(left, right, delta, '', 'key');
      expect(node.children?.[0].status).toBe(DiffStatus.Modified);
    });
  });
});
