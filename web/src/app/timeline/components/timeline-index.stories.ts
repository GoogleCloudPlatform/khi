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
import { TimelineIndexComponent } from './timeline-index.component';
import { componentWrapperDecorator } from '@storybook/angular';
import { TimelineHighlightType } from './interaction-model';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { StyleStore } from 'src/app/store/domain/style-store';
import { LogStore } from 'src/app/store/domain/log-store';
import { Timeline } from 'src/app/store/domain/timeline';

const meta: Meta<TimelineIndexComponent> = {
  title: 'Timeline/Index',
  component: TimelineIndexComponent,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
  decorators: [
    componentWrapperDecorator(
      (story) => `
      <div style="height: 100vh; width: 200px">
        ${story}
      </div>
    `,
    ),
  ],
  args: {
    timelines: createTimelines(),
  },
  argTypes: {
    hoverOnTimeline: {
      action: 'hoverOnTimeline',
    },
    clickOnTimeline: {
      action: 'clickOnTimeline',
    },
  },
};

export default meta;
type Story = StoryObj<TimelineIndexComponent>;

function createTimelines(): Timeline[] {
  const internPool = new InternPoolStore();
  const styleStore = new StyleStore();
  styleStore.addTimelineTypes([
    {
      id: 1,
      label: 'Kind',
      description: 'Kubernetes Resource Kind',
      icon: 'category',
      backgroundColor: { r: 63 / 255, g: 81 / 255, b: 181 / 255, a: 0.15 },
      foregroundColor: { r: 63 / 255, g: 81 / 255, b: 181 / 255, a: 1 },
      typeChipBackgroundColor: { r: 63 / 255, g: 81 / 255, b: 181 / 255, a: 1 },
      typeChipForegroundColor: { r: 1, g: 1, b: 1, a: 1 },
      visible: true,
      sortPriority: 1,
      height: 0.7,
    },
    {
      id: 2,
      label: 'Namespace',
      description: 'Kubernetes Namespace',
      icon: 'space_dashboard',
      backgroundColor: { r: 100 / 255, g: 100 / 255, b: 100 / 255, a: 0.15 },
      foregroundColor: { r: 100 / 255, g: 100 / 255, b: 100 / 255, a: 1 },
      typeChipBackgroundColor: {
        r: 100 / 255,
        g: 100 / 255,
        b: 100 / 255,
        a: 1,
      },
      typeChipForegroundColor: { r: 1, g: 1, b: 1, a: 1 },
      visible: true,
      sortPriority: 2,
      height: 0.7,
    },
    {
      id: 3,
      label: 'Resource',
      description: 'Kubernetes Resource Instance',
      icon: 'layers',
      backgroundColor: { r: 200 / 255, g: 200 / 255, b: 200 / 255, a: 0.15 },
      foregroundColor: { r: 50 / 255, g: 50 / 255, b: 50 / 255, a: 1 },
      typeChipBackgroundColor: {
        r: 200 / 255,
        g: 200 / 255,
        b: 200 / 255,
        a: 1,
      },
      typeChipForegroundColor: { r: 0.2, g: 0.2, b: 0.2, a: 1 },
      visible: true,
      sortPriority: 3,
      height: 1,
    },
    {
      id: 4,
      label: 'Subresource A',
      description: 'Kubernetes Subresource Type A',
      icon: 'mediation',
      backgroundColor: { r: 245 / 255, g: 245 / 255, b: 245 / 255, a: 0.15 },
      foregroundColor: { r: 100 / 255, g: 100 / 255, b: 100 / 255, a: 1 },
      typeChipBackgroundColor: { r: 1, g: 1, b: 1, a: 1 },
      typeChipForegroundColor: { r: 0.2, g: 0.2, b: 0.2, a: 1 },
      visible: true,
      sortPriority: 4,
      height: 0.5,
    },
    {
      id: 5,
      label: 'Subresource B',
      description: 'Kubernetes Subresource Type B',
      icon: 'schema',
      backgroundColor: { r: 245 / 255, g: 245 / 255, b: 245 / 255, a: 0.15 },
      foregroundColor: { r: 100 / 255, g: 100 / 255, b: 100 / 255, a: 1 },
      typeChipBackgroundColor: { r: 1, g: 1, b: 1, a: 1 },
      typeChipForegroundColor: { r: 0.2, g: 0.2, b: 0.2, a: 1 },
      visible: true,
      sortPriority: 5,
      height: 0.5,
    },
  ]);
  const logStore = new LogStore(internPool, styleStore);
  const timelineStore = new TimelineStore(internPool, styleStore, logStore);

  internPool.addStrings([
    { id: 1, value: 'core/v1' },
    { id: 2, value: 'foo' },
    { id: 3, value: 'resource-1' },
    { id: 4, value: 'sub0' },
    { id: 5, value: 'sub1' },
    { id: 6, value: 'sub2' },
    { id: 7, value: 'very-very-very-very-long-name-subresource' },
  ]);

  const timelinesData = [
    {
      id: 1, // t-kind
      timelineTypeId: 1,
      nameStringId: 1,
      parentTimelineId: 0,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 2, // t-namespace
      timelineTypeId: 2,
      nameStringId: 2,
      parentTimelineId: 1,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 3, // t-resource
      timelineTypeId: 3,
      nameStringId: 3,
      parentTimelineId: 2,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 4, // t-sub0
      timelineTypeId: 4,
      nameStringId: 4,
      parentTimelineId: 3,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 5, // t-sub1
      timelineTypeId: 5,
      nameStringId: 5,
      parentTimelineId: 3,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 6, // t-sub2
      timelineTypeId: 4,
      nameStringId: 6,
      parentTimelineId: 3,
      revisionIds: [],
      eventIds: [],
    },
    {
      id: 9, // t-sub5
      timelineTypeId: 5,
      nameStringId: 7,
      parentTimelineId: 3,
      revisionIds: [],
      eventIds: [],
    },
  ];

  timelineStore.initialize(timelinesData, timelinesData.length, [], 0, [], 0);

  return timelineStore.timelines as Timeline[];
}

export const Default: Story = {};

export const SelectionAndHover: Story = {
  args: {
    highlights: {
      3: TimelineHighlightType.Selected, // t-resource
      4: TimelineHighlightType.ChildrenOfSelected, // t-sub0
      5: TimelineHighlightType.ChildrenOfSelected, // t-sub1
      6: TimelineHighlightType.ChildrenOfSelected, // t-sub2
      9: TimelineHighlightType.Hovered, // t-sub5
    },
  },
};

export const SelectingKind: Story = {
  args: {
    highlights: {
      1: TimelineHighlightType.Selected, // t-kind
    },
  },
};

export const SelectingNamespace: Story = {
  args: {
    highlights: {
      2: TimelineHighlightType.Selected, // t-namespace
    },
  },
};
