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
import { SessionTimedOutLayoutComponent } from 'src/app/dialogs/session-timed-out/components/session-timed-out-layout.component';

export default {
  title: 'Dialogs/SessionTimedOut/SessionTimedOutLayout',
  component: SessionTimedOutLayoutComponent,
  decorators: [
    moduleMetadata({
      imports: [SessionTimedOutLayoutComponent],
    }),
  ],
} as Meta<SessionTimedOutLayoutComponent>;

type Story = StoryObj<SessionTimedOutLayoutComponent>;

export const Default: Story = {
  args: {
    errorMessage: null,
    isReconnecting: false,
  },
};

export const Reconnecting: Story = {
  args: {
    errorMessage: null,
    isReconnecting: true,
  },
};

export const ErrorState: Story = {
  args: {
    errorMessage: 'Failed to reconnect to the server. Connection timed out.',
    isReconnecting: false,
  },
};
