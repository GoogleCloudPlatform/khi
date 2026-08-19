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
import { Component, input } from '@angular/core';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatIconModule } from '@angular/material/icon';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Layout component for rendering the floating search index progress overlay at the bottom right.
 */
@Component({
  selector: 'khi-index-progress-layout',
  imports: [
    CommonModule,
    MatProgressBarModule,
    MatIconModule,
    KHIIconRegistrationModule,
  ],
  templateUrl: './index-progress-layout.component.html',
  styleUrls: ['./index-progress-layout.component.scss'],
})
export class IndexProgressLayoutComponent {
  /**
   * Whether the progress card should be displayed.
   */
  public readonly visible = input.required<boolean>();

  /**
   * Current progress percentage (0 - 100).
   */
  public readonly percent = input<number>(0);

  /**
   * Detail message describing current indexing phase.
   */
  public readonly message = input<string>('');

  /**
   * Whether the index is complete and ready.
   */
  public readonly isReady = input<boolean>(false);
}
