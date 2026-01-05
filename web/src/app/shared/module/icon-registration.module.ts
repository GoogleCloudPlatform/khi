/**
 * Copyright 2025 Google LLC
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

import { inject, NgModule } from '@angular/core';
import { MatIconModule, MatIconRegistry } from '@angular/material/icon';

/**
 * A module that registers icons used in KHI.
 * This module must be imported from components using MatIconModule.
 */
@NgModule({
  imports: [MatIconModule],
})
export class KHIIconRegistrationModule {
  private readonly iconRegistry = inject(MatIconRegistry);

  constructor() {
    this.iconRegistry.setDefaultFontSetClass('material-symbols-outlined');
  }
}
