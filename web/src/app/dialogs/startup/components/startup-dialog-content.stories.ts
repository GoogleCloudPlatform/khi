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

import { componentWrapperDecorator, Meta, StoryObj } from '@storybook/angular';
import { StartupDialogContentComponent } from './startup-dialog-content.component';

const meta: Meta<StartupDialogContentComponent> = {
  title: 'Dialogs/Startup/StartupDialogContent',
  component: StartupDialogContentComponent,
  tags: ['autodocs'],
  decorators: [
    componentWrapperDecorator(
      (story) => `<div style="height: 600px;">${story}</div>`,
    ),
  ],
};

export default meta;
type Story = StoryObj<StartupDialogContentComponent>;

export const Default: Story = {
  args: {
    items: [
      {
        id: '1',
        inspectionTimeLabel: '2026-04-06 12:00:00',
        label: 'Inspection 1',
        phase: 'RUNNING',
        totalProgress: {
          id: 'total',
          label: 'Total',
          message: 'Loading logs...',
          percentage: 50,
          percentageLabel: '50%',
          indeterminate: false,
        },
        progresses: [],
        errors: [],
      },
      {
        id: '2',
        inspectionTimeLabel: '2026-04-06 11:00:00',
        label: 'Inspection 2',
        phase: 'DONE',
        totalProgress: {
          id: 'total',
          label: 'Total',
          message: 'Done',
          percentage: 100,
          percentageLabel: '100%',
          indeterminate: false,
        },
        progresses: [],
        errors: [],
      },
    ],
    isLoading: false,
  },
};

export const Empty: Story = {
  args: {
    items: [],
    isLoading: false,
  },
};

export const Loading: Story = {
  args: {
    items: [],
    isLoading: true,
  },
};
