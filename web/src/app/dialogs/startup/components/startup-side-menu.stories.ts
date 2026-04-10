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
import { StartupSideMenuComponent } from './startup-side-menu.component';

const meta: Meta<StartupSideMenuComponent> = {
  title: 'Dialogs/Startup/StartupSideMenu',
  component: StartupSideMenuComponent,
  tags: ['autodocs'],
  argTypes: {
    newInvestigation: { action: 'newInvestigation' },
    openKhiFile: { action: 'openKhiFile' },
  },
};

export default meta;
type Story = StoryObj<StartupSideMenuComponent>;

export const Default: Story = {
  args: {
    version: 'v0.100.21',
    links: [
      {
        icon: 'description',
        label: 'Documentation',
        url: 'https://github.com/GoogleCloudPlatform/kubernetes-history-inspector',
      },
      {
        icon: 'bug_report',
        label: 'Report Bug',
        url: 'https://github.com/GoogleCloudPlatform/kubernetes-history-inspector/issues',
      },
      {
        icon: 'code',
        label: 'GitHub',
        url: 'https://github.com/GoogleCloudPlatform/kubernetes-history-inspector',
      },
    ],
  },
};
