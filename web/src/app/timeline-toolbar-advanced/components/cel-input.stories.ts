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
import { CelInputComponent } from 'src/app/timeline-toolbar-advanced/components/cel-input.component';

const meta: Meta<CelInputComponent> = {
  title: 'Timeline/Components/CelInput',
  component: CelInputComponent,
  tags: ['autodocs'],
  decorators: [
    moduleMetadata({
      imports: [CommonModule, BrowserAnimationsModule],
    }),
  ],
};

export default meta;
type Story = StoryObj<CelInputComponent>;

export const TimelineModeValid: Story = {
  args: {
    errorMessage: '',
    tooltip: 'Timeline CEL',
    placeholder: 'e.g. timeline.name == "pod-a"',
    value: 'timeline.name == "pod-a"',
    icon: 'view_timeline',
  },
};

export const TimelineModeInvalid: Story = {
  args: {
    errorMessage: 'no such variable: timelin',
    tooltip: 'Timeline CEL',
    placeholder: 'e.g. timeline.name == "pod-a"',
    value: 'timelin.name ==',
    icon: 'view_timeline',
  },
};

export const LogModeValid: Story = {
  args: {
    errorMessage: '',
    tooltip: 'Log CEL',
    placeholder: 'e.g. log.severity == ERROR',
    value: 'log.severity == ERROR',
    icon: 'view_timeline',
  },
};
