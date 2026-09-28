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
  GroupParameterFormField,
  ParameterFormValidationTiming,
  ParameterHintType,
  ParameterInputType,
} from 'src/app/common/schema/form-types';
import {
  FILE_UPLOADER,
  MockFileUploader,
} from 'src/app/dialogs/new-inspection/components/service/file-uploader';
import {
  DefaultParameterStore,
  PARAMETER_STORE,
} from 'src/app/dialogs/new-inspection/components/service/parameter-store';
import { GroupParameterComponent } from 'src/app/dialogs/new-inspection/components/group-parameter.component';
import { flattenDefaultValues } from 'src/app/dialogs/new-inspection/utils/new-inspection.utils';

const mockGroupParameter: GroupParameterFormField = {
  id: 'gcp-cluster-settings',
  label: 'GCP Cluster Settings',
  description: 'Target project and GKE cluster configuration.',
  type: ParameterInputType.Group,
  hint: '',
  hintType: ParameterHintType.None,
  collapsible: true,
  collapsedByDefault: false,
  children: [
    {
      id: 'project-id',
      label: 'Project ID',
      description: 'The Google Cloud project ID containing the cluster.',
      type: ParameterInputType.Text,
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
      description: 'The GKE cluster name to inspect.',
      type: ParameterInputType.Text,
      hint: '',
      hintType: ParameterHintType.None,
      default: 'prod-cluster-1',
      readonly: false,
      suggestions: ['prod-cluster-1', 'dev-cluster-1'],
      validationTiming: ParameterFormValidationTiming.Change,
    },
    {
      id: 'include-verbose',
      label: 'Include Verbose Logs',
      description: 'Fetch debug-level logs in addition to standard entries.',
      type: ParameterInputType.Checkbox,
      hint: '',
      hintType: ParameterHintType.None,
      default: false,
      readonly: false,
    },
  ],
};

const createInitializedParameterStore = () => {
  const store = new DefaultParameterStore();
  const defaults = flattenDefaultValues([mockGroupParameter]);
  store.setDefaultValues(defaults);
  store.setValidatedParameters(defaults);
  return store;
};

const meta: Meta<GroupParameterComponent> = {
  title: 'Dialogs/NewInspection/GroupParameter',
  component: GroupParameterComponent,
  tags: ['autodocs'],
  decorators: [
    moduleMetadata({
      imports: [BrowserAnimationsModule],
      providers: [
        {
          provide: PARAMETER_STORE,
          useFactory: createInitializedParameterStore,
        },
        {
          provide: FILE_UPLOADER,
          useClass: MockFileUploader,
        },
      ],
    }),
  ],
};

export default meta;
type Story = StoryObj<GroupParameterComponent>;

export const Expanded: Story = {
  args: {
    parameter: mockGroupParameter,
  },
};

export const CollapsedByDefault: Story = {
  args: {
    parameter: {
      ...mockGroupParameter,
      id: 'gcp-cluster-settings-collapsed',
      collapsedByDefault: true,
    },
  },
};

export const NonCollapsible: Story = {
  args: {
    parameter: {
      ...mockGroupParameter,
      id: 'gcp-cluster-settings-fixed',
      collapsible: false,
      collapsedByDefault: false,
    },
  },
};

export const WithWarningHint: Story = {
  args: {
    parameter: {
      ...mockGroupParameter,
      id: 'gcp-cluster-settings-warning',
      hint: 'Some optional sub-parameters use fallback defaults.',
      hintType: ParameterHintType.Warning,
    },
  },
};
