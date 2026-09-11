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
  TaskFilterEvaluation,
} from 'src/app/generated/api/v1/inspection_task_graph_pb';
import { Step2TypeFilterComponent } from './step2-type-filter.component';

const meta: Meta<Step2TypeFilterComponent> = {
  title: 'TaskGraphDebug/Step2TypeFilter',
  component: Step2TypeFilterComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<Step2TypeFilterComponent>;

const mockInspectionTypes: RegisteredInspectionTypeInfo[] = [
  {
    id: 'gke-standard',
    name: 'GKE Standard Cluster',
    description: 'Standard cluster with user managed node pools',
    labels: {
      platform: 'gke',
    },
  } as unknown as RegisteredInspectionTypeInfo,
  {
    id: 'gke-autopilot',
    name: 'GKE Autopilot Cluster',
    description: 'Fully automated serverless Kubernetes control plane',
    labels: {},
  } as unknown as RegisteredInspectionTypeInfo,
];

const mockFeatures: FeatureToggleInfo[] = [
  {
    taskImplementationId: 'feature.network.flow',
    label: 'Network Flow Logging',
    description: 'Capture container network traffic metrics',
    enabled: true,
  } as unknown as FeatureToggleInfo,
  {
    taskImplementationId: 'feature.security.threats',
    label: 'Security Posture & Threat Detection',
    description: 'Correlate audit logs with runtime vulnerability signals',
    enabled: false,
  } as unknown as FeatureToggleInfo,
];

const mockEvaluations: TaskFilterEvaluation[] = [
  {
    taskImplementationId: 'parser.audit.v1',
    taskReferenceId: 'parser.audit',
    isCompatible: true,
    isSelected: true,
    matchReason: 'Matched platform=gke label requirement',
    supersededByTaskImplementationId: '',
    priority: 100,
  } as unknown as TaskFilterEvaluation,
  {
    taskImplementationId: 'parser.audit.legacy',
    taskReferenceId: 'parser.audit',
    isCompatible: true,
    isSelected: false,
    matchReason: 'Superseded by higher-priority implementation parser.audit.v1',
    supersededByTaskImplementationId: 'parser.audit.v1',
    priority: 50,
  } as unknown as TaskFilterEvaluation,
  {
    taskImplementationId: 'parser.aws.cloudtrail',
    taskReferenceId: 'parser.audit',
    isCompatible: false,
    isSelected: false,
    matchReason: 'Platform label "aws" does not match required "gke"',
    supersededByTaskImplementationId: '',
    priority: 100,
  } as unknown as TaskFilterEvaluation,
];

export const Default: Story = {
  args: {
    inspectionTypes: mockInspectionTypes,
    selectedInspectionTypeId: 'gke-standard',
    availableFeatures: mockFeatures,
    evaluations: mockEvaluations,
  },
};

export const NoFeatures: Story = {
  args: {
    inspectionTypes: mockInspectionTypes,
    selectedInspectionTypeId: 'gke-autopilot',
    availableFeatures: [],
    evaluations: mockEvaluations,
  },
};
