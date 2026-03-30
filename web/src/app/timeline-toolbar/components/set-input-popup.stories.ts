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

import { Meta, StoryObj, moduleMetadata } from '@storybook/angular';
import { SetInputPopupComponent } from './set-input-popup.component';
import { CommonModule } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { SetInputComponent } from '../../shared/components/set-input/set-input.component';

const meta: Meta<SetInputPopupComponent> = {
  title: 'Timeline/Components/SetInputPopup',
  component: SetInputPopupComponent,
  tags: ['autodocs'],
  decorators: [
    moduleMetadata({
      imports: [CommonModule, MatButtonModule, SetInputComponent],
    }),
  ],
};

export default meta;
type Story = StoryObj<SetInputPopupComponent>;

export const Default: Story = {
  args: {
    label: 'Filter Kinds',
    choices: new Set(['Pod', 'Service', 'Deployment', 'ReplicaSet']),
    selectedItems: new Set(['Pod', 'Service']),
  },
};
