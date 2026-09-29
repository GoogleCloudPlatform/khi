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
  ParameterFormValidationTiming,
  ParameterHintType,
  ParameterInputType,
} from 'src/app/common/schema/form-types';
import {
  DefaultParameterStore,
  PARAMETER_STORE,
} from 'src/app/dialogs/new-inspection/components/service/parameter-store';
import {
  ParameterStepViewModel,
  TotalEstimatedLogsSeverity,
} from 'src/app/dialogs/new-inspection/types/new-inspection.types';
import { ParameterInputStepComponent } from 'src/app/dialogs/new-inspection/components/parameter-input-step.component';
import { flattenDefaultValues } from 'src/app/dialogs/new-inspection/utils/new-inspection.utils';

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
            id: 'project-id',
            label: 'Project ID',
            type: ParameterInputType.Text,
            description: 'Google Cloud project ID containing the cluster logs',
            hint: '',
            hintType: ParameterHintType.None,
            default: 'my-gcp-project',
            readonly: false,
            suggestions: ['my-gcp-project', 'staging-gcp-project'],
            validationTiming: ParameterFormValidationTiming.Blur,
          },
          {
            id: 'cluster-name',
            label: 'Cluster Name',
            type: ParameterInputType.Text,
            description: 'Name of the GKE cluster to inspect',
            hint: '',
            hintType: ParameterHintType.None,
            default: 'my-cluster',
            readonly: false,
            suggestions: ['my-cluster', 'prod-cluster-us-central1'],
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
      estimatedCount: 42000,
    },
    {
      id: 'q2',
      name: 'Container Logs',
      query: 'resource.type="k8s_container"',
      estimatedCount: 150000,
    },
  ],
  job: {
    command: 'khi run --target gke --cluster my-cluster',
  },
  fieldCount: 2,
  totalEstimatedSummary: {
    knownCount: 192000,
    isComplete: true,
    isEstimating: false,
    isIncomplete: false,
    displayText: '~192,000 total logs estimated',
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

const meta: Meta<ParameterInputStepComponent> = {
  title: 'Dialogs/NewInspection/ParameterInputStep',
  component: ParameterInputStepComponent,
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
type Story = StoryObj<ParameterInputStepComponent>;

export const Default: Story = {
  args: {
    parameterViewModel: mockParameterViewModel,
    parameterStore: storyParameterStore,
    taskGraphDebugUrl:
      '/debug/task-graph?tab=DAG_VIEWER&inspectionType=gke&features=k8s-audit,k8s-events',
  },
};

export const Loading: Story = {
  args: {
    parameterViewModel: null,
    parameterStore: storyParameterStore,
    taskGraphDebugUrl:
      '/debug/task-graph?tab=DAG_VIEWER&inspectionType=gke&features=k8s-audit,k8s-events',
  },
};

export const WithValidationErrors: Story = {
  args: {
    parameterViewModel: {
      ...mockParameterViewModel,
      rootGroupForm: {
        ...mockParameterViewModel.rootGroupForm,
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
                id: 'project-id',
                label: 'Project ID',
                type: ParameterInputType.Text,
                description:
                  'Google Cloud project ID containing the cluster logs',
                hint: 'Project ID is required.',
                hintType: ParameterHintType.Error,
                default: '',
                readonly: false,
                suggestions: [],
                validationTiming: ParameterFormValidationTiming.Blur,
              },
              {
                id: 'cluster-name',
                label: 'Cluster Name',
                type: ParameterInputType.Text,
                description: 'Name of the GKE cluster to inspect',
                hint: 'Cluster was not found in the specified project.',
                hintType: ParameterHintType.Error,
                default: 'invalid-cluster',
                readonly: false,
                suggestions: [],
                validationTiming: ParameterFormValidationTiming.Change,
              },
            ],
          },
        ],
      },
    },
    parameterStore: storyParameterStore,
    taskGraphDebugUrl:
      '/debug/task-graph?tab=DAG_VIEWER&inspectionType=gke&features=k8s-audit,k8s-events',
  },
};
