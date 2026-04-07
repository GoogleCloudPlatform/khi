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
import { StartupDialogLayoutComponent } from './startup-dialog-layout.component';
import { SidebarLink } from '../types/startup-side-menu.types';
import { InspectionListItemViewModel } from '../types/inspection-activity.model';

const meta: Meta<StartupDialogLayoutComponent> = {
  title: 'Dialogs/Startup/StartupDialogLayout',
  component: StartupDialogLayoutComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<StartupDialogLayoutComponent>;

const mockLinks: SidebarLink[] = [
  {
    icon: 'help',
    label: 'Documentation',
    url: 'https://github.com/GoogleCloudPlatform/khi',
  },
  {
    icon: 'bug_report',
    label: 'Report a bug',
    url: 'https://github.com/GoogleCloudPlatform/khi/issues',
  },
];

const mockItems: InspectionListItemViewModel[] = [
  {
    id: '1',
    inspectionTimeLabel: '2026-04-06 12:00:00',
    label: 'Completed Inspection',
    phase: 'DONE',
    totalProgress: {
      id: 'total',
      label: 'Total',
      message: 'Completed',
      percentage: 100,
      percentageLabel: '100%',
      indeterminate: false,
    },
    progresses: [],
    errors: [],
  },
  {
    id: '2',
    inspectionTimeLabel: '2026-04-06 13:00:00',
    label: 'Running Inspection',
    phase: 'RUNNING',
    totalProgress: {
      id: 'total',
      label: 'Total',
      message: 'Processing...',
      percentage: 45,
      percentageLabel: '45%',
      indeterminate: false,
    },
    progresses: [],
    errors: [],
  },
  {
    id: '3',
    inspectionTimeLabel: '2026-04-06 14:00:00',
    label: 'Failed Inspection',
    phase: 'ERROR',
    totalProgress: {
      id: 'total',
      label: 'Total',
      message: 'Failed',
      percentage: 0,
      percentageLabel: '0%',
      indeterminate: false,
    },
    progresses: [],
    errors: [{ message: 'Failed to fetch data', link: '' }],
  },
];

export const Default: Story = {
  args: {
    version: '1.0.0',
    links: mockLinks,
    items: mockItems,
    isLoading: false,
  },
  render: (args) => ({
    props: args,
    template: `
      <div style="height: 600px; border: 1px solid #ccc;">
        <khi-startup-dialog-layout [version]="version" [links]="links" [items]="items" [isLoading]="isLoading"></khi-startup-dialog-layout>
      </div>
    `,
  }),
};

export const Loading: Story = {
  args: {
    version: '1.0.0',
    links: mockLinks,
    items: [],
    isLoading: true,
  },
  render: (args) => ({
    props: args,
    template: `
      <div style="height: 600px; border: 1px solid #ccc;">
        <khi-startup-dialog-layout [version]="version" [links]="links" [items]="items" [isLoading]="isLoading"></khi-startup-dialog-layout>
      </div>
    `,
  }),
};
