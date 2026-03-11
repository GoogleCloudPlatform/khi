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
import { ResourceTimeline } from '../../store/timeline';
import { ParentRelationship } from '../../zzz-generated';

const mockTimeline = new ResourceTimeline(
  'timeline-id-1',
  'api/v1#pods#default#nginx-deployment-6fbb6b7d-xyz#status',
  [],
  [],
  ParentRelationship.RelationshipOwnerReference,
);

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
    timeline: new ResourceTimeline(
      'timeline-id-2',
      'api/v1#namespaces#default',
      [],
      [],
      ParentRelationship.RelationshipOwnerReference,
    ),
  },
};

export const WithoutTimeline: Story = {
  args: {
    timeline: null,
  },
};
