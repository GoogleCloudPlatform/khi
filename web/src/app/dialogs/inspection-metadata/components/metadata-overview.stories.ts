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
import { MetadataOverviewComponent } from './metadata-overview.component';

const meta: Meta<MetadataOverviewComponent> = {
  title: 'Dialogs/InspectionMetadata/MetadataOverview',
  component: MetadataOverviewComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<MetadataOverviewComponent>;

export const Default: Story = {
  args: {
    overview: {
      inspectionType: 'gcp-gke',
      inspectionName: 'Production GKE Cluster Inspection',
      inspectionTypeIconPath: '',
      formattedStartTime: '2023/11/14 22:13:20',
      formattedEndTime: '2023/11/15 00:13:20',
      durationText: '2h',
      suggestedFilename: 'production-gke-cluster.khi',
      fileSizeText: '12.4 MB',
    },
  },
};

export const SmallInspection: Story = {
  args: {
    overview: {
      inspectionType: 'local-file',
      inspectionName: 'Audit Log Test',
      inspectionTypeIconPath: '',
      formattedStartTime: '2023/11/14 22:13:20',
      formattedEndTime: '2023/11/14 22:18:20',
      durationText: '5m',
      suggestedFilename: 'audit-log-test.khi',
      fileSizeText: '450 KB',
    },
  },
};
