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
import { MaskIndexBannerComponent } from 'src/app/mask/components/mask-index-banner.component';
import { StringPoolIndexPhase } from 'src/app/mask/types/mask-viewmodel';

const meta: Meta<MaskIndexBannerComponent> = {
  title: 'Mask/MaskIndexBanner',
  component: MaskIndexBannerComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<MaskIndexBannerComponent>;

export const Building: Story = {
  args: {
    indexStatus: {
      phase: StringPoolIndexPhase.Building,
      progressPercentage: 48,
      message: 'Extracting trigrams (12,400 / 25,800 strings)...',
    },
  },
};

export const Ready: Story = {
  args: {
    indexStatus: {
      phase: StringPoolIndexPhase.Ready,
      progressPercentage: 100,
      message: 'String pool search index ready.',
    },
  },
};

export const Failed: Story = {
  args: {
    indexStatus: {
      phase: StringPoolIndexPhase.Failed,
      progressPercentage: 0,
      message: 'Trigram index failed.',
    },
  },
};
