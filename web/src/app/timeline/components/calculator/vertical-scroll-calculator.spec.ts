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

import { TimelineChartStyle } from '../style-model';
import { VerticalScrollCalculator } from './vertical-scroll-calculator';
import { ResourceTimeline, TimelineLayer } from 'src/app/store/timeline';

describe('VerticalScrollCalculator', () => {
  let mockStyle: TimelineChartStyle;

  beforeEach(() => {
    mockStyle = {
      heightsByLayer: {
        [TimelineLayer.Kind]: 100,
        [TimelineLayer.Namespace]: 100,
        [TimelineLayer.Name]: 100,
        [TimelineLayer.Subresource]: 50,
        [TimelineLayer.APIVersion]: 0,
      },
    } as unknown as TimelineChartStyle;
  });

  const createTimelines = (layers: TimelineLayer[]): ResourceTimeline[] => {
    return layers.map((layer) => ({ layer }) as ResourceTimeline);
  };

  describe('constructor', () => {
    it('should calculate totalHeight correctly', () => {
      const timelines = createTimelines([
        TimelineLayer.Kind, // 100
        TimelineLayer.Name, // 100
        TimelineLayer.Subresource, // 50
      ]);
      const calculator = new VerticalScrollCalculator(timelines, mockStyle, 0);
      expect(calculator.totalHeight).toBe(250);
    });

    it('should handle empty timelines', () => {
      const calculator = new VerticalScrollCalculator([], mockStyle, 0);
      expect(calculator.totalHeight).toBe(0);
    });
  });

  describe('topDrawAreaOffset', () => {
    it('should return 0 when timelines are empty', () => {
      const calculator = new VerticalScrollCalculator([], mockStyle, 0);
      expect(calculator.topDrawAreaOffset(100)).toBe(0);
    });

    it('should return the last offsetY when scrollY is greater than totalHeight', () => {
      const timelines = createTimelines([
        TimelineLayer.Kind,
        TimelineLayer.Name,
      ]); // 100,100
      const calculator = new VerticalScrollCalculator(timelines, mockStyle, 0);
      expect(calculator.topDrawAreaOffset(250)).toBe(100);
    });

    it('should return correct offset for scroll position within a timeline', () => {
      // Timeline 0: 0-100
      // Timeline 1: 100-200
      // Timeline 2: 200-250
      const timelines = createTimelines([
        TimelineLayer.Kind, // 100
        TimelineLayer.Name, // 100
        TimelineLayer.Subresource, // 50
      ]);
      const calculator = new VerticalScrollCalculator(timelines, mockStyle, 0);

      // scrollY at 0
      expect(calculator.topDrawAreaOffset(0)).toBe(0);

      // scrollY within first timeline
      expect(calculator.topDrawAreaOffset(50)).toBe(0);

      // scrollY at start of second timeline
      expect(calculator.topDrawAreaOffset(100)).toBe(100);

      // scrollY within second timeline
      expect(calculator.topDrawAreaOffset(150)).toBe(100);

      // scrollY at start of third timeline
      expect(calculator.topDrawAreaOffset(200)).toBe(200);
    });
  });

  describe('timelinesInDrawArea', () => {
    it('should return empty array when timelines are empty', () => {
      const calculator = new VerticalScrollCalculator([], mockStyle, 0);
      expect(calculator.timelinesInDrawArea(0, 100)).toEqual([]);
    });

    it('should return correct timelines overlapping the draw area', () => {
      // Timeline 0: 0-100
      // Timeline 1: 100-200
      // Timeline 2: 200-250
      const timelines = createTimelines([
        TimelineLayer.Kind, // 100
        TimelineLayer.Name, // 100
        TimelineLayer.Subresource, // 50
      ]);
      const calculator = new VerticalScrollCalculator(timelines, mockStyle, 0);

      // Case 1: Only first timeline visible (0-50)
      let result = calculator.timelinesInDrawArea(0, 50);
      expect(result.length).toBe(1);
      expect(result[0]).toBe(timelines[0]);

      // Case 2: Middle timeline visible (120-200)
      result = calculator.timelinesInDrawArea(120, 80);
      expect(result.length).toBe(2);
      expect(result[0]).toBe(timelines[1]);
      expect(result[1]).toBe(timelines[2]);

      // Case 3: Overlapping multiple (50-60)
      result = calculator.timelinesInDrawArea(50, 60);
      expect(result.length).toBe(2);
      expect(result[0]).toBe(timelines[0]);
      expect(result[1]).toBe(timelines[1]);
    });
  });

  describe('with marginTimelineCount = 2', () => {
    const margin = 2;
    it('should include margin timelines in timelinesInDrawArea', () => {
      // Timeline 0: 0-100
      // Timeline 1: 100-200
      // Timeline 2: 200-250
      // Timeline 3: 250-350
      // Timeline 4: 350-450
      const timelines = createTimelines([
        TimelineLayer.Kind, // 100
        TimelineLayer.Name, // 100
        TimelineLayer.Subresource, // 50
        TimelineLayer.Kind, // 100
        TimelineLayer.Kind, // 100
      ]);
      const calculator = new VerticalScrollCalculator(
        timelines,
        mockStyle,
        margin,
      );

      // Only Timeline 2 (200-250) is strictly visible
      // scrollY=210, visibleHeight=10
      // Visible range: 210-220
      // Timeline 2 covers 200-250.
      // Expected: T2 is visible.
      // Margins: T0, T1 (before), T3, T4 (after).
      // Total 5 timelines should be returned.
      // bisectRight(210) -> index 3 (value > 210 is 250 at index 3? No: [0, 100, 200, 250, 350])
      // 0: 0
      // 1: 100
      // 2: 200
      // 3: 250
      // 4: 350
      // bisectRight(210) -> 3 (250 > 210).
      // start index = 3 - 1 - 2 = 0.
      // bisectRight(220) -> 3 (250 > 220).
      // end index = 3 + 2 = 5.
      // slice(0, 5) -> T0, T1, T2, T3, T4.
      const result = calculator.timelinesInDrawArea(210, 10);
      expect(result.length).toBe(5);
      expect(result[0]).toBe(timelines[0]);
      expect(result[4]).toBe(timelines[4]);
    });

    it('should calculate totalRenderHeight with margin', () => {
      // maxTimelineHeight is 100.
      // margin is 2.
      // viewportHeight is 500.
      // totalRenderHeight = 500 + 2 * 2 * 100 = 500 + 400 = 900.
      const timelines = createTimelines([TimelineLayer.Kind]); // max 100
      const calculator = new VerticalScrollCalculator(
        timelines,
        mockStyle,
        margin,
      );
      expect(calculator.totalRenderHeight(500)).toBe(900);
    });
  });

  describe('stickyTimelines', () => {
    it('should return empty array when timelines are empty', () => {
      const calculator = new VerticalScrollCalculator([], mockStyle, 0);
      expect(calculator.stickyTimelines(100)).toEqual([]);
    });

    describe('sticky behavior scenarios', () => {
      let calculator: VerticalScrollCalculator;
      let timelines: ResourceTimeline[];

      beforeEach(() => {
        // Kind1
        //   Namespace1
        //     Pod1
        //     Pod2
        //   Namespace2
        //     Pod3
        // Kind2
        // ...
        timelines = createTimelines([
          TimelineLayer.Kind, // 0-100 (Kind1)
          TimelineLayer.Namespace, // 100-200 (Namespace1)
          TimelineLayer.Name, // 200-300 (Pod1)
          TimelineLayer.Name, // 300-400 (Pod2)
          TimelineLayer.Namespace, // 400-500 (Namespace2)
          TimelineLayer.Name, // 500-600 (Pod3)
          TimelineLayer.Subresource, // 600-650 (Subresource1)
          TimelineLayer.Kind, // 650-750 (Kind2)
          TimelineLayer.Namespace, // 750-850 (Namespace3)
          TimelineLayer.Name, // 850-950 (Pod4)
          TimelineLayer.Subresource, // 950-1050 (Subresource2)
        ]);
        calculator = new VerticalScrollCalculator(timelines, mockStyle, 0);
      });

      it('should return initial sticky header at scroll 0', () => {
        const result = calculator.stickyTimelines(0);
        expect(result.length).toBe(3);
        expect(result[0]).toBe(timelines[0]);
        expect(result[1]).toBe(timelines[1]);
        expect(result[2]).toBe(timelines[2]);
      });

      it('should maintain current sticky header before next header arrives (scroll 99)', () => {
        // 99 + 200 = 299 -> invades Pod1 (200-300).
        const result = calculator.stickyTimelines(99);
        expect(result.length).toBe(2);
        expect(result[0]).toBe(timelines[0]);
        expect(result[1]).toBe(timelines[1]);
        // The next item is also the Name layer, so the sticky header is up to the namespace layer.
      });

      it('should maintain current sticky header at exact boundary (scroll 100)', () => {
        // 100 + 200 = 300 -> reaches Pod2 (300-400).
        const result = calculator.stickyTimelines(100);
        expect(result.length).toBe(2);
        expect(result[0]).toBe(timelines[0]); // Kind1
        expect(result[1]).toBe(timelines[1]); // Namespace1
      });

      it('should switch to next sticky header after boundary when the next item is name resource (scroll 101)', () => {
        // 101 + 200 = 301 -> inside Pod2.
        const result = calculator.stickyTimelines(101);
        expect(result.length).toBe(2);
        expect(result[0]).toBe(timelines[0]); // Kind1
        expect(result[1]).toBe(timelines[1]); // Namespace1
        // The next item is also the Name layer, so the sticky header is up to the namespace layer.
      });

      it('should return sticky timelines only when the sticky timeline is fully visible (scroll 550)', () => {
        // Scroll 550 (inside Pod3, Namespace2, Kind1).
        // Kind2 starts at 600. There is no enough space to show Kind1 & Namespace1 as stickyTimelines
        // Scroll 550 + 300 = 850.
        // Returns [Kind1].
        const result = calculator.stickyTimelines(550);
        expect(result.length).toBe(1);
        expect(result[0]).toBe(timelines[0]); // Kind1
      });

      it('should maintain the last sticky header when scrolling past total height', () => {
        // Total height is 1050.
        // Scroll 1200.
        const result = calculator.stickyTimelines(1200);
        expect(result.length).toBe(3);
        expect(result[0]).toBe(timelines[7]); // Kind2
        expect(result[1]).toBe(timelines[8]); // Namespace3
        expect(result[2]).toBe(timelines[9]); // Pod4
      });
    });
  });
});
