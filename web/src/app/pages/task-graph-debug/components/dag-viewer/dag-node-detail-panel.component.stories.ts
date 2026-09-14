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
import { DagNodeDetailPanelComponent } from 'src/app/pages/task-graph-debug/components/dag-viewer/dag-node-detail-panel.component';
import { DagViewerNode } from 'src/app/pages/task-graph-debug/components/dag-viewer/dag-viewer.model';

const mockNode: DagViewerNode = {
  id: 'khi.k8s.pod-parser#d9a4f2',
  referenceId: 'khi.k8s.pod-parser',
  isFeature: false,
  isFormTask: false,
  topologicalOrder: 2,
  priority: 100,
  labels: {
    'khi.google.com/task-type': 'parser',
    'khi.google.com/subsystem': 'kubernetes',
    'khi.google.com/version': 'v1',
  },
  outputType: '*parser.ParsedPod',
  providedTags: [
    {
      $typeName: 'api.v1.ProvidedTagInfo',
      tag: 'parsed-pods',
      priority: 100,
      outputType: '*parser.ParsedPod',
    },
  ],
};

const mockUpstream: DagViewerNode[] = [
  {
    id: 'khi.source.log-reader#01',
    referenceId: 'khi.source.log-reader',
    isFeature: false,
    isFormTask: true,
    topologicalOrder: 0,
    priority: 100,
    labels: {},
    outputType: '*reader.LogStream',
    providedTags: [],
  },
];

const mockDownstream: DagViewerNode[] = [
  {
    id: 'khi.timeline.builder#05',
    referenceId: 'khi.timeline.builder',
    isFeature: false,
    isFormTask: false,
    topologicalOrder: 3,
    priority: 100,
    labels: {},
    outputType: '*timeline.Timeline',
    providedTags: [],
  },
];

const meta: Meta<DagNodeDetailPanelComponent> = {
  title: 'TaskGraphDebug/DagViewer/DagNodeDetailPanel',
  component: DagNodeDetailPanelComponent,
  tags: ['autodocs'],
  args: {
    node: mockNode,
    upstreamNodes: mockUpstream,
    downstreamNodes: mockDownstream,
  },
};

export default meta;
type Story = StoryObj<DagNodeDetailPanelComponent>;

export const Default: Story = {
  args: {},
};

export const FormTaskNoInputs: Story = {
  args: {
    node: {
      ...mockNode,
      id: 'khi.source.log-reader#01',
      referenceId: 'khi.source.log-reader',
      isFormTask: true,
      topologicalOrder: 0,
    },
    upstreamNodes: [],
  },
};
