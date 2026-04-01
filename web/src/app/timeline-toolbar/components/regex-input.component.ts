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

import {
  Component,
  ElementRef,
  input,
  model,
  effect,
  viewChild,
} from '@angular/core';
import { FormControl, ReactiveFormsModule } from '@angular/forms';
import { RegexValidator } from './regex-validator';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';

@Component({
  selector: 'khi-timeline-regex-input',
  templateUrl: './regex-input.component.html',
  styleUrls: ['./regex-input.component.scss'],
  imports: [MatFormFieldModule, MatInputModule, ReactiveFormsModule],
})
export class RegexInputComponent {
  private readonly regexInputElement =
    viewChild<ElementRef<HTMLInputElement>>('regexInputElement');

  /**
   * The label to display for the input field.
   */
  readonly label = input('');

  /**
   * The current regex filter value.
   */
  readonly value = model('');

  readonly regexInput: FormControl = new FormControl('', [RegexValidator()]);

  constructor() {
    // Sync model value to Form Control
    effect(() => {
      const val = this.value();
      if (this.regexInput.value !== val) {
        this.regexInput.setValue(val || '', { emitEvent: false });
      }
    });
  }

  protected regexFormErrorMessage(): string {
    return (this.regexInput.errors?.['regex'] as string) || '';
  }

  protected onFilterChange() {
    if (!this.regexInput.valid) return;
    this.value.set(this.regexInput.value || '');
  }

  /**
   * Focuses the regex input element.
   */
  public focus() {
    this.regexInputElement()?.nativeElement.focus();
  }
}
