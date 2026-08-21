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

import { signal, Signal } from '@angular/core';
import {
  PopupFormWithClient,
  PopupManager,
} from 'src/app/services/popup/popup-manager';

/**
 * MockPopupManager is a PopupManager implementation for testing purposes.
 */
export class MockPopupManager implements PopupManager {
  private readonly _currentPopup = signal<PopupFormWithClient | null>(null);

  public readonly currentPopup: Signal<PopupFormWithClient | null> =
    this._currentPopup.asReadonly();

  /**
   * Sets the active popup for testing.
   */
  setCurrentPopup(popup: PopupFormWithClient | null): void {
    this._currentPopup.set(popup);
  }
}
