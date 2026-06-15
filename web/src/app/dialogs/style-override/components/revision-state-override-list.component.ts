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
import { RevisionStateStyle } from 'src/app/store/domain/style';
import { RevisionStateStyleOverrideViewModel } from 'src/app/dialogs/style-override/types/style-override-viewmodel';

/**
 * Dumb component that presents a list of revision states and allows overriding their colors.
 */
@Component({
  selector: 'khi-revision-state-override-list',
  standalone: true,
  imports: [
    CommonModule,
    MatButtonModule,
    MatIconModule,
    KHIIconRegistrationModule,
  ],
  templateUrl: './revision-state-override-list.component.html',
  styleUrls: ['./revision-state-override-list.component.scss'],
})
export class RevisionStateOverrideListComponent {
  protected readonly RevisionStateStyle = RevisionStateStyle;

  /** List of revision state view models to display. */
  readonly revisionStates =
    input.required<RevisionStateStyleOverrideViewModel[]>();

  /** Emitted when a revision state's color is overridden. */
  readonly colorChange = output<{
    readonly id: number;
    readonly hexColor: string;
  }>();

  /** Emitted when a revision state's color override is reset. */
  readonly resetColor = output<number>();

  /**
   * Handles native color input element changes.
   * @param id The ID of the revision state.
   * @param event The native input event from color picker.
   */
  protected onColorPickerChange(id: number, event: Event): void {
    const inputElement = event.target as HTMLInputElement;
    if (inputElement) {
      this.colorChange.emit({ id, hexColor: inputElement.value });
    }
  }

  /**
   * Copies the provided text to the clipboard.
   * @param text The text to copy.
   */
  protected copyToClipboard(text: string): void {
    navigator.clipboard?.writeText(text);
  }
}
