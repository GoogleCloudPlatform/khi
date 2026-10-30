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
import { MaskStringPoolComponent } from 'src/app/mask/components/mask-string-pool.component';

const meta: Meta<MaskStringPoolComponent> = {
  title: 'Mask/MaskStringPool',
  component: MaskStringPoolComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<MaskStringPoolComponent>;

export const Default: Story = {
  args: {
    searchQuery: '^test-',
    searchRegexError: '',
    candidates: [
      { value: 'test-payments-namespace', selected: true, inRuleList: true },
      {
        value: 'test-sample-token-1',
        selected: false,
        inRuleList: false,
      },
      {
        value: 'test-service.example.com',
        selected: false,
        inRuleList: false,
      },
    ],
    totalCandidateCount: 18,
    isLoadingCandidates: false,
  },
};

export const WithRegexError: Story = {
  args: {
    searchQuery: '[unclosed',
    searchRegexError: 'error parsing regexp: missing closing ]: `[unclosed`',
    candidates: [],
    totalCandidateCount: 0,
    isLoadingCandidates: false,
  },
};
