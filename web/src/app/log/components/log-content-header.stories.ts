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
import { createMockInspectionData } from 'src/app/store/mock/inspection-data.mock';

const meta: Meta<LogContentHeaderComponent> = {
  title: 'Log/LogContentHeader',
  component: LogContentHeaderComponent,
  tags: ['autodocs'],
  args: {},
};

export default meta;
type Story = StoryObj<LogContentHeaderComponent>;

export const Default: Story = {
  loaders: [
    async () => ({
      mockData: await createMockInspectionData(),
    }),
  ],
  render: (args, { loaded: { mockData } }) => {
    const log = Array.from(mockData.logStore.logs())[0];
    return {
      props: {
        ...args,
        log,
      },
    };
  },
};
