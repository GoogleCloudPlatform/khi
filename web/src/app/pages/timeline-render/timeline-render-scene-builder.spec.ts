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
  TimelineCalloutPlacement,
  TimelineConnectorStyle,
  TimelineExcerptKind,
  TimelineExcerptLineKind,
  TimelineMarkerStyle,
  TimelineResolvedPaletteColors,
  TimelineSpanPattern,
} from 'src/app/common/schema/timeline-render-types';
import {
  TimelineRowStageBounds,
  VerticalStemSegment,
} from 'src/app/pages/timeline-render/types/timeline-render-scene.viewmodel';
import {
  DEFAULT_REFERENCE_PALETTE_COLORS,
  DEFAULT_TIMELINE_PALETTE_COLORS,
  buildCaliperConnectorPath,
  buildConnectorViewModels,
  buildCurveConnectorPath,
  buildMarkerViewModels,
  buildSpanViewModels,
  buildStepConnectorPath,
  buildTimelineSceneViewModel,
  collectMarkerVerticalStems,
  computeStageRowLayout,
  createPaletteColorResolver,
  createStageTimeProjector,
  extractTimelineRowMetrics,
  findStemFreeSpanIntervals,
  layoutSpanHeaderChip,
  packMarkerLanes,
  resolveConnectorPathGeometry,
  resolveSceneHorizontalBounds,
  resolveSpanBoxGeometry,
  resolveSpanVerticalBounds,
  selectVisibleAnnotations,
} from 'src/app/pages/timeline-render/timeline-render-scene-builder';
import {
  ExcerptDeckGridPreset,
  MAX_DECK_CARD_WIDTH_PX,
  MAX_STAGE_WIDTH_PX,
  MIN_CHART_VIEWPORT_WIDTH_PX,
  MIN_INDEX_PANE_WIDTH_PX,
  TIMELINE_GUTTER_WIDTH_PX,
  TIMELINE_HEADER_HEIGHT_PX,
  TextWidthMeasurer,
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

    const undefinedReturningStore = {
      severities: [],
      timelineTypes: [],
      getTimelineType: () => undefined,
    } as unknown as StyleStore;
    expect(
      extractTimelineRowMetrics(timelines, undefinedReturningStore),
    ).toEqual([
      { id: 100, heightPx: 1 * BASE_ROW_HEIGHT },
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

  it('computes zero topBandHeightPx, baseline minimum width, and empty deck when overlay is null', () => {
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
      deckPreset: vm.deck.gridPreset,
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
      deckPreset: ExcerptDeckGridPreset.EMPTY,
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

  it('merges contiguous row highlights, deduplicates repeated IDs, and emits dimmed background row masks', () => {
    const vm = buildTimelineSceneViewModel({
      indexPaneWidthPx: MIN_INDEX_PANE_WIDTH_PX,
      viewportLeftTimeMs: 1_000_000,
      pixelsPerMs: 0.01,
      rows: sampleRows,
      overlay: {
        palette: [victimPalette],
        rowHighlights: [
          {
            id: 'rh-1',
            roleLabel: 'VICTIM · Pod',
            timelineIds: [20, 20, 30],
            paletteId: 'victim',
          },
        ],
        dimUnlistedRows: true,
      },
    });

    expect(vm.rowHighlights).toEqual([
      {
        id: 'rh-1-run-0',
        roleLabel: 'VICTIM · Pod',
        paletteId: 'victim',
        colors: victimPalette,
        topPx: TIMELINE_HEADER_HEIGHT_PX + 50,
        heightPx: 100,
        badgeRightX: 294,
      },
    ]);
    expect(vm.dimmedRowMasks).toEqual([
      { id: 'dim-mask-0', topPx: 60, heightPx: 50 },
      { id: 'dim-mask-1', topPx: 210, heightPx: 50 },
    ]);

    const whitespaceVm = buildTimelineSceneViewModel({
      indexPaneWidthPx: MIN_INDEX_PANE_WIDTH_PX,
      viewportLeftTimeMs: 1_000_000,
      pixelsPerMs: 0.01,
      rows: sampleRows,
      overlay: {
        rowHighlights: [
          {
            id: 'rh-blank',
            roleLabel: '   ',
            timelineIds: [10],
            paletteId: 'default',
          },
        ],
      },
    });
    expect(whitespaceVm.rowHighlights[0].roleLabel).toBe('');
  });

  it('flips auto callouts, skips unrendered targets, orients vertical step arrows, and synchronizes deck cards', () => {
    const vm = buildTimelineSceneViewModel({
      indexPaneWidthPx: MIN_INDEX_PANE_WIDTH_PX,
      viewportLeftTimeMs: 1_000_000,
      pixelsPerMs: 0.01,
      rows: sampleRows,
      overlay: {
        palette: [triggerPalette, victimPalette],
        callouts: [
          {
            id: 'cal-right-edge',
            label: 'Container killed near right border',
            placement: TimelineCalloutPlacement.AUTO,
            dashed: true,
            paletteId: 'trigger',
            timeMs: 1_125_000,
            timelineId: 30,
          },
          {
            id: 'cal-unrendered',
            label: 'Hidden target',
            placement: TimelineCalloutPlacement.RIGHT,
            paletteId: 'trigger',
            timeMs: 1_020_000,
            timelineId: 999,
          },
        ],
        connectors: [
          {
            id: 'conn-curve',
            badge: '1',
            label: '+10s',
            style: TimelineConnectorStyle.CURVE,
            paletteId: 'victim',
            fromTimeMs: 1_010_000,
            fromTimelineId: 10,
            toTimeMs: 1_020_000,
            toTimelineId: 30,
          },
          {
            id: 'conn-caliper',
            label: '15s',
            style: TimelineConnectorStyle.CALIPER,
            paletteId: 'trigger',
            fromTimeMs: 1_020_000,
            fromTimelineId: 20,
            toTimeMs: 1_035_000,
            toTimelineId: 20,
          },
          {
            id: 'conn-step-vertical',
            style: TimelineConnectorStyle.STEP,
            paletteId: 'victim',
            fromTimeMs: 1_025_000,
            fromTimelineId: 10,
            toTimeMs: 1_025_000,
            toTimelineId: 30,
          },
          {
            id: 'conn-zero',
            style: TimelineConnectorStyle.CURVE,
            paletteId: 'victim',
            fromTimeMs: 1_025_000,
            fromTimelineId: 10,
            toTimeMs: 1_025_000,
            toTimelineId: 10,
          },
          {
            id: 'conn-unrendered',
            style: TimelineConnectorStyle.CURVE,
            paletteId: 'victim',
            fromTimeMs: 1_010_000,
            fromTimelineId: 10,
            toTimeMs: 1_020_000,
            toTimelineId: 999,
          },
        ],
        excerpts: [
          {
            id: 'ex-1',
            kind: TimelineExcerptKind.DIFF,
            badge: 'E1',
            paletteId: 'victim',
            timeMs: 1_010_000,
            timeLabel: '00:16:50',
            timelineId: 10,
            resourcePath: 'node · test-node-1',
            title: 'Patch · Resource exists',
            subtitle: '+1 / -1 · 2 diff lines',
            logTimestamp: '2026-10-01T00:16:50.123456789Z',
            insertId: 'audit-node-1-rev2',
            revisionIndex: 2,
            lines: [
              {
                kind: TimelineExcerptLineKind.DELETED,
                oldLineNum: 12,
                text: 'status: True',
              },
              {
                kind: TimelineExcerptLineKind.ADDED,
                newLineNum: 12,
                text: 'status: False',
                parts: [{ text: 'status: ' }, { text: 'False', matched: true }],
              },
            ],
            note: 'Node readiness flipped to False',
          },
          {
            id: 'ex-2',
            kind: TimelineExcerptKind.LOG,
            badge: 'E2',
            paletteId: 'trigger',
            timeMs: 1_030_000,
            timeLabel: '00:17:10',
            timelineId: 30,
            resourcePath: 'pod · default/test-pod-0',
            title: 'k8s-event · Warning',
            subtitle: '5 lines',
            logTimestamp: '2026-10-01T00:17:10Z',
            logId: 42,
            lines: [
              {
                kind: TimelineExcerptLineKind.OMITTED,
                text: '5 lines omitted',
                omitted: 5,
              },
            ],
          },
        ],
      },
    });

    expect(vm.callouts.length).toBe(1);
    expect(vm.callouts[0].cardLeftPx + vm.callouts[0].cardWidthPx).toBeLessThan(
      vm.callouts[0].anchorX,
    );
    expect(vm.connectors.map((c) => c.hasArrowHead)).toEqual([
      true,
      false,
      true,
      false,
    ]);
    // Vertical downward STEP arrow tip is at y=185 (row 30 center), with base points above at y=177.
    expect(vm.connectors[2].arrowHeadPoints).toBe(
      '558,185 553.5,177 562.5,177',
    );
    expect(vm.deck.gridPreset).toBe(ExcerptDeckGridPreset.COLS_2);
    expect(vm.deck.tethers[0].cardAnchorX).toBeLessThan(
      vm.deck.tethers[1].cardAnchorX,
    );
    expect(vm.excerptPins[0].colors).toEqual(victimPalette);
    expect(vm.deck.cards[0].colors).toEqual(victimPalette);
    expect(vm.deck.cards[0].hasQueryMetadata).toBe(true);
    expect(vm.deck.cards[0].logTimestamp).toBe(
      '2026-10-01T00:16:50.123456789Z',
    );
    expect(vm.deck.cards[0].insertId).toBe('audit-node-1-rev2');
    expect(vm.deck.cards[1].colors).toEqual(triggerPalette);
    expect(vm.deck.cards[1].insertId).toBe('');
    expect(vm.deck.cards.map((c) => c.actionLabel)).toEqual([
      'Open Diff',
      'Open Log',
    ]);
    expect(vm.estimatedTotalHeightPx).toBeGreaterThan(vm.stageHeightPx);
  });

  it('caps deck cards at 6 under the 3x2 grid preset', () => {
    const vm = buildTimelineSceneViewModel({
      indexPaneWidthPx: MIN_INDEX_PANE_WIDTH_PX,
      viewportLeftTimeMs: 1_000_000,
      pixelsPerMs: 0.01,
      rows: sampleRows,
      overlay: {
        excerpts: Array.from({ length: 7 }, (_, i) => ({
          id: `ex-${i}`,
          kind: TimelineExcerptKind.LOG,
          badge: `E${i + 1}`,
          paletteId: 'default',
          timeMs: 1_005_000 + i * 5_000,
          timeLabel: '00:16:45',
          timelineId: 10,
          resourcePath: 'pod · p',
          title: 'log',
          subtitle: '1 line',
          lines: [{ kind: TimelineExcerptLineKind.CONTEXT, text: 'err' }],
        })),
      },
    });

    expect(vm.deck.gridPreset).toBe(ExcerptDeckGridPreset.GRID_3X2);
    expect(vm.deck.cards.length).toBe(6);
    expect(vm.excerptPins.length).toBe(6);
    expect(vm.deck.tethers.slice(0, 3).map((t) => t.cardAnchorY)).toEqual([
      32, 32, 32,
    ]);
    for (const secondRowTether of vm.deck.tethers.slice(3, 6)) {
      expect(secondRowTether.cardAnchorY).toBeGreaterThan(32);
      expect(secondRowTether.jointCenterY).toBe(
        secondRowTether.cardAnchorY - 2,
      );
      expect(
        secondRowTether.svgPath.endsWith(
          `${secondRowTether.cardAnchorX} ${secondRowTether.cardAnchorY}`,
        ),
      ).toBe(true);
    }
  });

  it('expands stage width horizontally for wide evidence deck cards and falls back to 2x2 when 4 columns exceed 4096px', () => {
    const fixedMeasurer: TextWidthMeasurer = (text: string) => text.length * 7;
    const makeLogCard = (id: string, index: number, lineText: string) => ({
      id,
      kind: TimelineExcerptKind.LOG,
      badge: `E${index + 1}`,
      paletteId: 'default',
      timeMs: 1_010_000 + index * 5_000,
      timeLabel: '00:16:50',
      timelineId: 10,
      resourcePath: 'pod · test-pod-0',
      title: 'container-log',
      subtitle: '1 line',
      lines: [{ kind: TimelineExcerptLineKind.CONTEXT, text: lineText }],
    });

    // 4 moderate cards (100 chars -> 748px each): 24 + 4 * 748 + 36 = 3052 <= 4096 -> COLS_4.
    const fourColVm = buildTimelineSceneViewModel({
      indexPaneWidthPx: MIN_INDEX_PANE_WIDTH_PX,
      viewportLeftTimeMs: 1_000_000,
      pixelsPerMs: 0.01,
      rows: sampleRows,
      measureText: fixedMeasurer,
      overlay: {
        excerpts: [
          makeLogCard('e1', 0, 'a'.repeat(100)),
          makeLogCard('e2', 1, 'ok'),
          makeLogCard('e3', 2, 'ok'),
          makeLogCard('e4', 3, 'ok'),
        ],
      },
    });
    expect({
      gridPreset: fourColVm.deck.gridPreset,
      columns: fourColVm.deck.columns,
      indexPaneWidthPx: fourColVm.indexPaneWidthPx,
      stageWidthPx: fourColVm.stageWidthPx,
      chartViewportWidthPx: fourColVm.chartViewportWidthPx,
    }).toEqual({
      gridPreset: ExcerptDeckGridPreset.COLS_4,
      columns: 4,
      indexPaneWidthPx: 300,
      stageWidthPx: 3052,
      chartViewportWidthPx: 3052 - 308,
    });

    // 4 ultra-wide cards (200 chars -> 1448px unclamped): 4 cols = 5852 > 4096 -> GRID_2X2 clamped at 1000px.
    const twoByTwoVm = buildTimelineSceneViewModel({
      indexPaneWidthPx: MIN_INDEX_PANE_WIDTH_PX,
      viewportLeftTimeMs: 1_000_000,
      pixelsPerMs: 0.01,
      rows: sampleRows,
      measureText: fixedMeasurer,
      overlay: {
        excerpts: [
          makeLogCard('e1', 0, 'w'.repeat(200)),
          makeLogCard('e2', 1, 'ok'),
          makeLogCard('e3', 2, 'ok'),
          makeLogCard('e4', 3, 'ok'),
        ],
      },
    });
    const wantTwoByTwoWidth = 24 + 2 * MAX_DECK_CARD_WIDTH_PX + 12;
    expect({
      gridPreset: twoByTwoVm.deck.gridPreset,
      columns: twoByTwoVm.deck.columns,
      stageWidthPx: twoByTwoVm.stageWidthPx,
      chartViewportWidthPx: twoByTwoVm.chartViewportWidthPx,
    }).toEqual({
      gridPreset: ExcerptDeckGridPreset.GRID_2X2,
      columns: 2,
      stageWidthPx: wantTwoByTwoWidth,
      chartViewportWidthPx: wantTwoByTwoWidth - 308,
    });
    expect(twoByTwoVm.stageWidthPx).toBeLessThanOrEqual(MAX_STAGE_WIDTH_PX);
    expect(twoByTwoVm.deck.tethers[0].cardAnchorY).toBe(32);
    expect(twoByTwoVm.deck.tethers[1].cardAnchorY).toBe(32);
    expect(twoByTwoVm.deck.tethers[2].cardAnchorY).toBeGreaterThan(32);
    expect(twoByTwoVm.deck.tethers[3].cardAnchorY).toBe(
      twoByTwoVm.deck.tethers[2].cardAnchorY,
    );
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

  it('retains boundary markers at exact viewEndTimeMs under non-power-of-two pixelsPerMs rounding', () => {
    const startTimeMs = 1_000_000;
    const durationMs = 49_000;
    const endTimeMs = startTimeMs + durationMs;
    const vm = buildTimelineSceneViewModel({
      indexPaneWidthPx: MIN_INDEX_PANE_WIDTH_PX,
      viewportLeftTimeMs: startTimeMs,
      pixelsPerMs: MIN_CHART_VIEWPORT_WIDTH_PX / durationMs,
      rows: sampleRows,
      overlay: {
        markers: [
          {
            id: 'm-right-boundary',
            badge: '1',
            label: 'Exact right boundary',
            style: TimelineMarkerStyle.PIN,
            paletteId: 'default',
            timeMs: endTimeMs,
            timeLabel: '00:17:29',
            timelineId: 10,
          },
        ],
      },
    });

    expect(vm.markers.map((m) => m.id)).toEqual(['m-right-boundary']);
    expect(vm.markers[0].stemX).toBe(defaultStageWidthPx);
  });

  it('increases estimated deck height when clamped card columns wrap long log lines', () => {
    const fixedMeasurer: TextWidthMeasurer = (text: string) => text.length * 7;
    const buildWithLine = (lineText: string) =>
      buildTimelineSceneViewModel({
        indexPaneWidthPx: MIN_INDEX_PANE_WIDTH_PX,
        viewportLeftTimeMs: 1_000_000,
        pixelsPerMs: 0.01,
        rows: sampleRows,
        measureText: fixedMeasurer,
        overlay: {
          excerpts: [
            {
              id: 'e1',
              kind: TimelineExcerptKind.LOG,
              badge: 'E1',
              paletteId: 'default',
              timeMs: 1_010_000,
              timeLabel: '00:16:50',
              timelineId: 10,
              resourcePath: 'pod · test-pod-0',
              title: 'container-log',
              subtitle: '1 line',
              lines: [
                { kind: TimelineExcerptLineKind.CONTEXT, text: lineText },
              ],
            },
            {
              id: 'e2',
              kind: TimelineExcerptKind.LOG,
              badge: 'E2',
              paletteId: 'default',
              timeMs: 1_020_000,
              timeLabel: '00:17:00',
              timelineId: 10,
              resourcePath: 'pod · test-pod-0',
              title: 'container-log',
              subtitle: '1 line',
              lines: [{ kind: TimelineExcerptLineKind.CONTEXT, text: 'ok' }],
            },
          ],
        },
      });

    const singleLineVm = buildWithLine('short line');
    const wrappedLineVm = buildWithLine('w'.repeat(200));
    expect(wrappedLineVm.deck.estimatedHeightPx).toBeGreaterThan(
      singleLineVm.deck.estimatedHeightPx,
    );
  });
});

describe('resolveSceneHorizontalBounds', () => {
  it('clamps indexPaneWidthPx below minimum up to MIN_INDEX_PANE_WIDTH_PX', () => {
    const bounds = resolveSceneHorizontalBounds(150);
    expect(bounds.indexPaneWidthPx).toBe(MIN_INDEX_PANE_WIDTH_PX);
    expect(bounds.chartLeftPx).toBe(
      MIN_INDEX_PANE_WIDTH_PX + TIMELINE_GUTTER_WIDTH_PX,
    );
    expect(bounds.chartViewportWidthPx).toBe(MIN_CHART_VIEWPORT_WIDTH_PX);
    expect(bounds.stageWidthPx).toBe(
      MIN_INDEX_PANE_WIDTH_PX +
        TIMELINE_GUTTER_WIDTH_PX +
        MIN_CHART_VIEWPORT_WIDTH_PX,
    );
  });

  it('clamps indexPaneWidthPx above 1200 down to 1200', () => {
    const bounds = resolveSceneHorizontalBounds(2000);
    expect(bounds.indexPaneWidthPx).toBe(1200);
    expect(bounds.chartLeftPx).toBe(1200 + TIMELINE_GUTTER_WIDTH_PX);
  });

  it('handles non-finite values by defaulting to MIN_INDEX_PANE_WIDTH_PX', () => {
    const bounds = resolveSceneHorizontalBounds(Number.NaN);
    expect(bounds.indexPaneWidthPx).toBe(MIN_INDEX_PANE_WIDTH_PX);
  });

  it('caps total stageWidthPx at MAX_STAGE_WIDTH_PX', () => {
    const bounds = resolveSceneHorizontalBounds(1200);
    expect(bounds.stageWidthPx).toBeLessThanOrEqual(MAX_STAGE_WIDTH_PX);
  });
});

describe('createStageTimeProjector', () => {
  it('maps viewportLeftTimeMs to chartLeftPx and clamps sub-viewport times', () => {
    const project = createStageTimeProjector(10_000, 308, 1000, 0.5);
    expect(project(10_000)).toBe(308);
    expect(project(10_100)).toBe(358);
    // Before viewportLeftTimeMs: clamps to chartLeftPx.
    expect(project(9_000)).toBe(308);
    // Far past viewport right: clamps to chartLeftPx + chartViewportWidthPx.
    expect(project(20_000)).toBe(1308);
  });

  it('falls back to 0 pixelsPerMs when non-finite or non-positive value is supplied', () => {
    const project = createStageTimeProjector(10_000, 308, 1000, -1);
    expect(project(15_000)).toBe(308);
  });
});

describe('selectVisibleAnnotations', () => {
  const testMarker = (id: string, timeMs: number) => ({
    id,
    badge: 'M',
    label: id,
    style: TimelineMarkerStyle.PIN,
    paletteId: 'default',
    timeMs,
    timeLabel: '00:00:00',
  });
  const testSpan = (id: string, startMs: number, endMs: number) => ({
    id,
    label: id,
    startTimeMs: startMs,
    endTimeMs: endMs,
    durationLabel: '',
    pattern: TimelineSpanPattern.SOLID,
    paletteId: 'default',
  });

  it('filters annotations to those intersecting the visible chart viewport', () => {
    const overlay = {
      markers: [
        testMarker('m-before', 5_000),
        testMarker('m-inside', 15_000),
        testMarker('m-after', 30_000),
      ],
      spans: [
        testSpan('sp-before', 1_000, 5_000),
        testSpan('sp-straddle', 8_000, 12_000),
        testSpan('sp-inside', 14_000, 16_000),
        testSpan('sp-after', 25_000, 28_000),
      ],
    };
    const visible = selectVisibleAnnotations(overlay, 10_000, 1000, 0.1);
    // Viewport: [10_000, 10_000 + 1000 / 0.1 = 20_000].
    expect(visible.markers.map((m) => m.id)).toEqual(['m-inside']);
    expect(visible.spans.map((s) => s.id)).toEqual([
      'sp-straddle',
      'sp-inside',
    ]);
  });

  it('returns all annotations when pixelsPerMs is zero or non-positive', () => {
    const overlay = {
      markers: [testMarker('m1', 100)],
      spans: [testSpan('sp1', 50, 150)],
    };
    const visible = selectVisibleAnnotations(overlay, 10_000, 1000, 0);
    expect(visible.markers.length).toBe(1);
    expect(visible.spans.length).toBe(1);
  });
});

describe('packMarkerLanes', () => {
  it('returns zero lanes and empty list when markers array is empty', () => {
    const packed = packMarkerLanes([], 1500, (t) => t);
    expect(packed.laneCount).toBe(0);
    expect(packed.items).toEqual([]);
  });

  it('packs distant markers into lane 0 and overlapping markers into lane 1', () => {
    const markers = [
      {
        id: 'm1',
        badge: '1',
        label: 'First',
        style: TimelineMarkerStyle.PIN,
        paletteId: 'default',
        timeMs: 100,
        timeLabel: '00:01',
      },
      {
        id: 'm2',
        badge: '2',
        label: 'Second Overlapping',
        style: TimelineMarkerStyle.PIN,
        paletteId: 'default',
        timeMs: 110,
        timeLabel: '00:02',
      },
      {
        id: 'm3',
        badge: '3',
        label: 'Third Far',
        style: TimelineMarkerStyle.PIN,
        paletteId: 'default',
        timeMs: 2000,
        timeLabel: '00:30',
      },
    ];
    // Projector spreads timeMs directly to x coordinate.
    const packed = packMarkerLanes(markers, 3000, (t) => t);
    expect(packed.items.length).toBe(3);
    expect(packed.items[0].laneIndex).toBe(0);
    expect(packed.items[1].laneIndex).toBe(1);
    expect(packed.items[2].laneIndex).toBe(0);
    expect(packed.laneCount).toBe(2);
  });
});

describe('computeStageRowLayout', () => {
  it('computes layout geometry and row bounds with zero topBandHeightPx when laneCount is 0', () => {
    const rows = [
      { id: 1, heightPx: 40 },
      { id: 2, heightPx: 60 },
    ];
    const layout = computeStageRowLayout(rows, 0);
    expect(layout.topBandHeightPx).toBe(0);
    expect(layout.timelineFrameHeightPx).toBe(
      TIMELINE_HEADER_HEIGHT_PX + 40 + 60,
    );
    expect(layout.stageHeightPx).toBe(layout.timelineFrameHeightPx);
    expect(layout.rowBoundsList.length).toBe(2);
    expect(layout.rowBoundsById.get(1)).toEqual({
      index: 0,
      id: 1,
      topPx: TIMELINE_HEADER_HEIGHT_PX,
      bottomPx: TIMELINE_HEADER_HEIGHT_PX + 40,
      centerPx: TIMELINE_HEADER_HEIGHT_PX + 20,
      heightPx: 40,
    });
    expect(layout.rowBoundsById.get(2)).toEqual({
      index: 1,
      id: 2,
      topPx: TIMELINE_HEADER_HEIGHT_PX + 40,
      bottomPx: TIMELINE_HEADER_HEIGHT_PX + 100,
      centerPx: TIMELINE_HEADER_HEIGHT_PX + 70,
      heightPx: 60,
    });
  });

  it('allocates 30px per marker lane plus 12px padding when laneCount is positive', () => {
    const rows = [{ id: 1, heightPx: 50 }];
    const layout = computeStageRowLayout(rows, 3);
    // 3 * 30 + 12 = 102px.
    expect(layout.topBandHeightPx).toBe(102);
    expect(layout.rowBoundsById.get(1)?.topPx).toBe(
      102 + TIMELINE_HEADER_HEIGHT_PX,
    );
  });
});

describe('createPaletteColorResolver', () => {
  it('resolves color from provided palette or returns fallback', () => {
    const customPalette: TimelineResolvedPaletteColors = {
      id: 'custom',
      main: '#ff0000',
      onMain: '#ffffff',
      soft: 'rgba(255,0,0,0.1)',
      surface: '#fff0f0',
    };
    const resolver = createPaletteColorResolver([customPalette]);
    expect(resolver('custom')).toBe(customPalette);
    expect(resolver('unknown')).toBe(DEFAULT_TIMELINE_PALETTE_COLORS);
    expect(resolver('unknown', DEFAULT_REFERENCE_PALETTE_COLORS)).toBe(
      DEFAULT_REFERENCE_PALETTE_COLORS,
    );
  });
});

describe('buildMarkerViewModels', () => {
  const rowBoundsMap = new Map<number, TimelineRowStageBounds>([
    [
      10,
      {
        index: 0,
        id: 10,
        topPx: 100,
        bottomPx: 150,
        centerPx: 125,
        heightPx: 50,
      },
    ],
  ]);
  const defaultColors = DEFAULT_TIMELINE_PALETTE_COLORS;
  const resolver = () => defaultColors;

  it('constructs PIN marker with target dot at target row center', () => {
    const placements = [
      {
        marker: {
          id: 'm-pin',
          badge: '1',
          label: 'Pin Event',
          timeLabel: '00:01',
          style: TimelineMarkerStyle.PIN,
          paletteId: 'default',
          timeMs: 1000,
          timelineId: 10,
        },
        laneIndex: 0,
        pillLeftPx: 50,
        pillWidthPx: 120,
        stemX: 110,
      },
    ];
    const markers = buildMarkerViewModels(
      placements,
      500,
      rowBoundsMap,
      resolver,
    );
    expect(markers.length).toBe(1);
    expect(markers[0].hasTargetDot).toBe(true);
    expect(markers[0].dotY).toBe(125);
    expect(markers[0].stemHeightPx).toBe(125 - markers[0].stemTopPx);
    expect(markers[0].jumpTarget).toEqual({
      timelineId: 10,
      revisionIndex: undefined,
      logId: undefined,
    });
  });

  it('constructs REFERENCE marker with stem extending to full stage height', () => {
    const placements = [
      {
        marker: {
          id: 'm-ref',
          badge: 'REF',
          label: 'Alert',
          timeLabel: '00:02',
          style: TimelineMarkerStyle.REFERENCE,
          paletteId: 'default-reference',
          timeMs: 1000,
        },
        laneIndex: 1,
        pillLeftPx: 50,
        pillWidthPx: 120,
        stemX: 110,
      },
    ];
    const markers = buildMarkerViewModels(
      placements,
      500,
      rowBoundsMap,
      resolver,
    );
    expect(markers.length).toBe(1);
    expect(markers[0].hasTargetDot).toBe(false);
    expect(markers[0].dotY).toBe(500);
    expect(markers[0].stemTopPx + markers[0].stemHeightPx).toBe(500);
    expect(markers[0].jumpTarget).toBeNull();
  });
});

describe('collectMarkerVerticalStems', () => {
  it('extracts vertical segments for markers with positive stemHeightPx', () => {
    const markers = [
      {
        id: 'm1',
        badge: '1',
        label: 'A',
        timeLabel: '00:01',
        style: TimelineMarkerStyle.PIN,
        paletteId: 'default',
        colors: DEFAULT_TIMELINE_PALETTE_COLORS,
        pillLeftPx: 10,
        pillTopPx: 6,
        pillWidthPx: 100,
        pillHeightPx: 24,
        stemX: 60,
        stemTopPx: 30,
        stemHeightPx: 150,
        hasTargetDot: true,
        dotY: 180,
        jumpTarget: null,
      },
      {
        id: 'm2',
        badge: '2',
        label: 'B',
        timeLabel: '00:02',
        style: TimelineMarkerStyle.PIN,
        paletteId: 'default',
        colors: DEFAULT_TIMELINE_PALETTE_COLORS,
        pillLeftPx: 120,
        pillTopPx: 6,
        pillWidthPx: 100,
        pillHeightPx: 24,
        stemX: 170,
        stemTopPx: 30,
        stemHeightPx: 0,
        hasTargetDot: false,
        dotY: 30,
        jumpTarget: null,
      },
    ];
    const stems = collectMarkerVerticalStems(markers);
    expect(stems).toEqual([{ x: 60, topPx: 30, bottomPx: 180 }]);
  });
});

describe('resolveSpanVerticalBounds', () => {
  const rowBoundsMap = new Map<number, TimelineRowStageBounds>([
    [
      10,
      {
        index: 0,
        id: 10,
        topPx: 100,
        bottomPx: 140,
        centerPx: 120,
        heightPx: 40,
      },
    ],
    [
      20,
      {
        index: 1,
        id: 20,
        topPx: 140,
        bottomPx: 170,
        centerPx: 155,
        heightPx: 30,
      },
    ],
  ]);

  it('spans full stage when timelineIds is omitted', () => {
    const span = {
      id: 'sp1',
      label: 'Full',
      startTimeMs: 100,
      endTimeMs: 200,
      durationLabel: '',
      pattern: TimelineSpanPattern.SOLID,
      paletteId: 'default',
    };
    const bounds = resolveSpanVerticalBounds(span, 60, 400, rowBoundsMap);
    expect(bounds).toEqual({
      topPx: 60,
      bottomPx: 400,
      heightPx: 340,
      isRowScoped: false,
    });
  });

  it('applies 2px inset when union of target rows has raw height >= 20', () => {
    const span = {
      id: 'sp2',
      label: 'Scoped',
      startTimeMs: 100,
      endTimeMs: 200,
      durationLabel: '100ms',
      timelineIds: [10, 20],
      pattern: TimelineSpanPattern.SOLID,
      paletteId: 'default',
    };
    // minTop: 100, maxBottom: 170 -> rawHeight: 70 >= 20 -> 2px inset.
    const bounds = resolveSpanVerticalBounds(span, 60, 400, rowBoundsMap);
    expect(bounds).toEqual({
      topPx: 102,
      bottomPx: 168,
      heightPx: 66,
      isRowScoped: true,
    });
  });

  it('returns null when target row IDs are unmapped in rowBoundsMap', () => {
    const span = {
      id: 'sp-missing',
      label: 'Missing',
      startTimeMs: 100,
      endTimeMs: 200,
      durationLabel: '',
      timelineIds: [999],
      pattern: TimelineSpanPattern.SOLID,
      paletteId: 'default',
    };
    expect(resolveSpanVerticalBounds(span, 60, 400, rowBoundsMap)).toBeNull();
  });
});

describe('resolveSpanBoxGeometry', () => {
  const vertical = {
    topPx: 100,
    bottomPx: 200,
    heightPx: 100,
    isRowScoped: true,
  };
  const toStageX = (t: number) => t;

  it('computes left and width, setting showCornerBrackets when width >= 20', () => {
    const span = {
      id: 'sp1',
      label: 'Wide',
      startTimeMs: 300,
      endTimeMs: 350,
      durationLabel: '50ms',
      pattern: TimelineSpanPattern.SOLID,
      paletteId: 'default',
    };
    const box = resolveSpanBoxGeometry(span, 100, 1000, vertical, toStageX, []);
    expect(box.leftPx).toBe(300);
    expect(box.widthPx).toBe(50);
    expect(box.showCornerBrackets).toBe(true);
    expect(box.coincidentLeft).toBe(false);
    expect(box.coincidentRight).toBe(false);
  });

  it('detects coincidentLeft and coincidentRight when vertical stem matches edge within 3px', () => {
    const span = {
      id: 'sp2',
      label: 'Coincident',
      startTimeMs: 300,
      endTimeMs: 400,
      durationLabel: '100ms',
      pattern: TimelineSpanPattern.SOLID,
      paletteId: 'default',
    };
    const stems: readonly VerticalStemSegment[] = [
      { x: 302, topPx: 50, bottomPx: 250 },
      { x: 399, topPx: 50, bottomPx: 250 },
    ];
    const box = resolveSpanBoxGeometry(
      span,
      100,
      1000,
      vertical,
      toStageX,
      stems,
    );
    expect(box.coincidentLeft).toBe(true);
    expect(box.coincidentRight).toBe(true);
  });
});

describe('findStemFreeSpanIntervals', () => {
  it('sorts local stem Xs ascending and splits intervals around clearance gaps', () => {
    // Unsorted stems at local +50 and +20.
    const stems: readonly VerticalStemSegment[] = [
      { x: 150, topPx: 10, bottomPx: 50 },
      { x: 120, topPx: 10, bottomPx: 50 },
    ];
    // Span starts at leftPx = 100.
    // minLocalLeft = 6, maxLocalRight = 100.
    // Stems at local X = 20 and local X = 50.
    // Stem 20 creates exclusion [14, 26].
    // Stem 50 creates exclusion [44, 56].
    // Intervals: [6, 14], [26, 44], [56, 100].
    const intervals = findStemFreeSpanIntervals(stems, 100, 0, 60, 6, 100);
    expect(intervals).toEqual([
      { start: 6, end: 14 },
      { start: 26, end: 44 },
      { start: 56, end: 100 },
    ]);
  });
});

describe('layoutSpanHeaderChip', () => {
  const vertical = {
    topPx: 100,
    bottomPx: 200,
    heightPx: 100,
    isRowScoped: true,
  };

  it('formats headerText with durationLabel and places chip at minLocalLeft when clear', () => {
    const span = {
      id: 'sp1',
      label: 'Gap',
      durationLabel: '10s',
      startTimeMs: 100,
      endTimeMs: 300,
      pattern: TimelineSpanPattern.SOLID,
      paletteId: 'default',
    };
    const box = {
      leftPx: 200,
      widthPx: 150,
      showCornerBrackets: true,
      coincidentLeft: false,
      coincidentRight: false,
    };
    const chip = layoutSpanHeaderChip(span, box, vertical, []);
    expect(chip.headerText).toBe('Gap (10s)');
    expect(chip.labelLeftPx).toBe(6);
    expect(chip.chipStageLeftPx).toBe(206);
    expect(chip.chipStageTopPx).toBe(104);
  });

  it('shifts label past coincident left edge stem to avoid overlapping line', () => {
    const span = {
      id: 'sp2',
      label: 'Coincident Gap',
      startTimeMs: 100,
      endTimeMs: 300,
      durationLabel: '',
      pattern: TimelineSpanPattern.SOLID,
      paletteId: 'default',
    };
    const box = {
      leftPx: 200,
      widthPx: 150,
      showCornerBrackets: true,
      coincidentLeft: true,
      coincidentRight: false,
    };
    const chip = layoutSpanHeaderChip(span, box, vertical, []);
    expect(chip.labelLeftPx).toBe(14);
    expect(chip.chipStageLeftPx).toBe(214);
  });
});

describe('buildSpanViewModels', () => {
  const rowBoundsMap = new Map<number, TimelineRowStageBounds>([
    [
      10,
      {
        index: 0,
        id: 10,
        topPx: 100,
        bottomPx: 150,
        centerPx: 125,
        heightPx: 50,
      },
    ],
  ]);
  const defaultColors = DEFAULT_TIMELINE_PALETTE_COLORS;
  const resolver = () => defaultColors;

  it('skips spans with unmapped rows and constructs valid SpanItemViewModel', () => {
    const spans = [
      {
        id: 'sp-valid',
        label: 'Valid',
        startTimeMs: 100,
        endTimeMs: 200,
        durationLabel: '100ms',
        timelineIds: [10],
        pattern: TimelineSpanPattern.SOLID,
        paletteId: 'default',
      },
      {
        id: 'sp-invalid',
        label: 'Invalid',
        startTimeMs: 100,
        endTimeMs: 200,
        durationLabel: '',
        timelineIds: [999],
        pattern: TimelineSpanPattern.SOLID,
        paletteId: 'default',
      },
    ];
    const result = buildSpanViewModels(
      spans,
      50,
      1000,
      0,
      400,
      rowBoundsMap,
      (t) => t,
      resolver,
      [],
    );
    expect(result.length).toBe(1);
    expect(result[0].id).toBe('sp-valid');
    expect(result[0].colors).toBe(defaultColors);
  });
});

describe('buildCurveConnectorPath', () => {
  it('computes cubic Bezier SVG path and directional arrowhead', () => {
    const geom = buildCurveConnectorPath(100, 50, 300, 150);
    expect(geom.svgPath).toBe('M 100 50 C 200 50, 200 150, 300 150');
    expect(geom.hasArrowHead).toBe(true);
    expect(geom.arrowHeadPoints.length).toBeGreaterThan(0);
    expect(geom.labelCenterX).toBe(200);
    expect(geom.labelCenterY).toBe(100);
  });

  it('omits arrowhead when endpoints coincide', () => {
    const geom = buildCurveConnectorPath(100, 50, 100, 50);
    expect(geom.svgPath).toBe('M 100 50 C 100 50, 100 50, 100 50');
    expect(geom.hasArrowHead).toBe(false);
    expect(geom.arrowHeadPoints).toBe('');
    expect(geom.labelCenterX).toBe(100);
    expect(geom.labelCenterY).toBe(50);
  });
});

describe('buildStepConnectorPath', () => {
  it('computes orthogonal stepped SVG path and horizontal arrowhead', () => {
    const geom = buildStepConnectorPath(100, 50, 300, 150);
    expect(geom.svgPath).toBe('M 100 50 L 200 50 L 200 150 L 300 150');
    expect(geom.hasArrowHead).toBe(true);
    expect(geom.labelCenterX).toBe(200);
    expect(geom.labelCenterY).toBe(100);
  });

  it('orients vertical arrowhead when endpoints share X coordinate', () => {
    const geom = buildStepConnectorPath(100, 50, 100, 150);
    expect(geom.svgPath).toBe('M 100 50 L 100 50 L 100 150 L 100 150');
    expect(geom.hasArrowHead).toBe(true);
    expect(geom.arrowHeadPoints).toBe('100,150 95.5,142 104.5,142');
  });

  it('omits arrowhead when endpoints coincide', () => {
    const geom = buildStepConnectorPath(100, 50, 100, 50);
    expect(geom.hasArrowHead).toBe(false);
    expect(geom.arrowHeadPoints).toBe('');
  });
});

describe('buildCaliperConnectorPath', () => {
  it('computes tick marks and horizontal bar above min Y', () => {
    const geom = buildCaliperConnectorPath(100, 100, 300, 200, 20);
    const wantCaliperY = 100 - 12; // 88, which is > 20 + 10 = 30
    expect(geom.svgPath).toBe(
      `M 100 ${wantCaliperY - 5} L 100 ${wantCaliperY + 5} M 300 ${wantCaliperY - 5} L 300 ${wantCaliperY + 5} M 100 ${wantCaliperY} L 300 ${wantCaliperY}`,
    );
    expect(geom.hasArrowHead).toBe(false);
    expect(geom.arrowHeadPoints).toBe('');
    expect(geom.labelCenterX).toBe(200);
    expect(geom.labelCenterY).toBe(wantCaliperY - 12);
  });

  it('clamps caliper Y to chartTopPx + 10 when rows are near top', () => {
    const geom = buildCaliperConnectorPath(100, 30, 300, 30, 25);
    // min(30, 30) - 12 = 18 < 25 + 10 = 35 -> clamped to 35
    expect(geom.labelCenterY).toBe(35 - 12);
  });
});

describe('resolveConnectorPathGeometry', () => {
  it('delegates to specific geometry builder according to connector style', () => {
    const curve = resolveConnectorPathGeometry(
      TimelineConnectorStyle.CURVE,
      100,
      50,
      200,
      50,
      0,
    );
    const step = resolveConnectorPathGeometry(
      TimelineConnectorStyle.STEP,
      100,
      50,
      200,
      50,
      0,
    );
    const caliper = resolveConnectorPathGeometry(
      TimelineConnectorStyle.CALIPER,
      100,
      50,
      200,
      50,
      0,
    );
    expect(curve.svgPath).toContain('C');
    expect(step.svgPath).toContain('L');
    expect(caliper.svgPath).toContain('M 200');
  });
});

describe('buildConnectorViewModels', () => {
  const rowBoundsMap = new Map<number, TimelineRowStageBounds>([
    [
      10,
      {
        index: 0,
        id: 10,
        topPx: 40,
        bottomPx: 80,
        centerPx: 60,
        heightPx: 40,
      },
    ],
    [
      20,
      {
        index: 1,
        id: 20,
        topPx: 80,
        bottomPx: 120,
        centerPx: 100,
        heightPx: 40,
      },
    ],
  ]);

  it('skips connectors with unrendered target rows and applies resolved palette colors', () => {
    const connectors = [
      {
        id: 'conn-valid',
        style: TimelineConnectorStyle.CURVE,
        paletteId: 'default',
        fromTimeMs: 1000,
        fromTimelineId: 10,
        toTimeMs: 2000,
        toTimelineId: 20,
      },
      {
        id: 'conn-missing-from',
        style: TimelineConnectorStyle.CURVE,
        paletteId: 'default',
        fromTimeMs: 1000,
        fromTimelineId: 999,
        toTimeMs: 2000,
        toTimelineId: 20,
      },
      {
        id: 'conn-missing-to',
        style: TimelineConnectorStyle.CURVE,
        paletteId: 'default',
        fromTimeMs: 1000,
        fromTimelineId: 10,
        toTimeMs: 2000,
        toTimelineId: 999,
      },
    ];

    const vms = buildConnectorViewModels(
      connectors,
      0,
      rowBoundsMap,
      (t) => t / 10,
      () => DEFAULT_TIMELINE_PALETTE_COLORS,
    );

    expect(vms.length).toBe(1);
    expect(vms[0].id).toBe('conn-valid');
    expect(vms[0].colors).toEqual(DEFAULT_TIMELINE_PALETTE_COLORS);
    expect(vms[0].labelCenterY).toBe(80);
  });
});
