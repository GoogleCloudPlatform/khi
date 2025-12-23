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
import { SetInputComponent, SetInputItem } from './set-input.component';

const meta: Meta<SetInputComponent> = {
  title: 'Shared/Components/SetInput',
  component: SetInputComponent,
  tags: ['autodocs'],
  decorators: [
    moduleMetadata({
      imports: [],
    }),
  ],
};

export default meta;
type Story = StoryObj<SetInputComponent>;

const createItem = (id: string): SetInputItem => ({
  id,
  value: id,
});

export const Default: Story = {
  args: {
    choices: ['Angular', 'React', 'Vue', 'Svelte', 'Ember'].map(createItem),
    selectedItems: ['Angular'],
  },
};

export const Empty: Story = {
  args: {
    choices: ['A', 'B', 'C'].map(createItem),
    selectedItems: [],
  },
};

export const ALotOfItems: Story = {
  args: {
    choices: Array.from({ length: 50 }, (_, i) => createItem(`Item ${i + 1}`)),
    selectedItems: ['Item 1', 'Item 2', 'Item 3'],
  },
};

export const CustomRendering: Story = {
  args: {
    choices: ['A', 'B', 'C'].map(createItem),
    selectedItems: ['A'],
    chipTemplate: null, // Will be provided in render
    optionTemplate: null, // Will be provided in render
  },
  render: (args) => ({
    props: args,
    template: `
      <khi-shared-set-input
        [choices]="choices"
        [selectedItems]="selectedItems"
        [chipTemplate]="customChip"
        [optionTemplate]="customOption"
      ></khi-shared-set-input>
      <ng-template #customChip let-item let-remove="remove">
        <div style="background: #e0f7fa; padding: 4px 8px; border-radius: 12px; margin: 4px; display: inline-flex; align-items: center;">
          <span style="color: #006064; font-weight: bold;">{{ item.id }}</span>
          <button (click)="remove()" style="margin-left: 4px; border: none; background: none; cursor: pointer; color: #006064;">×</button>
        </div>
      </ng-template>
      <ng-template #customOption let-item>
        <div style="display: flex; align-items: center; gap: 8px;">
          <span style="color: #006064;">★</span>
          <span>{{ item.id }}</span>
        </div>
      </ng-template>
    `,
  }),
};

export const AllowCustomValues: Story = {
  args: {
    choices: ['A', 'B', 'C'].map(createItem),
    selectedItems: ['A', 'Custom One'],
    allowCustomValues: true,
  },
};
