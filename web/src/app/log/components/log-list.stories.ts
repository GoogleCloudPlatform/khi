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
import { createMockInspectionDataV2 } from 'src/app/store/mock/inspection-data.mock';

const meta: Meta<LogListComponent> = {
  title: 'Log/LogList',
  component: LogListComponent,
  tags: ['autodocs'],
  args: {
    allLogsCount: 100,
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
  loaders: [
    async () => ({
      mockData: await createMockInspectionDataV2(),
    }),
  ],
  render: (args, { loaded: { mockData } }) => {
    const filteredLogs = Array.from(mockData.logStore.logs());
    return {
      props: {
        ...args,
        filteredLogs,
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
    };
  },
};
