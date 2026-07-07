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
import { IconToggleButtonComponent } from './icon-toggle-button.component';

const meta: Meta<IconToggleButtonComponent> = {
  title: 'Shared/IconToggleButton',
  component: IconToggleButtonComponent,
  tags: ['autodocs'],
  args: {
    icon: 'filter_alt',
    tooltip: 'Default tooltip',
    selected: false,
    disabled: false,
  },
};

export default meta;
type Story = StoryObj<IconToggleButtonComponent>;

export const Default: Story = {};

export const Selected: Story = {
  args: {
    selected: true,
  },
};

export const Disabled: Story = {
  args: {
    disabled: true,
  },
};

export const SelectedAndDisabled: Story = {
  args: {
    selected: true,
    disabled: true,
  },
};
