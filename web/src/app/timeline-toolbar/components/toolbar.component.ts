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
  HostListener,
  input,
  model,
  output,
  inject,
  signal,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { OverlayModule } from '@angular/cdk/overlay';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatButtonModule } from '@angular/material/button';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatSelectModule } from '@angular/material/select';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { TimelineFilterBuilderComponent } from './timeline-filter-builder.component';
import { TimelineFilterConfig } from '../types/filter-config';
import { TimelineType } from 'src/app/store/domain/style';

export enum ToolbarPopupStatus {
  None = 'NONE_OPEN',
  KindFilter = 'KIND_FILTER_OPEN',
  NamespaceFilter = 'NAMESPACE_FILTER_OPEN',
  SubresourceFilter = 'SUBRESOURCE_FILTER_OPEN',
}

@Component({
  selector: 'khi-timeline-toolbar',
  templateUrl: './toolbar.component.html',
  styleUrls: ['./toolbar.component.scss'],
  imports: [
    CommonModule,
    MatIconModule,
    KHIIconRegistrationModule,
    OverlayModule,
    MatButtonModule,
    MatButtonToggleModule,
    MatTooltipModule,
    TimelineFilterBuilderComponent,
    MatSelectModule,
    MatInputModule,
    MatFormFieldModule,
  ],
})
export class ToolbarComponent {
  private readonly snackbar = inject(MatSnackBar);

  // Inputs (Signals)
  readonly showButtonLabel = input(false);
  readonly kinds = input<Set<string>>(new Set());
  readonly includedKinds = model<Set<string>>(new Set());
  readonly namespaces = input<Set<string>>(new Set());
  readonly includedNamespaces = model<Set<string>>(new Set());
  readonly subresourceRelationships = input<Set<string>>(new Set());
  readonly includedSubresourceRelationships = model<Set<string>>(new Set());
  readonly timezoneShift = model(0);
  readonly logOrTimelineNotSelected = input(true);
  readonly nameFilter = model('');
  readonly selectedSeverity = model<string>('ANY');
  readonly logSearchQuery = model<string>('');

  // Timeline Filters (Dumb state)
  readonly timelineFilters = model<TimelineFilterConfig[]>([]);
  readonly timelineTypes = input<TimelineType[]>([]);
  readonly candidates = input<string[]>([]);
  readonly selectedTimelineTypeForBuilder = model<string>('*');
  readonly typeIcons = input<Record<string, string>>({});
  readonly typeColors = input<
    Record<string, { backgroundColor: string; foregroundColor: string }>
  >({});
  readonly typeCandidateCounts = input<Record<string, number>>({});

  // Outputs (Outputs)
  readonly drawDiagram = output<void>();
  readonly switchToAdvanced = output<void>();

  protected getTypeColor(typeLabel: string): string {
    const colors = this.typeColors()[typeLabel.toLowerCase()];
    return colors ? colors.foregroundColor : '';
  }

  protected getTypeBgColor(typeLabel: string): string {
    const colors = this.typeColors()[typeLabel.toLowerCase()];
    return colors ? colors.backgroundColor : '';
  }

  /**
   * Returns the formatted display string for the filter value.
   * Show the raw pattern for regex mode, and "selectedCount/totalCount" for selection mode.
   */
  protected getFilterValueDisplay(filter: TimelineFilterConfig): string {
    if (filter.mode === 'regex') {
      return filter.value;
    }
    const selectedCount = filter.value ? filter.value.split('|').length : 0;
    const totalCount =
      this.typeCandidateCounts()[filter.timelineType.toLowerCase()] || 0;
    return `${selectedCount}/${totalCount}`;
  }

  /**
   * Returns the formatted tooltip string for the filter.
   * For selection mode, replaces '|' with ', ' for cleaner list presentation.
   */
  protected getFilterTooltip(filter: TimelineFilterConfig): string {
    if (filter.mode === 'regex') {
      return `${filter.timelineType}: ${filter.value}`;
    }
    const formattedValue = filter.value.split('|').join(', ');
    return `${filter.timelineType}: ${formattedValue}`;
  }

  protected readonly ToolbarPopupStatus = ToolbarPopupStatus;

