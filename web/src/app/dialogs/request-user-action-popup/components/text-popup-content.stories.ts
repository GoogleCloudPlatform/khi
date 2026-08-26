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
import { create } from '@bufbuild/protobuf';
import {
  PopupFormSchema,
  PopupService,
} from 'src/app/generated/api/v1/popup_pb';
import { Client } from '@connectrpc/connect';
import { TextPopupContentComponent } from 'src/app/dialogs/request-user-action-popup/components/text-popup-content.component';

const mockClient: Client<typeof PopupService> = {
  validatePopupAnswer: async () => ({
    id: 'test',
    validationError: '',
    $typeName: 'api.v1.ValidatePopupAnswerResponse',
  }),
  submitPopupAnswer: async () => ({
    $typeName: 'api.v1.SubmitPopupAnswerResponse',
  }),
  pullPopup: async () => ({
    popup: undefined,
    $typeName: 'api.v1.PullPopupResponse',
  }),
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  watchPopup: (() => {}) as any,
};

const meta: Meta<TextPopupContentComponent> = {
  title: 'Dialogs/RequestUserActionPopup/TextPopupContent',
  component: TextPopupContentComponent,
  tags: ['autodocs'],
  args: {
    form: create(PopupFormSchema, {
      id: 'test-text',
      title: 'Cluster Name',
      description: 'Enter your cluster name to query logs',
      payload: {
        case: 'text',
        value: { placeholder: 'e.g. gke-cluster-production' },
      },
    }),
    client: mockClient,
  },
};

export default meta;
type Story = StoryObj<TextPopupContentComponent>;

export const Default: Story = {};
