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
} from '@storybook/angular';
import { TimelineChartComponent } from './timeline-chart.component';
import {
  Component,
  DestroyRef,
  inject,
  NgZone,
  OnInit,
  computed,
  resource,
  input,
} from '@angular/core';
import { RenderingLoopManager } from './canvas/rendering-loop-manager';
import {
  generateDefaultChartStyle,
  generateDefaultRulerStyle,
  TimelineChartStyle,
  TimelineRulerStyle,
} from 'src/app/timeline/components/style-model';
import { createMockInspectionData } from 'src/app/store/mock/inspection-data.mock';
import { HistogramCache } from 'src/app/timeline/components/misc/histogram-cache';
import { getMinTimeSpanForHistogram } from 'src/app/timeline/components/calculator/human-friendly-tick';
import {
  RulerViewModelBuilder,
  TimelineRulerViewModel,
} from 'src/app/timeline/components/timeline-ruler.viewmodel';
import { IdBitset } from 'src/app/store/domain/filter/id-bitset';
import { TimelineChartViewModel } from 'src/app/timeline/components/timeline-chart.viewmodel';

interface TimelineChartStoryViewModelNotReady {
  readonly ready: false;
}

interface TimelineChartStoryViewModelReady {
  readonly ready: true;
  readonly chartViewModel: TimelineChartViewModel;
  readonly rulerViewModel: TimelineRulerViewModel;
  readonly filteredLogIds: IdBitset;
  readonly leftEdgeTime: number;
  readonly pixelsPerMs: number;
  readonly rulerStyle: TimelineRulerStyle;
  readonly chartStyle: TimelineChartStyle;
}

type TimelineChartStoryViewModel =
  TimelineChartStoryViewModelNotReady | TimelineChartStoryViewModelReady;

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

@Component({
  template: `
    @let vm = viewModel();
    @if (vm.ready) {
      <div style="height: 100vh; width: 100vw; display: grid;">
        <khi-timeline-chart
          [chartViewModel]="vm.chartViewModel"
          [rulerViewModel]="vm.rulerViewModel"
          [filteredLogIds]="vm.filteredLogIds"
          [leftEdgeTime]="vm.leftEdgeTime"
          [pixelsPerMs]="vm.pixelsPerMs"
          [rulerStyle]="vm.rulerStyle"
          [chartStyle]="vm.chartStyle"
          [forceNotReadyToRender]="forceNotReadyToRender()"
        ></khi-timeline-chart>
      </div>
    }
  `,
  imports: [TimelineChartComponent],
})
class TimelineChartStoriesComponent {
  readonly forceNotReadyToRender = input(false);

  readonly khiInspectionData = resource({
    loader: async () => {
      return await createMockInspectionData();
    },
  });

  viewModel = computed<TimelineChartStoryViewModel>(() => {
    const mockData = this.khiInspectionData.value();
    if (!mockData) {
      return {
        ready: false,
      };
    }
    const startTimeMs = mockData.metadata!.header!.startTimeUnixSeconds * 1000;
    const endTimeMs = mockData.metadata!.header!.endTimeUnixSeconds * 1000;
    const durationMs = endTimeMs - startTimeMs;

    const timelines = mockData.timelineStore.timelines.slice(0, 50);
    const logsList = Array.from(mockData.logStore.logs());

    const chartViewModel = {
      inspectionDataUniqueID: 'mock-unique-id',
      timelinesInDrawArea: timelines,
      logBeginTime: startTimeMs,
      logEndTime: endTimeMs,
      styleStore: mockData.styleStore,
    };

    const allLogIds = IdBitset.fromAll(logsList.map((log) => log.id));
    const allLogsCache = new HistogramCache(
      mockData.styleStore.severities,
      logsList,
      allLogIds,
      getMinTimeSpanForHistogram(10000, startTimeMs, endTimeMs),
      startTimeMs,
      endTimeMs,
    );
    const filteredLogsCache = new HistogramCache(
      mockData.styleStore.severities,
      logsList,
      allLogIds,
      getMinTimeSpanForHistogram(10000, startTimeMs, endTimeMs),
      startTimeMs,
      endTimeMs,
    );

    const rulerViewModelBuilder = new RulerViewModelBuilder();
    const rulerViewModel = rulerViewModelBuilder.generateRulerViewModel(
      startTimeMs,
      window.innerWidth / (durationMs || 1),
      window.innerWidth,
      0,
      allLogsCache,
      filteredLogsCache,
    );

    const filteredLogIds = IdBitset.fromAll(logsList.map((log) => log.id));

    return {
      ready: true,
      chartViewModel,
      rulerViewModel,
      filteredLogIds,
      leftEdgeTime: startTimeMs - 5000,
      pixelsPerMs: window.innerWidth / (durationMs + 10000),
      rulerStyle: generateDefaultRulerStyle(mockData.styleStore),
      chartStyle: generateDefaultChartStyle(),
    };
  });
}

const meta: Meta<TimelineChartStoriesComponent> = {
  title: 'Timeline/Main/TimelineChart',
  component: TimelineChartStoriesComponent,
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
  argTypes: {
    forceNotReadyToRender: {
      control: 'boolean',
    },
  },
};

export default meta;
type Story = StoryObj<TimelineChartStoriesComponent>;

export const Default: Story = {
  args: {},
};

export const NotReady: Story = {
  args: {
    forceNotReadyToRender: true,
  },
};