  protected popupStatus: ToolbarPopupStatus = ToolbarPopupStatus.None;

  // Popup and Editing States
  protected readonly isFilterBuilderOpen = signal<boolean>(false);
  protected readonly editingFilter = signal<TimelineFilterConfig | null>(null);
  protected readonly builderFilterMode = signal<'regex' | 'selection'>(
    'selection',
  );
  protected readonly builderRegexValue = signal<string>('');
  protected readonly builderSelectedCandidates = signal<string[]>([]);

  protected setPopupState(state: ToolbarPopupStatus) {
    this.popupStatus =
      state === this.popupStatus ? ToolbarPopupStatus.None : state;
  }

  onTimezoneshiftCommit(event: Event) {
    const value = +(event.target as HTMLInputElement).value;
    this.timezoneShift.set(value);
  }

  /**
   * Toggles the interactive timeline filter builder popover open state.
   */
  protected toggleFilterBuilder(): void {
    this.isFilterBuilderOpen.set(!this.isFilterBuilderOpen());
    if (!this.isFilterBuilderOpen()) {
      this.editingFilter.set(null);
      this.selectedTimelineTypeForBuilder.set('*');
      this.builderFilterMode.set('selection');
      this.builderRegexValue.set('');
      this.builderSelectedCandidates.set([]);
    }
  }

  /**
   * Triggers when the filter builder emits a confirmation event.
   */
  protected onFilterAddConfirm(event: {
    timelineType: string;
    mode: 'regex' | 'selection';
    value: string;
  }): void {
    const currentFilters = this.timelineFilters();
    const editFilter = this.editingFilter();

    if (editFilter != null) {
      // Update existing filter
      const updated = currentFilters.map((f) => {
        if (f.id === editFilter.id) {
          return {
            ...f,
            timelineType: event.timelineType,
            mode: event.mode,
            value: event.value,
          };
        }
        return f;
      });
      this.timelineFilters.set(updated);
    } else {
      // Add new filter with random ID
      const newFilter: TimelineFilterConfig = {
        id: Math.random().toString(36).substring(2, 9),
        timelineType: event.timelineType,
        mode: event.mode,
        value: event.value,
      };
      this.timelineFilters.set([...currentFilters, newFilter]);
    }

    // Close popup and reset states
    this.isFilterBuilderOpen.set(false);
    this.editingFilter.set(null);
    this.selectedTimelineTypeForBuilder.set('*');
    this.builderFilterMode.set('selection');
    this.builderRegexValue.set('');
    this.builderSelectedCandidates.set([]);
  }

  /**
   * Opens the popover in edit mode for an existing filter.
   */
  protected openEditFilterPopup(filter: TimelineFilterConfig): void {
    this.editingFilter.set(filter);
    this.selectedTimelineTypeForBuilder.set(filter.timelineType);
    this.builderFilterMode.set(filter.mode);
    if (filter.mode === 'regex') {
      this.builderRegexValue.set(filter.value);
      this.builderSelectedCandidates.set([]);
    } else {
      this.builderRegexValue.set('');
      this.builderSelectedCandidates.set(filter.value.split('|'));
    }
    this.isFilterBuilderOpen.set(true);
  }

  /**
   * Deletes an existing filter from the active list.
   */
  protected deleteFilter(id: string): void {
    const updated = this.timelineFilters().filter((f) => f.id !== id);
    this.timelineFilters.set(updated);

    // If we were editing the deleted filter, close the popover and reset states
    if (this.editingFilter()?.id === id) {
      this.isFilterBuilderOpen.set(false);
      this.editingFilter.set(null);
      this.selectedTimelineTypeForBuilder.set('*');
      this.builderFilterMode.set('selection');
      this.builderRegexValue.set('');
      this.builderSelectedCandidates.set([]);
    }
  }

  @HostListener('window:keydown', ['$event'])
  protected interceptBrowserSearch(event: KeyboardEvent) {
    if (event.key === 'f' && (event.ctrlKey || event.metaKey)) {
      this.snackbar.open(
        'In-browser search may not work on KHI because elements outside the visible area are not rendered. Please use the search text field on the toolbar instead.',
        'OK',
      );
    }
  }
}
