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
import {
  GroupParameterFormField,
  ParameterInputType,
} from 'src/app/common/schema/form-types';
import { TextParameterComponent } from './text-parameter.component';
import { FileParameterComponent } from './file-parameter.component';
import { ParameterHeaderComponent } from './parameter-header.component';
import { ParameterHintComponent } from './parameter-hint.component';

/**
 * A collection of form fields.
 */
@Component({
  selector: 'khi-new-inspection-group-parameter',
  templateUrl: './group-parameter.component.html',
  styleUrls: ['./group-parameter.component.sass'],
  imports: [
    TextParameterComponent,
    FileParameterComponent,
    ParameterHeaderComponent,
    ParameterHintComponent,
  ],
})
export class GroupParameterComponent {
  readonly ParameterInputType = ParameterInputType;
  /**
   * The setting of this group type form field.
   */
  parameter = input.required<GroupParameterFormField>();
}
