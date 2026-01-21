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
    it('calculates correct new scroll position when zooming in without extra offset', () => {
      const calculator = new HorizontalScrollCalculator(1000, 2000, 0);
      const currentPpm = 1;
      const newPpm = 2;
      const mousePos = 100;
      const currentScrollLeft = 0;

      // Initial state:
      // minScrollableTime = 1000
      // viewportLeftTime = 1000
      // mouseTime = 1000 + 100/1 = 1100

      // Expected state:
      // minScrollableTime = 1000
      // mouseTime = 1100
      // newViewportLeftTime = 1100 - 100/2 = 1050
      // newScrollLeft = (1050 - 1000) * 2 = 100

      const newScrollLeft = calculator.calculateZoomScrollLeft(
        currentPpm,
        newPpm,
        mousePos,
        currentScrollLeft,
      );
      expect(newScrollLeft).toBeCloseTo(100);
    });

    it('calculates correct new scroll position when zooming in with extra offset', () => {
      const calculator = new HorizontalScrollCalculator(1000, 2000, 300);
      const currentPpm = 1;
      const newPpm = 2;
      const mousePos = 100;
      // Let's assume scrolled a bit
      // minScrollableTime(1) = 700
      // scrollLeft=100 => viewportLeftTime = 700 + 100 = 800
      const currentScrollLeft = 100;

      // mouseTime = 800 + 100/1 = 900

      // New state:
      // minScrollableTime(2) = 1000 - 300/2 = 850
      // mouseTime should be 900
      // newViewportLeftTime = 900 - 100/2 = 850
      // newScrollLeft = (850 - 850) * 2 = 0

      const newScrollLeft = calculator.calculateZoomScrollLeft(
        currentPpm,
        newPpm,
        mousePos,
        currentScrollLeft,
      );
      expect(newScrollLeft).toBeCloseTo(0);
    });

    it('keeps the time at mouse position constant', () => {
      const calculator = new HorizontalScrollCalculator(1000, 5000, 300);
      const currentPpm = 2;
      const newPpm = 5.5; // Arbitrary zoom
      const mousePos = 453; // Arbitrary mouse position
      const currentScrollLeft = 1500; // Arbitrary scroll

      const prevTimeAtMouse =
        calculator.scrollToViewportLeftTime(currentScrollLeft, currentPpm) +
        mousePos / currentPpm;

      const newScrollLeft = calculator.calculateZoomScrollLeft(
        currentPpm,
        newPpm,
        mousePos,
        currentScrollLeft,
      );

      const newTimeAtMouse =
        calculator.scrollToViewportLeftTime(newScrollLeft, newPpm) +
        mousePos / newPpm;

      expect(newTimeAtMouse).toBeCloseTo(prevTimeAtMouse);
    });
  });
});
