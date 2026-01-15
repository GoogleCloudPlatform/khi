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
  getMinTimeSpanForHistogram,
  getRulerStep,
  getTickTimeMS,
} from './human-friendly-tick';

describe('human-friendly-tick', () => {
  describe('getMinTimeSpanForHistogram', () => {
    it('should return the smallest step that fits within maxHistogramSize', () => {
      // 1000ms duration, max size 100 -> step 10ms (1000/10 = 100 <= 100)
      // RULER_STEPS_MS has 1, 5, 10, 50, 100...
      // 1000 / 1 = 1000 > 100
      // 1000 / 5 = 200 > 100
      // 1000 / 10 = 100 <= 100 -> returns 10
      expect(getMinTimeSpanForHistogram(100, 0, 1000)).toBe(10);
    });

    it('should return the first step if it fits exactly', () => {
      // 100ms duration, max size 100 -> step 1ms (100/1 = 100 <= 100)
      expect(getMinTimeSpanForHistogram(100, 0, 100)).toBe(1);
    });
  });

  describe('getRulerStep', () => {
    it('should return the smallest step larger than minTimeSpanMS', () => {
      // pixelsPerMs = 1, minGridWidthPx = 100 -> minTimeSpanMS = 100
      // RULER_STEPS_MS usually contains ... 50, 100, 500 ...
      // Should pick 100
      const step = getRulerStep(1, 100);
      expect(step.low).toBe(100);
    });

    it('should return the step if minTimeSpanMS exactly matches a step', () => {
      const step = getRulerStep(1, 50); // 50ms
      expect(step.low).toBe(50);
    });

    it('should return the next larger step if minTimeSpanMS is between steps', () => {
      // minTimeSpanMS = 75 (between 50 and 100)
      // Should pick 100
      const step = getRulerStep(1, 75);
      expect(step.low).toBe(100);
    });
  });

  describe('getTickTimeMS', () => {
    it('should return the low value of the selected ruler step', () => {
      // Same logic as getRulerStep
      expect(getTickTimeMS(1, 75)).toBe(100);
    });
  });
});
