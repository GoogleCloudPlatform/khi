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
  Meta,
  moduleMetadata,
  StoryObj,
  componentWrapperDecorator,
  StoryContext,
} from '@storybook/angular';
import { TimelineChartComponent } from './timeline-chart.component';
import { Component, DestroyRef, inject, NgZone, OnInit } from '@angular/core';
import { RenderingLoopManager } from './canvas/rendering-loop-manager';
import {
  generateDefaultChartStyle,
  generateDefaultRulerStyle,
} from './style-model-v2';
import { InspectionDataV2 } from 'src/app/store/domain/inspection-data';
import { createMockInspectionDataV2 } from 'src/app/store/mock/inspection-data.mock';
import { HistogramCache } from 'src/app/timeline/components/misc/histogram-cache';
import { getMinTimeSpanForHistogram } from 'src/app/timeline/components/calculator/human-friendly-tick';
import { RulerViewModelBuilder } from './timeline-ruler.viewmodel';

@Component({
  selector: 'khi-rendering-loop-starter',
  template: `<ng-content></ng-content>`,
  standalone: true,
})
class RenderingLoopStarter implements OnInit {
  private readonly renderingLoopManager = inject(RenderingLoopManager);
  private readonly ngZone = inject(NgZone);
  private readonly destroyRef = inject(DestroyRef);

  ngOnInit() {
    this.renderingLoopManager.start(this.ngZone, this.destroyRef);
  }
}

// Dynamic renderer that consumes mockData from Storybook's loaded context
function renderTimelineChart(
  args: Record<string, unknown>,
  context: StoryContext,
) {
  const mockData = context.loaded['mockData'] as InspectionDataV2;
  const startTimeMs = mockData.metadata!.header!.startTimeUnixSeconds * 1000;
  const endTimeMs = mockData.metadata!.header!.endTimeUnixSeconds * 1000;
  const durationMs = endTimeMs - startTimeMs;

  const timelines = mockData.timelineStore.timelines;
  const logsList = Array.from(mockData.logStore.logs());

  const chartViewModel = {
    inspectionDataUniqueID: 'mock-unique-id',
    timelinesInDrawArea: timelines,
    logBeginTime: startTimeMs,
    logEndTime: endTimeMs,
    styleStore: mockData.styleStore,
  };

  const allLogsCache = new HistogramCache(
    mockData.styleStore.severities,
    logsList,
    getMinTimeSpanForHistogram(10000, startTimeMs, endTimeMs),
    startTimeMs,
    endTimeMs,
  );
  const filteredLogsCache = new HistogramCache(
    mockData.styleStore.severities,
    logsList,
    getMinTimeSpanForHistogram(10000, startTimeMs, endTimeMs),
    startTimeMs,
    endTimeMs,
  );

  const rulerViewModelBuilder = new RulerViewModelBuilder();
  const rulerViewModel = rulerViewModelBuilder.generateRulerViewModel(
    startTimeMs,
    window.innerWidth / durationMs,
    window.innerWidth,
    0,
    allLogsCache,
    filteredLogsCache,
  );

  const activeLogsIndices = new Set<number>();
  for (const log of logsList) {
    activeLogsIndices.add(log.logIndex);
  }

  return {
    props: {
      ...args,
      chartViewModel,
      rulerViewModel,
      activeLogsIndices,
      leftEdgeTime: startTimeMs - 5000,
      pixelsPerMs: window.innerWidth / (durationMs + 10000),
      rulerStyle: generateDefaultRulerStyle(mockData.styleStore),
      chartStyle: generateDefaultChartStyle(),
    },
  };
}

const meta: Meta<TimelineChartComponent> = {
  title: 'Timeline/TimelineChart',
  component: TimelineChartComponent,
  tags: ['autodocs'],
  decorators: [
    moduleMetadata({
      imports: [RenderingLoopStarter],
    }),
    componentWrapperDecorator(
      (story) => `
      <khi-rendering-loop-starter style="height: 100vh; display: grid;">
          ${story}
      </khi-rendering-loop-starter>`,
    ),
  ],
  parameters: {
    layout: 'fullscreen',
  },
  loaders: [
    async () => ({
      mockData: await createMockInspectionDataV2(),
    }),
  ],
  render: renderTimelineChart,
};

export default meta;
type Story = StoryObj<TimelineChartComponent>;

export const Default: Story = {
  args: {},
};

export const NotReady: Story = {
  args: {
    forceNotReadyToRender: true,
  },
};
