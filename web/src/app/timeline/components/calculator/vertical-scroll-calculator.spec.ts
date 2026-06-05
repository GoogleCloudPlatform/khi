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
  VerticalScrollCalculator,
  TimelineLayer,
} from './vertical-scroll-calculator';
import { Timeline } from 'src/app/store/domain/timeline';

describe('VerticalScrollCalculator', () => {
  const createTimelines = (layers: TimelineLayer[]): Timeline[] => {
    const result: Timeline[] = [];
    for (let index = 0; index < layers.length; index++) {
      const layer = layers[index];
      let height = 1.0;
      if (layer === TimelineLayer.Kind) height = 4.0; // 4.0 * 25.0 = 100
      if (layer === TimelineLayer.Namespace) height = 4.0; // 100
      if (layer === TimelineLayer.Name) height = 4.0; // 100
      if (layer === TimelineLayer.Subresource) height = 2.0; // 50
      if (layer === TimelineLayer.APIVersion) height = 0.0; // 0

      const mockTimeline = {
        id: index + 1,
        type: { height },
        layer,
        get parent() {
          for (let j = index - 1; j >= 0; j--) {
            if (result[j].layer < layer) {
              return result[j];
            }
          }
          return null;
        },
        get childrenCount() {
          return result.filter((t) => t.parent?.id === index + 1).length;
        },
      } as unknown as Timeline;
      result.push(mockTimeline);
    }
    return result;
  };

  describe('constructor', () => {
    it('should calculate totalHeight correctly', () => {
      const timelines = createTimelines([
        TimelineLayer.Kind, // 100
        TimelineLayer.Name, // 100
        TimelineLayer.Subresource, // 50
      ]);
      const calculator = new VerticalScrollCalculator(timelines, 0);
      expect(calculator.totalHeight).toBe(250);
    });

    it('should handle empty timelines', () => {
      const calculator = new VerticalScrollCalculator([], 0);
      expect(calculator.totalHeight).toBe(0);
    });
  });

  describe('topDrawAreaOffset', () => {
    it('should return 0 when timelines are empty', () => {
      const calculator = new VerticalScrollCalculator([], 0);
      expect(calculator.topDrawAreaOffset(100)).toBe(0);
    });

    it('should return the last offsetY when scrollY is greater than totalHeight', () => {
      const timelines = createTimelines([
        TimelineLayer.Kind,
        TimelineLayer.Name,
      ]); // 100,100
      const calculator = new VerticalScrollCalculator(timelines, 0);
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
      const calculator = new VerticalScrollCalculator(timelines, 0);

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
      const calculator = new VerticalScrollCalculator([], 0);
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
      const calculator = new VerticalScrollCalculator(timelines, 0);

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
      const calculator = new VerticalScrollCalculator(timelines, margin);

      // Only Timeline 2 (200-250) is strictly visible
      // scrollY=210, visibleHeight=10
      // Visible range: 210-220
      const result = calculator.timelinesInDrawArea(210, 10);
      expect(result.length).toBe(5);
      expect(result[0]).toBe(timelines[0]);
      expect(result[4]).toBe(timelines[4]);
    });

    it('should calculate totalRenderHeight with margin', () => {
      const timelines = createTimelines([TimelineLayer.Kind]); // max 100
      const calculator = new VerticalScrollCalculator(timelines, margin);
      expect(calculator.totalRenderHeight(500)).toBe(900);
    });
  });

  describe('stickyTimelines', () => {
    it('should return empty array when timelines are empty', () => {
      const calculator = new VerticalScrollCalculator([], 0);
      expect(calculator.stickyTimelines(100)).toEqual([]);
    });

    describe('sticky behavior scenarios', () => {
      let calculator: VerticalScrollCalculator;
      let timelines: Timeline[];

      beforeEach(() => {
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
        calculator = new VerticalScrollCalculator(timelines, 0);
      });

      it('should return initial sticky header at scroll 0', () => {
        const result = calculator.stickyTimelines(0);
        expect(result.length).toBe(2);
        expect(result[0]).toBe(timelines[0]);
        expect(result[1]).toBe(timelines[1]);
      });

      it('should maintain current sticky header before next header arrives (scroll 199)', () => {
        // Namespace2 starts at 400.
        // 400 - 199 = 201.
        // Sticky header area is 200.
        // So Namespace2 is NOT yet sticky.
        const result = calculator.stickyTimelines(199);
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
        // Kind2 starts at 650. There is not enough space to show Kind1 & Namespace2 as stickyTimelines.
        // Returns [].
        const result = calculator.stickyTimelines(550);
        expect(result.length).toBe(0);
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
