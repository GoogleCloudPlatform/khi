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
import { InspectionListComponent } from './inspection-list.component';
import { InspectionListItemViewModel } from '../types/inspection-activity.model';

const meta: Meta<InspectionListComponent> = {
  title: 'Dialogs/Startup/InspectionListComponent',
  component: InspectionListComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<InspectionListComponent>;

const mockItems: InspectionListItemViewModel[] = [
  {
    id: 'task-1',
    label: 'Long running task with sub tasks',
    phase: 'RUNNING',
    inspectionTimeLabel: '2026-04-06 12:00:00',
    totalProgress: {
      id: 'total-1',
      percentage: 45,
      percentageLabel: '45',
      label: 'Total Progress',
      message: 'Extracting logs...',
      indeterminate: false,
    },
    progresses: [
      {
        id: 'sub-1',
        percentage: 80,
        percentageLabel: '80',
        label: 'GKE Node Logs',
        message: 'Downloading...',
        indeterminate: false,
      },
      {
        id: 'sub-2',
        percentage: 10,
        percentageLabel: '10',
        label: 'Audit Logs',
        message: 'Parsing...',
        indeterminate: false,
      },
      {
        id: 'sub-3',
        percentage: 0,
        percentageLabel: '0',
        label: 'Controller Manager Logs',
        message: 'Waiting...',
        indeterminate: true,
      },
    ],
    errors: [],
  },
  {
    id: 'task-2',
    label: 'Successful completed inspection',
    phase: 'DONE',
    inspectionTimeLabel: '2026-04-06 11:30:00',
    totalProgress: {
      id: 'total-2',
      percentage: 100,
      percentageLabel: '100',
      label: 'Complete',
      message: 'Ready',
      indeterminate: false,
    },
    progresses: [],
    errors: [],
  },
  {
    id: 'task-3',
    label: 'Failed inspection',
    phase: 'ERROR',
    inspectionTimeLabel: '2026-04-06 10:00:00',
    totalProgress: {
      id: 'total-3',
      percentage: 0,
      percentageLabel: '0',
      label: 'Failed',
      message: 'Connection timeout',
      indeterminate: false,
    },
    progresses: [],
    errors: [
      {
        message: 'Failed to connect to cluster: connection timeout after 30s',
        link: '',
      },
    ],
  },
];

export const Default: Story = {
  args: {
    items: mockItems,
  },
};

export const Empty: Story = {
  args: {
    items: [],
  },
};

export const Loading: Story = {
  args: {
    items: [],
    isLoading: true,
  },
};
