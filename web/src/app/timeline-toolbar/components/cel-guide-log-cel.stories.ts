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
import { CelGuideLogCelComponent } from 'src/app/timeline-toolbar/components/cel-guide-log-cel.component';

const meta: Meta<CelGuideLogCelComponent> = {
  title: 'Timeline/Components/CelGuideLogCel',
  component: CelGuideLogCelComponent,
  tags: ['autodocs'],
  decorators: [
    moduleMetadata({
      imports: [CommonModule, BrowserAnimationsModule],
    }),
  ],
};

export default meta;
type Story = StoryObj<CelGuideLogCelComponent>;

/**
 * Default render state of the Log CEL Guide tab.
 */
export const Default: Story = {};
