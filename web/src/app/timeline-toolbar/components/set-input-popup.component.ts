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

import { Component, computed, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { SetInputComponent } from '../../shared/components/set-input/set-input.component';

/**
 * A wrapper component for SetInputComponent to be used in a popup.
 * It handles the conversion between Set<string> used in toolbar and
 * SetInputItem[] used in SetInputComponent.
 * Uses modern Angular Signal-based inputs and outputs.
 */
@Component({
  selector: 'khi-timeline-set-input-popup',
  templateUrl: './set-input-popup.component.html',
  styleUrls: ['./set-input-popup.component.scss'],
  imports: [CommonModule, MatButtonModule, MatIconModule, SetInputComponent],
})
export class SetInputPopupComponent {
  /** The label to display in the popup header. */
  readonly label = input<string>('');

  /** The available choices as a Set of strings. */
  readonly choices = input<Set<string>>(new Set());

  /** The currently selected items as a Set of strings. */
  readonly selectedItems = input<Set<string>>(new Set());

  /** Emits the updated selected items when selection changes. */
  readonly selectedItemsChange = output<Set<string>>();

  /** Emits when the close button is clicked. */
  readonly closeButtonClicked = output<void>();

  /**
   * Computes the `SetInputItem` objects for the choices.
   */
  protected readonly mappedChoices = computed(() => {
    return Array.from(this.choices()).map((id) => ({
      id,
      value: id,
    }));
  });

  /**
   * Computes the array of selected item IDs.
   */
  protected readonly mappedSelectedItems = computed(() => {
    return Array.from(this.selectedItems());
  });

  protected onSelectionChange(selectedIds: string[]): void {
    this.selectedItemsChange.emit(new Set(selectedIds));
  }

  protected onClose(): void {
    this.closeButtonClicked.emit();
  }
}
