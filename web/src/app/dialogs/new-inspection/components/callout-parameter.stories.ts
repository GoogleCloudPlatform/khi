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
import {
  ParameterHintType,
  ParameterInputType,
} from 'src/app/common/schema/form-types';
import { CalloutParameterComponent } from 'src/app/dialogs/new-inspection/components/callout-parameter.component';

const meta: Meta<CalloutParameterComponent> = {
  title: 'Dialogs/NewInspection/CalloutParameter',
  component: CalloutParameterComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<CalloutParameterComponent>;

export const Info: Story = {
  args: {
    parameter: {
      id: 'log-volume-advisory',
      type: ParameterInputType.Callout,
      label: 'Log Volume Advisory',
      description: 'Applies when inspecting high-traffic clusters.',
      hint: 'Narrowing the query time window reduces Cloud Logging scan volume and speeds up timeline generation.',
      hintType: ParameterHintType.Info,
    },
  },
};

export const Warning: Story = {
  args: {
    parameter: {
      id: 'api-quota-warning',
      type: ParameterInputType.Callout,
      label: 'API Quota Warning',
      description: 'Detected elevated read request usage in test-project.',
      hint: 'Concurrent inspections against the same Google Cloud project may encounter Cloud Logging rate limiting.',
      hintType: ParameterHintType.Warning,
    },
  },
};

export const Error: Story = {
  args: {
    parameter: {
      id: 'missing-log-bucket',
      type: ParameterInputType.Callout,
      label: 'Missing Required Log Bucket',
      description: 'Cluster prerequisite check failed.',
      hint: 'No accessible Cloud Logging bucket was found in project test-project. Grant roles/logging.viewer to continue.',
      hintType: ParameterHintType.Error,
    },
  },
};

export const Pending: Story = {
  args: {
    parameter: {
      id: 'cluster-prerequisites-pending',
      type: ParameterInputType.Callout,
      label: 'Cluster Prerequisites',
      description: 'Verifying Cloud Logging access for prod-cluster-1.',
      hint: 'Checking IAM permissions and available log buckets...',
      hintType: ParameterHintType.Info,
      pending: true,
    },
  },
};

export const MultilineMessage: Story = {
  args: {
    parameter: {
      id: 'cluster-prerequisites-multiline',
      type: ParameterInputType.Callout,
      label: 'Cluster Prerequisites',
      description: 'Required roles for GKE audit and node log inspection:',
      hint: '1. roles/logging.viewer on project test-project\n2. roles/container.clusterViewer for cluster discovery\n3. Active Application Default Credentials via gcloud auth application-default login',
      hintType: ParameterHintType.Info,
    },
  },
};

export const MessageOnlyWithoutDescription: Story = {
  args: {
    parameter: {
      id: 'callout-message-only',
      type: ParameterInputType.Callout,
      label: '',
      description: '',
      hint: 'Audit log queries default to UTC timestamps.',
      hintType: ParameterHintType.Info,
    },
  },
};

export const TitleAndMessageWithoutDescription: Story = {
  args: {
    parameter: {
      id: 'callout-title-and-message',
      type: ParameterInputType.Callout,
      label: 'Log Volume Advisory',
      description: '',
      hint: 'Audit log queries default to UTC timestamps.',
      hintType: ParameterHintType.Info,
    },
  },
};
