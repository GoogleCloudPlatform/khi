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
import { Step1RegistryTableComponent } from './step1-registry-table.component';

const meta: Meta<Step1RegistryTableComponent> = {
  title: 'TaskGraphDebug/Step1RegistryTable',
  component: Step1RegistryTableComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<Step1RegistryTableComponent>;

const mockGroups: RegisteredTaskGroupInfo[] = [
  {
    taskReferenceId: 'parser.k8s.audit',
    tasks: [
      {
        taskImplementationId: 'parser.k8s.audit-log',
        taskReferenceId: 'parser.k8s.audit',
        priority: 100,
        isFeature: false,
        isDefaultFeature: false,
        featureLabel: '',
        featureDescription: '',
        dependencies: [],
        selectorRequirements: [],
        compatibleInspectionTypes: [],
        labels: {
          log_type: 'audit',
          'khi.google.com/inspection/is-form-task': 'true',
        },
        outputType: '*parser.ParsedAuditLogs',
        providedTags: [],
      } as unknown as RegisteredTaskInfo,
      {
        taskImplementationId: 'parser.k8s.audit-log-specialized',
        taskReferenceId: 'parser.k8s.audit',
        priority: 200,
        isFeature: false,
        isDefaultFeature: false,
        featureLabel: '',
        featureDescription: '',
        dependencies: [],
        selectorRequirements: [],
        compatibleInspectionTypes: [],
        labels: {
          log_type: 'audit',
          environment: 'production',
        },
        outputType: '*parser.ParsedAuditLogs',
        providedTags: [],
      } as unknown as RegisteredTaskInfo,
    ],
  } as unknown as RegisteredTaskGroupInfo,
  {
    taskReferenceId: 'feature.container-network',
    tasks: [
      {
        taskImplementationId: 'feature.container-network.cilium',
        taskReferenceId: 'feature.container-network',
        priority: 100,
        isFeature: true,
        isDefaultFeature: true,
        featureLabel: 'Container Networking',
        featureDescription: 'Inspect Cilium eBPF network flows and policies',
        dependencies: [
          {
            cardinality: TaskDependencyCardinality.FAN_IN,
            scope: TaskDependencyScope.ALL,
            targetReferenceId: '',
            targetTag: 'network-events',
            outputType: '[]*cilium.FlowEvent',
          },
        ],
        selectorRequirements: [],
        compatibleInspectionTypes: [],
        labels: {
          cni: 'cilium',
        },
        outputType: '*cilium.NetworkInspectionResult',
        providedTags: [
          {
            tag: 'network-events',
            outputType: '[]*cilium.FlowEvent',
          },
        ],
      } as unknown as RegisteredTaskInfo,
    ],
  } as unknown as RegisteredTaskGroupInfo,
];

export const Default: Story = {
  args: {
    taskGroups: mockGroups,
  },
};

export const Empty: Story = {
  args: {
    taskGroups: [],
  },
};

export const PanelOpen: Story = {
  args: {
    taskGroups: mockGroups,
  },
  play: async ({ canvasElement }) => {
    const card = canvasElement.querySelector(
      '#task-impl-parser\\.k8s\\.audit-log',
    ) as HTMLElement;
    if (card) {
      card.click();
    }
  },
};
