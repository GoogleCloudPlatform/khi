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
  TimelineExcerptKind,
  TimelineExcerptLineKind,
  TimelineResolvedExcerptCard,
  TimelineSceneOverlay,
} from 'src/app/common/schema/timeline-render-types';
import {
  ExcerptDeckGridPreset,
  ExcerptDeckLayoutPlan,
  MAX_DECK_CARD_WIDTH_PX,
  MAX_INDEX_PANE_WIDTH_PX,
  MAX_STAGE_WIDTH_PX,
  MIN_CHART_VIEWPORT_WIDTH_PX,
  MIN_INDEX_PANE_WIDTH_PX,
  TIMELINE_GUTTER_WIDTH_PX,
  TIMELINE_HEADER_HEIGHT_PX,
  TextWidthMeasurer,
  TimelineStageDimensions,
  calculateTimelineStageDimensions,
  createCanvasTextWidthMeasurer,
  planExcerptDeckLayout,
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

function makeExcerptCard(
  id: string,
  kind: TimelineExcerptKind,
  lineText: string,
  note?: string,
): TimelineResolvedExcerptCard {
  return {
    id,
    kind,
    badge: 'E1',
    paletteId: 'default',
    timeMs: 1_000_000,
    timeLabel: '00:16:40',
    timelineId: 1,
    resourcePath: 'pod/test-pod',
    title: 'Event',
    subtitle: '',
    lines: [{ kind: TimelineExcerptLineKind.CONTEXT, text: lineText }],
    note,
  };
}

describe('timeline-render-sizing', () => {
  interface SizingCase {
    readonly desc: string;
    readonly timelines: readonly ReadonlyDomainElement<Timeline>[];
    readonly styleStore: StyleStoreLike;
    readonly overlay?: TimelineSceneOverlay | null;
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
    {
      desc: 'widens index pane so a primary row highlight role badge does not overlap the resource label',
      // Row 1 ('test-pod-alpha-01', layer 2): leftContent = 5 + 8 + 54 + 12 + 119 + 80 = 278 (fits 300px without badge).
      // Role badge 'VICTIM · Pod' (12 chars): pillWidth = 84 + 6 + 14 = 104 -> rightAllowance = 8 + 104 + 6 = 118.
      // Required index width = 278 + 118 + 4 = 400.
      timelines: [
        makeTimeline(1, 'test-pod-alpha-01', 2),
        makeTimeline(2, 'container', 3),
      ],
      styleStore: defaultStyleStore,
      overlay: {
        rowHighlights: [
          {
            id: 'rh-victim',
            roleLabel: 'VICTIM · Pod',
            timelineIds: [1, 2],
            paletteId: 'default',
          },
        ],
      },
      want: {
        indexPaneWidthPx: 400,
        chartViewportWidthPx: 1280,
        stageWidthPx: 400 + 8 + 1280,
        timelineFrameHeightPx: 160,
      },
    },
    {
      desc: 'charge role badge width only to the topmost matched row of a highlight group',
      // Row 1 ('test-pod', layer 1) receives the badge: leftContent (5+4+54+12+56+80 = 211) + 118 + 4 = 333.
      // Row 2 ('test-pod-alpha-01', layer 2) has no badge: 278 + 8 + 4 = 290 <= 333 (would be 400 if charged).
      timelines: [
        makeTimeline(1, 'test-pod', 1),
        makeTimeline(2, 'test-pod-alpha-01', 2),
      ],
      styleStore: defaultStyleStore,
      overlay: {
        rowHighlights: [
          {
            id: 'rh-top-only',
            roleLabel: 'VICTIM · Pod',
            timelineIds: [2, 1],
            paletteId: 'default',
          },
        ],
      },
      want: {
        indexPaneWidthPx: 333,
        chartViewportWidthPx: 1280,
        stageWidthPx: 333 + 8 + 1280,
        timelineFrameHeightPx: 160,
      },
    },
    {
      desc: 'expands stage and chart viewport horizontally when 3 wide diff cards exceed minimum timeline width',
      // 80-char diff line: 80 * 7 + 106 + 4 = 670px per card -> 24 + 3 * 670 + 24 = 2058px stage width.
      timelines: [makeTimeline(1, 'pod', 1)],
      styleStore: defaultStyleStore,
      overlay: {
        excerpts: [
          makeExcerptCard('e1', TimelineExcerptKind.DIFF, 'x'.repeat(80)),
          makeExcerptCard('e2', TimelineExcerptKind.DIFF, 'short'),
          makeExcerptCard('e3', TimelineExcerptKind.LOG, 'short'),
        ],
      },
      want: {
        indexPaneWidthPx: 300,
        chartViewportWidthPx: 2058 - 308,
        stageWidthPx: 2058,
        timelineFrameHeightPx: 110,
      },
    },
    {
      desc: 'clamps pathological deck line widths at MAX_DECK_CARD_WIDTH_PX and keeps stage within MAX_STAGE_WIDTH_PX',
      timelines: [makeTimeline(1, 'pod', 1)],
      styleStore: defaultStyleStore,
      overlay: {
        excerpts: [
          makeExcerptCard('e1', TimelineExcerptKind.LOG, 'm'.repeat(1000)),
          makeExcerptCard('e2', TimelineExcerptKind.LOG, 'm'.repeat(1000)),
          makeExcerptCard('e3', TimelineExcerptKind.LOG, 'm'.repeat(1000)),
        ],
      },
      want: {
        indexPaneWidthPx: 300,
        chartViewportWidthPx: 24 + 3 * MAX_DECK_CARD_WIDTH_PX + 24 - 308,
        stageWidthPx: 24 + 3 * MAX_DECK_CARD_WIDTH_PX + 24,
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
        tc.overlay,
      );
      expect(got).toEqual(tc.want);
    });
  }

  interface DeckPlanCase {
    readonly desc: string;
    readonly excerpts: readonly TimelineResolvedExcerptCard[];
    readonly want: ExcerptDeckLayoutPlan;
  }

  const deckPlanCases: readonly DeckPlanCase[] = [
    {
      desc: 'return empty plan with zero required width when no excerpts exist',
      excerpts: [],
      want: {
        preset: ExcerptDeckGridPreset.EMPTY,
        columns: 0,
        requiredDeckWidthPx: 0,
      },
    },
    {
      desc: 'use 4 horizontal columns when 4 wide diff cards fit within MAX_STAGE_WIDTH_PX',
      // 125-char diff line -> 125 * 7 + 106 + 4 = 985px -> 24 + 4 * 985 + 36 = 4000 <= 4096.
      excerpts: [
        makeExcerptCard('e1', TimelineExcerptKind.DIFF, 'a'.repeat(125)),
        makeExcerptCard('e2', TimelineExcerptKind.DIFF, 'short'),
        makeExcerptCard('e3', TimelineExcerptKind.DIFF, 'short'),
        makeExcerptCard('e4', TimelineExcerptKind.DIFF, 'short'),
      ],
      want: {
        preset: ExcerptDeckGridPreset.COLS_4,
        columns: 4,
        requiredDeckWidthPx: 4000,
      },
    },
    {
      desc: 'wrap 4 wide cards to GRID_2X2 when unclamped card width exceeds MAX_DECK_CARD_WIDTH_PX even below MAX_STAGE_WIDTH_PX',
      // 128-char diff line -> 128 * 7 + 106 + 4 = 1006px (> 1000px cap, though 24 + 4 * 1006 + 36 = 4084 <= 4096) -> wraps to 2x2.
      excerpts: [
        makeExcerptCard('e1', TimelineExcerptKind.DIFF, 'a'.repeat(128)),
        makeExcerptCard('e2', TimelineExcerptKind.DIFF, 'short'),
        makeExcerptCard('e3', TimelineExcerptKind.DIFF, 'short'),
        makeExcerptCard('e4', TimelineExcerptKind.DIFF, 'short'),
      ],
      want: {
        preset: ExcerptDeckGridPreset.GRID_2X2,
        columns: 2,
        requiredDeckWidthPx: 24 + 2 * MAX_DECK_CARD_WIDTH_PX + 12,
      },
    },
    {
      desc: 'wrap 4 ultra-wide cards to GRID_2X2 when 4 unclamped columns would exceed MAX_STAGE_WIDTH_PX',
      // 130-char diff line -> 130 * 7 + 106 + 4 = 1020px -> 4-col width 4140 > 4096 -> wraps to 2x2 clamped at 1000px.
      excerpts: [
        makeExcerptCard('e1', TimelineExcerptKind.DIFF, 'a'.repeat(130)),
        makeExcerptCard('e2', TimelineExcerptKind.DIFF, 'short'),
        makeExcerptCard('e3', TimelineExcerptKind.DIFF, 'short'),
        makeExcerptCard('e4', TimelineExcerptKind.DIFF, 'short'),
      ],
      want: {
        preset: ExcerptDeckGridPreset.GRID_2X2,
        columns: 2,
        requiredDeckWidthPx: 24 + 2 * MAX_DECK_CARD_WIDTH_PX + 12,
      },
    },
    {
      desc: 'expand card width to fit long footer notes cleanly',
      // 90-char note -> 46 + 90 * 7 + 26 + 4 = 706px for a single COLS_1 card -> 24 + 706 = 730px.
      excerpts: [
        makeExcerptCard('e1', TimelineExcerptKind.LOG, 'ok', 'n'.repeat(90)),
      ],
      want: {
        preset: ExcerptDeckGridPreset.COLS_1,
        columns: 1,
        requiredDeckWidthPx: 730,
      },
    },
  ];

  for (const tc of deckPlanCases) {
    it(`should ${tc.desc}`, () => {
      const got = planExcerptDeckLayout(tc.excerpts, fixedCharWidthMeasurer);
      expect(got).toEqual(tc.want);
      expect(got.requiredDeckWidthPx).toBeLessThanOrEqual(MAX_STAGE_WIDTH_PX);
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
