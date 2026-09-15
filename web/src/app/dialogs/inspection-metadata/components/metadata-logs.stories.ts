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
import { MetadataLogsComponent } from './metadata-logs.component';

const meta: Meta<MetadataLogsComponent> = {
  title: 'Dialogs/InspectionMetadata/MetadataLogs',
  component: MetadataLogsComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<MetadataLogsComponent>;

export const Default: Story = {
  args: {
    logs: [
      {
        id: 'k8s.io/audit-log/query',
        name: 'Audit Log Query Task',
        log: `2026/09/14 15:30:00 INFO Starting audit log querying...
2026/09/14 15:30:02 INFO Connecting to Cloud Logging API...
2026/09/14 15:30:04 INFO 3,420 entries fetched successfully.
2026/09/14 15:30:05 INFO Parsing audit log payloads completed.`,
      },
      {
        id: 'k8s.io/event-log/query',
        name: 'Kubernetes Event Query Task',
        log: `2026/09/14 15:30:01 INFO Starting event log querying...
2026/09/14 15:30:03 INFO 850 events fetched.
2026/09/14 15:30:04 INFO Event timeline integration finished.`,
      },
    ],
  },
};
