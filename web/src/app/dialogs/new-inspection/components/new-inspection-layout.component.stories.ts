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

import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { Meta, StoryObj, moduleMetadata } from '@storybook/angular';
import {
  InspectionFeature,
  InspectionType,
} from 'src/app/common/schema/api-types';
import {
  ParameterFormValidationTiming,
  ParameterHintType,
  ParameterInputType,
} from 'src/app/common/schema/form-types';
import {
  DefaultParameterStore,
  PARAMETER_STORE,
} from 'src/app/dialogs/new-inspection/components/service/parameter-store';
import {
  NewInspectionStepIndex,
  ParameterStepViewModel,
  TotalEstimatedLogsSeverity,
} from 'src/app/dialogs/new-inspection/types/new-inspection.types';
import { NewInspectionLayoutComponent } from 'src/app/dialogs/new-inspection/components/new-inspection-layout.component';
import { flattenDefaultValues } from 'src/app/dialogs/new-inspection/utils/new-inspection.utils';

const mockInspectionTypes: readonly InspectionType[] = [
  {
    id: 'gke',
    name: 'Google Kubernetes Engine',
    description:
      'Inspect Google Kubernetes Engine cluster issues by analyzing Cloud Logging audit logs, events, and container logs.',
    icon: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="80" height="80" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" fill="%234285F4"/></svg>',
  },
  {
    id: 'composer',
    name: 'Cloud Composer',
    description:
      'Inspect Cloud Composer environment issues by analyzing Airflow task logs and worker states.',
    icon: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="80" height="80" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" fill="%230F9D58"/></svg>',
  },
];

const mockFeatures: readonly InspectionFeature[] = [
  {
    id: 'k8s-audit',
    label: 'Kubernetes Audit Log',
    description:
      'Audit logs record the actions taken by users, administrators, and system components in the cluster.',
    enabled: true,
  },
  {
    id: 'k8s-events',
    label: 'Kubernetes Events',
    description:
      'Kubernetes events provide insight into what is happening inside a cluster, such as pod scheduling or node reboot.',
    enabled: true,
  },
  {
    id: 'k8s-node-system',
    label: 'Node System Logs',
    description:
      'System logs from node components like kubelet, containerd, and systemd journal.',
    enabled: false,
  },
];

const mockParameterViewModel: ParameterStepViewModel = {
  rootGroupForm: {
    id: 'root',
    label: '',
    type: ParameterInputType.Group,
    description: '',
    hint: '',
    hintType: ParameterHintType.None,
    collapsible: false,
    collapsedByDefault: false,
    children: [
      {
        id: 'gcp-settings',
        label: 'GCP settings',
        type: ParameterInputType.Group,
        description: 'Google Cloud project and cluster settings',
        hint: '',
        hintType: ParameterHintType.None,
        collapsible: true,
        collapsedByDefault: false,
        children: [
          {
            id: 'cluster-name',
            label: 'Cluster Name',
            type: ParameterInputType.Text,
            description: 'Name of the GKE cluster',
            hint: '',
            hintType: ParameterHintType.None,
            default: 'production-cluster',
            readonly: false,
            suggestions: [],
            validationTiming: ParameterFormValidationTiming.Change,
          },
        ],
      },
    ],
  },
  queries: [
    {
      id: 'q1',
      name: 'Kubernetes Audit Logs',
      query:
        'logName="projects/test-project/logs/cloudaudit.googleapis.com%2Factivity"',
      estimatedCount: 50000,
    },
  ],
  plan: {
    taskGraph: 'digraph G {\n  A -> B;\n}',
  },
  job: {
    command: 'khi run --target gke --cluster production-cluster',
  },
  fieldCount: 1,
  totalEstimatedSummary: {
    knownCount: 50000,
    isComplete: true,
    isEstimating: false,
    isIncomplete: false,
    displayText: '~50,000 total logs estimated',
    severity: TotalEstimatedLogsSeverity.Normal,
  },
};

const createInitializedParameterStore = () => {
  const store = new DefaultParameterStore();
  const defaults = flattenDefaultValues(
    mockParameterViewModel.rootGroupForm.children,
  );
  store.setDefaultValues(defaults);
  store.setValidatedParameters(defaults);
  return store;
};

const storyParameterStore = createInitializedParameterStore();

const meta: Meta<NewInspectionLayoutComponent> = {
  title: 'Dialogs/NewInspection/NewInspectionLayout',
  component: NewInspectionLayoutComponent,
  tags: ['autodocs'],
  decorators: [
    moduleMetadata({
      imports: [BrowserAnimationsModule],
      providers: [
        {
          provide: PARAMETER_STORE,
          useValue: storyParameterStore,
        },
      ],
    }),
  ],
};

export default meta;
type Story = StoryObj<NewInspectionLayoutComponent>;

export const Step1SelectTarget: Story = {
  args: {
    selectedStepIndex: NewInspectionStepIndex.InspectionType,
    inspectionTypes: mockInspectionTypes,
    currentInspectionType: null,
    features: [],
    parameterStore: storyParameterStore,
    parameterViewModel: null,
  },
};

export const Step2SelectFeatures: Story = {
  args: {
    selectedStepIndex: NewInspectionStepIndex.FeatureSelection,
    inspectionTypes: mockInspectionTypes,
    currentInspectionType: mockInspectionTypes[0],
    features: mockFeatures,
    parameterStore: storyParameterStore,
    parameterViewModel: null,
  },
};

export const Step3InputParameters: Story = {
  args: {
    selectedStepIndex: NewInspectionStepIndex.ParameterInput,
    inspectionTypes: mockInspectionTypes,
    currentInspectionType: mockInspectionTypes[0],
    features: mockFeatures,
    parameterStore: storyParameterStore,
    parameterViewModel: mockParameterViewModel,
  },
};
