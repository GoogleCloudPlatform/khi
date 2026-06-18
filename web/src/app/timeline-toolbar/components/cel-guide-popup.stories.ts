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
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import {
  CelGuidePopupComponent,
  CelGuideTab,
} from 'src/app/timeline-toolbar/components/cel-guide-popup.component';

const meta: Meta<CelGuidePopupComponent> = {
  title: 'Timeline/Components/CelGuidePopup',
  component: CelGuidePopupComponent,
  tags: ['autodocs'],
  decorators: [
    moduleMetadata({
      imports: [
        CommonModule,
        BrowserAnimationsModule,
        KHIIconRegistrationModule,
      ],
    }),
  ],
};

export default meta;
type Story = StoryObj<CelGuidePopupComponent>;

/**
 * Render state of the popup showing the Overview tab.
 */
export const OverviewTab: Story = {
  args: {
    activeTab: CelGuideTab.Overview,
  },
};

/**
 * Render state of the popup showing the Timeline CEL tab.
 */
export const TimelineCelTab: Story = {
  args: {
    activeTab: CelGuideTab.TimelineCel,
  },
};

/**
 * Render state of the popup showing the Log CEL tab.
 */
export const LogCelTab: Story = {
  args: {
    activeTab: CelGuideTab.LogCel,
  },
};
