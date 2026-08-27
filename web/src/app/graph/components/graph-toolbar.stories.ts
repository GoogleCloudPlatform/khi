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
import { DEFAULT_DELETION_THRESHOLD_SECONDS } from 'src/app/common/schema/graph-schema';
import {
  GraphToolbarComponent,
  MAX_DELETION_THRESHOLD_SECONDS,
  MIN_DELETION_THRESHOLD_SECONDS,
} from './graph-toolbar.component';

const meta: Meta<GraphToolbarComponent> = {
  title: 'Graph/GraphToolbar',
  component: GraphToolbarComponent,
  tags: ['autodocs'],
  argTypes: {
    fitToView: { action: 'fitToView' },
    downloadSvg: { action: 'downloadSvg' },
    downloadPng: { action: 'downloadPng' },
  },
  args: {
    deletionThresholdSeconds: DEFAULT_DELETION_THRESHOLD_SECONDS,
    isLoading: false,
  },
};

export default meta;
type Story = StoryObj<GraphToolbarComponent>;

export const Default: Story = {
  args: {},
};

export const Loading: Story = {
  args: {
    isLoading: true,
  },
};

export const MinDeletionThreshold: Story = {
  args: {
    deletionThresholdSeconds: MIN_DELETION_THRESHOLD_SECONDS,
  },
};

export const LongDeletionThreshold: Story = {
  args: {
    deletionThresholdSeconds: 1800,
  },
};

export const MaxDeletionThreshold: Story = {
  args: {
    deletionThresholdSeconds: MAX_DELETION_THRESHOLD_SECONDS,
  },
};
