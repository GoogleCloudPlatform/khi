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
import { TimelineLegendComponent } from './timeline-legend.component';
import { Timeline } from 'src/app/store/domain/timeline';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { StyleStore } from 'src/app/store/domain/style-store';
import { LogStore } from 'src/app/store/domain/log-store';
import { RevisionStateStyle } from 'src/app/store/domain/style';

const meta: Meta<TimelineLegendComponent> = {
  title: 'Timeline/Legend',
  component: TimelineLegendComponent,
  tags: ['autodocs'],
  args: {
    expanded: true,
    legendType: 'revisions',
  },
};

export default meta;
type Story = StoryObj<TimelineLegendComponent>;

function createMockTimeline(isKind: boolean): Timeline {
  const internPool = InternPoolStore.create();
  const styleStore = new StyleStore();
  const logStore = LogStore.create(internPool, styleStore);
  const timelineStore = TimelineStore.create(internPool, styleStore, logStore);

  internPool.addStrings([
    { id: 1, value: 'core/v1#default' },
    { id: 2, value: 'core/v1#pods#default#foo' },
  ]);

  styleStore.addTimelineTypes([
    {
      id: 1,
      label: 'Pod',
      description: 'Kubernetes Pod Resource',
      icon: 'pod',
      backgroundColor: { r: 0.129, g: 0.588, b: 0.953, a: 1 },
      foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
      typeChipBackgroundColor: { r: 0.129, g: 0.588, b: 0.953, a: 1 },
      typeChipForegroundColor: { r: 1, g: 1, b: 1, a: 1 },
      visible: true,
      sortPriority: 1,
      height: 20,
    },
  ]);

  styleStore.addRevisionStates([
    {
      id: 1,
      label: 'Inferred',
      description: 'Inferred state',
      icon: 'help',
      style: RevisionStateStyle.PARTIAL_INFO,
      backgroundColor: { r: 0.62, g: 0.62, b: 0.62, a: 1 },
    },
    {
      id: 2,
      label: 'Provisioning',
      description: 'Provisioning state',
      icon: 'autorenew',
      style: RevisionStateStyle.NORMAL,
      backgroundColor: { r: 1, g: 0.757, b: 0.027, a: 1 },
    },
    {
      id: 3,
      label: 'Deleting',
      description: 'Deleting state',
      icon: 'delete_sweep',
      style: RevisionStateStyle.PARTIAL_INFO,
      backgroundColor: { r: 0.957, g: 0.263, b: 0.212, a: 1 },
    },
    {
      id: 4,
      label: 'Deleted',
      description: 'Deleted state',
      icon: 'delete',
      style: RevisionStateStyle.DELETED,
      backgroundColor: { r: 0.459, g: 0.459, b: 0.459, a: 1 },
    },
  ]);

  styleStore.addLogTypes([
    {
      id: 1,
      label: 'Audit',
      description: 'Audit log',
      backgroundColor: { r: 0.298, g: 0.686, b: 0.314, a: 1 },
      foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
    },
    {
      id: 2,
      label: 'ComputeApi',
      description: 'Compute API log',
      backgroundColor: { r: 0.612, g: 0.153, b: 0.69, a: 1 },
      foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
    },
    {
      id: 3,
      label: 'GkeAudit',
      description: 'GKE Audit log',
      backgroundColor: { r: 0.247, g: 0.318, b: 0.71, a: 1 },
      foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
    },
  ]);

  styleStore.addSeverities([
    {
      id: 1,
      label: 'Info',
      shortLabel: 'I',
      backgroundColor: { r: 0, g: 0, b: 0, a: 0 },
      foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
      order: 1,
    },
  ]);

  styleStore.addVerbs([
    {
      id: 1,
      label: 'Create',
      backgroundColor: { r: 0, g: 0, b: 0, a: 0 },
      foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
      visible: true,
    },
  ]);

  if (isKind) {
    const timelineData = [
      {
        id: 1,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [],
        eventIds: [],
      },
    ];
    timelineStore.initialize(timelineData, 1, [], 0, [], 0);
    return timelineStore.getTimeline(1) as Timeline;
  } else {
    const timelineData = [
      {
        id: 1,
        timelineTypeId: 1,
        nameStringId: 2,
        parentTimelineId: 0,
        revisionIds: [101, 102, 103, 104],
        eventIds: [201, 202, 203],
      },
    ];
    const revisionsData = [
      {
        id: 101,
        logId: 1,
        changedTime: 0n,
        principalStringId: 0,
        verbTypeId: 1,
        stateTypeId: 1,
      },
      {
        id: 102,
        logId: 2,
        changedTime: 1000n,
        principalStringId: 0,
        verbTypeId: 1,
        stateTypeId: 2,
      },
      {
        id: 103,
        logId: 3,
        changedTime: 2000n,
        principalStringId: 0,
        verbTypeId: 1,
        stateTypeId: 3,
      },
      {
        id: 104,
        logId: 4,
        changedTime: 3000n,
        principalStringId: 0,
        verbTypeId: 1,
        stateTypeId: 4,
      },
    ];
    const eventsData = [
      { id: 201, logId: 1 },
      { id: 202, logId: 2 },
      { id: 203, logId: 3 },
    ];
    const logsData = [
      {
        id: 1,
        ts: 0n,
        logTypeId: 1,
        severityTypeId: 1,
        summaryStringId: 0,
        body: undefined,
      },
      {
        id: 2,
        ts: 1000n,
        logTypeId: 2,
        severityTypeId: 1,
        summaryStringId: 0,
        body: undefined,
      },
      {
        id: 3,
        ts: 2000n,
        logTypeId: 3,
        severityTypeId: 1,
        summaryStringId: 0,
        body: undefined,
      },
      {
        id: 4,
        ts: 3000n,
        logTypeId: 1,
        severityTypeId: 1,
        summaryStringId: 0,
        body: undefined,
      },
    ];

    logStore.initialize(logsData, logsData.length);
    timelineStore.initialize(timelineData, 1, revisionsData, 4, eventsData, 3);
    return timelineStore.getTimeline(1) as Timeline;
  }
}

export const Default: Story = {
  args: {
    timeline: createMockTimeline(false),
  },
};

export const NoSelection: Story = {
  args: {
    timeline: null,
  },
};
