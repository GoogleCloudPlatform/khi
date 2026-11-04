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
  TimelineMarkerStyle,
  TimelineResolvedPaletteColors,
  TimelineSpanPattern,
} from 'src/app/common/schema/timeline-render-types';
import {
  DEFAULT_REFERENCE_PALETTE_COLORS,
  buildTimelineSceneViewModel,
  extractTimelineRowMetrics,
} from 'src/app/pages/timeline-render/types/timeline-render-scene.viewmodel';
import {
  MIN_CHART_VIEWPORT_WIDTH_PX,
  MIN_INDEX_PANE_WIDTH_PX,
  TIMELINE_GUTTER_WIDTH_PX,
  TIMELINE_HEADER_HEIGHT_PX,
} from 'src/app/pages/timeline-render/types/timeline-render-sizing';
import { HDRColor4, TimelineType } from 'src/app/store/domain/style';
import { StyleStore } from 'src/app/store/domain/style-store';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { BASE_ROW_HEIGHT } from 'src/app/timeline/components/style-model';

describe('extractTimelineRowMetrics', () => {
  const zeroColor: HDRColor4 = { r: 0, g: 0, b: 0, a: 1 };

  const makeTimelineType = (id: number, height: number): TimelineType => ({
    id,
    label: `type-${id}`,
    description: '',
    icon: '',
    backgroundColor: zeroColor,
    foregroundColor: zeroColor,
    typeChipBackgroundColor: zeroColor,
    typeChipForegroundColor: zeroColor,
    visible: true,
    sortPriority: 0,
    height,
  });

  it('uses overridden timeline heights from styleStore and falls back to element type height', () => {
    const styleStore = new StyleStore();
    styleStore.addTimelineTypes([makeTimelineType(1, 2)]);

    const timelines = [
      { id: 100, type: makeTimelineType(1, 1) },
      { id: 200, type: makeTimelineType(99, 1.5) },
    ] as unknown as readonly ReadonlyDomainElement<Timeline>[];

    expect(extractTimelineRowMetrics(timelines, styleStore)).toEqual([
      { id: 100, heightPx: 2 * BASE_ROW_HEIGHT },
      { id: 200, heightPx: 1.5 * BASE_ROW_HEIGHT },
    ]);
  });
});

