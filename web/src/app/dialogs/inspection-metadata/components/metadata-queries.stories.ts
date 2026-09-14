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
import { MetadataQueriesComponent } from './metadata-queries.component';

const meta: Meta<MetadataQueriesComponent> = {
  title: 'Dialogs/InspectionMetadata/MetadataQueries',
  component: MetadataQueriesComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<MetadataQueriesComponent>;

export const Default: Story = {
  args: {
    queries: [
      {
        id: 'q1',
        name: 'Kubernetes Audit Log Query',
        query: `resource.type="k8s_cluster"
logName="projects/test-project/logs/cloudaudit.googleapis.com%2Factivity"
timestamp >= "2023-11-14T22:00:00Z" AND timestamp <= "2023-11-15T00:00:00Z"`,
      },
      {
        id: 'q2',
        name: 'Node System Log Query',
        query: `resource.type="gce_instance"
logName=~"projects/test-project/logs/syslog"`,
      },
      {
        id: 'q3',
        name: 'Container Startup Events',
        query: `resource.type="k8s_container"
jsonPayload.reason="Started"`,
      },
    ],
  },
};

export const SingleQuery: Story = {
  args: {
    queries: [
      {
        id: 'q-single',
        name: 'Simple Event Query',
        query: 'SELECT * FROM `k8s.events` WHERE level="ERROR"',
      },
    ],
  },
};
