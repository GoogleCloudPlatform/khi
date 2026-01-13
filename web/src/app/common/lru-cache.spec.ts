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

import { LRUCache } from './lru-cache';

describe('LRUCache', () => {
  it('should throw error if capacity is less than or equal to 0', () => {
    expect(() => new LRUCache(0)).toThrowError(
      'Capacity must be greater than 0.',
    );
    expect(() => new LRUCache(-1)).toThrowError(
      'Capacity must be greater than 0.',
    );
  });

  it('should put and get values', () => {
    const cache = new LRUCache<string, number>(3);
    cache.put('a', 1);
    cache.put('b', 2);
    expect(cache.get('a')).toBe(1);
    expect(cache.get('b')).toBe(2);
    expect(cache.get('c')).toBeUndefined();
  });

  it('should update value and MRU on put of existing key', () => {
    const cache = new LRUCache<string, number>(2);
    cache.put('a', 1);
    cache.put('b', 2);

    // Update 'a', making it MRU. 'b' becomes LRU.
    cache.put('a', 3);
    expect(cache.get('a')).toBe(3);

    // Add 'c'. Should evict 'b' (LRU).
    cache.put('c', 4);
    expect(cache.get('b')).toBeUndefined();
    expect(cache.get('a')).toBe(3);
    expect(cache.get('c')).toBe(4);
  });

  it('should call onDispose when item is evicted', () => {
    const onDispose = jasmine.createSpy('onDispose');
    const cache = new LRUCache<string, number>(2, onDispose);
    cache.put('a', 1);
    cache.put('b', 2);

    cache.put('c', 3);
    expect(onDispose).toHaveBeenCalledWith(1); // 'a' was evicted
    expect(onDispose).toHaveBeenCalledTimes(1);
  });

  it('should call onDispose when item is overwritten with different value', () => {
    const onDispose = jasmine.createSpy('onDispose');
    const cache = new LRUCache<string, number>(2, onDispose);
    cache.put('a', 1);

    cache.put('a', 2);
    expect(onDispose).toHaveBeenCalledWith(1);
    expect(onDispose).toHaveBeenCalledTimes(1);

    cache.put('a', 2);
    expect(onDispose).toHaveBeenCalledTimes(1);
  });

  it('should call onDispose for all items on clear', () => {
    const onDispose = jasmine.createSpy('onDispose');
    const cache = new LRUCache<string, number>(2, onDispose);
    cache.put('a', 1);
    cache.put('b', 2);

    cache.clear();
    expect(onDispose).toHaveBeenCalledWith(1);
    expect(onDispose).toHaveBeenCalledWith(2);
    expect(onDispose).toHaveBeenCalledTimes(2);
    expect(cache.size).toBe(0);
  });

  it('should check existence with has without updating MRU', () => {
    const cache = new LRUCache<string, number>(2);
    cache.put('a', 1);
    cache.put('b', 2);

    expect(cache.has('a')).toBeTrue();
    expect(cache.has('c')).toBeFalse();

    cache.put('c', 3);

    // 'a' should be evicted because 'has' does NOT update MRU
    expect(cache.get('a')).toBeUndefined();
    expect(cache.get('b')).toBe(2);
    expect(cache.get('c')).toBe(3);
  });

  it('should return correct size', () => {
    const cache = new LRUCache<string, number>(2);
    expect(cache.size).toBe(0);
    cache.put('a', 1);
    expect(cache.size).toBe(1);
    cache.put('b', 2);
    expect(cache.size).toBe(2);
    cache.put('c', 3); // Evicts 'a', size remains 2
    expect(cache.size).toBe(2);
    cache.clear();
    expect(cache.size).toBe(0);
  });

  it('should iterate with foreach', () => {
    const cache = new LRUCache<string, number>(3);
    cache.put('a', 1);
    cache.put('b', 2);

    const keys: string[] = [];
    const values: number[] = [];
    cache.forEach((val, key) => {
      keys.push(key);
      values.push(val);
    });

    expect(keys).toEqual(['a', 'b']);
    expect(values).toEqual([1, 2]);
  });
});
