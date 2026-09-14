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

import { Meta, moduleMetadata, StoryObj } from '@storybook/angular';
import { JobCommandInputLayoutComponent } from 'src/app/dialogs/job-command-input/components/job-command-input-layout.component';

export default {
  title: 'Dialogs/JobCommandInput/JobCommandInputLayout',
  component: JobCommandInputLayoutComponent,
  decorators: [
    moduleMetadata({
      imports: [JobCommandInputLayoutComponent],
    }),
  ],
  argTypes: {
    submitCommand: { action: 'submitCommand' },
    cancelDialog: { action: 'cancelDialog' },
  },
} as Meta<JobCommandInputLayoutComponent>;

type Story = StoryObj<JobCommandInputLayoutComponent>;

export const Empty: Story = {
  args: {
    command: '',
    errorMessage: null,
  },
};

export const Filled: Story = {
  args: {
    command:
      './khi --job-mode --job-inspection-type="gke-audit-log" --job-inspection-features="feature-a" --job-inspection-values=\'{"project-id":"my-project"}\'',
    errorMessage: null,
  },
};

export const WithError: Story = {
  args: {
    command: './khi --job-mode --job-inspection-type="unknown"',
    errorMessage: 'Missing required flag: --job-inspection-values',
  },
};
