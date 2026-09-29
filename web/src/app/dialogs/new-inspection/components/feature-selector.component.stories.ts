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
import { InspectionFeature } from 'src/app/common/schema/api-types';
import { FeatureSelectorComponent } from 'src/app/dialogs/new-inspection/components/feature-selector.component';

const mockFeatures: readonly InspectionFeature[] = [
  {
    id: 'k8s-audit',
    label: 'Kubernetes Audit Log',
    description:
      'Audit logs record the actions taken by users, administrators, and system components in the cluster.',
    enabled: true,
  },
  {
    id: 'k8s-events',
    label: 'Kubernetes Events',
    description:
      'Kubernetes events provide insight into what is happening inside a cluster, such as pod scheduling or node reboot.',
    enabled: true,
  },
  {
    id: 'k8s-node-system',
    label: 'Node System Logs',
    description:
      'System logs from node components like kubelet, containerd, and systemd journal.',
    enabled: false,
  },
  {
    id: 'cloud-monitoring',
    label: 'Cloud Monitoring Metrics',
    description:
      'Time series metric data for CPU, memory, and network utilization across cluster nodes.',
    enabled: false,
  },
];

const mockNoneSelectedFeatures: readonly InspectionFeature[] = mockFeatures.map(
  (feature) => ({
    ...feature,
    enabled: false,
  }),
);

const meta: Meta<FeatureSelectorComponent> = {
  title: 'Dialogs/NewInspection/FeatureSelector',
  component: FeatureSelectorComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<FeatureSelectorComponent>;

export const Default: Story = {
  args: {
    features: mockFeatures,
  },
};

export const NoneSelected: Story = {
  args: {
    features: mockNoneSelectedFeatures,
  },
};
