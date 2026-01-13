/**
 * Copyright 2025 Google LLC
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

import { bisectLeft, bisectRight } from './misc-util';

describe('misc-util', () => {
  describe('bisectLeft', () => {
    it('should return 0 for empty array', () => {
      expect(bisectLeft([], 1)).toBe(0);
    });

    it('should return 0 if value is smaller than all elements', () => {
      expect(bisectLeft([1, 2, 3], 0)).toBe(0);
    });

    it('should return length if value is larger than all elements', () => {
      expect(bisectLeft([1, 2, 3], 4)).toBe(3);
    });

    it('should return index of first occurrence if value exists', () => {
      expect(bisectLeft([1, 2, 2, 2, 3], 2)).toBe(1);
    });

    it('should return insertion point if value does not exist', () => {
      expect(bisectLeft([1, 3, 5], 2)).toBe(1);
      expect(bisectLeft([1, 3, 5], 4)).toBe(2);
    });

    it('should respect custom lo and hi', () => {
      // search in [3, 5] (indices 1 to 3)
      expect(bisectLeft([1, 3, 5, 7], 4, 1, 3)).toBe(2);
      // search in [1, 3] (indices 0 to 2)
      expect(bisectLeft([1, 3, 5, 7], 2, 0, 2)).toBe(1);
    });
  });

  describe('bisectRight', () => {
    it('should return 0 for empty array', () => {
      expect(bisectRight([], 1)).toBe(0);
    });

    it('should return 0 if value is smaller than all elements', () => {
      expect(bisectRight([1, 2, 3], 0)).toBe(0);
    });

    it('should return length if value is larger than all elements', () => {
      expect(bisectRight([1, 2, 3], 4)).toBe(3);
    });

    it('should return index after last occurrence if value exists', () => {
      expect(bisectRight([1, 2, 2, 2, 3], 2)).toBe(4);
    });

    it('should return insertion point if value does not exist', () => {
      expect(bisectRight([1, 3, 5], 2)).toBe(1);
      expect(bisectRight([1, 3, 5], 4)).toBe(2);
    });

    it('should respect custom lo and hi', () => {
      // search in [3, 5] (indices 1 to 3)
      expect(bisectRight([1, 3, 5, 7], 4, 1, 3)).toBe(2);
      // search in [1, 3] (indices 0 to 2)
      expect(bisectRight([1, 3, 5, 7], 2, 0, 2)).toBe(1);
    });
  });
});
