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
import { IndexProgressLayoutComponent } from './index-progress-layout.component';

const meta: Meta<IndexProgressLayoutComponent> = {
  title: 'IndexProgress/IndexProgressLayout',
  component: IndexProgressLayoutComponent,
  tags: ['autodocs'],
  args: {
    visible: true,
    percent: 45,
    message: 'Building search index posting lists...',
    isReady: false,
  },
};

export default meta;
type Story = StoryObj<IndexProgressLayoutComponent>;

export const Building: Story = {
  args: {
    visible: true,
    percent: 65,
    message: 'Indexing log body struct trigrams (65%)...',
    isReady: false,
  },
};

export const Ready: Story = {
  args: {
    visible: true,
    percent: 100,
    message: 'Search index ready.',
    isReady: true,
  },
};

export const Hidden: Story = {
  args: {
    visible: false,
    percent: 100,
    message: 'Search index ready.',
    isReady: true,
  },
};
