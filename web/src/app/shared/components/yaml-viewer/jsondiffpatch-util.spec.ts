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

import {
  isAddedDelta,
  isModifiedDelta,
  isDeletedDelta,
  isMovedDelta,
} from './jsondiffpatch-util';

describe('jsondiffpatch-util', () => {
  describe('isAddedDelta', () => {
    it('should return true for addition delta', () => {
      expect(isAddedDelta(['new'])).toBe(true);
    });

    it('should return false for other deltas', () => {
      expect(isAddedDelta(undefined)).toBe(false);
      expect(isAddedDelta(['old', 'new'])).toBe(false);
      expect(isAddedDelta(['old', 0, 0])).toBe(false);
    });
  });

  describe('isModifiedDelta', () => {
    it('should return true for modification delta', () => {
      expect(isModifiedDelta(['old', 'new'])).toBe(true);
    });

    it('should return false for other deltas', () => {
      expect(isModifiedDelta(undefined)).toBe(false);
      expect(isModifiedDelta(['new'])).toBe(false);
      expect(isModifiedDelta(['old', 0, 0])).toBe(false);
    });
  });

  describe('isDeletedDelta', () => {
    it('should return true for deletion delta', () => {
      expect(isDeletedDelta(['old', 0, 0])).toBe(true);
    });

    it('should return false for other deltas', () => {
      expect(isDeletedDelta(undefined)).toBe(false);
      expect(isDeletedDelta(['new'])).toBe(false);
      expect(isDeletedDelta(['old', 'new'])).toBe(false);
      expect(isDeletedDelta(['old', 1, 3])).toBe(false);
    });
  });

  describe('isMovedDelta', () => {
    it('should return true for move delta', () => {
      expect(isMovedDelta(['old', 1, 3])).toBe(true);
    });

    it('should return false for other deltas', () => {
      expect(isMovedDelta(undefined)).toBe(false);
      expect(isMovedDelta(['new'])).toBe(false);
      expect(isMovedDelta(['old', 0, 0])).toBe(false);
    });
  });
});
