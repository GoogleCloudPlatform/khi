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

import {
  Component,
  ElementRef,
  input,
  output,
  TemplateRef,
  ViewChild,
  computed,
} from '@angular/core';
import { FormControl, ReactiveFormsModule } from '@angular/forms';
import { MatChipInputEvent, MatChipsModule } from '@angular/material/chips';
import {
  MatAutocompleteModule,
  MatAutocompleteSelectedEvent,
} from '@angular/material/autocomplete';
import { startWith } from 'rxjs';
import { CommonModule } from '@angular/common';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { toSignal } from '@angular/core/rxjs-interop';

/**
 * Represents an item in the set input.
 */
export interface SetInputItem {
  /**
   * Unique identifier for the item.
   * This is also used as the display label.
   */
  id: string;
  /**
   * The actual value associated with the item.
   */
  value: unknown;
}

import { SetInputDefaultChipComponent } from './default-chip.component';
import { SetInputDefaultOptionComponent } from './default-option.component';
import { COMMA, ENTER } from '@angular/cdk/keycodes';

/**
 * A component that allows selecting multiple items from a set of choices.
 * Supports custom rendering for chips and autocomplete options.
 * Can optionally allow custom values that are not present in the choices.
 */
@Component({
  selector: 'khi-shared-set-input',
  templateUrl: './set-input.component.html',
  styleUrls: ['./set-input.component.scss'],
  imports: [
    CommonModule,
    MatFormFieldModule,
    MatInputModule,
    MatChipsModule,
    MatIconModule,
    MatButtonModule,
    ReactiveFormsModule,
    MatAutocompleteModule,
    MatTooltipModule,
    SetInputDefaultChipComponent,
    SetInputDefaultOptionComponent,
  ],
})
export class SetInputComponent {
  separatorKeysCodes: number[] = [ENTER, COMMA];
  /** The currently selected item IDs. */
  selectedItems = input<string[]>([]);
  /** The available choices. */
  public choices = input<SetInputItem[]>([]);
  /** Custom template for rendering selected items (chips). */
  public chipTemplate = input<TemplateRef<unknown> | null>(null);
  /** Custom template for rendering autocomplete options. */
  public optionTemplate = input<TemplateRef<unknown> | null>(null);
  /** Whether to allow adding values that are not in the choices list. Defaults to false. */
  public allowCustomValues = input<boolean>(false);
  /** Whether to show the "Add all" button. Defaults to true. */
  public showAddAll = input<boolean>(true);
  /** Whether to show the "Remove all" button. Defaults to true. */
  public showRemoveAll = input<boolean>(true);

  /** Whether to allow subtractive values. Defaults to false. */
  public allowSubtractiveValue = input<boolean>(false);

  /** Emits the updated list of selected item IDs when selection changes. */
  public selectedItemsChange = output<string[]>();

  inputCtrl = new FormControl<string>('', { nonNullable: true });

  @ViewChild('inputElement') inputElement!: ElementRef<HTMLInputElement>;

  /**
   * Computes the `SetInputItem` objects for the selected IDs.
   * If an ID is not found in choices, a transient item is created.
   */
  viewSelectedItems = computed(() => {
    const choices = this.choices();
    const selected = this.selectedItems();
    const choiceMap = new Map(choices.map((c) => [c.id, c]));
    return selected.map((id) => {
      // Return existing choice or creating a transient one for display
      return choiceMap.get(id) ?? { id, value: id };
    });
  });

  private inputValue = toSignal(
    this.inputCtrl.valueChanges.pipe(startWith('')),
    { initialValue: '' },
  );

  /**
   * Computes the available candidates for the text field autocomplete.
   * Filters out already selected items and matches against the input text.
   */
  textFieldCandidates = computed(() => {
    const value = this.inputValue();
    const name = value;
    const selectedIdSet = new Set(this.selectedItems());
    const available = this.choices().filter((c) => !selectedIdSet.has(c.id));

    if (!name) return available;
    let lowerName = name.toLowerCase();
    if (this.allowSubtractiveValue() && lowerName.startsWith('-')) {
      lowerName = lowerName.substring(1);
    }
    // Simple filtering by id
    return available.filter((item) =>
      item.id.toLowerCase().includes(lowerName),
    );
  });

  /** Removes a selected item. */
  removeItem(removedItem: SetInputItem) {
    const newItems = this.selectedItems().filter((id) => id !== removedItem.id);
    this.selectedItemsChange.emit(newItems);
  }

  /** Adds an item from the text input (chip input event). */
  addItemFromText(event: MatChipInputEvent): void {
    const inputValue = (event.value || '').trim();

    if (inputValue) {
      // Find ID if it matches a choice id
      const existingChoice = this.choices().find((c) => c.id === inputValue);
      if (existingChoice || this.allowCustomValues()) {
        const idToAdd = existingChoice ? existingChoice.id : inputValue;
        const newItems = this.getUniqueString([
          ...this.selectedItems(),
          idToAdd,
        ]);
        this.selectedItemsChange.emit(newItems);
      }
    }

    // Clear the input value
    event.chipInput!.clear();
    this.inputCtrl.setValue('');
  }

  /** Adds all available choices to the selection. */
  addAll(): void {
    // Add all choices that are not already selected
    const allIds = this.getUniqueString([
      ...this.selectedItems(),
      ...this.choices().map((c) => c.id),
    ]);
    this.selectedItemsChange.emit(allIds);
  }

  /** Removes all selected items. */
  removeAll(): void {
    this.selectedItemsChange.emit([]);
  }

  /** Selects only the specified item, clearing others. */
  selectOnly(item: SetInputItem) {
    this.selectedItemsChange.emit([item.id]);
  }

  /** Handle selection from autocomplete. */
  selected(event: MatAutocompleteSelectedEvent): void {
    const id = event.option.value as string;
    let newItem = id;
    if (this.allowSubtractiveValue() && this.inputValue().startsWith('-')) {
      newItem = '-' + id;
    }
    const newItems = this.getUniqueString([...this.selectedItems(), newItem]);
    this.selectedItemsChange.emit(newItems);
    this.inputElement.nativeElement.value = '';
    this.inputCtrl.setValue('');
  }

  private getUniqueString(items: string[]): string[] {
    return Array.from(new Set(items));
  }
}
