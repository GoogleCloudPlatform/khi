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
 * A generic Least Recently Used (LRU) cache implementation.
 * * This implementation leverages the native JavaScript `Map` object, which preserves
 * the insertion order of keys. This allows for O(1) time complexity for both
 * read (`get`) and write (`put`) operations.
 * * @typeParam K - The type of the keys held in the cache.
 * @typeParam V - The type of the values held in the cache.
 */
export class LRUCache<K, V> {
  /**
   * The underlying store for cache items.
   * `Map` maintains the order of entries: the first item is the LRU (Least Recently Used),
   * and the last item is the MRU (Most Recently Used).
   */
  private readonly cache: Map<K, V> = new Map<K, V>();

  /**
   * Creates an instance of LRUCache.
   * * @param capacity - The maximum number of entries allowed in the cache. Must be a positive integer.
   * @throws {Error} If the capacity is less than or equal to 0.
   */
  constructor(
    private readonly capacity: number,
    private onDispose?: (value: V) => void,
  ) {
    if (capacity <= 0) {
      throw new Error('Capacity must be greater than 0.');
    }
  }

  /**
   * Retrieves the value associated with the specified key.
   * * If the key exists, the entry is marked as the most recently used (MRU)
   * by moving it to the end of the internal Map.
   * * @param key - The key of the element to return.
   * @returns The value associated with the key, or `undefined` if the key does not exist.
   */
  public get(key: K): V | undefined {
    if (!this.cache.has(key)) {
      return undefined;
    }

    // Retrieve the value before deleting
    const value = this.cache.get(key)!;

    this.cache.delete(key);
    this.cache.set(key, value);

    return value;
  }

  /**
   * Adds or updates an element with the specified key and value.
   * * - If the key already exists, its value is updated and it is marked as most recently used (MRU).
   * - If the key does not exist and the cache is full, the least recently used (LRU) item
   * (the first item in the Map) is evicted before adding the new item.
   * * @param key - The key to add or update.
   * @param value - The value to store.
   */
  public put(key: K, value: V): void {
    if (this.cache.has(key)) {
      // Remove the existing entry so it can be re-inserted at the end
      const oldValue = this.cache.get(key)!;
      if (oldValue !== value && this.onDispose) {
        this.onDispose(oldValue);
      }
      this.cache.delete(key);
    } else if (this.cache.size >= this.capacity) {
      // Evict the least recently used item (the first key in the iterator)
      // Map.prototype.keys() returns an iterator in insertion order.
      const lruKey = this.cache.keys().next().value;

      // Strict check although lruKey should be present if size > 0
      if (lruKey !== undefined) {
        if (this.onDispose) {
          this.onDispose(this.cache.get(lruKey)!);
        }
        this.cache.delete(lruKey);
      }
    }

    // Insert the new key-value pair at the end (MRU position)
    this.cache.set(key, value);
  }

  /**
   * Checks if a key exists in the cache without updating its recentness.
   * * @param key - The key to check.
   * @returns `true` if the key exists, `false` otherwise.
   */
  public has(key: K): boolean {
    return this.cache.has(key);
  }

  /**
   * Returns the number of elements currently in the cache.
   * * @returns The current size of the cache.
   */
  public get size(): number {
    return this.cache.size;
  }

  /**
   * Iterates over the cache entries.
   * @param callback - The callback function to execute for each entry.
   */
  public forEach(callback: (value: V, key: K) => void): void {
    this.cache.forEach(callback);
  }

  /**
   * Removes all elements from the cache.
   */
  public clear(): void {
    if (this.onDispose) {
      this.cache.forEach((v) => this.onDispose?.(v));
    }
    this.cache.clear();
  }
}
