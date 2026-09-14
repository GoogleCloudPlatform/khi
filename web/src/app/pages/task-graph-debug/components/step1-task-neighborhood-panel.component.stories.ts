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
import {
  RegisteredTaskGroupInfo,
  RegisteredTaskInfo,
  TaskDependencyCardinality,
  TaskDependencyScope,
} from 'src/app/generated/api/v1/inspection_task_graph_pb';
import { Step1TaskNeighborhoodPanelComponent } from 'src/app/pages/task-graph-debug/components/step1-task-neighborhood-panel.component';

const meta: Meta<Step1TaskNeighborhoodPanelComponent> = {
  title: 'TaskGraphDebug/Step1TaskNeighborhoodPanel',
  component: Step1TaskNeighborhoodPanelComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<Step1TaskNeighborhoodPanelComponent>;

const mockTaskGroups: readonly RegisteredTaskGroupInfo[] = [
  {
    $typeName: 'api.v1.RegisteredTaskGroupInfo',
    taskReferenceId: 'input.k8s.audit-log',
    tasks: [
      {
        $typeName: 'api.v1.RegisteredTaskInfo',
        taskImplementationId: 'input.k8s.audit-log.reader',
        taskReferenceId: 'input.k8s.audit-log',
        priority: 100,
        isFeature: false,
        isDefaultFeature: false,
        featureDescription: '',
        dependencies: [],
        labels: {},
        outputType: '*audit.LogReader',
        providedTags: [
          {
            tag: 'raw-k8s-audit',
            outputType: '[]*audit.RawLog',
          },
        ],
      } as unknown as RegisteredTaskInfo,
    ],
  } as unknown as RegisteredTaskGroupInfo,
  {
    $typeName: 'api.v1.RegisteredTaskGroupInfo',
    taskReferenceId: 'parser.k8s.audit-log',
    tasks: [
      {
        $typeName: 'api.v1.RegisteredTaskInfo',
        taskImplementationId: 'parser.k8s.audit-log.v2',
        taskReferenceId: 'parser.k8s.audit-log',
        priority: 100,
        isFeature: false,
        isDefaultFeature: false,
        featureDescription: '',
        dependencies: [
          {
            $typeName: 'api.v1.TaskDependencyInfo',
            cardinality: TaskDependencyCardinality.FAN_IN,
            scope: TaskDependencyScope.ACTIVE_GRAPH,
            targetReferenceId: '',
            targetTag: 'raw-k8s-audit',
            outputType: '[]*audit.RawLog',
          },
        ],
        labels: {},
        outputType: '*parser.AuditParserResult',
        providedTags: [
          {
            tag: 'parsed-audit-events',
            outputType: '[]*audit.ParsedEvent',
          },
        ],
      } as unknown as RegisteredTaskInfo,
      {
        $typeName: 'api.v1.RegisteredTaskInfo',
        taskImplementationId: 'parser.k8s.audit-log.legacy',
        taskReferenceId: 'parser.k8s.audit-log',
        priority: 50,
        isFeature: true,
        isDefaultFeature: false,
        featureDescription: 'Legacy audit parser fallback',
        dependencies: [
          {
            $typeName: 'api.v1.TaskDependencyInfo',
            cardinality: TaskDependencyCardinality.POINT_TO_POINT,
            scope: TaskDependencyScope.ALL,
            targetReferenceId: 'input.k8s.audit-log',
            targetTag: '',
            outputType: '*audit.LogReader',
          },
        ],
        labels: {},
        outputType: '*parser.LegacyAuditResult',
        providedTags: [],
      } as unknown as RegisteredTaskInfo,
    ],
  } as unknown as RegisteredTaskGroupInfo,
  {
    $typeName: 'api.v1.RegisteredTaskGroupInfo',
    taskReferenceId: 'timeline.k8s.events',
    tasks: [
      {
        $typeName: 'api.v1.RegisteredTaskInfo',
        taskImplementationId: 'timeline.k8s.events.builder',
        taskReferenceId: 'timeline.k8s.events',
        priority: 100,
        isFeature: false,
        isDefaultFeature: false,
        featureDescription: '',
        dependencies: [
          {
            $typeName: 'api.v1.TaskDependencyInfo',
            cardinality: TaskDependencyCardinality.FAN_IN,
            scope: TaskDependencyScope.ACTIVE_GRAPH,
            targetReferenceId: '',
            targetTag: 'parsed-audit-events',
            outputType: '[]*audit.ParsedEvent',
          },
        ],
        labels: {},
        outputType: '*timeline.Timeline',
        providedTags: [],
      } as unknown as RegisteredTaskInfo,
    ],
  } as unknown as RegisteredTaskGroupInfo,
];

export const MiddleTaskSelected: Story = {
  args: {
    taskGroups: mockTaskGroups,
    centerTaskId: 'parser.k8s.audit-log.v2',
    canGoBack: false,
    canGoForward: false,
  },
};

export const EntryTaskSelected: Story = {
  args: {
    taskGroups: mockTaskGroups,
    centerTaskId: 'input.k8s.audit-log.reader',
    canGoBack: false,
    canGoForward: false,
  },
};

export const LeafTaskSelected: Story = {
  args: {
    taskGroups: mockTaskGroups,
    centerTaskId: 'timeline.k8s.events.builder',
    canGoBack: true,
    canGoForward: false,
  },
};

export const WithHistoryNavigation: Story = {
  args: {
    taskGroups: mockTaskGroups,
    centerTaskId: 'parser.k8s.audit-log.v2',
    canGoBack: true,
    canGoForward: true,
  },
};
