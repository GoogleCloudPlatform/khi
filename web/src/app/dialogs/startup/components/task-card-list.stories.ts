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
import { TaskCardListComponent } from './task-card-list.component';

const meta: Meta<TaskCardListComponent> = {
  title: 'Startup/TaskCardList',
  component: TaskCardListComponent,
  tags: ['autodocs'],
  render: (args) => ({
    props: args,
    template: `<div style="height:300px"><khi-task-card-list 
      [tasks]="tasks"
      [isViewerMode]="isViewerMode"
      (openInspectionResult)="openInspectionResult($event)"
      (openInspectionMetadata)="openInspectionMetadata($event)"
      (cancelInspection)="cancelInspection($event)"
      (downloadInspectionResult)="downloadInspectionResult($event)"
      (changeInspectionTitle)="changeInspectionTitle($event)"
      ></khi-task-card-list></div>`,
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
type Story = StoryObj<TaskCardListComponent>;

export const ViewerMode: Story = {
  args: {
    tasks: undefined,
    isViewerMode: true,
  },
};

export const Loading: Story = {
  args: {
    tasks: undefined,
    isViewerMode: false,
  },
};

export const Empty: Story = {
  args: {
    tasks: [],
    isViewerMode: false,
  },
};

export const WithItems: Story = {
  args: {
    isViewerMode: false,
    tasks: [
      {
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
        progresses: [],
        errors: [],
      },
      {
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
    ],
  },
};
