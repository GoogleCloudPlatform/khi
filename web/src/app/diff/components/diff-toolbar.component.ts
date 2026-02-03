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

import { Component, input, model, output } from '@angular/core';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { MatTooltip } from '@angular/material/tooltip';

@Component({
  selector: 'khi-diff-toolbar',
  templateUrl: './diff-toolbar.component.html',
  styleUrls: ['./diff-toolbar.component.scss'],
  imports: [
    MatSlideToggleModule,
    MatButtonModule,
    MatIconModule,
    KHIIconRegistrationModule,
    MatTooltip,
  ],
})
export class DiffToolbarComponent {
  /**
   * showManagedFields is the model for the slide toggle to show managed fields.
   */
  showManagedFields = model(false);

  /**
   * showOpenInNewTabButton is the input for the open in new tab button.
   * The diff window apart from the main diff window should not have the open in new tab button.
   */
  showOpenInNewTabButton = input(true);

  /**
   * copyContent is the output for the copy content button.
   */
  copyContent = output<void>();

  /**
   * openInNewTab is the output for the open in new tab button.
   */
  openInNewTab = output<void>();
}
