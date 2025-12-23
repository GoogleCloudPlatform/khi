/**
 * Copyright 2024 Google LLC
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
import {
  Component,
  inject,
  input,
  OnInit,
  OnDestroy,
  computed,
  signal,
} from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { Subject, takeUntil } from 'rxjs';
import {
  ParameterHintType,
  SetParameterFormField,
} from 'src/app/common/schema/form-types';
import {
  SetInputComponent,
  SetInputItem,
} from 'src/app/shared/components/set-input/set-input.component';
import { ParameterHeaderComponent } from './parameter-header.component';
import { ParameterHintComponent } from './parameter-hint.component';
import { PARAMETER_STORE } from './service/parameter-store';
import { SetInputAliasChipComponent } from './alias-chip.component';
import { SetInputDescriptionOptionComponent } from './option-item.component';

/**
 * A form field for set type parameter in the new-inspection dialog.
 */
@Component({
  selector: 'khi-new-inspection-set-parameter',
  templateUrl: './set-parameter.component.html',
  styleUrls: ['./set-parameter.component.scss'],
  imports: [
    CommonModule,
    ParameterHeaderComponent,
    SetInputComponent,
    ReactiveFormsModule,
    ParameterHintComponent,
    SetInputAliasChipComponent,
    SetInputDescriptionOptionComponent,
  ],
})
export class SetParameterComponent implements OnInit, OnDestroy {
  readonly ParameterHintType = ParameterHintType;
  readonly parameter = input.required<SetParameterFormField>();

  private readonly store = inject(PARAMETER_STORE);
  private readonly destroyed = new Subject<void>();
  readonly stagingInput = signal<string[]>([]);

  choices = computed(() => {
    return this.parameter().options.map(
      (opt): SetInputItem => ({ id: opt.id, value: opt }),
    );
  });

  ngOnInit(): void {
    this.store
      .watch<string[]>(this.parameter().id)
      .pipe(takeUntil(this.destroyed))
      .subscribe((value) => {
        this.stagingInput.set(value);
      });
  }

  ngOnDestroy(): void {
    this.destroyed.next();
    this.destroyed.complete();
  }

  onSelectionChange(selectedItems: string[]): void {
    this.stagingInput.set(selectedItems);
    this.store.set(this.parameter().id, selectedItems);
  }
}
