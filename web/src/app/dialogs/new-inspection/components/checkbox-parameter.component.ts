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
import { Component, computed, inject, input } from '@angular/core';
import { MatCheckboxModule } from '@angular/material/checkbox';
import {
  CheckboxParameterFormField,
  ParameterHintType,
} from 'src/app/common/schema/form-types';
import { ParameterHeaderComponent } from './parameter-header.component';
import { ParameterHintComponent } from './parameter-hint.component';
import { PARAMETER_STORE } from './service/parameter-store';

/**
 * A form field for checkbox type parameter in the new-inspection dialog.
 */
@Component({
  selector: 'khi-new-inspection-checkbox-parameter',
  templateUrl: './checkbox-parameter.component.html',
  styleUrls: ['./checkbox-parameter.component.scss'],
  imports: [
    CommonModule,
    MatCheckboxModule,
    ParameterHeaderComponent,
    ParameterHintComponent,
  ],
})
export class CheckboxParameterComponent {
  /**
   * Exposes ParameterHintType enum to the template.
   */
  protected readonly ParameterHintType = ParameterHintType;

  /**
   * The spec of this checkbox type parameter.
   */
  readonly parameter = input.required<CheckboxParameterFormField>();

  /**
   * Injects the PARAMETER_STORE service.
   */
  private readonly store = inject(PARAMETER_STORE);

  /**
   * Computed checked state of the checkbox parameter.
   */
  readonly isChecked = computed(() => {
    return (
      this.store.get<boolean>(this.parameter().id)() ?? this.parameter().default
    );
  });

  /**
   * Handles toggle change events from the checkbox.
   */
  onToggle(checked: boolean): void {
    if (!this.parameter().readonly) {
      this.store.set(this.parameter().id, checked);
    }
  }
}
