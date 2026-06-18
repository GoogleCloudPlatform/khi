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

import {
  Component,
  ElementRef,
  input,
  model,
  output,
  viewChild,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Provides a slim text field optimized for CEL expression input with real-time validation.
 */
@Component({
  selector: 'khi-timeline-cel-input',
  templateUrl: './cel-input.component.html',
  styleUrls: ['./cel-input.component.scss'],
  imports: [
    CommonModule,
    FormsModule,
    MatIconModule,
    MatTooltipModule,
    KHIIconRegistrationModule,
  ],
})
export class CelInputComponent {
  /**
   * Specifies the validation error message to display.
   */
  readonly errorMessage = input('');

  /**
   * Specifies the tooltip text displayed when hovering over the icon.
   */
  readonly tooltip = input('');

  /**
   * Specifies the icon name (Material symbol) displayed next to the input field.
   */
  readonly icon = input('');

  /**
   * Specifies the placeholder text for the input field.
   */
  readonly placeholder = input('Enter CEL expression');

  /**
   * Holds the two-way bound CEL expression string.
   */
  readonly value = model('');

  /**
   * Emits when the input receives focus.
   */
  readonly focused = output<void>();

  private readonly inputElement =
    viewChild<ElementRef<HTMLInputElement>>('inputElement');

  /**
   * Focuses the internal input element.
   */
  public focus(): void {
    this.inputElement()?.nativeElement.focus();
  }
}
