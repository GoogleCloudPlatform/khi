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

import { align } from 'src/app/common/memory-util';

describe('memory-util', () => {
  describe('align', () => {
    it('returns 0 when offset is 0', () => {
      expect(align(0, 4)).toBe(0);
      expect(align(0, 8)).toBe(0);
    });

    it('returns same offset when already aligned', () => {
      expect(align(4, 4)).toBe(4);
      expect(align(8, 4)).toBe(8);
      expect(align(8, 8)).toBe(8);
      expect(align(16, 8)).toBe(16);
    });

    it('aligns unaligned offset to next boundary', () => {
      expect(align(1, 4)).toBe(4);
      expect(align(2, 4)).toBe(4);
      expect(align(3, 4)).toBe(4);
      expect(align(5, 4)).toBe(8);
      expect(align(6, 4)).toBe(8);
      expect(align(7, 4)).toBe(8);
      expect(align(10, 4)).toBe(12);
      expect(align(10, 8)).toBe(16);
    });
  });
});
