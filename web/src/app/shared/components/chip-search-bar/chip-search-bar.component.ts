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
  computed,
  input,
  model,
  signal,
  viewChild,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Splits raw input text by pipe delimiters or newlines into trimmed, non-empty search terms.
 * @param raw The raw string containing delimiters.
 */
function splitSearchTerms(raw: string): string[] {
  return raw
    .split(/[|\r\n]+/)
    .map((seg) => seg.trim())
    .filter((seg) => seg.length > 0);
}

/**
 * Dumb component responsible for rendering an inline multi-chip search input with OR logic.
 */
@Component({
  selector: 'khi-chip-search-bar',
  templateUrl: './chip-search-bar.component.html',
  styleUrls: ['./chip-search-bar.component.scss'],
  imports: [
    CommonModule,
    MatIconModule,
    MatTooltipModule,
    KHIIconRegistrationModule,
  ],
})
export class ChipSearchBarComponent {
  /**
   * The list of committed search term chips represented as a two-way model.
   */
  public searchTerms = model<string[]>([]);

  /**
   * Placeholder displayed when no search terms or draft text are entered.
   */
  public placeholder = input<string>('Search in log body...');

  /**
   * Reference to the text input element for direct focus management.
   */
  public readonly inputElement =
    viewChild<ElementRef<HTMLInputElement>>('inputElement');

  /**
   * The currently uncommitted draft text in the input.
   */
  public readonly draft = signal<string>('');

  /**
   * The index of the chip currently being edited, or null if no chip is in edit mode.
   */
  public readonly editingIndex = signal<number | null>(null);

  /**
   * The draft text currently being edited inside the active chip.
   */
  public readonly editingText = signal<string>('');

  /**
   * Reference to the inline chip edit input element.
   */
  public readonly chipEditInput =
    viewChild<ElementRef<HTMLInputElement>>('chipEditInput');

  /**
   * Indicates whether any chips or draft text exist.
   */
  public readonly hasQuery = computed(
    () =>
      this.searchTerms().length > 0 ||
      this.draft().trim().length > 0 ||
      this.editingIndex() !== null,
  );

  /**
   * Dynamic placeholder text indicating OR condition when chips are present.
   */
  public readonly dynamicPlaceholder = computed(() => {
    if (this.searchTerms().length > 0) {
      return 'Add OR condition...';
    }
    return this.placeholder();
  });

  /**
   * Handles user typing in the input field.
   * @param value The raw string value from the input field.
   */
  onDraftInput(value: string) {
    this.draft.set(value);
  }

  /**
   * Focuses on the text input element.
   */
  focus() {
    this.inputElement()?.nativeElement.focus();
  }

  /**
   * Handles container click to focus on the text input.
   */
  onContainerClick() {
    this.focus();
  }

  /**
   * Handles input blur to commit any pending draft as a chip.
   */
  onBlur() {
    this.commitDraft();
  }

  /**
   * Handles clicking on a chip to switch it into edit mode.
   * @param index Index of the clicked chip.
   * @param event The associated mouse click event.
   */
  onChipClick(index: number, event: MouseEvent) {
    event.stopPropagation();
    this.commitDraft();
    this.startChipEdit(index);
  }

  /**
   * Starts editing the chip at the specified index.
   * @param index Index of the chip to edit.
   */
  public startChipEdit(index: number) {
    if (index < 0 || index >= this.searchTerms().length) {
      return;
    }
    if (this.editingIndex() !== null && this.editingIndex() !== index) {
      const prevIndex = this.editingIndex()!;
      const prevLen = this.searchTerms().length;
      this.commitChipEdit();
      if (prevIndex < index) {
        index += this.searchTerms().length - prevLen;
      }
      if (index < 0 || index >= this.searchTerms().length) {
        return;
      }
    }
    this.editingIndex.set(index);
    this.editingText.set(this.searchTerms()[index] ?? '');
    setTimeout(() => {
      const input = this.chipEditInput()?.nativeElement;
      if (input) {
        input.focus();
        input.select();
      }
    }, 0);
  }

  /**
   * Handles user typing in the active chip edit input.
   * @param value The raw string value from the input element.
   */
  onChipEditInput(value: string) {
    this.editingText.set(value);
  }

