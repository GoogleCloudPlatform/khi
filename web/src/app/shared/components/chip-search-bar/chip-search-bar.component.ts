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
   * Indicates whether any chips or draft text exist.
   */
  public readonly hasQuery = computed(
    () => this.searchTerms().length > 0 || this.draft().trim().length > 0,
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
   * Removes a chip at the specified index.
   * @param index Index of the chip to remove.
   * @param event The associated mouse click event.
   */
  onRemoveChip(index: number, event: MouseEvent) {
    event.stopPropagation();
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
      const rawSegments = pastedText.split(/[|\r\n]+/);
      const validSegments: string[] = [];
      for (const seg of rawSegments) {
        const trimmed = seg.trim();
        if (trimmed) {
          validSegments.push(trimmed);
        }
      }
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
