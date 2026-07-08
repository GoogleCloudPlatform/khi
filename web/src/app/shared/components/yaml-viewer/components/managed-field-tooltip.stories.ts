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

import { ManagedFieldTooltipComponent } from 'src/app/shared/components/yaml-viewer/components/managed-field-tooltip.component';

const meta: Meta<ManagedFieldTooltipComponent> = {
  title: 'Shared/YamlViewer/ManagedFieldTooltip',
  component: ManagedFieldTooltipComponent,
  decorators: [
    moduleMetadata({
      imports: [ManagedFieldTooltipComponent],
    }),
  ],
  argTypes: {
    manager: { control: 'text' },
    timezoneShift: { control: 'number' },
  },
};
export default meta;
type Story = StoryObj<ManagedFieldTooltipComponent>;

export const Default: Story = {
  args: {
    manager: 'kubectl-client-side-apply',
    time: 1696939200000000000n, // 2023-10-10T12:00:00Z in ns
    timezoneShift: 9,
  },
};
