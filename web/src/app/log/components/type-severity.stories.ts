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
import { TypeSeverityComponent } from './type-severity.component';
import { createMockInspectionDataV2 } from 'src/app/store/mock/inspection-data.mock';
import { InspectionDataV2 } from 'src/app/store/domain/inspection-data';

const meta: Meta<TypeSeverityComponent> = {
  title: 'Log/TypeSeverityComponent',
  component: TypeSeverityComponent,
  tags: ['autodocs'],
  args: {},
};

export default meta;
type Story = StoryObj<TypeSeverityComponent>;

export const Info: Story = {
  loaders: [
    async () => ({
      mockData: await createMockInspectionDataV2(),
    }),
  ],
  render: (args, { loaded: { mockData } }) => {
    const data = mockData as InspectionDataV2;
    const log =
      Array.from(data.logStore.logs()).find(
        (l) => l.severity.label === 'INFO',
      ) ?? Array.from(data.logStore.logs())[0];
    return {
      props: {
        ...args,
        logType: log.logType,
        severity: log.severity,
      },
    };
  },
};

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
        logType: log.logType,
        severity: log.severity,
      },
    };
  },
};

export const ErrorSeverity: Story = {
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
        logType: log.logType,
        severity: log.severity,
      },
    };
  },
};

export const Unknown: Story = {
  loaders: [
    async () => ({
      mockData: await createMockInspectionDataV2(),
    }),
  ],
  render: (args, { loaded: { mockData } }) => {
    const data = mockData as InspectionDataV2;
    const log =
      Array.from(data.logStore.logs()).find(
        (l) => l.severity.label === 'UNKNOWN',
      ) ?? Array.from(data.logStore.logs())[0];
    return {
      props: {
        ...args,
        logType: log.logType,
        severity: log.severity,
      },
    };
  },
};
