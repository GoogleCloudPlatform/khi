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
    store = InternPoolStore.initialize();
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
    const smallStore = InternPoolStore.initialize(
      [
        { id: 1, value: 'abcdefgh' }, // 8 bytes -> fits in 1st buffer
        { id: 2, value: 'ijklmnop' }, // 8 bytes -> exceeds remaining 2 bytes, goes to 2nd buffer
        { id: 3, value: 'qrstuvwxyz12345' }, // 15 bytes -> exceeds 10 bytes maxBufferSize, allocated standalone
        { id: 4, value: 'abc' }, // 3 bytes -> fits in next buffer
      ],
      10,
    );

    expect(smallStore.getString(1)).toBe('abcdefgh');
    expect(smallStore.getString(2)).toBe('ijklmnop');
    expect(smallStore.getString(3)).toBe('qrstuvwxyz12345');
    expect(smallStore.getString(4)).toBe('abc');
  });

  it('should resize metadata TypedArrays when string ID is large', () => {
    store.addStrings([{ id: 2000, value: 'large-id-string' }]);

    expect(store.getString(2000)).toBe('large-id-string');
  });

  describe('ArrayBuffer allocation', () => {
    it('should allocate ArrayBuffer and perform operations successfully', () => {
      const fallbackStore = InternPoolStore.initialize();
      fallbackStore.addStrings([
        { id: 10, value: 'fallback-string-1' },
        { id: 20, value: 'fallback-string-2' },
      ]);

      expect(fallbackStore.getString(10)).toBe('fallback-string-1');
      expect(fallbackStore.getString(20)).toBe('fallback-string-2');
    });
  });
});
