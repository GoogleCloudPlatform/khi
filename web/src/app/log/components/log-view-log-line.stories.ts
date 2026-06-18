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

import { Meta, StoryObj, componentWrapperDecorator } from '@storybook/angular';
import { LogViewLogLineComponent } from './log-view-log-line.component';
import { createMockInspectionDataV2 } from 'src/app/store/mock/inspection-data.mock';
import { InspectionDataV2 } from 'src/app/store/domain/inspection-data';

const meta: Meta<LogViewLogLineComponent> = {
  title: 'Log/LogViewLogLineComponent',
  component: LogViewLogLineComponent,
  tags: ['autodocs'],
  decorators: [
    componentWrapperDecorator(
      (story) => `
      <div style="display: grid; grid-template: 'type severity ts message' / auto auto auto 1fr; width: 100%;">
        ${story}
      </div>
    `,
    ),
  ],
  args: {},
};

export default meta;
type Story = StoryObj<LogViewLogLineComponent>;

export const Warning: Story = {
  loaders: [
    async () => ({
      mockData: await createMockInspectionDataV2(),
    }),
  ],
  render: (args, { loaded: { mockData } }) => {
    const data = mockData as InspectionDataV2;
    const log =
      Array.from(data.logStore.logs()).find(
        (l) => l.severity.label === 'WARNING',
      ) ?? Array.from(data.logStore.logs())[0];
    return {
      props: {
        ...args,
        log,
      },
    };
  },
};

export const ErrorLog: Story = {
  loaders: [
    async () => ({
      mockData: await createMockInspectionDataV2(),
    }),
  ],
  render: (args, { loaded: { mockData } }) => {
    const data = mockData as InspectionDataV2;
    const log =
      Array.from(data.logStore.logs()).find(
        (l) => l.severity.label === 'ERROR',
      ) ?? Array.from(data.logStore.logs())[0];
    return {
      props: {
        ...args,
        log,
      },
    };
  },
};

export const Highlighted: Story = {
  loaders: [
    async () => ({
      mockData: await createMockInspectionDataV2(),
    }),
  ],
  render: (args, { loaded: { mockData } }) => {
    const data = mockData as InspectionDataV2;
    const log = Array.from(data.logStore.logs())[0];
    return {
      props: {
        ...args,
        log,
        highlighted: true,
      },
    };
  },
};

export const Selected: Story = {
  loaders: [
    async () => ({
      mockData: await createMockInspectionDataV2(),
    }),
  ],
  render: (args, { loaded: { mockData } }) => {
    const data = mockData as InspectionDataV2;
    const log = Array.from(data.logStore.logs())[0];
    return {
      props: {
        ...args,
        log,
        selected: true,
      },
    };
  },
};