  /**
   * Commits the edited chip text back into searchTerms.
   */
  public commitChipEdit() {
    const index = this.editingIndex();
    if (index === null) {
      return;
    }
    const segments = splitSearchTerms(this.editingText());
    if (segments.length === 0) {
      this.searchTerms.update((terms) => terms.filter((_, i) => i !== index));
    } else {
      this.searchTerms.update((terms) => [
        ...terms.slice(0, index),
        ...segments,
        ...terms.slice(index + 1),
      ]);
    }
    this.editingIndex.set(null);
    this.editingText.set('');
  }

  /**
   * Cancels the current chip edit and exits edit mode without modifying searchTerms.
   */
  public cancelChipEdit() {
    this.editingIndex.set(null);
    this.editingText.set('');
  }

  /**
   * Handles keyboard navigation in the active chip edit input.
   * @param event The keyboard event.
   */
  onChipEditKeyDown(event: KeyboardEvent) {
    if (event.key === 'Enter') {
      event.preventDefault();
      this.commitChipEdit();
      this.focus();
    } else if (event.key === 'Escape') {
      event.preventDefault();
      this.cancelChipEdit();
      this.focus();
    }
  }

  /**
   * Handles blur on the active chip edit input to commit changes.
   * @param index Index of the chip reporting the blur event.
   */
  onChipEditBlur(index: number) {
    if (this.editingIndex() !== index) {
      return;
    }
    this.commitChipEdit();
  }

  /**
   * Removes a chip at the specified index.
   * @param index Index of the chip to remove.
   * @param event The associated mouse click event.
   */
  onRemoveChip(index: number, event: MouseEvent) {
    event.stopPropagation();
    if (this.editingIndex() === index) {
      this.editingIndex.set(null);
      this.editingText.set('');
    } else if (this.editingIndex() !== null && this.editingIndex()! > index) {
      this.editingIndex.update((curr) => (curr !== null ? curr - 1 : null));
      setTimeout(() => {
        const input = this.chipEditInput()?.nativeElement;
        if (input) {
          input.focus();
          input.select();
        }
      }, 0);
    }
    this.searchTerms.update((terms) => terms.filter((_, i) => i !== index));
  }

  /**
   * Handles keyboard navigation and delimiter triggers (Enter, |, Backspace, Escape).
   * @param event The keyboard event.
   */
  onKeyDown(event: KeyboardEvent) {
    if (event.key === 'Enter' || event.key === '|') {
      event.preventDefault();
      this.commitDraft();
    } else if (event.key === 'Backspace') {
      if (!this.draft() && this.searchTerms().length > 0) {
        event.preventDefault();
        const currentTerms = this.searchTerms();
        const lastTerm = currentTerms[currentTerms.length - 1];
        this.searchTerms.set(currentTerms.slice(0, -1));
        this.draft.set(lastTerm);
      }
    } else if (event.key === 'Escape') {
      if (this.draft()) {
        this.draft.set('');
      } else if (this.searchTerms().length > 0) {
        this.clearAll();
      }
    }
  }

  /**
   * Handles paste events, splitting on pipe delimiters or newlines into distinct chips.
   * @param event The clipboard paste event.
   */
  onPaste(event: ClipboardEvent) {
    const pastedText = event.clipboardData?.getData('text') ?? '';
    if (pastedText.includes('|') || pastedText.includes('\n')) {
      event.preventDefault();
      const input = this.inputElement()?.nativeElement;
      const start = input?.selectionStart ?? 0;
      const end = input?.selectionEnd ?? 0;
      const currentVal = this.draft();
      const combined =
        currentVal.slice(0, start) + pastedText + currentVal.slice(end);

      const validSegments = splitSearchTerms(combined);
      if (validSegments.length > 0) {
        this.searchTerms.update((terms) => [...terms, ...validSegments]);
        this.draft.set('');
      }
    }
  }

  /**
   * Clears all chips and uncommitted draft text.
   * @param event Optional mouse event if triggered by clear button click.
   */
  onClearClick(event?: MouseEvent) {
    event?.stopPropagation();
    this.clearAll();
  }

  /**
   * Clears all chips and draft.
   */
  clearAll() {
    this.editingIndex.set(null);
    this.editingText.set('');
    this.searchTerms.set([]);
    this.draft.set('');
  }

  /**
   * Commits the current draft text as a chip.
   */
  public commitDraft() {
    const trimmed = this.draft().trim();
    if (trimmed) {
      this.searchTerms.update((terms) => [...terms, trimmed]);
      this.draft.set('');
    }
  }
}
