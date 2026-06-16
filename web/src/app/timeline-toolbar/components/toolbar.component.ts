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
  viewChild,
  ElementRef,
  input,
  model,
  output,
  signal,
  computed,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { OverlayModule } from '@angular/cdk/overlay';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatButtonModule } from '@angular/material/button';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatSelectModule } from '@angular/material/select';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { TimelineFilterBuilderComponent } from './timeline-filter-builder.component';
import { SearchScope } from 'src/app/services/view-state.service';
import { TimelineFilterConfig } from '../types/filter-config';
import { TimelineType } from 'src/app/store/domain/style';
import { RendererConvertUtil } from 'src/app/timeline/components/canvas/convertutil';

/**
 * Visual theme representation for a timeline type chip.
 */
export interface TimelineTypeColor {
  readonly backgroundColor: string;
  readonly foregroundColor: string;
}

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
  /**
   * Reference to the Log Search input element for search focus management.
   */
  public readonly logSearchInput =
    viewChild<ElementRef<HTMLInputElement>>('logSearchInput');

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

  /**
   * Holds the current active search scope.
   */
  readonly activeSearchScope = input<SearchScope>(SearchScope.Global);

  // Timeline Filters (Dumb state)
  readonly timelineFilters = model<TimelineFilterConfig[]>([]);
  readonly timelineTypes = input<TimelineType[]>([]);
  readonly candidates = input<string[]>([]);
  readonly selectedTimelineTypeForBuilder = model<string>('*');
  readonly typeCandidateCounts = input<Record<string, number>>({});

  /** Map matching timeline type labels to standard icons. */
  protected readonly typeIcons = computed<Record<string, string>>(() => {
    const map: Record<string, string> = {};
    for (const type of this.timelineTypes()) {
      map[type.label.toLowerCase()] = type.icon;
    }
    return map;
  });

  /** Theme colors mapped per timeline type. */
  protected readonly typeColors = computed<Record<string, TimelineTypeColor>>(
    () => {
      const map: Record<string, TimelineTypeColor> = {};
      for (const type of this.timelineTypes()) {
        map[type.label.toLowerCase()] = {
          backgroundColor: RendererConvertUtil.hdrColorToCSSColor([
            type.backgroundColor.r,
            type.backgroundColor.g,
            type.backgroundColor.b,
            type.backgroundColor.a,
          ]),
          foregroundColor: RendererConvertUtil.hdrColorToCSSColor([
            type.foregroundColor.r,
            type.foregroundColor.g,
            type.foregroundColor.b,
            type.foregroundColor.a,
          ]),
        };
      }
      return map;
    },
  );

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

  /**
   * Intercepts Ctrl+F or Cmd+F to focus the Log Search input when KHI log or diff search is not active.
   * @param event The keyboard event.
   */
  @HostListener('window:keydown', ['$event'])
  public onKeyDown(event: KeyboardEvent): void {
    if ((event.ctrlKey || event.metaKey) && event.key === 'f') {
      if (event.defaultPrevented) {
        return;
      }
      if (this.activeSearchScope() === SearchScope.Global) {
        event.preventDefault();
        this.logSearchInput()?.nativeElement.focus();
      }
    }
  }
}
