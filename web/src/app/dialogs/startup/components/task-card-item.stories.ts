/**
 * Copyright 2024 Google LLC
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
import { TaskCardItemComponent } from './task-card-item.component';

const meta: Meta<TaskCardItemComponent> = {
  title: 'Startup/TaskCardItem',
  component: TaskCardItemComponent,
  tags: ['autodocs'],
  render: (args) => ({
    props: args,
    template: `<khi-task-card-item 
      [task]="task"
      (openInspectionResult)="openInspectionResult($event)"
      (openInspectionMetadata)="openInspectionMetadata($event)"
      (cancelInspection)="cancelInspection($event)"
      (downloadInspectionResult)="downloadInspectionResult($event)"
      (changeInspectionTitle)="changeInspectionTitle($event)"
      ></khi-task-card-item>`,
  }),
  argTypes: {
    openInspectionResult: { action: 'openInspectionResult' },
    openInspectionMetadata: { action: 'openInspectionMetadata' },
    cancelInspection: { action: 'cancelInspection' },
    downloadInspectionResult: { action: 'downloadInspectionResult' },
    changeInspectionTitle: { action: 'changeInspectionTitle' },
  },
};

export default meta;
type Story = StoryObj<TaskCardItemComponent>;

export const Running: Story = {
  args: {
    task: {
      id: 'task-1',
      inspectionTimeLabel: '2 minutes ago',
      label: 'GKE Cluster Inspection',
      phase: 'RUNNING',
      totalProgress: {
        id: 'total',
        label: 'Total Progress',
        message: '50%',
        percentage: 50,
        percentageLabel: '50',
        indeterminate: false,
      },
      progresses: [
        {
          id: 'p1',
          label: 'Fetching logs',
          message: 'Processing...',
          percentage: 30,
          percentageLabel: '30',
          indeterminate: false,
        },
        {
          id: 'p2',
          label: 'Analyzing events',
          message: 'Waiting...',
          percentage: 0,
          percentageLabel: '0',
          indeterminate: true,
        },
      ],
      errors: [],
    },
  },
};

export const Done: Story = {
  args: {
    task: {
      id: 'task-2',
      inspectionTimeLabel: '1 hour ago',
      label: 'Local File Inspection',
      phase: 'DONE',
      totalProgress: {
        id: 'total',
        label: 'Completed',
        message: '100%',
        percentage: 100,
        percentageLabel: '100',
        indeterminate: false,
      },
      progresses: [],
      errors: [],
    },
  },
};

export const Error: Story = {
  args: {
    task: {
      id: 'task-3',
      inspectionTimeLabel: '5 minutes ago',
      label: 'Failed Inspection',
      phase: 'ERROR',
      totalProgress: {
        id: 'total',
        label: 'Error',
        message: 'Failed',
        percentage: 100,
        percentageLabel: '100',
        indeterminate: false,
      },
      progresses: [],
      errors: [
        {
          message: 'Failed to connect to cluster',
          link: 'https://example.com/troubleshooting',
        },
      ],
    },
  },
};
