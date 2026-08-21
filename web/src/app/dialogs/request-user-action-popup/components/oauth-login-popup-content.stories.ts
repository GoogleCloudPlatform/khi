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
import { PopupFormSchema } from 'src/app/generated/api/v1/popup_pb';
import { OAuthLoginPopupContentComponent } from 'src/app/dialogs/request-user-action-popup/components/oauth-login-popup-content.component';

const meta: Meta<OAuthLoginPopupContentComponent> = {
  title: 'Dialogs/RequestUserActionPopup/OAuthLoginPopupContent',
  component: OAuthLoginPopupContentComponent,
  tags: ['autodocs'],
  args: {
    form: create(PopupFormSchema, {
      id: 'test-oauth',
      title: 'OAuth Token',
      description:
        'Please login to your Google account to get the access token.',
      payload: {
        case: 'oauthLogin',
        value: { authUrl: 'https://accounts.google.com/o/oauth2/auth' },
      },
    }),
  },
};

export default meta;
type Story = StoryObj<OAuthLoginPopupContentComponent>;

export const Default: Story = {};
