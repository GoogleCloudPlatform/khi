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

import { Meta, StoryObj, moduleMetadata } from '@storybook/angular';
import { DiffListComponent } from './diff-list.component';
import { Timeline } from '../../store/domain/timeline';
import { Log } from '../../store/domain/log';
import { ReadonlyDomainElement } from '../../store/domain/types';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

const mockLogs = [
  {
    id: 1,
    logIndex: 0,
    summary: 'Created pod',
  },
  {
    id: 2,
    logIndex: 1,
    summary: 'Updated pod',
  },
  {
    id: 3,
    logIndex: 2,
    summary: 'Deleted pod',
  },
] as unknown as ReadonlyDomainElement<Log>[];

const mockTimeline = {
  id: 1,
  revisions: [
    {
      id: 1,
      timelineId: 1,
      legacyChangedTimeMs: new Date('2025-01-01T00:00:00Z').getTime(),
      principal: 'system:serviceaccount:kube-system:replicaset-controller',
      verb: {
        id: 1,
        label: 'CREATE',
        backgroundColor: { r: 0.1, g: 0.5, b: 0.9, a: 1.0 },
        foregroundColor: { r: 1.0, g: 1.0, b: 1.0, a: 1.0 },
        visible: true,
      },
      bodyYAML: 'apiVersion: v1\nkind: Pod\nmetadata:\n  name: my-pod\n',
      logIndex: 0,
    },
    {
      id: 2,
      timelineId: 1,
      legacyChangedTimeMs: new Date('2025-01-01T00:00:01Z').getTime(),
      principal: 'user@example.com',
      verb: {
        id: 2,
        label: 'UPDATE',
        backgroundColor: { r: 0.9, g: 0.8, b: 0.2, a: 1.0 },
        foregroundColor: { r: 1.0, g: 1.0, b: 1.0, a: 1.0 },
        visible: true,
      },
      bodyYAML:
        'apiVersion: v1\nkind: Pod\nmetadata:\n  name: my-pod\nspec:\n  containers:\n  - name: nginx\n',
      logIndex: 1,
    },
    {
      id: 3,
      timelineId: 1,
      legacyChangedTimeMs: new Date('2025-01-01T00:00:02Z').getTime(),
      principal: 'admin@example.com',
      verb: {
        id: 3,
        label: 'DELETE',
        backgroundColor: { r: 0.9, g: 0.2, b: 0.2, a: 1.0 },
        foregroundColor: { r: 1.0, g: 1.0, b: 1.0, a: 1.0 },
        visible: true,
      },
      bodyYAML: '',
      logIndex: 2,
    },
  ],
} as unknown as ReadonlyDomainElement<Timeline>;

const meta: Meta<DiffListComponent> = {
  title: 'Diff/DiffList',
  component: DiffListComponent,
  tags: ['autodocs'],
  decorators: [
    moduleMetadata({
      imports: [BrowserAnimationsModule],
    }),
  ],
  args: {
    timeline: mockTimeline,
    logs: mockLogs,
    selectedLogIndex: 1,
    highlightedLogIndices: new Set([0]),
    timezoneShift: 0,
  },
};

export default meta;
type Story = StoryObj<DiffListComponent>;

export const Default: Story = {
  render: (args) => ({
    props: {
      ...args,
    },
    template: `
      <div style="height: 300px; display: flex; flex-direction: column;">
        <khi-diff-list
          [timeline]="timeline"
          [logs]="logs"
          [selectedLogIndex]="selectedLogIndex"
          [highlightedLogIndices]="highlightedLogIndices"
          [timezoneShift]="timezoneShift"
          (selectRevision)="selectRevision($event)"
          (highlightRevision)="highlightRevision($event)"
          (moveSelection)="moveSelection($event)"></khi-diff-list>
      </div>
    `,
  }),
};

export const NoSelection: Story = {
  ...Default,
  args: {
    selectedLogIndex: -1,
    highlightedLogIndices: new Set(),
  },
};

export const WithoutTimeline: Story = {
  ...Default,
  args: {
    timeline: null,
  },
};
