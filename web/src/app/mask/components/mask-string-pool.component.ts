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

import { CommonModule } from '@angular/common';
import { Component, computed, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import {
  CandidateToggleEvent,
  MaskCandidateItemViewModel,
} from 'src/app/mask/types/mask-viewmodel';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Displays the dataset string pool search controls, custom substring input, and selectable candidate list.
 */
@Component({
  selector: 'khi-mask-string-pool',
  imports: [
    CommonModule,
    FormsModule,
    MatButtonModule,
    MatCheckboxModule,
    MatIconModule,
    MatTooltipModule,
    KHIIconRegistrationModule,
  ],
  templateUrl: './mask-string-pool.component.html',
  styleUrls: ['./mask-string-pool.component.scss'],
})
export class MaskStringPoolComponent {
  /**
   * Active search query filtering candidate strings in the string pool.
   */
  public readonly searchQuery = input.required<string>();

  /**
   * Inline RE2 syntax validation error message for the active search query.
   */
  public readonly searchRegexError = input<string>('');

  /**
   * Candidate strings currently loaded from the Workbench string pool.
   */
  public readonly candidates =
    input.required<readonly MaskCandidateItemViewModel[]>();

  /**
   * Total count of unique strings matching the search query on the backend.
   */
  public readonly totalCandidateCount = input.required<number>();

  /**
   * Whether candidate strings are currently being fetched.
   */
  public readonly isLoadingCandidates = input.required<boolean>();

  /**
   * Emits when the candidate search query text changes.
   */
  public readonly searchQueryChange = output<string>();

  /**
   * Emits when the user selects all currently displayed candidate strings in bulk.
   */
  public readonly selectShownCandidates = output<void>();

  /**
   * Emits when the user requests the next page of candidate strings.
   */
  public readonly loadMoreCandidates = output<void>();

  /**
   * Emits when a candidate string checkbox is toggled.
   */
  public readonly candidateToggle = output<CandidateToggleEvent>();

  /**
   * Emits when the user submits a custom substring to mask.
   */
  public readonly addCustomSubstring = output<string>();

  /**
   * Local draft text for adding an arbitrary substring rule.
   */
  protected readonly customSubstringDraft = signal<string>('');

  /**
   * Whether additional pages of candidate strings remain to be loaded.
   */
  protected readonly hasMoreCandidates = computed<boolean>(
    () => this.candidates().length < this.totalCandidateCount(),
  );

  /**
   * Whether the custom substring input contains a non-empty value.
   */
  protected readonly canAddCustomSubstring = computed<boolean>(
    () => this.customSubstringDraft().length > 0,
  );

  /**
   * Whether an inline RE2 syntax validation error should be displayed below the search input.
   */
  protected readonly hasSearchRegexError = computed<boolean>(
    () => this.searchRegexError().length > 0,
  );

  /**
   * Count of currently displayed candidate strings that are not yet selected.
   */
  protected readonly unselectedShownCount = computed<number>(
    () => this.candidates().filter((c) => !c.selected).length,
  );

  /**
   * Whether the bulk "Select shown" button can be triggered.
   */
  protected readonly canSelectShown = computed<boolean>(
    () => this.unselectedShownCount() > 0 && !this.isLoadingCandidates(),
  );

  /**
   * Button label displaying how many unselected candidate strings will be added in bulk.
   */
  protected readonly selectShownLabel = computed<string>(() => {
    const count = this.unselectedShownCount();
    return count > 0 ? `Select shown (${count})` : 'Select shown';
  });

  protected onSearchInputChange(value: string): void {
    this.searchQueryChange.emit(value);
  }

  protected onClearSearch(): void {
    this.searchQueryChange.emit('');
  }

  protected onCustomSubstringInput(value: string): void {
    this.customSubstringDraft.set(value);
  }

  protected onSubmitCustomSubstring(): void {
    const target = this.customSubstringDraft();
    if (target.length === 0) {
      return;
    }
    this.addCustomSubstring.emit(target);
    this.customSubstringDraft.set('');
  }

  protected onCandidateCheckboxChange(value: string, checked: boolean): void {
    this.candidateToggle.emit({ target: value, checked });
  }
}
