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

import { VerticalScrollCalculator } from './vertical-scroll-calculator';
import { Timeline } from 'src/app/store/domain/timeline';
import { StyleStoreLike } from 'src/app/store/domain/style-store';
import { TimelineType } from 'src/app/store/domain/style';

describe('VerticalScrollCalculator', () => {
  interface TimelineSpecConfig {
    readonly height: number;
    readonly parentIndex?: number;
    readonly childrenCount?: number;
  }

  const mockStyleStore: StyleStoreLike = {
    severities: [],
    logTypes: [],
    verbs: [],
    revisionStates: [],
    timelineTypes: [],
    getSeverity: () => {
      throw new Error('not implemented');
    },
    getLogType: () => {
      throw new Error('not implemented');
    },
    getVerb: () => {
      throw new Error('not implemented');
    },
    getRevisionState: () => {
      throw new Error('not implemented');
    },
    getTimelineType: () => undefined as unknown as TimelineType,
    getIconAtlas: () => undefined,
  };

  const createMockTimelines = (
    configs: readonly TimelineSpecConfig[],
  ): Timeline[] => {
    const result: Timeline[] = [];
    for (let index = 0; index < configs.length; index++) {
      const config = configs[index];
      const mockTimeline = {
        id: index + 1,
        type: { height: config.height },
        get parent() {
          if (config.parentIndex !== undefined) {
            return result[config.parentIndex];
          }
          return null;
        },
        get childrenCount() {
          return (
            config.childrenCount ??
            result.filter((t) => t.parent?.id === index + 1).length
          );
        },
      } as unknown as Timeline;
      result.push(mockTimeline);
    }
    return result;
  };

  describe('constructor', () => {
    it('should calculate totalHeight correctly', () => {
      const timelines = createMockTimelines([
        { height: 4.0 }, // 4.0 * 25 = 100
        { height: 4.0 }, // 100
        { height: 2.0 }, // 50
      ]);
      const calculator = new VerticalScrollCalculator(
        timelines,
        0,
        mockStyleStore,
      );
      expect(calculator.totalHeight).toBe(250);
    });

    it('should handle empty timelines', () => {
      const calculator = new VerticalScrollCalculator([], 0, mockStyleStore);
      expect(calculator.totalHeight).toBe(0);
    });

    it('should respect overridden timeline heights from styleStore', () => {
      const timelines = createMockTimelines([{ height: 4.0 }]);
      (timelines[0].type as { id: number }).id = 101;
      const customStyleStore: StyleStoreLike = {
        ...mockStyleStore,
        getTimelineType: (id: number) => {
          if (id === 101) {
            return { height: 2.0 } as unknown as TimelineType;
          }
          return undefined as unknown as TimelineType;
        },
      };
      const calculator = new VerticalScrollCalculator(
        timelines,
        0,
        customStyleStore,
      );
      expect(calculator.totalHeight).toBe(50);
    });
  });

  describe('topDrawAreaOffset', () => {
    it('should return 0 when timelines are empty', () => {
      const calculator = new VerticalScrollCalculator([], 0, mockStyleStore);
      expect(calculator.topDrawAreaOffset(100)).toBe(0);
    });

    it('should return the last offsetY when scrollY is greater than totalHeight', () => {
      const timelines = createMockTimelines([
        { height: 4.0 }, // 100
        { height: 4.0 }, // 100
      ]);
      const calculator = new VerticalScrollCalculator(
        timelines,
        0,
        mockStyleStore,
      );
      expect(calculator.topDrawAreaOffset(250)).toBe(100);
    });

    it('should return correct offset for scroll position within a timeline', () => {
      const timelines = createMockTimelines([
        { height: 4.0 }, // 100
        { height: 4.0 }, // 100
        { height: 2.0 }, // 50
      ]);
      const calculator = new VerticalScrollCalculator(
        timelines,
        0,
        mockStyleStore,
      );

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
      const calculator = new VerticalScrollCalculator([], 0, mockStyleStore);
      expect(calculator.timelinesInDrawArea(0, 100)).toEqual([]);
    });

    it('should return correct timelines overlapping the draw area', () => {
      const timelines = createMockTimelines([
        { height: 4.0 }, // 100
        { height: 4.0 }, // 100
        { height: 2.0 }, // 50
      ]);
      const calculator = new VerticalScrollCalculator(
        timelines,
        0,
        mockStyleStore,
      );

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
      const timelines = createMockTimelines([
        { height: 4.0 }, // 100
        { height: 4.0 }, // 100
        { height: 2.0 }, // 50
        { height: 4.0 }, // 100
        { height: 4.0 }, // 100
      ]);
      const calculator = new VerticalScrollCalculator(
        timelines,
        margin,
        mockStyleStore,
      );

      // Only Timeline 2 (200-250) is strictly visible
      // scrollY=210, visibleHeight=10
      // Visible range: 210-220
      const result = calculator.timelinesInDrawArea(210, 10);
      expect(result.length).toBe(5);
      expect(result[0]).toBe(timelines[0]);
      expect(result[4]).toBe(timelines[4]);
    });

    it('should calculate totalRenderHeight with margin', () => {
      const timelines = createMockTimelines([{ height: 4.0 }]); // max 100
      const calculator = new VerticalScrollCalculator(
        timelines,
        margin,
        mockStyleStore,
      );
      expect(calculator.totalRenderHeight(500)).toBe(900);
    });
  });

  describe('stickyTimelines', () => {
    it('should return empty array when timelines are empty', () => {
      const calculator = new VerticalScrollCalculator([], 0, mockStyleStore);
      expect(calculator.stickyTimelines(100)).toEqual([]);
    });

    describe('sticky behavior scenarios', () => {
      let calculator: VerticalScrollCalculator;
      let timelines: Timeline[];

      beforeEach(() => {
        timelines = createMockTimelines([
          { height: 4.0 }, // index 0 (Kind1), parent = null
          { height: 4.0, parentIndex: 0 }, // index 1 (Namespace1)
          { height: 4.0, parentIndex: 1 }, // index 2 (Pod1)
          { height: 4.0, parentIndex: 1 }, // index 3 (Pod2)
          { height: 4.0, parentIndex: 0 }, // index 4 (Namespace2)
          { height: 4.0, parentIndex: 4 }, // index 5 (Pod3)
          { height: 2.0, parentIndex: 5 }, // index 6 (Subresource1)
          { height: 4.0 }, // index 7 (Kind2), parent = null
          { height: 4.0, parentIndex: 7 }, // index 8 (Namespace3)
          { height: 4.0, parentIndex: 8 }, // index 9 (Pod4)
          { height: 2.0, parentIndex: 9 }, // index 10 (Subresource2)
        ]);
        calculator = new VerticalScrollCalculator(timelines, 0, mockStyleStore);
      });

      it('should return initial sticky header at scroll 0', () => {
        const result = calculator.stickyTimelines(0);
        expect(result.length).toBe(2);
        expect(result[0]).toBe(timelines[0]);
        expect(result[1]).toBe(timelines[1]);
      });

      it('should maintain current sticky header before next header arrives (scroll 99)', () => {
        // 99 + 200 = 299 -> invades Pod1 (200-300).
        const result = calculator.stickyTimelines(99);
        expect(result.length).toBe(2);
        expect(result[0]).toBe(timelines[0]);
        expect(result[1]).toBe(timelines[1]);
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
