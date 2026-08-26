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

import { Component, computed, inject, input } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { BreaklinePipe } from 'src/app/common/breakline.pipe';
import {
  ParameterFormFieldBase,
  ParameterHintType,
} from 'src/app/common/schema/form-types';
import { PARAMETER_STORE } from './service/parameter-store';
import { CommonModule } from '@angular/common';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';

/**
 * Component of common parameter headers used in new inspection dialog.
 */
@Component({
  selector: 'khi-new-inspection-parameter-header',
  templateUrl: './parameter-header.component.html',
  styleUrls: ['./parameter-header.component.scss'],
  imports: [
    CommonModule,
    BreaklinePipe,
    MatIconModule,
    MatTooltipModule,
    MatProgressSpinnerModule,
  ],
})
export class ParameterHeaderComponent {
  readonly ParameterHintType = ParameterHintType;

  readonly dirtyIconTooltip =
    "This field modified once and won't follow the default value when KHI updated the default dynamatically.";
  /**
   * The spec of this text type parameter.
   */
  readonly parameter = input.required<ParameterFormFieldBase>();

  /**
   * If the status of validation should show on header or not.
   */
  readonly showValidationStatus = input(true);

  private readonly store = inject(PARAMETER_STORE);

  /**
   * Computed signal indicating if the parameter is currently being validated.
   */
  readonly isValidating = computed(() => {
    return this.store.isValidating(this.parameter().id)();
  });

  /**
   * Computed signal indicating if the parameter was modified by the user.
   */
  readonly isDirty = computed(() => {
    return this.store.isDirty(this.parameter().id)();
  });
}
