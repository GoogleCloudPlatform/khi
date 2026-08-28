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

describe('InternPoolStore', () => {
  let store: InternPoolStore;

  beforeEach(() => {
    store = InternPoolStore.create();
  });

  it('should add and get strings from the pool', () => {
    store.addStrings([
      { id: 1, value: 'foo' },
      { id: 3, value: 'bar' },
    ]);

    expect(store.getString(1)).toBe('foo');
    expect(store.getString(3)).toBe('bar');
  });

  it('should throw an error if string ID is missing', () => {
    expect(() => store.getString(999)).toThrowError(
      'String ID 999 not found in pool',
    );
  });

  it('should split buffer if the string size exceeds maxBufferSize', () => {
    const smallStore = InternPoolStore.create(4, 10);
    smallStore.addStrings([
      { id: 1, value: 'abcdefgh' }, // 8 bytes -> fits in 1st buffer
      { id: 2, value: 'ijklmnop' }, // 8 bytes -> exceeds remaining 2 bytes, goes to 2nd buffer
      { id: 3, value: 'qrstuvwxyz12345' }, // 15 bytes -> exceeds 10 bytes maxBufferSize, allocated standalone
      { id: 4, value: 'abc' }, // 3 bytes -> fits in next buffer
    ]);
    smallStore.shrinkToFit();

    expect(smallStore.getString(1)).toBe('abcdefgh');
    expect(smallStore.getString(2)).toBe('ijklmnop');
    expect(smallStore.getString(3)).toBe('qrstuvwxyz12345');
    expect(smallStore.getString(4)).toBe('abc');
  });

  it('should resize metadata TypedArrays when string ID is large', () => {
    store.addStrings([{ id: 2000, value: 'large-id-string' }]);

    expect(store.getString(2000)).toBe('large-id-string');
  });

  it('should correctly encode and decode multi-byte UTF-8 strings including emojis', () => {
    store.addStrings([
      { id: 1, value: 'こんにちは世界' },
      { id: 2, value: 'Kubernetes 🚀 クラスタ' },
      { id: 3, value: '' },
    ]);

    expect(store.getString(1)).toBe('こんにちは世界');
    expect(store.getString(2)).toBe('Kubernetes 🚀 クラスタ');
    expect(store.getString(3)).toBe('');
  });

  describe('create and dynamic methods', () => {
    it('should create an empty store and allow adding strings with addString', () => {
      const emptyStore = InternPoolStore.create();
      expect(emptyStore.count).toBe(0);
      expect(emptyStore.maxStringId).toBe(0);

      emptyStore.addString({ id: 1, value: 'hello' });
      emptyStore.addString({ id: 5, value: 'world' });

      expect(emptyStore.count).toBe(2);
      expect(emptyStore.maxStringId).toBe(5);
      expect(emptyStore.getString(1)).toBe('hello');
      expect(emptyStore.getString(5)).toBe('world');
    });

    it('should handle multiple buffer expansions starting from capacity 1', () => {
      const dynamicStore = InternPoolStore.create(1);
      for (let i = 1; i <= 50; i++) {
        dynamicStore.addString({ id: i, value: `str-${i}` });
      }

      expect(dynamicStore.count).toBe(50);
      expect(dynamicStore.maxStringId).toBe(50);
      for (let i = 1; i <= 50; i++) {
        expect(dynamicStore.getString(i)).toBe(`str-${i}`);
      }
    });

    it('should not double-count duplicate additions for the same string ID', () => {
      const dynamicStore = InternPoolStore.create();
      dynamicStore.addString({ id: 1, value: 'v1' });
      dynamicStore.addString({ id: 1, value: 'v2' });

      expect(dynamicStore.count).toBe(1);
      expect(dynamicStore.getString(1)).toBe('v2');
    });

    it('should shrink metadataBuffer to fit string count', () => {
      const emptyStore = InternPoolStore.create(2048);
      emptyStore.addString({ id: 10, value: 'test' });
      expect(emptyStore.count).toBe(1);
      expect(emptyStore.maxStringId).toBe(10);

      emptyStore.shrinkToFit();

      expect(emptyStore.getString(10)).toBe('test');
      expect(() => emptyStore.getString(11)).toThrowError(
        'String ID 11 not found in pool',
      );
    });
  });
});
