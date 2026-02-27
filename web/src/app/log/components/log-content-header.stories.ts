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
import { LogContentHeaderComponent } from './log-content-header.component';
import { LogEntry } from 'src/app/store/log';
import { LogType, Severity } from 'src/app/zzz-generated';
import { ToTextReferenceFromKHIFileBinary } from 'src/app/common/loader/reference-type';

const meta: Meta<LogContentHeaderComponent> = {
  title: 'Log/LogContentHeader',
  component: LogContentHeaderComponent,
  tags: ['autodocs'],
  args: {},
};

export default meta;
type Story = StoryObj<LogContentHeaderComponent>;

export const Default: Story = {
  args: {
    log: new LogEntry(
      0,
      'foobar',
      LogType.LogTypeAudit,
      Severity.SeverityWarning,
      1234567890,
      'summary',
      ToTextReferenceFromKHIFileBinary(null),
      [],
    ),
  },
};