describe('buildTimelineSceneViewModel', () => {
  const defaultChartLeftPx = MIN_INDEX_PANE_WIDTH_PX + TIMELINE_GUTTER_WIDTH_PX;
  const defaultStageWidthPx = defaultChartLeftPx + MIN_CHART_VIEWPORT_WIDTH_PX;

  const sampleRows = [
    { id: 10, heightPx: 50 },
    { id: 20, heightPx: 50 },
    { id: 30, heightPx: 50 },
    { id: 40, heightPx: 50 },
  ];

  const triggerPalette: TimelineResolvedPaletteColors = {
    id: 'trigger',
    main: '#b45309',
    onMain: '#fafafa',
    soft: 'rgba(180, 83, 9, 0.13)',
    surface: '#f8eee6',
  };

  const victimPalette: TimelineResolvedPaletteColors = {
    id: 'victim',
    main: '#d32f2f',
    onMain: '#fafafa',
    soft: 'rgba(211, 47, 47, 0.13)',
    surface: '#fbeaea',
  };

  it('computes zero topBandHeightPx and baseline minimum width when overlay is null', () => {
    const vm = buildTimelineSceneViewModel({
      indexPaneWidthPx: MIN_INDEX_PANE_WIDTH_PX,
      viewportLeftTimeMs: 1_000_000,
      pixelsPerMs: 0.01,
      rows: sampleRows,
      overlay: null,
    });

    expect({
      indexPaneWidthPx: vm.indexPaneWidthPx,
      chartLeftPx: vm.chartLeftPx,
      chartViewportWidthPx: vm.chartViewportWidthPx,
      stageWidthPx: vm.stageWidthPx,
      topBandHeightPx: vm.topBandHeightPx,
      timelineFrameHeightPx: vm.timelineFrameHeightPx,
      stageHeightPx: vm.stageHeightPx,
      estimatedTotalHeightPx: vm.estimatedTotalHeightPx,
      hasOverlayItems: vm.hasOverlayItems,
    }).toEqual({
      indexPaneWidthPx: 300,
      chartLeftPx: 308,
      chartViewportWidthPx: 1280,
      stageWidthPx: defaultStageWidthPx,
      topBandHeightPx: 0,
      timelineFrameHeightPx: 260,
      stageHeightPx: 260,
      estimatedTotalHeightPx: 260,
      hasOverlayItems: false,
    });
  });

  it('expands stage width horizontally with indexPaneWidthPx while preserving minimum chart viewport width', () => {
    interface WidthCase {
      readonly desc: string;
      readonly indexPaneWidthPx: number;
      readonly wantIndexPaneWidthPx: number;
      readonly wantChartLeftPx: number;
      readonly wantStageWidthPx: number;
    }

    const widthCases: readonly WidthCase[] = [
      {
        desc: 'clamps narrow index pane up to 300px minimum',
        indexPaneWidthPx: 180,
        wantIndexPaneWidthPx: 300,
        wantChartLeftPx: 308,
        wantStageWidthPx: 1588,
      },
      {
        desc: 'grows stage horizontally for wide index labels without shrinking 1280px chart viewport',
        indexPaneWidthPx: 480,
        wantIndexPaneWidthPx: 480,
        wantChartLeftPx: 488,
        wantStageWidthPx: 1768,
      },
      {
        desc: 'caps pathological index pane width at 1200px',
        indexPaneWidthPx: 1600,
        wantIndexPaneWidthPx: 1200,
        wantChartLeftPx: 1208,
        wantStageWidthPx: 2488,
      },
    ];

    for (const tc of widthCases) {
      const vm = buildTimelineSceneViewModel({
        indexPaneWidthPx: tc.indexPaneWidthPx,
        viewportLeftTimeMs: 1_000_000,
        pixelsPerMs: 0.01,
        rows: sampleRows,
        overlay: {
          markers: [
            {
              id: 'm1',
              badge: '1',
              label: 'Event',
              style: TimelineMarkerStyle.PIN,
              paletteId: 'default',
              timeMs: 1_020_000,
              timeLabel: '00:17:00',
              timelineId: 10,
            },
          ],
        },
      });

      expect({
        indexPaneWidthPx: vm.indexPaneWidthPx,
        chartLeftPx: vm.chartLeftPx,
        chartViewportWidthPx: vm.chartViewportWidthPx,
        stageWidthPx: vm.stageWidthPx,
        stemX: vm.markers[0].stemX,
      })
        .withContext(tc.desc)
        .toEqual({
          indexPaneWidthPx: tc.wantIndexPaneWidthPx,
          chartLeftPx: tc.wantChartLeftPx,
          chartViewportWidthPx: 1280,
          stageWidthPx: tc.wantStageWidthPx,
          stemX: tc.wantChartLeftPx + 200,
        });
    }
  });

  it('staggers overlapping marker pills into separate top-band lanes and resolves named and default-reference palettes', () => {
    const vm = buildTimelineSceneViewModel({
      indexPaneWidthPx: MIN_INDEX_PANE_WIDTH_PX,
      viewportLeftTimeMs: 1_000_000,
      pixelsPerMs: 0.01,
      rows: sampleRows,
      overlay: {
        palette: [victimPalette],
        markers: [
          {
            id: 'm1',
            badge: '1',
            label: 'Node NotReady transition',
            style: TimelineMarkerStyle.PIN,
            paletteId: 'victim',
            timeMs: 1_020_000,
            timeLabel: '00:17:00',
            timelineId: 20,
            revisionIndex: 1,
          },
          {
            id: 'm2',
            badge: 'ALERT',
            label: 'External alert fired',
            style: TimelineMarkerStyle.REFERENCE,
            paletteId: 'default-reference',
            timeMs: 1_021_000,
            timeLabel: '00:17:01',
          },
        ],
      },
    });

    // Two nearby markers overlap horizontally and pack into 2 lanes -> 2 * 30 + 12 = 72px topBandHeightPx.
    expect(vm.topBandHeightPx).toBe(72);
    expect(vm.markers.length).toBe(2);
    expect(vm.markers[0].pillTopPx).toBe(6);
    expect(vm.markers[0].colors).toEqual(victimPalette);
    expect(vm.markers[1].pillTopPx).toBe(36);
    expect(vm.markers[1].colors).toEqual(DEFAULT_REFERENCE_PALETTE_COLORS);
    // Row 20 starts at 72 (band) + 60 (header) + 50 (row 10) = 182, center = 207.
    expect(vm.markers[0].hasTargetDot).toBe(true);
    expect(vm.markers[0].dotY).toBe(207);
    expect(vm.markers[0].stemX).toBe(defaultChartLeftPx + 200);
    // Reference marker extends to full stageHeightPx (72 + 260 = 332).
    expect(vm.markers[1].hasTargetDot).toBe(false);
    expect(vm.markers[1].stemTopPx + vm.markers[1].stemHeightPx).toBe(
      vm.stageHeightPx,
    );
  });

  it('scopes spans vertically to target rows, skips unrendered row-scoped spans, and clamps right-edge minimum width', () => {
    const vm = buildTimelineSceneViewModel({
      indexPaneWidthPx: MIN_INDEX_PANE_WIDTH_PX,
      viewportLeftTimeMs: 1_000_000,
      pixelsPerMs: 0.01,
      rows: sampleRows,
      overlay: {
        palette: [triggerPalette],
        spans: [
          {
            id: 'sp-scoped',
            label: 'Silent kubelet gap',
            startTimeMs: 1_010_000,
            endTimeMs: 1_040_000,
            durationLabel: '30s',
            timelineIds: [20, 30],
            pattern: TimelineSpanPattern.HATCHED,
            paletteId: 'trigger',
          },
          {
            id: 'sp-unrendered',
            label: 'Hidden row span',
            startTimeMs: 1_010_000,
            endTimeMs: 1_040_000,
            durationLabel: '30s',
            timelineIds: [999],
            pattern: TimelineSpanPattern.SOLID,
            paletteId: 'trigger',
          },
          {
            id: 'sp-right-edge',
            label: 'Edge spike',
            startTimeMs: 1_128_000,
            endTimeMs: 1_128_100,
            durationLabel: '100ms',
            pattern: TimelineSpanPattern.SOLID,
            paletteId: 'trigger',
          },
        ],
      },
    });

    expect(vm.spans.length).toBe(2);
    expect(vm.spans[0]).toEqual({
      id: 'sp-scoped',
      headerText: 'Silent kubelet gap (30s)',
      pattern: TimelineSpanPattern.HATCHED,
      paletteId: 'trigger',
      colors: triggerPalette,
      isRowScoped: true,
      showCornerBrackets: true,
      coincidentLeft: false,
      coincidentRight: false,
      labelLeftPx: 6,
      chipStageLeftPx: defaultChartLeftPx + 106,
      chipStageTopPx: TIMELINE_HEADER_HEIGHT_PX + 56,
      chipMaxWidthPx: 288,
      leftPx: defaultChartLeftPx + 100,
      topPx: TIMELINE_HEADER_HEIGHT_PX + 52,
      widthPx: 300,
      heightPx: 96,
    });
    expect(vm.spans[1].id).toBe('sp-right-edge');
    expect(vm.spans[1].leftPx).toBe(defaultStageWidthPx - 4);
    expect(vm.spans[1].widthPx).toBe(4);
  });

  it('disambiguates coinciding span boundaries, slides labels into stem-free sub-intervals, and suppresses brackets on narrow spans', () => {
    const vm = buildTimelineSceneViewModel({
      indexPaneWidthPx: MIN_INDEX_PANE_WIDTH_PX,
      viewportLeftTimeMs: 1_000_000,
      pixelsPerMs: 0.01,
      rows: sampleRows,
      overlay: {
        palette: [triggerPalette, victimPalette],
        markers: [
          {
            id: 'm-start',
            badge: '1',
            label: 'Outage began',
            style: TimelineMarkerStyle.PIN,
            paletteId: 'victim',
            timeMs: 1_010_000,
            timeLabel: '00:16:50',
            timelineId: 30,
          },
          {
            id: 'm-mid',
            badge: '2',
            label: 'Mid-span alert',
            style: TimelineMarkerStyle.REFERENCE,
            paletteId: 'trigger',
            timeMs: 1_018_000,
            timeLabel: '00:16:58',
          },
          {
            id: 'm-end',
            badge: '3',
            label: 'Recovered',
            style: TimelineMarkerStyle.PIN,
            paletteId: 'trigger',
            timeMs: 1_040_000,
            timeLabel: '00:17:20',
            timelineId: 30,
          },
        ],
        spans: [
          {
            id: 'sp-coinciding',
            label: 'Outage window',
            startTimeMs: 1_010_000,
            endTimeMs: 1_040_000,
            durationLabel: '30s',
            timelineIds: [20, 30],
            pattern: TimelineSpanPattern.SOLID,
            paletteId: 'victim',
          },
          {
            id: 'sp-clear',
            label: 'Recovery window',
            startTimeMs: 1_050_000,
            endTimeMs: 1_080_000,
            durationLabel: '30s',
            pattern: TimelineSpanPattern.OUTLINE,
            paletteId: 'trigger',
          },
          {
            id: 'sp-narrow',
            label: 'Tiny spike',
            startTimeMs: 1_090_000,
            endTimeMs: 1_091_000,
            durationLabel: '1s',
            pattern: TimelineSpanPattern.SOLID,
            paletteId: 'trigger',
          },
        ],
      },
    });

    // sp-coinciding starts at m-start (local 0), crosses m-mid (local 80), and ends at m-end (local 300).
    // Label slides past m-mid to local 86 so no vertical stem crosses the text.
    expect(vm.spans[0].showCornerBrackets).toBe(true);
    expect(vm.spans[0].coincidentLeft).toBe(true);
    expect(vm.spans[0].coincidentRight).toBe(true);
    expect(vm.spans[0].labelLeftPx).toBe(86);
    expect(vm.spans[0].chipStageLeftPx).toBe(vm.spans[0].leftPx + 86);
    expect(vm.spans[0].chipMaxWidthPx).toBe(204);

    // sp-clear has no coinciding or crossing stems -> labelLeftPx stays at 6.
    expect(vm.spans[1].showCornerBrackets).toBe(true);
    expect(vm.spans[1].coincidentLeft).toBe(false);
    expect(vm.spans[1].coincidentRight).toBe(false);
    expect(vm.spans[1].labelLeftPx).toBe(6);
    expect(vm.spans[1].chipMaxWidthPx).toBe(288);

    // sp-narrow has width 10px (< 20px) -> corner brackets are suppressed.
    expect(vm.spans[2].widthPx).toBe(10);
    expect(vm.spans[2].showCornerBrackets).toBe(false);
    expect(vm.spans[2].labelLeftPx).toBe(6);
  });

  it('excludes markers and spans strictly outside the visible viewport time range', () => {
    const marker = (id: string, timeMs: number) => ({
      id,
      badge: '1',
      label: id,
      style: TimelineMarkerStyle.PIN,
      paletteId: 'default',
      timeMs,
      timeLabel: '00:16:39',
    });
    const span = (id: string, startTimeMs: number, endTimeMs: number) => ({
      id,
      label: id,
      startTimeMs,
      endTimeMs,
      durationLabel: '1s',
      pattern: TimelineSpanPattern.SOLID,
      paletteId: 'default',
    });
    const vm = buildTimelineSceneViewModel({
      indexPaneWidthPx: MIN_INDEX_PANE_WIDTH_PX,
      viewportLeftTimeMs: 1_000_000,
      pixelsPerMs: 0.01,
      rows: sampleRows,
      overlay: {
        markers: [marker('m-before', 999_999), marker('m-after', 1_128_001)],
        spans: [
          span('sp-before', 950_000, 999_000),
          span('sp-after', 1_128_001, 1_140_000),
        ],
      },
    });

    expect(vm.markers).toEqual([]);
    expect(vm.spans).toEqual([]);
    expect(vm.topBandHeightPx).toBe(0);
    expect(vm.hasOverlayItems).toBe(false);
  });
});
