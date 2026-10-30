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
import { MaskRuleListComponent } from 'src/app/mask/components/mask-rule-list.component';

const meta: Meta<MaskRuleListComponent> = {
  title: 'Mask/MaskRuleList',
  component: MaskRuleListComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<MaskRuleListComponent>;

export const WithRules: Story = {
  args: {
    rules: [
      {
        target: 'test-payments-namespace',
        replacement: '<masked_1>',
        defaultReplacement: '<masked_1>',
        isDeletion: false,
        isDefaultReplacement: true,
        enabled: true,
      },
      {
        target: 'secret-token-value',
        replacement: '',
        defaultReplacement: '<masked_2>',
        isDeletion: true,
        isDefaultReplacement: false,
        enabled: true,
      },
      {
        target: 'disabled-target-node',
        replacement: '<masked_3>',
        defaultReplacement: '<masked_3>',
        isDeletion: false,
        isDefaultReplacement: true,
        enabled: false,
      },
    ],
  },
};

export const Empty: Story = {
  args: {
    rules: [],
  },
};
