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

import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import {
  LogTypeStyleOverrideViewModel,
  LogTypeOverrideEvent,
} from 'src/app/dialogs/style-override/types/style-override-viewmodel';

/**
 * Dumb component that presents a list of log types and allows overriding their background and foreground colors.
 */
@Component({
  selector: 'khi-log-type-override-list',
  standalone: true,
  imports: [
    CommonModule,
    MatButtonModule,
    MatIconModule,
    KHIIconRegistrationModule,
  ],
  templateUrl: './log-type-override-list.component.html',
  styleUrls: ['./log-type-override-list.component.scss'],
})
export class LogTypeOverrideListComponent {
  /** List of log type view models to display. */
  readonly logTypes = input.required<LogTypeStyleOverrideViewModel[]>();

  /** Emitted when a log type property is overridden. */
  readonly propertyChange = output<LogTypeOverrideEvent>();

  /** Emitted when a log type's styles are reset. */
  readonly resetColor = output<number>();

  /**
   * Handles color picker changes for a specific color property.
   * @param id The ID of the log type.
   * @param prop The property to override ('backgroundColor' | 'foregroundColor').
   * @param event The native input event from the color picker.
   */
  protected onColorChange(
    id: number,
    prop: 'backgroundColor' | 'foregroundColor',
    event: Event,
  ): void {
    const inputElement = event.target as HTMLInputElement;
    if (inputElement) {
      this.propertyChange.emit({
        id,
        [prop]: inputElement.value,
      });
    }
  }

  /**
   * Converts a hex color string to Go style.Color code format.
   * @param hex The hex color string (e.g. '#ff0000').
   */
  protected getGoColorCode(hex: string): string {
    const r = parseInt(hex.substring(1, 3), 16) / 255;
    const g = parseInt(hex.substring(3, 5), 16) / 255;
    const b = parseInt(hex.substring(5, 7), 16) / 255;
    return `style.Color{R: ${r.toFixed(3)}, G: ${g.toFixed(3)}, B: ${b.toFixed(3)}, A: 1.0}`;
  }

  /**
   * Copies the provided text to the clipboard.
   * @param text The text to copy.
   */
  protected copyToClipboard(text: string): void {
    navigator.clipboard?.writeText(text);
  }
}
