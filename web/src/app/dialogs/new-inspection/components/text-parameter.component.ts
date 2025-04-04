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
import { Component, inject, input, OnInit } from '@angular/core';
import { ParameterHeaderComponent } from './parameter-header.component';
import { MatFormFieldModule } from '@angular/material/form-field';
import { ReactiveFormsModule } from '@angular/forms';
import { MatInputModule } from '@angular/material/input';
import { ParameterHintComponent } from './parameter-hint.component';
import {
  ParameterHintType,
  TextParameterFormField,
} from 'src/app/common/schema/form-types';
import {
  MatAutocompleteModule,
  MatAutocompleteSelectedEvent,
} from '@angular/material/autocomplete';
import { PARAMETER_STORE } from './service/parameter-store';
import { Observable, Subject, takeUntil } from 'rxjs';

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
export class TextParameterComponent implements OnInit {
  readonly ParameterHintType = ParameterHintType;
  readonly destroyed = new Subject();
  /**
   * The spec of this text type parameter.
   */
  parameter = input.required<TextParameterFormField>();

  store = inject(PARAMETER_STORE);

  value!: Observable<string>;

  ngOnInit(): void {
    this.value = this.store
      .watch<string>(this.parameter().id)
      .pipe(takeUntil(this.destroyed));
  }

  onInput(ev: Event) {
    this.store.set(this.parameter().id, (ev.target as HTMLInputElement).value);
  }

  onOptionSelected(ev: MatAutocompleteSelectedEvent) {
    this.store.set(this.parameter().id, ev.option.value);
  }
}
