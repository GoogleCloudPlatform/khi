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
  DEFAULT_REFERENCE_PALETTE_COLORS,
  ExcerptDeckGridPreset,
  TIMELINE_CHART_LEFT_PX,
  TIMELINE_HEADER_HEIGHT_PX,
  buildTimelineSceneViewModel,
  extractTimelineRowMetrics,
} from 'src/app/pages/timeline-render/types/timeline-render-scene.viewmodel';
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

  it('computes zero topBandHeightPx and empty deck when overlay is null', () => {
    const vm = buildTimelineSceneViewModel({
      stageWidthPx: 1308,
      viewportLeftTimeMs: 1_000_000,
      pixelsPerMs: 0.01,
      rows: sampleRows,
      overlay: null,
    });

    expect({
      topBandHeightPx: vm.topBandHeightPx,
      timelineFrameHeightPx: vm.timelineFrameHeightPx,
      stageHeightPx: vm.stageHeightPx,
      estimatedTotalHeightPx: vm.estimatedTotalHeightPx,
      hasOverlayItems: vm.hasOverlayItems,
      deckPreset: vm.deck.gridPreset,
    }).toEqual({
      topBandHeightPx: 0,
      timelineFrameHeightPx: 260,
      stageHeightPx: 260,
      estimatedTotalHeightPx: 260,
      hasOverlayItems: false,
      deckPreset: ExcerptDeckGridPreset.EMPTY,
    });
  });

  it('staggers overlapping marker pills into separate top-band lanes and resolves named and default-reference palettes', () => {
    const vm = buildTimelineSceneViewModel({
      stageWidthPx: 1308,
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
    expect(vm.markers[0].stemX).toBe(TIMELINE_CHART_LEFT_PX + 200);
    // Reference marker extends to full stageHeightPx (72 + 260 = 332).
    expect(vm.markers[1].hasTargetDot).toBe(false);
    expect(vm.markers[1].stemTopPx + vm.markers[1].stemHeightPx).toBe(
      vm.stageHeightPx,
    );
  });

  it('scopes spans vertically to target rows, skips unrendered row-scoped spans, and clamps right-edge minimum width', () => {
    const vm = buildTimelineSceneViewModel({
      stageWidthPx: 1308,
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
            startTimeMs: 1_100_000,
            endTimeMs: 1_100_100,
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
      chipStageLeftPx: TIMELINE_CHART_LEFT_PX + 106,
      chipStageTopPx: TIMELINE_HEADER_HEIGHT_PX + 56,
      chipMaxWidthPx: 288,
      leftPx: TIMELINE_CHART_LEFT_PX + 100,
      topPx: TIMELINE_HEADER_HEIGHT_PX + 52,
      widthPx: 300,
      heightPx: 96,
    });
    expect(vm.spans[1].id).toBe('sp-right-edge');
    expect(vm.spans[1].leftPx).toBe(1304);
    expect(vm.spans[1].widthPx).toBe(4);
  });

  it('merges contiguous row highlights, deduplicates repeated IDs, and emits dimmed background row masks', () => {
    const vm = buildTimelineSceneViewModel({
      stageWidthPx: 1308,
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
  });

  it('flips auto callouts, skips unrendered targets, orients vertical step arrows, and synchronizes deck cards', () => {
    const vm = buildTimelineSceneViewModel({
      stageWidthPx: 1600,
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
      stageWidthPx: 1308,
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
          lines: [{ kind: TimelineExcerptLineKind.LOG, text: 'err' }],
        })),
      },
    });

    expect(vm.deck.gridPreset).toBe(ExcerptDeckGridPreset.GRID_3X2);
    expect(vm.deck.cards.length).toBe(6);
    expect(vm.excerptPins.length).toBe(6);
  });

  it('disambiguates coinciding span boundaries, slides labels into stem-free sub-intervals, and suppresses brackets on narrow spans', () => {
    const vm = buildTimelineSceneViewModel({
      stageWidthPx: 1308,
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
});
