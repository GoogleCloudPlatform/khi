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
import { Component, computed, input } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import {
  StringPoolIndexPhase,
  StringPoolIndexStatusViewModel,
} from 'src/app/mask/types/mask-viewmodel';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Displays the construction status and progress of the on-demand string pool trigram index.
 */
@Component({
  selector: 'khi-mask-index-banner',
  imports: [
    CommonModule,
    MatIconModule,
    MatProgressBarModule,
    KHIIconRegistrationModule,
  ],
  templateUrl: './mask-index-banner.component.html',
  styleUrls: ['./mask-index-banner.component.scss'],
})
export class MaskIndexBannerComponent {
  protected readonly StringPoolIndexPhase = StringPoolIndexPhase;

  /**
   * Current construction status of the on-demand string pool trigram index.
   */
  public readonly indexStatus =
    input.required<StringPoolIndexStatusViewModel>();

  /**
   * Rounded percentage displayed in the string pool index progress banner.
   */
  protected readonly roundedIndexProgress = computed<number>(() =>
    Math.min(
      100,
      Math.max(0, Math.round(this.indexStatus().progressPercentage)),
    ),
  );
}
