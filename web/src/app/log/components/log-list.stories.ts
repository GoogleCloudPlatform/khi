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
import { LogListComponent } from './log-list.component';
import { LogEntry } from 'src/app/store/log';
import { LogType, Severity } from 'src/app/zzz-generated';
import { ToTextReferenceFromKHIFileBinary } from 'src/app/common/loader/reference-type';

const mockLogs: LogEntry[] = [
  new LogEntry(
    0,
    'a1',
    LogType.LogTypeAudit,
    Severity.SeverityInfo,
    new Date('2025-01-01T00:00:00Z').getTime(),
    'Created pod',
    ToTextReferenceFromKHIFileBinary(null),
    [],
  ),
  new LogEntry(
    1,
    'a2',
    LogType.LogTypeNode,
    Severity.SeverityError,
    new Date('2025-01-01T00:00:01Z').getTime(),
    'Failed to pull image',
    ToTextReferenceFromKHIFileBinary(null),
    [],
  ),
];

const meta: Meta<LogListComponent> = {
  title: 'Log/LogList',
  component: LogListComponent,
  tags: ['autodocs'],
  args: {
    allLogsCount: 100,
    filteredLogs: mockLogs,
    selectedLogIndex: 1,
    highlightLogIndices: new Set([0]),
    selectedTimelinesWithChildren: [],
    filterByTimeline: true,
    includeTimelineChildren: false,
  },
};

export default meta;
type Story = StoryObj<LogListComponent>;

export const Default: Story = {
  render: (args) => ({
    props: {
      ...args,
    },
    template: `
      <div style="height: 500px; border: 1px solid #ccc; position: relative;">
        <khi-log-list
          [allLogsCount]="allLogsCount"
          [filteredLogs]="filteredLogs"
          [selectedLogIndex]="selectedLogIndex"
          [highlightLogIndices]="highlightLogIndices"
          [selectedTimelinesWithChildren]="selectedTimelinesWithChildren"
          [filterByTimeline]="filterByTimeline"
          (filterByTimelineChange)="filterByTimelineChange($event)"
          [includeTimelineChildren]="includeTimelineChildren"
          (includeTimelineChildrenChange)="includeTimelineChildrenChange($event)"></khi-log-list>
      </div>
    `,
  }),
};
