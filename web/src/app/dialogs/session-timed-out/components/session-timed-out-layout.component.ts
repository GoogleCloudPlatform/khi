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

import { Component, input, output } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatDialogModule } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Layout component for the session timed out dialog.
 * Renders the session timeout prompt or failure states and delegates user actions.
 */
@Component({
  selector: 'khi-session-timed-out-layout',
  imports: [
    MatButtonModule,
    MatDialogModule,
    MatIconModule,
    KHIIconRegistrationModule,
  ],
  templateUrl: './session-timed-out-layout.component.html',
  styleUrls: ['./session-timed-out-layout.component.scss'],
})
export class SessionTimedOutLayoutComponent {
  /**
   * Optional error message if a reconnection attempt failed.
   */
  readonly errorMessage = input<string | null>(null);

  /**
   * Indicates whether a reconnection request is currently pending.
   */
  readonly isReconnecting = input<boolean>(false);

  /**
   * Emits when the user requests reconnecting or retrying to connect to the server.
   */
  readonly reconnect = output<void>();

  /**
   * Emits when the user chooses to return to the startup dialog.
   */
  readonly returnToStartup = output<void>();
}
