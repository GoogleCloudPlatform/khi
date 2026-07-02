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
import { LogContentComponent } from './log-content.component';
import { createMockInspectionData } from 'src/app/store/mock/inspection-data.mock';

const meta: Meta<LogContentComponent> = {
  title: 'Log/LogContent',
  component: LogContentComponent,
  tags: ['autodocs'],
  args: {},
};

export default meta;
type Story = StoryObj<LogContentComponent>;

export const Default: Story = {
  loaders: [
    async () => ({
      mockData: await createMockInspectionData(),
    }),
  ],
  render: (args, { loaded: { mockData } }) => {
    const logEntry = Array.from(mockData.logStore.logs())[0];
    return {
      props: {
        ...args,
        vm: {
          logEntry,
          logBody: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod`,
          parsedLogBody: {
            apiVersion: 'v1',
            kind: 'Pod',
            metadata: {
              name: 'test-pod',
            },
          },
          referencedTimelineIds: [],
        },
        timezoneShift: 0,
      },
    };
  },
};

export const NoSelectedLog: Story = {
  args: {
    vm: null,
    timezoneShift: 0,
  },
};
