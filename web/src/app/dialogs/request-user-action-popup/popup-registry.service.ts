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

import { Injectable, Type } from '@angular/core';
import { TextPopupContentComponent } from 'src/app/dialogs/request-user-action-popup/components/text-popup-content.component';
import { OAuthLoginPopupContentComponent } from 'src/app/dialogs/request-user-action-popup/components/oauth-login-popup-content.component';

/**
 * PopupRegistry manages mapping between popup payload case names and their rendering components.
 */
@Injectable({ providedIn: 'root' })
export class PopupRegistry {
  private readonly registry = new Map<string, Type<unknown>>();

  constructor() {
    this.register('text', TextPopupContentComponent);
    this.register('oauthLogin', OAuthLoginPopupContentComponent);
  }

  /**
   * Registers a component for a specific popup payload case.
   */
  register(caseName: string, component: Type<unknown>): void {
    this.registry.set(caseName, component);
  }

  /**
   * Returns the registered component for a specific popup payload case.
   */
  getComponent(caseName: string): Type<unknown> | undefined {
    return this.registry.get(caseName);
  }
}
