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
import { Severity } from 'src/app/generated';
import {
  RulerViewModelBuilder,
  TimelineRulerViewModel,
} from './timeline-ruler.viewmodel';
import { HistogramCache } from './misc/histogram-cache';
import { LogEntry } from 'src/app/store/log';

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
  },
};

export default meta;
type Story = StoryObj<TimelineRulerComponent>;

const START_TIME = Date.parse('2025-12-31T23:30:00Z');
const DURATION = 60 * 60 * 1000; // 1 hour
const VIEWPORT_WIDTH = window.innerWidth;

function generateMockLogs(
  count: number,
  severityRatio: { [severity in Severity]?: number },
): LogEntry[] {
  const logs: LogEntry[] = [];
  const culmativeRatios: number[] = [];
  for (const severity of [
    Severity.SeverityUnknown,
    Severity.SeverityInfo,
    Severity.SeverityWarning,
    Severity.SeverityError,
    Severity.SeverityFatal,
  ]) {
    const lastRatio: number = culmativeRatios[culmativeRatios.length - 1] || 0;
    culmativeRatios.push(lastRatio + (severityRatio[severity] || 0));
  }
  for (let i = 0; i < count; i++) {
    const time = START_TIME + Math.random() * DURATION;
    const rand = Math.random();
    let severity: Severity = Severity.SeverityInfo;
    for (let j = 0; j < culmativeRatios.length; j++) {
      if (
        rand <
        culmativeRatios[j] / culmativeRatios[culmativeRatios.length - 1]
      ) {
        severity = j as Severity;
        break;
      }
    }
    logs.push({
      time,
      severity,
    } as LogEntry);
  }
  return logs;
}

function generateViewModel(
  logs: LogEntry[],
  filteredLogs: LogEntry[] = logs,
): TimelineRulerViewModel {
  const calculator = new RulerViewModelBuilder();
  const allLogsCache = new HistogramCache(
    logs,
    1000,
    START_TIME,
    START_TIME + DURATION,
  ); // 1s bucket
  const filteredLogsCache = new HistogramCache(
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
  logs: LogEntry[],
  rate: number,
): {
  allLogs: LogEntry[];
  filteredLogs: LogEntry[];
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
      generateMockLogs(100000, {
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
      generateMockLogs(100000, {
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
  generateMockLogs(100000, {
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
