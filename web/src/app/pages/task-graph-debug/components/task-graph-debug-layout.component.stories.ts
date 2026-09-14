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
  FeatureToggleInfo,
  RegisteredInspectionTypeInfo,
  RegisteredTaskGroupInfo,
  TaskDependencyCardinality,
  TaskFilterEvaluation,
} from 'src/app/generated/api/v1/inspection_task_graph_pb';
import {
  DagViewerEdge,
  DagViewerNode,
} from 'src/app/shared/components/dag-viewer/dag-viewer.model';
import { TaskGraphDebugTab } from 'src/app/pages/task-graph-debug/types/task-graph-debug.model';
import { TaskGraphDebugLayoutComponent } from './task-graph-debug-layout.component';

const meta: Meta<TaskGraphDebugLayoutComponent> = {
  title: 'TaskGraphDebug/TaskGraphDebugLayout',
  component: TaskGraphDebugLayoutComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<TaskGraphDebugLayoutComponent>;

const mockInspectionTypes: RegisteredInspectionTypeInfo[] = [
  {
    id: 'gke-standard',
    name: 'GKE Standard Cluster',
    description: 'Standard cluster with user managed node pools',
    labels: { platform: 'gke' },
  } as unknown as RegisteredInspectionTypeInfo,
  {
    id: 'gke-autopilot',
    name: 'GKE Autopilot Cluster',
    description: 'Fully managed Autopilot cluster',
    labels: {},
  } as unknown as RegisteredInspectionTypeInfo,
];

const mockFeatures: FeatureToggleInfo[] = [
  {
    taskImplementationId: 'feature.container-network',
    label: 'Container Network Flow',
    description: 'Analyze container network traffic and DNS queries',
    enabled: true,
  } as unknown as FeatureToggleInfo,
];

const mockGroups: RegisteredTaskGroupInfo[] = [
  {
    taskReferenceId: 'parser.k8s.audit',
    tasks: [],
  } as unknown as RegisteredTaskGroupInfo,
];

const mockEvaluations: TaskFilterEvaluation[] = [
  {
    taskImplementationId: 'parser.k8s.audit-log',
    taskReferenceId: 'parser.k8s.audit',
    isCompatible: true,
    isSelected: true,
    matchReason: 'Matches inspection type criteria',
    supersededByTaskImplementationId: '',
    priority: 100,
  } as unknown as TaskFilterEvaluation,
];

const mockNodes: DagViewerNode[] = [
  {
    id: 'root-task',
    referenceId: 'root',
    priority: 100,
    isFeature: false,
    isFormTask: true,
    topologicalOrder: 0,
    labels: {},
    outputType: '*config.ClusterConfig',
    providedTags: [],
  },
  {
    id: 'child-task-1',
    referenceId: 'worker-1',
    priority: 100,
    isFeature: false,
    isFormTask: false,
    topologicalOrder: 1,
    labels: {},
    outputType: '[]*worker.TaskResult',
    providedTags: [],
  },
  {
    id: 'child-task-2',
    referenceId: 'worker-2',
    priority: 100,
    isFeature: false,
    isFormTask: false,
    topologicalOrder: 2,
    labels: {},
    outputType: '[]*worker.TaskResult',
    providedTags: [],
  },
];

const mockEdges: DagViewerEdge[] = [
  {
    id: 'edge-1',
    sourceId: 'root-task',
    destinationId: 'child-task-1',
    sourceReferenceId: 'root',
    cardinality: TaskDependencyCardinality.POINT_TO_POINT,
    tag: '',
    priority: 0,
    outputType: '*config.ClusterConfig',
  },
  {
    id: 'edge-2',
    sourceId: 'root-task',
    destinationId: 'child-task-2',
    sourceReferenceId: 'root',
    cardinality: TaskDependencyCardinality.POINT_TO_POINT,
    tag: '',
    priority: 0,
    outputType: '*config.ClusterConfig',
  },
];

export const Default: Story = {
  args: {
    activeTab: TaskGraphDebugTab.DAG_VIEWER,
    taskGroups: mockGroups,
    inspectionTypes: mockInspectionTypes,
    selectedInspectionTypeId: 'gke-standard',
    evaluations: mockEvaluations,
    availableFeatures: mockFeatures,
    dagNodes: mockNodes,
    dagEdges: mockEdges,
    isResolutionSuccess: true,
    resolutionErrorMessage: '',
    isLoading: false,
  },
};

export const Step1Registry: Story = {
  args: {
    ...Default.args,
    activeTab: TaskGraphDebugTab.REGISTRY,
  },
};

export const Step2Filter: Story = {
  args: {
    ...Default.args,
    activeTab: TaskGraphDebugTab.INSPECTION_TYPE_FILTER,
  },
};

export const ResolutionError: Story = {
  args: {
    ...Default.args,
    isResolutionSuccess: false,
    resolutionErrorMessage:
      'Cyclic dependency detected: task-a -> task-b -> task-c -> task-a',
  },
};

export const Empty: Story = {
  args: {
    activeTab: TaskGraphDebugTab.DAG_VIEWER,
    taskGroups: [],
    inspectionTypes: [],
    selectedInspectionTypeId: '',
    evaluations: [],
    availableFeatures: [],
    dagNodes: [],
    dagEdges: [],
    isResolutionSuccess: true,
    resolutionErrorMessage: '',
    isLoading: false,
  },
};
