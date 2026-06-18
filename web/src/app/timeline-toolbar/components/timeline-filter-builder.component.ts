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
  computed,
  effect,
  input,
  model,
  output,
  signal,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import {
  SetInputComponent,
  SetInputItem,
} from 'src/app/shared/components/set-input/set-input.component';
import { TimelineType } from 'src/app/store/domain/style';
import { RendererConvertUtil } from 'src/app/timeline/components/canvas/convertutil';

/**
 * An interactive timeline filter builder component.
 * This is a dumb component that provides a clean UI for building timeline filters.
 */
@Component({
  selector: 'khi-timeline-filter-builder',
  templateUrl: './timeline-filter-builder.component.html',
  styleUrls: ['./timeline-filter-builder.component.scss'],
  imports: [
    CommonModule,
    MatAutocompleteModule,
    MatInputModule,
    MatFormFieldModule,
    MatButtonToggleModule,
    MatButtonModule,
    MatIconModule,
    KHIIconRegistrationModule,
    SetInputComponent,
  ],
})
export class TimelineFilterBuilderComponent {
  /** The list of available timeline types. */
  readonly timelineTypes = input<TimelineType[]>([]);

  /** The list of matching candidates for the selected timeline type. */
  readonly candidates = input<string[]>([]);

  /** Map of timeline type labels (lowercase) to their candidate counts. */
  readonly typeCandidateCounts = input<Record<string, number>>({});

  /** Whether to display the delete trash can icon button. */
  readonly showDeleteButton = input<boolean>(false);

  /** The selected timeline type (model). */
  readonly selectedTimelineType = model<string>('*');

  /** The selected filter mode: regex or selection (model). */
  readonly filterMode = model<'regex' | 'selection'>('selection');

  /** The current raw regex filter string input (model). */
  readonly regexValue = model<string>('');

  /** The selected candidate values in selection mode (model). */
  readonly selectedCandidates = model<string[]>([]);

  /** Holds the current text input query for autocomplete filtering. */
  protected readonly typeInputQuery = signal<string>('');

  private lastSyncedType = '*';

  /** Filters the timeline types case-insensitively based on the current text input query and sorts them by candidate count. */
  protected readonly filteredTimelineTypes = computed<TimelineType[]>(() => {
    const query = this.typeInputQuery().trim().toLowerCase();
    const allTypes = this.timelineTypes();
    const counts = this.typeCandidateCounts();

    const filtered = query
      ? allTypes.filter((t) => t.label.toLowerCase().includes(query))
      : [...allTypes];

    return filtered.sort((a, b) => {
      const countA = counts[a.label.toLowerCase()] || 0;
      const countB = counts[b.label.toLowerCase()] || 0;
      if (countA !== countB) {
        return countB - countA; // Descending order
      }
      return a.label.localeCompare(b.label); // Alphabetical fallback
    });
  });

  /** Computes candidate choices in the format expected by SetInputComponent. */
  protected readonly mappedCandidates = computed<SetInputItem[]>(() => {
    return this.candidates().map((c) => ({
      id: c,
      value: c,
    }));
  });

  /** Emits when the close button is clicked. */
  readonly closeButtonClicked = output<void>();

  /** Emits when the delete trash can button is clicked. */
  readonly deleteButtonClicked = output<void>();

  /** Emits the final filter configuration. */
  readonly confirm = output<{
    timelineType: string;
    mode: 'regex' | 'selection';
    value: string;
  }>();

  constructor() {
    // Synchronizes the autocomplete input field and handles states when selection updates.
    effect(() => {
      const currentType = this.selectedTimelineType();
      if (currentType === '*') {
        this.filterMode.set('regex');
        this.selectedCandidates.set([]);
      }
      if (currentType !== this.lastSyncedType) {
        this.lastSyncedType = currentType;
        this.typeInputQuery.set(currentType === '*' ? '' : currentType);
      }
    });
  }

  /**
   * Returns the computed CSS color for a timeline type's foreground color.
   */
  protected getTypeForegroundColor(type: TimelineType): string {
    return RendererConvertUtil.hdrColorToCSSColor([
      type.typeChipForegroundColor.r,
      type.typeChipForegroundColor.g,
      type.typeChipForegroundColor.b,
      type.typeChipForegroundColor.a,
    ]);
  }

  /**
   * Returns the computed CSS color for a timeline type's background color.
   */
  protected getTypeBackgroundColor(type: TimelineType): string {
    return RendererConvertUtil.hdrColorToCSSColor([
      type.typeChipBackgroundColor.r,
      type.typeChipBackgroundColor.g,
      type.typeChipBackgroundColor.b,
      type.typeChipBackgroundColor.a,
    ]);
  }

  /**
   * Returns the computed CSS color for a timeline type's row background color.
   */
  protected getTypeRowBackgroundColor(type: TimelineType): string {
    return RendererConvertUtil.hdrColorToCSSColor([
      type.backgroundColor.r,
      type.backgroundColor.g,
      type.backgroundColor.b,
      type.backgroundColor.a,
    ]);
  }

  /**
   * Returns the computed CSS color for a timeline type's row foreground color.
   */
  protected getTypeRowForegroundColor(type: TimelineType): string {
    return RendererConvertUtil.hdrColorToCSSColor([
      type.foregroundColor.r,
      type.foregroundColor.g,
      type.foregroundColor.b,
      type.foregroundColor.a,
    ]);
  }

  /**
   * Handles changes to the autocomplete text input query and matches against exact timeline types.
   */
  protected onTypeInputChange(value: string): void {
    this.typeInputQuery.set(value);
    const match = this.timelineTypes().find(
      (t) => t.label.toLowerCase() === value.trim().toLowerCase(),
    );
    const newType = match ? match.label : '*';
    this.lastSyncedType = newType;
    this.selectedTimelineType.set(newType);
  }

  /**
   * Handles selection of a timeline type from the autocomplete dropdown.
   */
  protected onTimelineTypeSelected(value: string): void {
    this.lastSyncedType = value;
    this.selectedTimelineType.set(value);
    this.typeInputQuery.set(value === '*' ? '' : value);
  }

  /**
   * A computed signal indicating if the "Add Filter" button should be disabled.
   */
  protected readonly isAddDisabled = computed(() => {
    if (this.filterMode() === 'regex') {
      return !this.regexValue().trim();
    } else {
      return !this.selectedCandidates().length;
    }
  });

  /**
   * Triggers when the user clicks "Add Filter".
   */
  protected onAdd(): void {
    if (this.isAddDisabled()) {
      return;
    }

    const value =
      this.filterMode() === 'regex'
        ? this.regexValue().trim()
        : this.selectedCandidates().join('|');

    this.confirm.emit({
      timelineType: this.selectedTimelineType(),
      mode: this.filterMode(),
      value,
    });
  }
}
