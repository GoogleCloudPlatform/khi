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

import { Injectable, inject, signal, OnDestroy } from '@angular/core';
import { PopupForm } from 'src/app/generated/api/v1/popup_pb';
import { ConnectClientService } from 'src/app/services/api/connect-client.service';
import {
  PopupFormWithClient,
  PopupManager,
} from 'src/app/services/popup/popup-manager';

/**
 * PopupManagerImpl continuously watches the server for popup lifecycle events using Connect-RPC streaming.
 */
@Injectable({ providedIn: 'root' })
export class PopupManagerImpl implements PopupManager, OnDestroy {
  private readonly connectClient = inject(ConnectClientService);

  private readonly _currentPopup = signal<PopupFormWithClient | null>(null);

  /**
   * Signal providing the currently active popup form and its client, or null if no popup is active.
   */
  public readonly currentPopup = this._currentPopup.asReadonly();

  private readonly abortController = new AbortController();

  constructor() {
    this.startWatchLoop();
  }

  private async startWatchLoop(): Promise<void> {
    while (!this.abortController.signal.aborted) {
      try {
        const responseStream = this.connectClient.popupClient.watchPopup(
          {},
          { signal: this.abortController.signal },
        );

        for await (const res of responseStream) {
          if (this.abortController.signal.aborted) {
            break;
          }

          switch (res.event.case) {
            case 'popup':
              this.handlePopupEvent(res.event.value);
              break;
            case 'dismissed':
              this.handleDismissedEvent();
              break;
          }
        }
      } catch {
        if (this.abortController.signal.aborted) {
          break;
        }
        // Reconnect after 1 second if a connection drop or network failure occurred.
        await new Promise((resolve) => setTimeout(resolve, 1000));
      }
    }
  }

  private handlePopupEvent(form: PopupForm): void {
    const current = this._currentPopup();
    if (current?.form.id !== form.id) {
      this._currentPopup.set({
        form,
        client: this.connectClient.popupClient,
      });
    }
  }

  private handleDismissedEvent(): void {
    this._currentPopup.set(null);
  }

  ngOnDestroy(): void {
    this.abortController.abort();
  }
}
