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
import { ProgressDialogComponent } from './progress.component';
import {
  PROGRESS_DIALOG_STATUS_OBSERVER,
  CurrentProgress,
} from 'src/app/services/progress/progress-interface';
import { of } from 'rxjs';

const createProgressDecorator = (progress: CurrentProgress) =>
  moduleMetadata({
    providers: [
      {
        provide: PROGRESS_DIALOG_STATUS_OBSERVER,
        useValue: {
          status: () => of(progress as CurrentProgress),
        },
      },
    ],
  });

const meta: Meta<ProgressDialogComponent> = {
  title: 'Dialogs/Progress/ProgressDialog',
  component: ProgressDialogComponent,
  tags: ['autodocs'],
  decorators: [
    createProgressDecorator({
      mode: 'indeterminate',
      percent: 0,
      message: 'Loading resources...',
    } as CurrentProgress),
  ],
};

export default meta;
type Story = StoryObj<ProgressDialogComponent>;

export const Indeterminate: Story = {};

export const DeterminateStart: Story = {
  decorators: [
    createProgressDecorator({
      mode: 'determinate',
      percent: 0,
      message: 'Starting download...',
    } as CurrentProgress),
  ],
};

export const DeterminateHalf: Story = {
  decorators: [
    createProgressDecorator({
      mode: 'determinate',
      percent: 50,
      message: 'Processing data (50%)...',
    } as CurrentProgress),
  ],
};

export const DeterminateComplete: Story = {
  decorators: [
    createProgressDecorator({
      mode: 'determinate',
      percent: 100,
      message: 'Completed!',
    } as CurrentProgress),
  ],
};

export const MultiTierClientAndServer: Story = {
  decorators: [
    createProgressDecorator({
      mode: 'determinate',
      percent: 50,
      message: 'Opening dataset...',
      tasks: [
        {
          id: 'client',
          label: 'Client',
          message: 'Parsing local chunks (80%)...',
          percent: 80,
          mode: 'determinate',
        },
        {
          id: 'server',
          label: 'Server',
          message: 'Reading backend stream (40%)...',
          percent: 40,
          mode: 'determinate',
        },
      ],
    }),
  ],
};

export const MultiTierThreeTasks: Story = {
  decorators: [
    createProgressDecorator({
      mode: 'determinate',
      percent: 60,
      message: 'Processing multi-stage pipeline...',
      tasks: [
        {
          id: 'download',
          label: 'Download',
          message: 'Complete',
          percent: 100,
          mode: 'determinate',
        },
        {
          id: 'client-parse',
          label: 'Client Parse',
          message: 'Parsing logs (65%)...',
          percent: 65,
          mode: 'determinate',
        },
        {
          id: 'server-index',
          label: 'Server Parse',
          message: 'Indexing chunks...',
          percent: 25,
          mode: 'determinate',
        },
      ],
    }),
  ],
};
