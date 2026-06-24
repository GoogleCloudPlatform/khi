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
import { ResourceHierarchyOverlayComponent } from 'src/app/log/components/resource-hierarchy-overlay.component';
import { TimelineType } from 'src/app/store/domain/style';

export default {
  title: 'Log/ResourceHierarchyOverlay',
  component: ResourceHierarchyOverlayComponent,
} as Meta<ResourceHierarchyOverlayComponent>;

type Story = StoryObj<ResourceHierarchyOverlayComponent>;

const mockClusterType: TimelineType = {
  id: 1,
  label: 'Cluster',
  description: '',
  icon: 'cloud',
  backgroundColor: { r: 0.91, g: 0.92, b: 0.96, a: 1 },
  foregroundColor: { r: 0.1, g: 0.14, b: 0.49, a: 1 },
  typeChipBackgroundColor: { r: 0.77, g: 0.79, b: 0.91, a: 1 },
  typeChipForegroundColor: { r: 0.1, g: 0.14, b: 0.49, a: 1 },
  visible: true,
  sortPriority: 1,
  height: 1,
};

const mockNamespaceType: TimelineType = {
  id: 2,
  label: 'Namespace',
  description: '',
  icon: 'folder',
  backgroundColor: { r: 0.88, g: 0.95, b: 0.95, a: 1 },
  foregroundColor: { r: 0, g: 0.3, b: 0.25, a: 1 },
  typeChipBackgroundColor: { r: 0.7, g: 0.87, b: 0.86, a: 1 },
  typeChipForegroundColor: { r: 0, g: 0.3, b: 0.25, a: 1 },
  visible: true,
  sortPriority: 2,
  height: 1,
};

const mockPodType: TimelineType = {
  id: 3,
  label: 'Pod',
  description: '',
  icon: 'widgets',
  backgroundColor: { r: 1, g: 0.95, b: 0.88, a: 1 },
  foregroundColor: { r: 0.9, g: 0.32, b: 0, a: 1 },
  typeChipBackgroundColor: { r: 1, g: 0.88, b: 0.7, a: 1 },
  typeChipForegroundColor: { r: 0.9, g: 0.32, b: 0, a: 1 },
  visible: true,
  sortPriority: 3,
  height: 1,
};

export const Default: Story = {
  args: {
    pathNodes: [
      {
        id: 1,
        label: 'my-cluster',
        type: mockClusterType,
      },
      {
        id: 2,
        label: 'default',
        type: mockNamespaceType,
      },
      {
        id: 3,
        label: 'my-pod-123',
        type: mockPodType,
      },
    ],
  },
};
