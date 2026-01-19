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

import { HorizontalScrollCalculator } from './horizontal-scroll-calculator';

describe('HorizontalScrollCalculator', () => {
  describe('totalWidth', () => {
    it('returns total needed width for rendering without offset & margin', () => {
      const calculator = new HorizontalScrollCalculator(0, 1000, 0);
      // 1000ms & 1px/ms => 1000 px
      expect(calculator.totalWidth(1)).toBeCloseTo(1000);
      expect(calculator.totalWidth(3)).toBeCloseTo(3000);
    });

    it('returns total needed width for rendering with offset', () => {
      const calculator = new HorizontalScrollCalculator(1000, 2000, 0);
      expect(calculator.totalWidth(1)).toBeCloseTo(1000);
      expect(calculator.totalWidth(3)).toBeCloseTo(3000);
    });

    it('returns total needed width for rendering with offset & margin', () => {
      const calculator = new HorizontalScrollCalculator(1000, 2000, 300);
      // 1000ms & 1px/ms => 1000 px
      // 300px margin on both sides
      expect(calculator.totalWidth(1)).toBeCloseTo(1000 + 300 * 2);
      expect(calculator.totalWidth(3)).toBeCloseTo(3000 + 300 * 2);
    });
  });

  describe('totalRenderWidth', () => {
    it('returns viewport width + extra offset width', () => {
      const calculator = new HorizontalScrollCalculator(0, 1000, 300);
      expect(calculator.totalRenderWidth(1000)).toBeCloseTo(1000 + 300 * 2);
    });
  });

  describe('leftDrawAreaTimeMS', () => {
    it('returns aligned time based on tickTimeMS', () => {
      const calculator = new HorizontalScrollCalculator(0, 1000, 300);
      // tickTimeMS = 100
      // extraOffsetTimeMS (at 1px/ms) = 300ms
      // viewportLeftTimeMS = 550
      // (550 - 300) / 100 = 2.5 -> floor -> 2 -> 200
      expect(calculator.leftDrawAreaTimeMS(550, 100, 1)).toBeCloseTo(200);
    });

    it('returns aligned time based on tickTimeMS with different pixelsPerMs', () => {
      const calculator = new HorizontalScrollCalculator(0, 1000, 300);
      // tickTimeMS = 100
      // extraOffsetTimeMS (at 10px/ms) = 300/10 = 30ms
      // viewportLeftTimeMS = 155
      // (155 - 30) / 100 = 1.25 -> floor -> 1 -> 100
      expect(calculator.leftDrawAreaTimeMS(155, 100, 10)).toBeCloseTo(100);
    });
  });

  describe('calculateZoomScrollLeft', () => {
    // 0~1000ms. extraOffset 300px.
    const calculator = new HorizontalScrollCalculator(0, 1000, 300);

    it('calculates scrollLeft to keep mouse position static in time', () => {
      const currentPixelsPerMs = 1;
      const currentViewportLeftTime = 0;
      const viewportWidth = 500;
      const mouseX = 250; // Center

      // Zoom in 2x
      const newPixelsPerMs = 2;

      const newScrollLeft = calculator.calculateZoomScrollLeft(
        currentPixelsPerMs,
        currentViewportLeftTime,
        newPixelsPerMs,
        viewportWidth,
        mouseX,
      );

      // Verify time at mouse is preserved.
      // New time at mouse = scrollTime(newScrollLeft) + mouseX / newPPS
      // Or simply: leftDrawAreaTime(scrollTime(...)) ... wait.
      // Frame uses: vpLT = scrollTime(scrollLeft).
      // So let's calculate newVpLT.
      const newVpLT = calculator.scrollTime(newScrollLeft, newPixelsPerMs);
      const newTimeAtMouse = newVpLT + mouseX / newPixelsPerMs;

      // Original time at mouse:
      // vpLT + mouseX / currentPPS = 0 + 250 / 1 = 250.
      expect(newTimeAtMouse).toBeCloseTo(250, 0);
    });
  });

  describe('leftDrawAreaOffset', () => {
    it('returns offset in pixels', () => {
      const calculator = new HorizontalScrollCalculator(0, 1000, 300);
      // tickTimeMS = 100
      // pixelsPerMs = 1
      // extraOffsetTimeMS = 300
      // minScrollableTimeMS = 0 - 300 = -300
      // viewportLeftTimeMS = 550
      // leftDrawAreaTimeMS = 200 (calculated above)
      // (200 - (-300)) * 1 = 500
      expect(calculator.leftDrawAreaOffset(550, 100, 1)).toBeCloseTo(500);
    });

    it('returns offset in pixels with different pixelsPerMs', () => {
      const calculator = new HorizontalScrollCalculator(0, 1000, 300);
      // tickTimeMS = 100
      // pixelsPerMs = 10
      // extraOffsetTimeMS = 30
      // minScrollableTimeMS = 0 - 30 = -30
      // viewportLeftTimeMS = 155
      // leftDrawAreaTimeMS = 100 (calculated above)
      // (100 - (-30)) * 10 = 130 * 10 = 1300
      expect(calculator.leftDrawAreaOffset(155, 100, 10)).toBeCloseTo(1300);
    });
  });
});
