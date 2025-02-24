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

import { Component, input } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { CommonModule } from '@angular/common';
import { BreaklinePipe } from '../../../common/breakline.pipe';
import {
  ParameterFormField,
  ParameterHintType,
} from 'src/app/common/schema/form-types';

/**
 * Hint message (error, warning or info) at the bottom of parameter form field.
 */
@Component({
  selector: 'khi-new-inspection-parameter-hint',
  templateUrl: './parameter-hint.component.html',
  styleUrls: ['./parameter-hint.component.sass'],
  imports: [CommonModule, MatIconModule, BreaklinePipe],
})
export class ParameterHintComponent {
  /**
   * Type of ParameterHintType enum. Exporting this to be used in the template.
   */
  readonly ParameterHintType = ParameterHintType;

  /**
   * The spec of this parameter field.
   */
  parameter = input.required<ParameterFormField>();
}
