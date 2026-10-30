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
  getTickTimeNs,
} from './human-friendly-tick';

describe('human-friendly-tick', () => {
  describe('getMinTimeSpanForHistogram', () => {
    it('should return the smallest step that fits within maxHistogramSize', () => {
      // 1000ms (1_000_000_000ns) duration, max size 100 -> step 10ms (10_000_000ns)
      // RULER_STEPS_NS has 1ms, 5ms, 10ms, 50ms, 100ms...
      expect(getMinTimeSpanForHistogram(100, 0n, 1_000_000_000n)).toBe(
        10_000_000n,
      );
    });

    it('should return the first step if it fits exactly', () => {
      // 100ms (100_000_000ns) duration, max size 100 -> step 1ms (1_000_000ns)
      expect(getMinTimeSpanForHistogram(100, 0n, 100_000_000n)).toBe(
        1_000_000n,
      );
    });
  });

  describe('getRulerStep', () => {
    it('should return the smallest step larger than minTimeSpanNs', () => {
      // pixelsPerNs = 1e-6 (1 px/ms), minGridWidthPx = 100 -> minTimeSpanNs = 100_000_000
      // RULER_STEPS_NS contains ... 50ms, 100ms, 500ms ...
      // Should pick 100ms (100_000_000n)
      const step = getRulerStep(1e-6, 100);
      expect(step.low).toBe(100_000_000n);
    });

    it('should return the step if minTimeSpanNs exactly matches a step', () => {
      const step = getRulerStep(1e-6, 50); // 50ms (50_000_000n)
      expect(step.low).toBe(50_000_000n);
    });

    it('should return the next larger step if minTimeSpanNs is between steps', () => {
      // minTimeSpanNs = 75_000_000 (between 50ms and 100ms)
      // Should pick 100ms (100_000_000n)
      const step = getRulerStep(1e-6, 75);
      expect(step.low).toBe(100_000_000n);
    });
  });

  describe('getTickTimeNs', () => {
    it('should return the low value of the selected ruler step', () => {
      // Same logic as getRulerStep
      expect(getTickTimeNs(1e-6, 75)).toBe(100_000_000n);
    });
  });
});
