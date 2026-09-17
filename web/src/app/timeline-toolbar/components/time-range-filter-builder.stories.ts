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
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { TimeRangeFilterBuilderComponent } from './time-range-filter-builder.component';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

const meta: Meta<TimeRangeFilterBuilderComponent> = {
  title: 'Timeline/Toolbar/TimeRangeFilterBuilder',
  component: TimeRangeFilterBuilderComponent,
  tags: ['autodocs'],
  decorators: [
    moduleMetadata({
      imports: [
        CommonModule,
        MatInputModule,
        MatFormFieldModule,
        MatButtonModule,
        MatIconModule,
        MatTooltipModule,
        NoopAnimationsModule,
        KHIIconRegistrationModule,
      ],
    }),
  ],
};

export default meta;
type Story = StoryObj<TimeRangeFilterBuilderComponent>;

export const Default: Story = {
  args: {
    startTime: null,
    endTime: null,
    defaultStartTime: 1700000000000000000n,
    defaultEndTime: 1700003600000000000n,
    timezoneShift: 0,
    showDeleteButton: false,
  },
};

export const WithExistingRange: Story = {
  args: {
    startTime: 1700000000000000000n,
    endTime: 1700001800000000000n,
    defaultStartTime: 1700000000000000000n,
    defaultEndTime: 1700003600000000000n,
    timezoneShift: 9,
    showDeleteButton: true,
  },
};

export const InvalidRange: Story = {
  args: {
    startTime: 1700003600000000000n,
    endTime: 1700000000000000000n,
    defaultStartTime: 1700000000000000000n,
    defaultEndTime: 1700003600000000000n,
    timezoneShift: 0,
    showDeleteButton: true,
  },
};
