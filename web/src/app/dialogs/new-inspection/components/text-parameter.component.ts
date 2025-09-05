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
  ParameterFormValidationTiming,
  ParameterHintType,
  TextParameterFormField,
} from 'src/app/common/schema/form-types';
import {
  MatAutocompleteModule,
  MatAutocompleteSelectedEvent,
} from '@angular/material/autocomplete';
import { PARAMETER_STORE } from './service/parameter-store';
import {
  distinctUntilChanged,
  merge,
  Observable,
  ReplaySubject,
  Subject,
  takeUntil,
} from 'rxjs';

/**
 * A form field of parameter in the new-inspection dialog.
 */
@Component({
  selector: 'khi-new-inspection-text-parameter',
  templateUrl: './text-parameter.component.html',
  styleUrls: ['./text-parameter.component.scss'],
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
  /**
   * Exposes ParameterHintType enum to the template.
   */
  readonly ParameterHintType = ParameterHintType;
  /**
   * Subject that emits when the component is destroyed. Used for unsubscribing from observables.
   */
  readonly destroyed = new Subject();
  /**
   * The spec of this text type parameter.
   */
  parameter = input.required<TextParameterFormField>();

  /**
   * Injects the PARAMETER_STORE service.
   */
  store = inject(PARAMETER_STORE);

  /**
   * Observable that emits the current display value of the parameter.
   */
  value!: Observable<string>;

  /**
   * ReplaySubject that keeps input values when validationTiming is 'onblur'.
   */
  private readonly stagingInput = new ReplaySubject<string>(1);

  /**
   * Initializes the component.
   * Subscribes to the parameter store and staging input to update the `value` observable.
   */
  ngOnInit(): void {
    this.value = merge(
      this.store.watch<string>(this.parameter().id),
      this.stagingInput,
    ).pipe(distinctUntilChanged(), takeUntil(this.destroyed));
  }

  /**
   * Handles input events from the text field.
   * Updates the parameter store immediately if validationTiming is 'ParameterFormValidationTiming.Change', otherwise stages the input.
   */
  onInput(ev: Event) {
    if (
      this.parameter().validationTiming === ParameterFormValidationTiming.Change
    ) {
      this.store.set(
        this.parameter().id,
        (ev.target as HTMLInputElement).value,
      );
    } else {
      this.stagingInput.next((ev.target as HTMLInputElement).value);
    }
  }

  /**
   * Handles blur events from the text field.
   * Updates the parameter store if validationTiming is 'ParameterFormValidationTiming.Blur'.
   */
  onBlur(ev: Event) {
    if (
      this.parameter().validationTiming === ParameterFormValidationTiming.Blur
    ) {
      this.store.set(
        this.parameter().id,
        (ev.target as HTMLInputElement).value,
      );
    }
  }

  /**
   * Handles the selection of an option from the autocomplete dropdown.
   */
  onOptionSelected(ev: MatAutocompleteSelectedEvent) {
    this.store.set(this.parameter().id, ev.option.value);
  }
}
