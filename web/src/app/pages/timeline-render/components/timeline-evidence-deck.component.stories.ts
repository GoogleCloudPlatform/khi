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
        id: 'ex-manifest',
        kind: TimelineExcerptKind.MANIFEST,
        badge: 'E2',
        paletteId: 'trigger',
        timeMs: 1_024_000,
        timeLabel: '10:00:24',
        timelineId: 2,
        resourcePath: 'horizontalpodautoscaler · prod/checkout-hpa',
        title: 'Manifest · Resource exists',
        subtitle: '28 lines',
        revisionIndex: 0,
        lines: [
          {
            kind: TimelineExcerptLineKind.OMITTED,
            text: '⋮ 11 lines omitted',
            omitted: 11,
          },
          {
            kind: TimelineExcerptLineKind.CONTEXT,
            newLineNum: 12,
            text: '      target:',
          },
          {
            kind: TimelineExcerptLineKind.CONTEXT,
            newLineNum: 13,
            text: '        averageUtilization: 40',
            parts: [
              { text: '        ' },
              { text: 'averageUtilization: 40', matched: true },
            ],
          },
        ],
        note: 'Static CPU target threshold remained 40% throughout incident',
      },
      {
        id: 'ex2',
        kind: TimelineExcerptKind.LOG,
        badge: 'E3',
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

const wrappedGridScene = buildTimelineSceneViewModel({
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
      {
        id: 'exporter',
        main: '#5e35b1',
        onMain: '#fafafa',
        soft: 'rgba(94, 53, 177, 0.13)',
        surface: '#efebf7',
      },
    ],
    excerpts: [
      {
        id: 'ex-grid-1',
        kind: TimelineExcerptKind.DIFF,
        badge: 'E1',
        paletteId: 'trigger',
        timeMs: 1_012_000,
        timeLabel: '10:00:12',
        timelineId: 1,
        resourcePath: 'node · gke-pool-1',
        title: 'Patch · Resource exists',
        subtitle: '+2 / -2 · 14 diff lines',
        revisionIndex: 2,
        lines: [
          {
            kind: TimelineExcerptLineKind.DELETED,
            oldLineNum: 16,
            text: '  lastHeartbeatTime: "2026-10-01T09:55:12Z"',
          },
          {
            kind: TimelineExcerptLineKind.ADDED,
            newLineNum: 16,
            text: '  lastHeartbeatTime: "2026-10-01T10:00:12Z"',
          },
          {
            kind: TimelineExcerptLineKind.CONTEXT,
            oldLineNum: 17,
            newLineNum: 17,
            text: '  reason: NodeSysctlChange',
          },
        ],
        note: 'Node heartbeat update before HPA conflict',
      },
      {
        id: 'ex-grid-2',
        kind: TimelineExcerptKind.DIFF,
        badge: 'E2',
        paletteId: 'victim',
        timeMs: 1_025_000,
        timeLabel: '10:00:25',
        timelineId: 2,
        resourcePath: 'horizontalpodautoscaler · gke-managed-cim/metrics',
        title: 'Update · Resource exists',
        subtitle: '+1 / -1 · 8 diff lines',
        revisionIndex: 4,
        lines: [
          {
            kind: TimelineExcerptLineKind.DELETED,
            oldLineNum: 13,
            text: '  resourceVersion: "1789351196414191018"',
          },
          {
            kind: TimelineExcerptLineKind.ADDED,
            newLineNum: 13,
            text: '  resourceVersion: "1789351406375103018"',
          },
        ],
        note: 'HPA resourceVersion advanced concurrently',
      },
      {
        id: 'ex-grid-3',
        kind: TimelineExcerptKind.LOG,
        badge: 'E3',
        paletteId: 'victim',
        timeMs: 1_025_050,
        timeLabel: '10:00:25',
        timelineId: 2,
        resourcePath: 'horizontalpodautoscaler · gke-managed-cim/metrics',
        title: 'k8s audit · ERROR',
        subtitle: '183 lines',
        logId: 11601,
        lines: [
          {
            kind: TimelineExcerptLineKind.SUMMARY,
            text: 'Operation cannot be fulfilled on horizontalpodautoscalers.autoscaling "metrics": the object has been modified; please apply your changes to the latest version and try again',
          },
          {
            kind: TimelineExcerptLineKind.CONTEXT,
            newLineNum: 166,
            text: '  code: 409',
            parts: [{ text: '  ' }, { text: 'code: 409', matched: true }],
          },
        ],
        note: 'vpa-recommender status update rejected with HTTP 409 Conflict',
      },
      {
        id: 'ex-grid-4',
        kind: TimelineExcerptKind.LOG,
        badge: 'E4',
        paletteId: 'exporter',
        timeMs: 1_090_000,
        timeLabel: '10:01:30',
        timelineId: 2,
        resourcePath: 'pod · kube-system/core-metrics-exporter',
        title: 'container · ERROR',
        subtitle: '21 lines',
        logId: 81953,
        lines: [
          {
            kind: TimelineExcerptLineKind.SUMMARY,
            text: 'Exporting failed: rpc error: code = DeadlineExceeded desc = context deadline exceeded while flushing metric batch to Google Cloud Monitoring',
          },
        ],
        note: 'Exporter RPC deadline exceeded during contention',
      },
    ],
  },
});

export const WrappedTwoByTwoGrid: Story = {
  args: {
    deck: wrappedGridScene.deck,
    stageWidthPx: wrappedGridScene.stageWidthPx,
    interactiveMode: true,
    hoveredBadge: null,
  },
};
