/**
 * Copyright 2025 Google LLC
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

import { Meta, StoryObj, moduleMetadata } from '@storybook/angular';
import { SetParameterComponent } from './set-parameter.component';
import { PARAMETER_STORE } from './service/parameter-store';
import {
  ParameterHintType,
  ParameterInputType,
} from 'src/app/common/schema/form-types';
import { of } from 'rxjs';

const createParameterStoreMock = (initialValue: string[] = []) => ({
  watch: () => of(initialValue),
  watchDirty: () => of(false),
  set: () => {},
});

const meta: Meta<SetParameterComponent> = {
  title: 'Dialogs/NewInspection/Components/SetParameter',
  component: SetParameterComponent,
  tags: ['autodocs'],
  decorators: [
    moduleMetadata({
      imports: [],
      providers: [
        {
          provide: PARAMETER_STORE,
          useValue: createParameterStoreMock(),
        },
      ],
    }),
  ],
};

export default meta;
type Story = StoryObj<SetParameterComponent>;

export const Default: Story = {
  args: {
    parameter: {
      id: 'test-set-param',
      type: ParameterInputType.Set,
      label: 'Select Options',
      description: 'Choose one or more options from the list.',
      hint: 'This is a hint.',
      hintType: ParameterHintType.Info,
      options: [
        { id: '@managed', description: 'Managed namespaces' },
        { id: '-@any', description: 'Exclude any' },
        { id: '-pods', description: 'Exclude pods' },
        { id: '-nodes', description: 'Exclude nodes' },
      ],
      default: ['@managed', '-@any', '-pods', '-nodes'],
      allowAddAll: false,
      allowRemoveAll: false,
      allowCustomValue: true,
    },
  },
  decorators: [
    moduleMetadata({
      providers: [
        {
          provide: PARAMETER_STORE,
          useValue: createParameterStoreMock([
            '@managed',
            '-@any',
            '-pods',
            '-nodes',
          ]),
        },
      ],
    }),
  ],
};

export const WithPreselectedValues: Story = {
  args: {
    parameter: {
      id: 'test-set-param-preselected',
      type: ParameterInputType.Set,
      label: 'Preselected Options',
      description: 'Some options are already selected.',
      hint: '',
      hintType: ParameterHintType.None,
      options: [
        { id: 'Option 1', description: 'First option' },
        { id: 'Option 2', description: 'Second option' },
        { id: 'Option 3', description: 'Third option' },
        { id: 'Option 4', description: 'Fourth option' },
      ],
      default: ['Option 1', 'Option 2'],
      allowAddAll: false,
      allowRemoveAll: true,
      allowCustomValue: true,
    },
  },
  decorators: [
    moduleMetadata({
      providers: [
        {
          provide: PARAMETER_STORE,
          useValue: createParameterStoreMock(['Option 2', 'Option 4']),
        },
      ],
    }),
  ],
};

export const WithError: Story = {
  args: {
    parameter: {
      id: 'test-set-param-error',
      type: ParameterInputType.Set,
      label: 'Invalid Selection',
      description: 'This field has an error.',
      hint: 'You must select at least one option.',
      hintType: ParameterHintType.Error,
      options: [
        { id: 'Red', description: 'Color Red' },
        { id: 'Green', description: 'Color Green' },
        { id: 'Blue', description: 'Color Blue' },
      ],
      default: [],
      allowAddAll: false,
      allowRemoveAll: false,
      allowCustomValue: true,
    },
  },
};
