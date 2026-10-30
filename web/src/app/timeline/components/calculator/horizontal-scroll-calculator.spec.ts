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
      const calculator = new HorizontalScrollCalculator(0n, 1000n, 0);
      // 1000ns & 1px/ns => 1000 px
      expect(calculator.totalWidth(1)).toBeCloseTo(1000);
      expect(calculator.totalWidth(3)).toBeCloseTo(3000);
    });

    it('returns total needed width for rendering with offset', () => {
      const calculator = new HorizontalScrollCalculator(1000n, 2000n, 0);
      expect(calculator.totalWidth(1)).toBeCloseTo(1000);
      expect(calculator.totalWidth(3)).toBeCloseTo(3000);
    });

    it('returns total needed width for rendering with offset & margin', () => {
      const calculator = new HorizontalScrollCalculator(1000n, 2000n, 300);
      // 1000ns & 1px/ns => 1000 px
      // 300px margin on both sides
      expect(calculator.totalWidth(1)).toBeCloseTo(1000 + 300 * 2);
      expect(calculator.totalWidth(3)).toBeCloseTo(3000 + 300 * 2);
    });
  });

  describe('totalRenderWidth', () => {
    it('returns viewport width + extra offset width', () => {
      const calculator = new HorizontalScrollCalculator(0n, 1000n, 300);
      expect(calculator.totalRenderWidth(1000)).toBeCloseTo(1000 + 300 * 2);
    });
  });

  describe('leftDrawAreaTimeNs', () => {
    it('returns aligned time based on tickTimeNs', () => {
      const calculator = new HorizontalScrollCalculator(0n, 1000n, 300);
      // tickTimeNs = 100n
      // extraOffsetTimeNs (at 1px/ns) = 300n
      // viewportLeftTimeNs = 550n
      // (550 - 300) / 100 = 2.5 -> floor -> 2 -> 200n
      expect(calculator.leftDrawAreaTimeNs(550n, 100n, 1)).toBe(200n);
    });

    it('returns aligned time based on tickTimeNs with different pixelsPerNs', () => {
      const calculator = new HorizontalScrollCalculator(0n, 1000n, 300);
      // tickTimeNs = 100n
      // extraOffsetTimeNs (at 10px/ns) = 300/10 = 30n
      // viewportLeftTimeNs = 155n
      // (155 - 30) / 100 = 1.25 -> floor -> 1 -> 100n
      expect(calculator.leftDrawAreaTimeNs(155n, 100n, 10)).toBe(100n);
    });
  });

  describe('calculateZoomScrollLeft', () => {
    it('calculates correct new scroll position when zooming in without extra offset', () => {
      const calculator = new HorizontalScrollCalculator(1000n, 2000n, 0);
      const currentPpn = 1;
      const newPpn = 2;
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
        currentPpn,
        newPpn,
        mousePos,
        currentScrollLeft,
      );
      expect(newScrollLeft).toBeCloseTo(100);
    });

    it('calculates correct new scroll position when zooming in with extra offset', () => {
      const calculator = new HorizontalScrollCalculator(1000n, 2000n, 300);
      const currentPpn = 1;
      const newPpn = 2;
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
        currentPpn,
        newPpn,
        mousePos,
        currentScrollLeft,
      );
      expect(newScrollLeft).toBeCloseTo(0);
    });

    it('keeps the time at mouse position constant', () => {
      const calculator = new HorizontalScrollCalculator(
        1_000_000_000n,
        5_000_000_000n,
        300,
      );
      const currentPpn = 2e-6;
      const newPpn = 5.5e-6; // Arbitrary zoom
      const mousePos = 453; // Arbitrary mouse position
      const currentScrollLeft = 1500; // Arbitrary scroll

      const prevTimeAtMouse =
        Number(
          calculator.scrollToViewportLeftTimeNs(currentScrollLeft, currentPpn),
        ) +
        mousePos / currentPpn;

      const newScrollLeft = calculator.calculateZoomScrollLeft(
        currentPpn,
        newPpn,
        mousePos,
        currentScrollLeft,
      );

      const newTimeAtMouse =
        Number(calculator.scrollToViewportLeftTimeNs(newScrollLeft, newPpn)) +
        mousePos / newPpn;

      expect(newTimeAtMouse).toBeCloseTo(prevTimeAtMouse, -1);
    });
  });
});
