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
import { TimelineRenderLayoutComponent } from 'src/app/pages/timeline-render/components/timeline-render-layout.component';
import { buildTimelineSceneViewModel } from 'src/app/pages/timeline-render/timeline-render-scene-builder';
import { IdBitset } from 'src/app/store/domain/filter/id-bitset';
import { StyleStoreLike } from 'src/app/store/domain/style-store';
import {
  generateDefaultChartStyle,
  generateDefaultRulerStyle,
} from 'src/app/timeline/components/style-model';

const meta: Meta<TimelineRenderLayoutComponent> = {
  title: 'TimelineRender/TimelineRenderLayout',
  component: TimelineRenderLayoutComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<TimelineRenderLayoutComponent>;

const mockStyleStore = {
  severities: [],
  timelineTypes: [],
  getTimelineType: () => undefined,
} as unknown as StyleStoreLike;

const mockScene = buildTimelineSceneViewModel({
  indexPaneWidthPx: 300,
  viewportLeftTimeMs: 1_000_000,
  pixelsPerMs: 0.01,
  rows: [{ id: 1, heightPx: 50 }],
  overlay: null,
});

export const InteractiveErrorBanner: Story = {
  args: {
    inspectionDataUniqueID: 'story-1',
    styleStore: mockStyleStore,
    chartStyle: generateDefaultChartStyle(),
    rulerStyle: generateDefaultRulerStyle(mockStyleStore),
    timelines: [],
    collapsedTimelineIds: new Set<number>(),
    timezoneShiftHours: 0,
    pixelsPerNs: 1e-8,
    viewportLeftTimeNs: 1_000_000_000_000n,
    minQueryLogTimeNs: 1_000_000_000_000n,
    maxQueryLogTimeNs: 2_000_000_000_000n,
    allLogs: [],
    filteredLogIds: IdBitset.createEmpty(),
    scene: mockScene,
    interactiveMode: true,
    sceneId: 'scn_expired01',
    inspectionId: '',
    errorMessage: 'Scene scn_expired01 has expired.',
    hoveredBadge: null,
  },
};
