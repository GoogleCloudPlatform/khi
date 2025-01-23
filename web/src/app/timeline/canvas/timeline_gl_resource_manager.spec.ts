/**
 * Copyright 2024 Google LLC
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

import { LRULifetimeManager } from './timeline_gl_resource_manager';

describe('LRULifetimeManager', () => {
  it("returns null when it's not hitting the capacity", () => {
    const lru = new LRULifetimeManager<number>(3);
    expect(lru.touch(1)).toBe(null);
    expect(lru.touch(2)).toBe(null);
    expect(lru.touch(3)).toBe(null);
  });

  it("returns null when it's not hitting the capacity and ignore the same element", () => {
    const lru = new LRULifetimeManager<number>(1);
    expect(lru.touch(1)).toBe(null);
    expect(lru.touch(1)).toBe(null);
    expect(lru.touch(1)).toBe(null);
  });

  it("returns the element to remove when it's hitting the capacity", () => {
    const lru = new LRULifetimeManager<number>(2);
    expect(lru.touch(1)).toBe(null);
    expect(lru.touch(1)).toBe(null);
    expect(lru.touch(2)).toBe(null);
    expect(lru.touch(3)).toBe(1);
  });
});
