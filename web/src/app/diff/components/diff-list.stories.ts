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
import { ResourceTimeline } from '../../store/timeline';
import {
  ParentRelationship,
  RevisionState,
  RevisionVerb,
  LogType,
  Severity,
} from '../../zzz-generated';
import { ResourceRevision } from '../../store/revision';
import { LogEntry } from '../../store/log';
import { ToTextReferenceFromKHIFileBinary } from '../../common/loader/reference-type';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

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
    LogType.LogTypeAudit,
    Severity.SeverityInfo,
    new Date('2025-01-01T00:00:01Z').getTime(),
    'Updated pod',
    ToTextReferenceFromKHIFileBinary(null),
    [],
  ),
  new LogEntry(
    2,
    'a3',
    LogType.LogTypeAudit,
    Severity.SeverityInfo,
    new Date('2025-01-01T00:00:02Z').getTime(),
    'Deleted pod',
    ToTextReferenceFromKHIFileBinary(null),
    [],
  ),
];

const mockRevisions: ResourceRevision[] = [
  new ResourceRevision(
    mockLogs[0].time,
    mockLogs[1].time,
    RevisionState.RevisionStateExisting,
    RevisionVerb.RevisionVerbCreate,
    'content1',
    'system:serviceaccount:kube-system:replicaset-controller',
    false,
    false,
    0,
  ),
  new ResourceRevision(
    mockLogs[1].time,
    mockLogs[2].time,
    RevisionState.RevisionStateExisting,
    RevisionVerb.RevisionVerbUpdate,
    'content2',
    'user@example.com',
    false,
    false,
    1,
  ),
  new ResourceRevision(
    mockLogs[2].time,
    mockLogs[2].time + 1000,
    RevisionState.RevisionStateDeleted,
    RevisionVerb.RevisionVerbDelete,
    'content3',
    'admin@example.com',
    true,
    false,
    2,
  ),
];

// Add an unknown/inferred revision at the end
mockRevisions.push(
  new ResourceRevision(
    mockLogs[2].time + 1000,
    mockLogs[2].time + 2000,
    RevisionState.RevisionStateInferred,
    RevisionVerb.RevisionVerbUnknown,
    'content4',
    '',
    false,
    true,
    -1,
  ),
);

const mockTimeline = new ResourceTimeline(
  'timeline-id-1',
  'api/v1#pods#default#my-pod',
  mockRevisions,
  [], // events
  ParentRelationship.RelationshipOwnerReference,
);

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
