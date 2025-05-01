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

import { Component, inject, input } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { BreaklinePipe } from 'src/app/common/breakline.pipe';
import {
  ParameterFormFieldBase,
  ParameterHintType,
} from 'src/app/common/schema/form-types';
import { PARAMETER_STORE } from './service/parameter-store';
import { CommonModule } from '@angular/common';
import { MatTooltipModule } from '@angular/material/tooltip';

/**
 * Component of common parameter headers used in new inspection dialog.
 */
@Component({
  selector: 'khi-new-inspection-parameter-header',
  templateUrl: './parameter-header.component.html',
  styleUrls: ['./parameter-header.component.sass'],
  imports: [CommonModule, BreaklinePipe, MatIconModule, MatTooltipModule],
})
export class ParameterHeaderComponent {
  readonly ParameterHintType = ParameterHintType;

  readonly dirtyIconTooltip =
    "This field modified once and won't follow the default value when KHI updated the default dynamatically.";
  /**
   * The spec of this text type parameter.
   */
  parameter = input.required<ParameterFormFieldBase>();

  /**
   * If the status of validation should show on header or not.
   */
  showValidationStatus = input(true);

  store = inject(PARAMETER_STORE);
}
