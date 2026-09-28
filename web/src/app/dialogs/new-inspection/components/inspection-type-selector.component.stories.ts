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
import { InspectionType } from 'src/app/common/schema/api-types';
import { InspectionTypeSelectorComponent } from 'src/app/dialogs/new-inspection/components/inspection-type-selector.component';

const mockInspectionTypes: readonly InspectionType[] = [
  {
    id: 'gke',
    name: 'Google Kubernetes Engine',
    description:
      'Inspect Google Kubernetes Engine cluster issues by analyzing Cloud Logging audit logs, events, and container logs.',
    icon: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="80" height="80" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" fill="%234285F4"/></svg>',
  },
  {
    id: 'composer',
    name: 'Cloud Composer',
    description:
      'Inspect Cloud Composer environment issues by analyzing Airflow task logs and worker states.',
    icon: 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="80" height="80" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" fill="%230F9D58"/></svg>',
  },
];

const meta: Meta<InspectionTypeSelectorComponent> = {
  title: 'Dialogs/NewInspection/InspectionTypeSelector',
  component: InspectionTypeSelectorComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<InspectionTypeSelectorComponent>;

export const Default: Story = {
  args: {
    inspectionTypes: mockInspectionTypes,
  },
};

export const Loading: Story = {
  args: {
    inspectionTypes: null,
  },
};
