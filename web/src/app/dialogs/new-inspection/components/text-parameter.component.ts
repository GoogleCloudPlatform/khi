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

import { CommonModule } from '@angular/common';
import { Component, input } from '@angular/core';
import { ParameterHeaderComponent } from './parameter-header.component';
import { MatFormFieldModule } from '@angular/material/form-field';
import { ReactiveFormsModule } from '@angular/forms';
import { MatInputModule } from '@angular/material/input';
import { ParameterHintComponent } from './parameter-hint.component';
import {
  ParameterHintType,
  TextParameterFormField,
} from 'src/app/common/schema/form-types';
import { MatAutocompleteModule } from '@angular/material/autocomplete';

/**
 * A form field of parameter in the new-inspection dialog.
 */
@Component({
  selector: 'khi-new-inspection-text-parameter',
  templateUrl: './text-parameter.component.html',
  styleUrls: ['./text-parameter.component.sass'],
  imports: [
    CommonModule,
    ParameterHeaderComponent,
    MatInputModule,
    MatFormFieldModule,
    ReactiveFormsModule,
    ParameterHintComponent,
    MatAutocompleteModule,
  ],
})
export class TextParameterComponent {
  readonly ParameterHintType = ParameterHintType;
  /**
   * The spec of this text type parameter.
   */
  parameter = input.required<TextParameterFormField>();
}
