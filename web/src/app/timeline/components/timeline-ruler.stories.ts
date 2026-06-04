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
  componentWrapperDecorator,
  Meta,
  moduleMetadata,
  StoryObj,
} from '@storybook/angular';
import { Component, DestroyRef, inject, NgZone, OnInit } from '@angular/core';
import { RenderingLoopManager } from './canvas/rendering-loop-manager';
import { TimelineRulerComponent } from './timeline-ruler.component';
import { Severity } from 'src/app/zzz-generated';
import {
  RulerViewModelBuilder,
  TimelineRulerViewModel,
} from './timeline-ruler.viewmodel';
import { HistogramCache } from './misc/histogram-cache';
import { Log } from 'src/app/store/domain/log';
import { LogStore } from 'src/app/store/domain/log-store';
import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { StyleStore } from 'src/app/store/domain/style-store';
import { generateDefaultRulerStyle } from './style-model-v2';

@Component({
  selector: 'khi-rendering-loop-starter',
  template: `<ng-content></ng-content>`,
  imports: [],
})
class RenderingLoopStarter implements OnInit {
  private readonly renderingLoopManager = inject(RenderingLoopManager);
  private readonly ngZone = inject(NgZone);
  private readonly destroyRef = inject(DestroyRef);

  ngOnInit() {
    this.renderingLoopManager.start(this.ngZone, this.destroyRef);
  }
}

const sharedStyleStore = new StyleStore();
sharedStyleStore.addSeverities([
  {
    id: 0,
    label: 'Unknown',
    shortLabel: 'U',
    backgroundColor: { r: 0, g: 0, b: 0, a: 1 },
    foregroundColor: { r: 0.667, g: 0.667, b: 0.667, a: 1 },
    order: 0,
  },
  {
    id: 1,
    label: 'Info',
    shortLabel: 'I',
    backgroundColor: { r: 0, g: 0, b: 1, a: 1 },
    foregroundColor: { r: 0.118, g: 0.533, b: 0.898, a: 1 },
    order: 1,
  },
  {
    id: 2,
    label: 'Warning',
    shortLabel: 'W',
    backgroundColor: { r: 1, g: 0.667, b: 0.267, a: 1 },
    foregroundColor: { r: 0.992, g: 0.847, b: 0.208, a: 1 },
    order: 2,
  },
  {
    id: 3,
    label: 'Error',
    shortLabel: 'E',
    backgroundColor: { r: 1, g: 0.224, b: 0.208, a: 1 },
    foregroundColor: { r: 1, g: 0.533, b: 0.533, a: 1 },
    order: 3,
  },
  {
    id: 4,
    label: 'Fatal',
    shortLabel: 'F',
    backgroundColor: { r: 0.667, g: 0.4, b: 0.667, a: 1 },
    foregroundColor: { r: 1, g: 0.6, b: 1, a: 1 },
    order: 4,
  },
]);
sharedStyleStore.addLogTypes([
  {
    id: 1,
    label: 'K8sAudit',
    description: 'Kubernetes Audit Log',
    backgroundColor: { r: 0, g: 0, b: 0, a: 1 },
    foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
  },
]);

const meta: Meta<TimelineRulerComponent> = {
  title: 'Timeline/TimelineRuler',
  component: TimelineRulerComponent,
  tags: ['autodocs'],
  decorators: [
    moduleMetadata({
      imports: [RenderingLoopStarter],
      providers: [RenderingLoopManager],
    }),
    componentWrapperDecorator(
      (story) => `
      <khi-rendering-loop-starter>
        <div style="width: 100%; height: 400px;">
          ${story}
        </div>
      </khi-rendering-loop-starter>
    `,
    ),
  ],
  argTypes: {
    viewModel: { control: 'object' },
    timezoneShift: { control: 'number' },
  },
  args: {
    rulerStyle: generateDefaultRulerStyle(sharedStyleStore),
  },
};

export default meta;
type Story = StoryObj<TimelineRulerComponent>;

const START_TIME = Date.parse('2025-12-31T23:30:00Z');
const DURATION = 60 * 60 * 24 * 1000; // 24 hour
const VIEWPORT_WIDTH = window.innerWidth;

