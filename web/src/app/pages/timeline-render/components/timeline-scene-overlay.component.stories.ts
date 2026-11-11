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

import { Meta, StoryObj } from '@storybook/angular';
import {
  TimelineCalloutPlacement,
  TimelineConnectorStyle,
  TimelineExcerptKind,
  TimelineExcerptLineKind,
  TimelineMarkerStyle,
  TimelineSpanPattern,
} from 'src/app/common/schema/timeline-render-types';
import { TimelineSceneOverlayComponent } from 'src/app/pages/timeline-render/components/timeline-scene-overlay.component';
import { buildTimelineSceneViewModel } from 'src/app/pages/timeline-render/timeline-render-scene-builder';

const meta: Meta<TimelineSceneOverlayComponent> = {
  title: 'TimelineRender/TimelineSceneOverlay',
  component: TimelineSceneOverlayComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<TimelineSceneOverlayComponent>;

const mockScene = buildTimelineSceneViewModel({
  indexPaneWidthPx: 420,
  viewportLeftTimeMs: 1_000_000,
  pixelsPerMs: 0.01,
  rows: [
    { id: 1, heightPx: 50 },
    { id: 2, heightPx: 50 },
    { id: 3, heightPx: 50 },
  ],
  overlay: {
    palette: [
      {
        id: 'trigger',
        main: '#b45309',
        onMain: '#fafafa',
        soft: 'rgba(180, 83, 9, 0.13)',
        surface: '#f8eee6',
      },
      {
        id: 'victim',
        main: '#d32f2f',
        onMain: '#fafafa',
        soft: 'rgba(211, 47, 47, 0.13)',
        surface: '#fbeaea',
      },
    ],
    markers: [
      {
        id: 'm1',
        badge: '1',
        label: 'Node NotReady',
        style: TimelineMarkerStyle.PIN,
        paletteId: 'victim',
        timeMs: 1_015_000,
        timeLabel: '10:00:15',
        timelineId: 1,
      },
      {
        id: 'm2',
        badge: 'TICKET',
        label: 'Customer alert reported',
        style: TimelineMarkerStyle.REFERENCE,
        paletteId: 'default-reference',
        timeMs: 1_042_000,
        timeLabel: '10:00:42',
      },
    ],
    spans: [
      {
        id: 'sp1',
        label: 'Missing kubelet logs',
        startTimeMs: 1_015_000,
        endTimeMs: 1_060_000,
        durationLabel: '45s',
        timelineIds: [1, 2],
        pattern: TimelineSpanPattern.HATCHED,
        paletteId: 'trigger',
      },
    ],
    rowHighlights: [
      {
        id: 'rh1',
        roleLabel: 'TRIGGER · Node',
        timelineIds: [1],
        paletteId: 'victim',
      },
    ],
    dimUnlistedRows: true,
    connectors: [
      {
        id: 'c1',
        badge: '1',
        label: '+18s',
        style: TimelineConnectorStyle.CURVE,
        paletteId: 'victim',
        fromTimeMs: 1_015_000,
        fromTimelineId: 1,
        toTimeMs: 1_033_000,
        toTimelineId: 3,
      },
    ],
    callouts: [
      {
        id: 'cal1',
        label: 'Pod evicted after taint',
        placement: TimelineCalloutPlacement.RIGHT,
        paletteId: 'default',
        timeMs: 1_033_000,
        timelineId: 3,
      },
    ],
    excerpts: [
      {
        id: 'ex1',
        kind: TimelineExcerptKind.DIFF,
        badge: 'E1',
        paletteId: 'victim',
        timeMs: 1_015_000,
        timeLabel: '10:00:15',
        timelineId: 1,
        resourcePath: 'node · gke-pool-1',
        title: 'Patch · Resource exists',
        subtitle: '+1 / -1 · 6 diff lines',
        logTimestamp: '2026-10-01T10:00:15Z',
        insertId: 'audit-node-pool-1',
        lines: [
          {
            kind: TimelineExcerptLineKind.ADDED,
            newLineNum: 10,
            text: 'status: False',
          },
        ],
      },
    ],
  },
});

export const Default: Story = {
  args: {
    scene: mockScene,
    interactiveMode: true,
    hoveredBadge: null,
  },
};

const compactRowAboveSpanScene = buildTimelineSceneViewModel({
  indexPaneWidthPx: 420,
  viewportLeftTimeMs: 1_000_000,
  pixelsPerMs: 0.01,
  rows: [
    { id: 1, heightPx: 25 },
    { id: 2, heightPx: 25 },
    { id: 3, heightPx: 50 },
  ],
  overlay: {
    palette: [
      {
        id: 'before',
        main: '#1976d2',
        onMain: '#fafafa',
        soft: 'rgba(25, 118, 210, 0.13)',
        surface: '#e8f1fb',
      },
      {
        id: 'after',
        main: '#b45309',
        onMain: '#fafafa',
        soft: 'rgba(180, 83, 9, 0.13)',
        surface: '#f8eee6',
      },
    ],
    spans: [
      {
        id: 'sp-before',
        label: 'averageUtilization: 20 ~ 27%',
        startTimeMs: 1_010_000,
        endTimeMs: 1_032_000,
        durationLabel: '',
        timelineIds: [2],
        pattern: TimelineSpanPattern.OUTLINE,
        hideDuration: true,
        paletteId: 'before',
      },
      {
        id: 'sp-after',
        label: 'averageUtilization: 1 ~ 2%',
        startTimeMs: 1_035_000,
        endTimeMs: 1_070_000,
        durationLabel: '',
        timelineIds: [2],
        pattern: TimelineSpanPattern.SOLID,
        hideDuration: true,
        paletteId: 'after',
      },
    ],
  },
});

export const CompactRowAboveSpanLabels: Story = {
  args: {
    scene: compactRowAboveSpanScene,
    interactiveMode: true,
    hoveredBadge: null,
  },
};
