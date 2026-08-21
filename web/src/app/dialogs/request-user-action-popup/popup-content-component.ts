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

import { Client } from '@connectrpc/connect';
import { PopupForm, PopupService } from 'src/app/generated/api/v1/popup_pb';

/**
 * Inputs provided dynamically to popup content child components.
 */
export interface PopupContentInputs extends Record<string, unknown> {
  /** The active popup form message from the server. */
  readonly form: PopupForm;
  /** The Connect-RPC client used to interact with PopupService. */
  readonly client: Client<typeof PopupService>;
  /** Callback to notify the parent dialog when the popup interaction has completed. */
  readonly onComplete?: () => void;
}
