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

import { Injectable } from '@angular/core';

/**
 * Key used in localStorage to persist the unique user ID across browser tabs.
 */
export const KHI_USER_ID_STORAGE_KEY = 'khi_user_id';

/**
 * Service that provides a persistent unique user identifier for Workbench sessions.
 */
@Injectable({
  providedIn: 'root',
})
export class UserIdentityService {
  /**
   * The persistent unique identifier for this user/client instance.
   */
  public readonly userId: string;

  constructor() {
    let storedId: string | null = null;
    try {
      storedId = localStorage.getItem(KHI_USER_ID_STORAGE_KEY);
    } catch {
      // Ignore localStorage access errors in restricted environments.
    }

    if (storedId) {
      this.userId = storedId;
    } else {
      this.userId = this.generateUUID();
      try {
        localStorage.setItem(KHI_USER_ID_STORAGE_KEY, this.userId);
      } catch {
        // Ignore localStorage write errors.
      }
    }
  }

  private generateUUID(): string {
    if (
      typeof crypto !== 'undefined' &&
      typeof crypto.randomUUID === 'function'
    ) {
      return crypto.randomUUID();
    }
    return (
      'usr-' +
      Math.random().toString(36).substring(2, 15) +
      Math.random().toString(36).substring(2, 15)
    );
  }
}