function generateMockLogs(
  count: number,
  severityRatio: { [severity in Severity]?: number },
): Log[] {
  const internPool = new InternPoolStore();
  const logStore = new LogStore(internPool, sharedStyleStore);

  const culmativeRatios: number[] = [];
  const severitiesList = [
    Severity.SeverityUnknown,
    Severity.SeverityInfo,
    Severity.SeverityWarning,
    Severity.SeverityError,
    Severity.SeverityFatal,
  ];
  for (const severity of severitiesList) {
    const lastRatio: number = culmativeRatios[culmativeRatios.length - 1] || 0;
    culmativeRatios.push(lastRatio + (severityRatio[severity] || 0));
  }

  const logDataList = [];
  for (let i = 0; i < count; i++) {
    const time = START_TIME + Math.random() * DURATION;
    const rand = Math.random();
    let severity: Severity = Severity.SeverityInfo;
    for (let j = 0; j < culmativeRatios.length; j++) {
      if (
        rand <
        culmativeRatios[j] / culmativeRatios[culmativeRatios.length - 1]
      ) {
        severity = severitiesList[j];
        break;
      }
    }
    logDataList.push({
      id: i + 1,
      ts: BigInt(Math.floor(time)) * 1000000n,
      logTypeId: 1,
      severityTypeId: severity,
      summaryStringId: 0,
      body: undefined,
    });
  }
  logDataList.sort((a, b) => Number(a.ts - b.ts));
  logStore.initialize(logDataList, count);
  return Array.from(logStore.logs()) as Log[];
}

function generateViewModel(
  logs: Log[],
  filteredLogs: Log[] = logs,
): TimelineRulerViewModel {
  const calculator = new RulerViewModelBuilder();
  const allLogsCache = new HistogramCache(
    sharedStyleStore.severities,
    logs,
    1000,
    START_TIME,
    START_TIME + DURATION,
  ); // 1s bucket
  const filteredLogsCache = new HistogramCache(
    sharedStyleStore.severities,
    filteredLogs,
    1000,
    START_TIME,
    START_TIME + DURATION,
  );

  return calculator.generateRulerViewModel(
    START_TIME,
    VIEWPORT_WIDTH / DURATION, // pixelsPerMs
    VIEWPORT_WIDTH, // viewportWidth
    0, // timezoneShiftHours
    allLogsCache,
    filteredLogsCache,
  );
}

function filterLogs(
  logs: Log[],
  rate: number,
): {
  allLogs: Log[];
  filteredLogs: Log[];
} {
  const allLogs = logs;
  const filteredLogs = logs.filter(() => {
    return Math.random() < rate;
  });
  return { allLogs, filteredLogs };
}

export const Default: Story = {
  args: {
    viewModel: generateViewModel(
      generateMockLogs(10000, {
        [Severity.SeverityUnknown]: 1,
        [Severity.SeverityInfo]: 1,
        [Severity.SeverityWarning]: 1,
        [Severity.SeverityError]: 1,
        [Severity.SeverityFatal]: 1,
      }),
    ),
    leftEdgeTime: START_TIME,
    pixelsPerMs: VIEWPORT_WIDTH / DURATION,
  },
};

export const NoLogs: Story = {
  args: {
    viewModel: generateViewModel([]),
    leftEdgeTime: START_TIME,
    pixelsPerMs: VIEWPORT_WIDTH / DURATION,
  },
};

export const HighError: Story = {
  args: {
    viewModel: generateViewModel(
      generateMockLogs(10000, {
        [Severity.SeverityUnknown]: 1,
        [Severity.SeverityInfo]: 1,
        [Severity.SeverityWarning]: 1,
        [Severity.SeverityError]: 5,
        [Severity.SeverityFatal]: 1,
      }),
    ),
    leftEdgeTime: START_TIME,
    pixelsPerMs: VIEWPORT_WIDTH / DURATION,
  },
};

const filtered = filterLogs(
  generateMockLogs(10000, {
    [Severity.SeverityUnknown]: 1,
    [Severity.SeverityInfo]: 30,
    [Severity.SeverityWarning]: 10,
    [Severity.SeverityError]: 5,
    [Severity.SeverityFatal]: 1,
  }),
  0.3,
);
export const Filtered: Story = {
  args: {
    viewModel: generateViewModel(filtered.allLogs, filtered.filteredLogs),
    leftEdgeTime: START_TIME,
    pixelsPerMs: VIEWPORT_WIDTH / DURATION,
  },
};

export const WithTimezoneshift: Story = {
  args: {
    viewModel: generateViewModel(
      generateMockLogs(10000, {
        [Severity.SeverityUnknown]: 1,
        [Severity.SeverityInfo]: 1,
        [Severity.SeverityWarning]: 1,
        [Severity.SeverityError]: 1,
        [Severity.SeverityFatal]: 1,
      }),
    ),
    leftEdgeTime: START_TIME,
    pixelsPerMs: VIEWPORT_WIDTH / DURATION,
    timezoneShift: 5.5,
  },
};
