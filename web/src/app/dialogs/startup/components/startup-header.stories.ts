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
import { StartupHeaderComponent } from './startup-header.component';

const meta: Meta<StartupHeaderComponent> = {
  title: 'Startup/StartupHeader',
  component: StartupHeaderComponent,
  tags: ['autodocs'],
  render: (args) => ({
    props: args,
    template: `<div style="background-color: #f8f8f8; padding: 20px; position: relative;"><khi-startup-header [version]="version" [isViewerMode]="isViewerMode"></khi-startup-header></div>`,
  }),
  argTypes: {
    version: {
      control: 'text',
      description: 'Version string to display',
    },
    isViewerMode: {
      control: 'boolean',
    },
  },
};

export default meta;
type Story = StoryObj<StartupHeaderComponent>;

export const Default: Story = {
  args: {
    version: '0.0.1',
    isViewerMode: false,
  },
};

export const ViewerMode: Story = {
  args: {
    version: '0.0.1',
    isViewerMode: true,
  },
};
