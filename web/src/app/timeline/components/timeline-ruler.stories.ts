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

const START_TIME = 1000000;
const DURATION = 60 * 60 * 1000; // 1 hour

function generateMockLogs(count: number, errorRatio: number = 0): LogEntry[] {
  const logs: LogEntry[] = [];
  for (let i = 0; i < count; i++) {
    const time = START_TIME + Math.random() * DURATION;
    const isError = Math.random() < errorRatio;
    logs.push({
      time,
      severity: isError ? Severity.SeverityError : Severity.SeverityInfo,
      // Minimal properties required by HistogramCache (assuming LogEntry interface)
    } as LogEntry);
  }
  return logs;
}

function generateViewModel(
  logs: LogEntry[],
  filteredLogs: LogEntry[] = [],
): TimelineRulerViewModel {
  const calculator = new RulerViewModelBuilder();
  const allLogsCache = new HistogramCache(logs, 1000); // 1s bucket
  const filteredLogsCache = new HistogramCache(
    filteredLogs.length > 0 ? filteredLogs : logs,
    1000,
  );

  return calculator.generateRulerViewModel(
    START_TIME,
    1000 / DURATION, // pixelsPerMs
    1000, // viewportWidth
    0, // timezoneShiftHours
    allLogsCache,
    filteredLogsCache,
  );
}

export const Default: Story = {
  args: {
    viewModel: generateViewModel(generateMockLogs(1000, 0.1)),
    leftEdgeTime: START_TIME,
    pixelsPerMs: 1000 / DURATION,
  },
};

export const NoLogs: Story = {
  args: {
    viewModel: generateViewModel([]),
    leftEdgeTime: START_TIME,
    pixelsPerMs: 1000 / DURATION,
  },
};

export const HighError: Story = {
  args: {
    viewModel: generateViewModel(generateMockLogs(1000, 0.8)),
    leftEdgeTime: START_TIME,
    pixelsPerMs: 1000 / DURATION,
  },
};
