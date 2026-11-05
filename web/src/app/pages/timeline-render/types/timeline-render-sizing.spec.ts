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
  MAX_INDEX_PANE_WIDTH_PX,
  MIN_CHART_VIEWPORT_WIDTH_PX,
  MIN_INDEX_PANE_WIDTH_PX,
  TIMELINE_GUTTER_WIDTH_PX,
  TIMELINE_HEADER_HEIGHT_PX,
  TextWidthMeasurer,
  TimelineStageDimensions,
  calculateTimelineStageDimensions,
  createCanvasTextWidthMeasurer,
} from 'src/app/pages/timeline-render/types/timeline-render-sizing';
import { TimelineType } from 'src/app/store/domain/style';
import { StyleStoreLike } from 'src/app/store/domain/style-store';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { BASE_ROW_HEIGHT } from 'src/app/timeline/components/style-model';

const podTimelineType: TimelineType = {
  id: 1,
  label: 'Resource',
  description: 'Kubernetes resource',
  icon: 'pod',
  backgroundColor: { r: 0, g: 0, b: 0, a: 1 },
  foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
  typeChipBackgroundColor: { r: 0, g: 0, b: 0, a: 1 },
  typeChipForegroundColor: { r: 1, g: 1, b: 1, a: 1 },
  visible: true,
  sortPriority: 0,
  height: 2,
};

function makeTimeline(
  id: number,
  name: string,
  layer: number,
  type: TimelineType = podTimelineType,
): ReadonlyDomainElement<Timeline> {
  return {
    id,
    name,
    rawName: name,
    type,
    path: [],
    layer,
    revisions: [],
    events: [],
    parent: null,
    childrenCount: 0,
  } as unknown as ReadonlyDomainElement<Timeline>;
}

const fixedCharWidthMeasurer: TextWidthMeasurer = (text: string) =>
  text.length * 7;

const defaultStyleStore: StyleStoreLike = {
  severities: [],
  timelineTypes: [podTimelineType],
  getTimelineType: () => podTimelineType,
} as unknown as StyleStoreLike;

describe('timeline-render-sizing', () => {
  interface SizingCase {
    readonly desc: string;
    readonly timelines: readonly ReadonlyDomainElement<Timeline>[];
    readonly styleStore: StyleStoreLike;
    readonly want: TimelineStageDimensions;
  }

  const cases: readonly SizingCase[] = [
    {
      desc: 'returns baseline minimum stage and frame dimensions when no timelines are rendered',
      timelines: [],
      styleStore: defaultStyleStore,
      want: {
        indexPaneWidthPx: MIN_INDEX_PANE_WIDTH_PX,
        chartViewportWidthPx: MIN_CHART_VIEWPORT_WIDTH_PX,
        stageWidthPx:
          MIN_INDEX_PANE_WIDTH_PX +
          TIMELINE_GUTTER_WIDTH_PX +
          MIN_CHART_VIEWPORT_WIDTH_PX,
        timelineFrameHeightPx: TIMELINE_HEADER_HEIGHT_PX + BASE_ROW_HEIGHT,
      },
    },
    {
      desc: 'keeps 300px index width when short labels fit within the minimum pane size',
      timelines: [makeTimeline(1, 'core/v1', 0), makeTimeline(2, 'pod', 1)],
      styleStore: defaultStyleStore,
      want: {
        indexPaneWidthPx: 300,
        chartViewportWidthPx: 1280,
        stageWidthPx: 1588,
        timelineFrameHeightPx: 160,
      },
    },
    {
      desc: 'expands index pane and stage width horizontally for long nested names without shrinking chart width',
      // layer 4 -> leftPadding = 4 + 3*4 = 16; rowHeight = 50 -> icon = 18;
      // name length 40 * 7 = 280; chip 'Resource' (8 chars) -> text = 56 + 4 = 60 -> block = 8 + 60 + 12 = 80;
      // total = 5 + 16 + 54 + 12 + 280 + 80 + 8 + 4 = 459.
      timelines: [makeTimeline(1, 'a'.repeat(40), 4)],
      styleStore: defaultStyleStore,
      want: {
        indexPaneWidthPx: 459,
        chartViewportWidthPx: 1280,
        stageWidthPx: 459 + 8 + 1280,
        timelineFrameHeightPx: 110,
      },
    },
    {
      desc: 'caps pathological label width at MAX_INDEX_PANE_WIDTH_PX while preserving minimum chart width',
      timelines: [makeTimeline(1, 'x'.repeat(400), 5)],
      styleStore: defaultStyleStore,
      want: {
        indexPaneWidthPx: MAX_INDEX_PANE_WIDTH_PX,
        chartViewportWidthPx: MIN_CHART_VIEWPORT_WIDTH_PX,
        stageWidthPx:
          MAX_INDEX_PANE_WIDTH_PX +
          TIMELINE_GUTTER_WIDTH_PX +
          MIN_CHART_VIEWPORT_WIDTH_PX,
        timelineFrameHeightPx: 110,
      },
    },
    {
      desc: 'falls back to timeline.type height when styleStore throws for an unknown type ID',
      timelines: [makeTimeline(1, 'short', 1)],
      styleStore: {
        severities: [],
        timelineTypes: [],
        getTimelineType: () => {
          throw new Error('unknown timeline type');
        },
      } as unknown as StyleStoreLike,
      want: {
        indexPaneWidthPx: 300,
        chartViewportWidthPx: 1280,
        stageWidthPx: 1588,
        timelineFrameHeightPx: 110,
      },
    },
  ];

  for (const tc of cases) {
    it(`should ${tc.desc}`, () => {
      const got = calculateTimelineStageDimensions(
        tc.timelines,
        tc.styleStore,
        fixedCharWidthMeasurer,
      );
      expect(got).toEqual(tc.want);
    });
  }

  it('should measure monotonic widths with createCanvasTextWidthMeasurer', () => {
    const measure = createCanvasTextWidthMeasurer();
    expect(measure('', '400 13px Roboto, sans-serif')).toBe(0);
    const shortWidth = measure('pod', '400 13px Roboto, sans-serif');
    const longWidth = measure(
      'gke-test-cluster-default-pool-12345678-abcd',
      '400 13px Roboto, sans-serif',
    );
    expect(shortWidth).toBeGreaterThan(0);
    expect(longWidth).toBeGreaterThan(shortWidth);
  });
});
