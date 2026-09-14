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
import { MetadataCodeViewerComponent } from './metadata-code-viewer.component';

const meta: Meta<MetadataCodeViewerComponent> = {
  title: 'Dialogs/InspectionMetadata/MetadataCodeViewer',
  component: MetadataCodeViewerComponent,
  tags: ['autodocs'],
  argTypes: {
    contentCopied: { action: 'contentCopied' },
  },
};

export default meta;
type Story = StoryObj<MetadataCodeViewerComponent>;

export const Default: Story = {
  args: {
    title: 'Audit Log Query',
    badge: 'Estimated: 1,420',
    code: `SELECT
  timestamp,
  protoPayload.methodName,
  protoPayload.authenticationInfo.principalEmail
FROM
  \`project.dataset.cloudaudit_googleapis_com_activity\`
WHERE
  resource.type = "k8s_cluster"
ORDER BY
  timestamp DESC
LIMIT 100`,
    maxHeight: '240px',
  },
};

export const WithoutHeader: Story = {
  args: {
    code: `graph TD
    A[Fetch Cluster Info] --> B[Query Audit Logs]
    B --> C[Generate Timeline Events]
    C --> D[Save KHI File]`,
    maxHeight: '200px',
  },
};

export const LongLogOutput: Story = {
  args: {
    title: 'k8s.io/audit-log/query',
    badge: 'Task Log',
    code: `2026/09/14 15:30:00 INFO [audit-log] Starting query execution
2026/09/14 15:30:01 DEBUG [audit-log] Connecting to BigQuery client
2026/09/14 15:30:03 INFO [audit-log] Query returned 1,420 rows
2026/09/14 15:30:04 DEBUG [audit-log] Parsing events into timeline model
2026/09/14 15:30:05 INFO [audit-log] Finished processing in 5.2s`,
    maxHeight: '180px',
  },
};
