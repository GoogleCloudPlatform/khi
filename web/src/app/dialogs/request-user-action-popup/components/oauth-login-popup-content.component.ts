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

import { CommonModule } from '@angular/common';
import { Component, OnInit, input } from '@angular/core';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { Client } from '@connectrpc/connect';
import { PopupForm, PopupService } from 'src/app/generated/api/v1/popup_pb';

/**
 * OAuthLoginPopupContentComponent manages the OAuth login popup view and window dispatch.
 */
@Component({
  selector: 'khi-oauth-login-popup-content',
  standalone: true,
  templateUrl: './oauth-login-popup-content.component.html',
  styleUrls: ['./oauth-login-popup-content.component.scss'],
  imports: [CommonModule, MatProgressBarModule],
})
export class OAuthLoginPopupContentComponent implements OnInit {
  /** Active popup form configuration. */
  readonly form = input.required<PopupForm>();

  /** Connect-RPC client for popup operations. */
  readonly client = input<Client<typeof PopupService>>();

  /** Optional callback to notify container when interaction completes. */
  readonly onComplete = input<() => void>();

  ngOnInit(): void {
    this.openAuthWindow();
  }

  /**
   * Opens the OAuth login URL in a popup window.
   */
  openAuthWindow(): void {
    const payload = this.form().payload;
    if (payload.case !== 'oauthLogin') {
      return;
    }
    const url = payload.value.authUrl;
    if (!url) {
      throw new Error('OAuth authentication URL is missing');
    }
    window.open(url, 'oauth login', 'width=400px,height=500px');
  }
}
