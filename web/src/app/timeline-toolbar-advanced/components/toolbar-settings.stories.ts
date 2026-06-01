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

import { Meta, StoryObj, moduleMetadata } from '@storybook/angular';
import { CommonModule } from '@angular/common';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { ToolbarSettingsComponent } from 'src/app/timeline-toolbar-advanced/components/toolbar-settings.component';

const meta: Meta<ToolbarSettingsComponent> = {
  title: 'Timeline/Components/ToolbarSettings',
  component: ToolbarSettingsComponent,
  tags: ['autodocs'],
  decorators: [
    moduleMetadata({
      imports: [CommonModule, BrowserAnimationsModule],
    }),
  ],
};

export default meta;
type Story = StoryObj<ToolbarSettingsComponent>;

export const Default: Story = {
  args: {
    hideSubresourcesWithoutMatchingLogs: false,
    hideResourcesWithoutMatchingLogs: false,
  },
};

export const AllChecked: Story = {
  args: {
    hideSubresourcesWithoutMatchingLogs: true,
    hideResourcesWithoutMatchingLogs: true,
  },
};
