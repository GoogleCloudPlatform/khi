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
import { DiffListHeaderComponent } from './diff-list-header.component';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

const mockTimeline = {
  id: 1,
  path: [
    {
      id: 1,
      type: {
        id: 1,
        label: 'APIVersion',
        description: '',
        icon: 'settings',
        backgroundColor: { r: 0, g: 0, b: 0, a: 0 },
        foregroundColor: { r: 0, g: 0, b: 0, a: 0 },
        visible: true,
        sortPriority: 1,
      },
      label: 'v1',
    },
    {
      id: 2,
      type: {
        id: 2,
        label: 'Kind',
        description: '',
        icon: 'workspaces',
        backgroundColor: { r: 0, g: 0, b: 0, a: 0 },
        foregroundColor: { r: 0, g: 0, b: 0, a: 0 },
        visible: true,
        sortPriority: 2,
      },
      label: 'Pod',
    },
    {
      id: 3,
      type: {
        id: 3,
        label: 'Namespace',
        description: '',
        icon: 'folder',
        backgroundColor: { r: 0, g: 0, b: 0, a: 0 },
        foregroundColor: { r: 0, g: 0, b: 0, a: 0 },
        visible: true,
        sortPriority: 3,
      },
      label: 'default',
    },
    {
      id: 4,
      type: {
        id: 4,
        label: 'Resource',
        description: '',
        icon: 'description',
        backgroundColor: { r: 0, g: 0, b: 0, a: 0 },
        foregroundColor: { r: 0, g: 0, b: 0, a: 0 },
        visible: true,
        sortPriority: 4,
      },
      label: 'nginx-deployment-6fbb6b7d-xyz',
    },
    {
      id: 5,
      type: {
        id: 5,
        label: 'Subresource',
        description: '',
        icon: 'page_info',
        backgroundColor: { r: 0, g: 0, b: 0, a: 0 },
        foregroundColor: { r: 0, g: 0, b: 0, a: 0 },
        visible: true,
        sortPriority: 5,
      },
      label: 'status',
    },
  ],
} as unknown as ReadonlyDomainElement<Timeline>;

const meta: Meta<DiffListHeaderComponent> = {
  title: 'Diff/DiffListHeader',
  component: DiffListHeaderComponent,
  tags: ['autodocs'],
  args: {
    timeline: mockTimeline,
  },
};

export default meta;
type Story = StoryObj<DiffListHeaderComponent>;

export const Default: Story = {};

export const RootResource: Story = {
  args: {
    timeline: {
      id: 2,
      path: [
        {
          id: 1,
          type: {
            id: 1,
            label: 'APIVersion',
            description: '',
            icon: 'settings',
            backgroundColor: { r: 0, g: 0, b: 0, a: 0 },
            foregroundColor: { r: 0, g: 0, b: 0, a: 0 },
            visible: true,
            sortPriority: 1,
          },
          label: 'v1',
        },
        {
          id: 2,
          type: {
            id: 2,
            label: 'Kind',
            description: '',
            icon: 'workspaces',
            backgroundColor: { r: 0, g: 0, b: 0, a: 0 },
            foregroundColor: { r: 0, g: 0, b: 0, a: 0 },
            visible: true,
            sortPriority: 2,
          },
          label: 'Namespace',
        },
        {
          id: 3,
          type: {
            id: 3,
            label: 'Namespace',
            description: '',
            icon: 'folder',
            backgroundColor: { r: 0, g: 0, b: 0, a: 0 },
            foregroundColor: { r: 0, g: 0, b: 0, a: 0 },
            visible: true,
            sortPriority: 3,
          },
          label: 'default',
        },
      ],
    } as unknown as ReadonlyDomainElement<Timeline>,
  },
};

export const WithoutTimeline: Story = {
  args: {
    timeline: null,
  },
};
