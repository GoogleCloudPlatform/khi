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
import { ResourceReferenceListComponent } from './resource-reference-list.component';

export default {
  title: 'Log/ResourceReferenceList',
  component: ResourceReferenceListComponent,
} as Meta<ResourceReferenceListComponent>;

type Story = StoryObj<ResourceReferenceListComponent>;

export const Default: Story = {
  args: {
    refs: [
      { label: 'my-config of default', timelineId: 101 },
      { label: 'my-secret of default', timelineId: 102 },
    ],
  },
};

export const Empty: Story = {
  args: {
    refs: [],
  },
};
