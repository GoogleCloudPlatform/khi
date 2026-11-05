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
  TimelineExcerptKind,
  TimelineExcerptLineKind,
} from 'src/app/common/schema/timeline-render-types';
import { TimelineEvidenceDeckComponent } from 'src/app/pages/timeline-render/components/timeline-evidence-deck.component';
import { buildTimelineSceneViewModel } from 'src/app/pages/timeline-render/timeline-render-scene-builder';

const meta: Meta<TimelineEvidenceDeckComponent> = {
  title: 'TimelineRender/TimelineEvidenceDeck',
  component: TimelineEvidenceDeckComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<TimelineEvidenceDeckComponent>;

const mockScene = buildTimelineSceneViewModel({
  indexPaneWidthPx: 420,
  viewportLeftTimeMs: 1_000_000,
  pixelsPerMs: 0.01,
  rows: [
    { id: 1, heightPx: 50 },
    { id: 2, heightPx: 50 },
  ],
  overlay: {
    palette: [
      {
        id: 'victim',
        main: '#d32f2f',
        onMain: '#fafafa',
        soft: 'rgba(211, 47, 47, 0.13)',
        surface: '#fbeaea',
      },
      {
        id: 'trigger',
        main: '#b45309',
        onMain: '#fafafa',
        soft: 'rgba(180, 83, 9, 0.13)',
        surface: '#f8eee6',
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
        subtitle: '+2 / -1 · 12 diff lines',
        logTimestamp: '2026-10-01T10:00:15.204918Z',
        insertId: '18f9c2a4e7b01d3f',
        revisionIndex: 3,
        lines: [
          {
            kind: TimelineExcerptLineKind.CONTEXT,
            oldLineNum: 18,
            newLineNum: 18,
            text: '  type: Ready',
          },
          {
            kind: TimelineExcerptLineKind.DELETED,
            oldLineNum: 19,
            text: '  status: "True"',
          },
          {
            kind: TimelineExcerptLineKind.ADDED,
            newLineNum: 19,
            text: '  status: "False"',
            parts: [{ text: '  status: ' }, { text: '"False"', matched: true }],
          },
          {
            kind: TimelineExcerptLineKind.OMITTED,
            text: '6 unchanged lines omitted',
            omitted: 6,
          },
          {
            kind: TimelineExcerptLineKind.ADDED,
            newLineNum: 26,
            text: '  reason: KubeletNotReady',
            parts: [
              { text: '  reason: ' },
              { text: 'KubeletNotReady', matched: true },
            ],
          },
        ],
        note: 'Node readiness flipped to False due to PLEG stall',
      },
      {
        id: 'ex2',
        kind: TimelineExcerptKind.LOG,
        badge: 'E2',
        paletteId: 'trigger',
        timeMs: 1_033_000,
        timeLabel: '10:00:33',
        timelineId: 2,
        resourcePath: 'pod · prod/checkout-7d9c',
        title: 'k8s-event · Warning',
        subtitle: '9 lines',
        logTimestamp: '2026-10-01T10:00:33.008412Z',
        insertId: 'evt-probe-checkout-7d9c',
        logId: 1094,
        lines: [
          {
            kind: TimelineExcerptLineKind.SUMMARY,
            text: 'Liveness probe failed: HTTP 503',
          },
          {
            kind: TimelineExcerptLineKind.CONTEXT,
            newLineNum: 7,
            text: 'reason: Unhealthy',
            parts: [{ text: 'reason: ' }, { text: 'Unhealthy', matched: true }],
          },
        ],
        note: 'Container killed after three consecutive probe failures',
      },
    ],
  },
});

export const Default: Story = {
  args: {
    deck: mockScene.deck,
    stageWidthPx: mockScene.stageWidthPx,
    interactiveMode: true,
    hoveredBadge: null,
  },
};
