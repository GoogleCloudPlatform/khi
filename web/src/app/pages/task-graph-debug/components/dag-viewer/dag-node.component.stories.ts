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

import { Meta, StoryObj, componentWrapperDecorator } from '@storybook/angular';
import { DagNodeComponent } from 'src/app/pages/task-graph-debug/components/dag-viewer/dag-node.component';
import { DagPositionedNode } from 'src/app/pages/task-graph-debug/components/dag-viewer/dag-viewer.model';

const mockNode: DagPositionedNode = {
  id: 'khi.google.com/k8s/pod-parser#d9a4f2',
  referenceId: 'khi.google.com/k8s/pod-parser',
  isFeature: false,
  isFormTask: false,
  topologicalOrder: 3,
  priority: 100,
  labels: {
    'khi.google.com/task-type': 'parser',
  },
  outputType: '*parser.ParsedPod',
  providedTags: [],
  x: 20,
  y: 20,
  width: 420,
  height: 88,
  layer: 1,
};

const meta: Meta<DagNodeComponent> = {
  title: 'TaskGraphDebug/DagViewer/DagNode',
  component: DagNodeComponent,
  tags: ['autodocs'],
  decorators: [
    componentWrapperDecorator(
      (story) => `
        <svg width="460" height="140" style="background: #fafafa; border: 1px solid #e0e0e0; border-radius: 4px;">
          ${story}
        </svg>
      `,
    ),
  ],
  render: (args) => ({
    props: args,
    template: `
      <g
        khi-dag-node
        [node]="node"
        [isSelected]="isSelected"
        [isHighlighted]="isHighlighted"
        [isDimmed]="isDimmed"
      ></g>
    `,
  }),
  args: {
    node: mockNode,
    isSelected: false,
    isHighlighted: false,
    isDimmed: false,
  },
};

export default meta;
type Story = StoryObj<DagNodeComponent>;

export const Default: Story = {
  args: {},
};

export const Selected: Story = {
  args: {
    isSelected: true,
  },
};

export const Highlighted: Story = {
  args: {
    isHighlighted: true,
  },
};

export const Dimmed: Story = {
  args: {
    isDimmed: true,
  },
};

export const FeatureTask: Story = {
  args: {
    node: {
      ...mockNode,
      isFeature: true,
      referenceId: 'khi.google.com/feature/audit-timeline',
      id: 'khi.google.com/feature/audit-timeline#8f3c1a',
    },
  },
};

export const InputFormTask: Story = {
  args: {
    node: {
      ...mockNode,
      isFormTask: true,
      referenceId: 'khi.google.com/source/log-reader',
      id: 'khi.google.com/source/log-reader#0b4e2d',
    },
  },
};

export const FeatureAndInputFormTask: Story = {
  args: {
    node: {
      ...mockNode,
      isFeature: true,
      isFormTask: true,
      priority: 200,
    },
  },
};

export const WithProvidedTags: Story = {
  args: {
    node: {
      ...mockNode,
      providedTags: [
        {
          $typeName: 'api.v1.ProvidedTagInfo',
          tag: 'timeline',
          priority: 100,
          outputType: '[]*timeline.Fragment',
        },
        {
          $typeName: 'api.v1.ProvidedTagInfo',
          tag: 'k8s-event',
          priority: 100,
          outputType: '[]*event.Event',
        },
      ],
      labels: {},
    },
  },
};
