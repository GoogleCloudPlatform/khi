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
  computed,
  ElementRef,
  input,
  output,
  viewChild,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltip } from '@angular/material/tooltip';
import { MatButtonModule } from '@angular/material/button';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Component responsible for rendering a search input bar with match counts and navigation controls.
 */
@Component({
  selector: 'khi-search-bar',
  templateUrl: './search-bar.component.html',
  styleUrls: ['./search-bar.component.scss'],
  imports: [
    CommonModule,
    MatIconModule,
    MatTooltip,
    MatButtonModule,
    KHIIconRegistrationModule,
  ],
})
export class SearchBarComponent {
  /**
   * The current search query string.
   */
  public query = input<string>('');

  /**
   * The total number of matches found.
   */
  public matchCount = input<number>(0);

  /**
   * The 1-based index of the currently active match.
   */
  public currentMatchIndex = input<number>(0);

  /**
   * Computed string label representing the current match count or no matches state.
   */
  public readonly matchLabel = computed(() => {
    const count = this.matchCount();
    if (count > 0) {
      return `${this.currentMatchIndex()} / ${count}`;
    }
    if (this.query()) {
      return 'No matches';
    }
    return '0 / 0';
  });

  /**
   * Emitted when the search query string is updated.
   */
  public queryChange = output<string>();

  /**
   * Emitted when the next match navigation is triggered.
   */
  public nextMatch = output<void>();

  /**
   * Emitted when the previous match navigation is triggered.
   */
  public prevMatch = output<void>();

  /**
   * Emitted when the search bar close button is clicked.
   */
  public closeSearch = output<void>();

  /**
   * Reference to the search input element for focus management.
   */
  public readonly searchInput =
    viewChild<ElementRef<HTMLInputElement>>('searchInput');

  /**
   * Emits the updated query string when the input field value changes.
   * @param query The new search query string.
   */
  onInput(query: string) {
    this.queryChange.emit(query);
  }

  /**
   * Focuses on the search input element.
   */
  focus() {
    this.searchInput()?.nativeElement.focus();
  }
}
