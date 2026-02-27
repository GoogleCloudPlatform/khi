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

import { Meta, StoryObj, moduleMetadata } from '@storybook/angular';
import { LogViewLogLineComponent } from './log-view-log-line.component';
import { of } from 'rxjs';
import { ViewStateService } from 'src/app/services/view-state.service';
import { LogEntry } from 'src/app/store/log';
import { ToTextReferenceFromKHIFileBinary } from 'src/app/common/loader/reference-type';

export default {
  title: 'log/LogViewLogLineComponent',
  component: LogViewLogLineComponent,
  decorators: [
    moduleMetadata({
      providers: [
        {
          provide: ViewStateService,
          useValue: {
            timezoneShift: of(0),
          },
        },
      ],
    }),
  ],
} as Meta<LogViewLogLineComponent>;

type Story = StoryObj<LogViewLogLineComponent>;

const mockLog: LogEntry = {
  logIndex: 123,
  time: 1700000000000,
  insertId: 'mock-insert-id',
  logType: 0,
  logTypeLabel: 'k8s_audit',
  severity: 3,
  logSeverityLabel: 'WARNING',
  summary: 'Mock log entry summary explaining what happened.',
  body: ToTextReferenceFromKHIFileBinary({ offset: 0, len: 0, buffer: 0 }), // Mock TextReference
  annotations: [],
} as unknown as LogEntry;

const mockErrorLog: LogEntry = {
  ...mockLog,
  logTypeLabel: 'k8s_container',
  severity: 4,
  logSeverityLabel: 'ERROR',
  summary: 'A critical container error occurred.',
} as unknown as LogEntry;

export const Warning: Story = {
  args: {
    log: mockLog,
  },
};

export const ErrorLog: Story = {
  args: {
    log: mockErrorLog,
  },
};
