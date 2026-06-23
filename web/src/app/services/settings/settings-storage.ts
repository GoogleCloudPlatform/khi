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

import { InjectionToken, Injectable } from '@angular/core';

/**
 * Provides access to key-value storage for application persistence settings.
 * This abstraction decouples components from specific browser storage mechanisms.
 */
export interface SettingsStorage {
  /**
   * Retrieves the stored value associated with the specified key.
   */
  getItem(key: string): string | null;

  /**
   * Stores the specified key-value pair in the persistent storage.
   */
  setItem(key: string, value: string): void;
}

/**
 * Default implementation of SettingsStorage utilizing browser localStorage.
 * Automatically catches storage access errors in restricted environments.
 */
@Injectable({ providedIn: 'root' })
export class LocalStorageSettingsStorage implements SettingsStorage {
  getItem(key: string): string | null {
    try {
      return localStorage.getItem(key);
    } catch (error) {
      console.warn(`Failed to read key "${key}" from localStorage:`, error);
      return null;
    }
  }

  setItem(key: string, value: string): void {
    try {
      localStorage.setItem(key, value);
    } catch (error) {
      console.warn(`Failed to write key "${key}" to localStorage:`, error);
    }
  }
}

/**
 * InjectionToken for accessing the application settings storage mechanism.
 */
export const SETTINGS_STORAGE = new InjectionToken<SettingsStorage>(
  'SETTINGS_STORAGE',
  {
    providedIn: 'root',
    factory: () => new LocalStorageSettingsStorage(),
  },
);
