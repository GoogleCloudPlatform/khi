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

import { Meta, StoryObj, moduleMetadata } from '@storybook/angular';
import { DiffContentComponent } from './diff-content.component';
import { ResourceRevision } from '../../store/revision';
import { RevisionState, RevisionVerb } from '../../zzz-generated';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

const mockCurrentRevision = new ResourceRevision(
  new Date('2025-01-01T00:00:01Z').getTime(),
  new Date('2025-01-01T00:00:02Z').getTime(),
  RevisionState.RevisionStateExisting,
  RevisionVerb.RevisionVerbUpdate,
  'content2',
  'user@example.com',
  false,
  false,
  1,
);

const currentContentWithManagedFields = `apiVersion: v1
kind: Pod
metadata:
  name: my-pod
  namespace: default
  managedFields:
    - manager: kubectl
      operation: Update
spec:
  containers:
  - name: nginx
    image: nginx:latest
    ports:
    - containerPort: 80`;

const previousContentWithManagedFields = `apiVersion: v1
kind: Pod
metadata:
  name: my-pod
  namespace: default
spec:
  containers:
  - name: nginx
    image: nginx:1.14.2
    ports:
    - containerPort: 80`;

const currentContent = `apiVersion: v1
kind: Pod
metadata:
  name: my-pod
  namespace: default
spec:
  containers:
  - name: nginx
    image: nginx:latest
    ports:
    - containerPort: 80`;

const previousContent = `apiVersion: v1
kind: Pod
metadata:
  name: my-pod
  namespace: default
spec:
  containers:
  - name: nginx
    image: nginx:1.14.2
    ports:
    - containerPort: 80`;

const meta: Meta<DiffContentComponent> = {
  title: 'Diff/DiffContent',
  component: DiffContentComponent,
  tags: ['autodocs'],
  decorators: [
    moduleMetadata({
      imports: [BrowserAnimationsModule],
    }),
  ],
  args: {
    currentRevision: mockCurrentRevision,
    currentRevisionContent: currentContent,
    previousRevisionContent: previousContent,
    showManagedFields: false,
  },
};

export default meta;
type Story = StoryObj<DiffContentComponent>;

export const Default: Story = {
  render: (args) => ({
    props: {
      ...args,
    },
    template: `
      <div style="height: 500px; display: flex; flex-direction: column;">
        <khi-diff-content
          [currentRevision]="currentRevision"
          [currentRevisionContent]="currentRevisionContent"
          [previousRevisionContent]="previousRevisionContent"
          [(showManagedFields)]="showManagedFields"
          (openInNewTab)="openInNewTab($event)"></khi-diff-content>
      </div>
    `,
  }),
};

export const WithoutManagedFields: Story = {
  ...Default,
  args: {
    showManagedFields: false,
  },
};

export const WithManagedFields: Story = {
  ...Default,
  args: {
    showManagedFields: true,
    currentRevisionContent: currentContentWithManagedFields,
    previousRevisionContent: previousContentWithManagedFields,
  },
};

export const NoRevision: Story = {
  ...Default,
  args: {
    currentRevision: null,
    currentRevisionContent: '',
    previousRevisionContent: '',
  },
};
