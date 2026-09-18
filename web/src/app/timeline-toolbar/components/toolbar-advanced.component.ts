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
  HostListener,
  input,
  model,
  output,
  signal,
  viewChild,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatTooltipModule } from '@angular/material/tooltip';
import { OverlayModule } from '@angular/cdk/overlay';
import { CelInputComponent } from 'src/app/timeline-toolbar/components/cel-input.component';
import { ToolbarSettingsComponent } from 'src/app/timeline-toolbar/components/toolbar-settings.component';
import { SearchScope } from 'src/app/services/view-state.service';
import {
  CelGuidePopupComponent,
  CelGuideTab,
} from 'src/app/timeline-toolbar/components/cel-guide-popup.component';
import { TimeRangeFilterBuilderComponent } from 'src/app/timeline-toolbar/components/time-range-filter-builder.component';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { TimeRangeFilter } from 'src/app/services/view-state.service';
import {
  formatTimeRangeChipLines,
  formatTimeRangeTooltip,
  TimeRangeChipLines,
} from 'src/app/utils/time-format-util';
import { isEventFromOverlay, isSearchShortcut } from 'src/app/common/dom-util';

/**
 * Provides an advanced toolbar for the timeline view featuring two-row CEL expression text fields.
 */
@Component({
  selector: 'khi-timeline-toolbar-advanced',
  templateUrl: './toolbar-advanced.component.html',
  styleUrls: ['./toolbar-advanced.component.scss'],
  imports: [
    CommonModule,
    MatIconModule,
    KHIIconRegistrationModule,
    CelInputComponent,
    ToolbarSettingsComponent,
    CelGuidePopupComponent,
    TimeRangeFilterBuilderComponent,
    MatButtonModule,
    OverlayModule,
    MatTooltipModule,
  ],
})
export class ToolbarAdvancedComponent {
  protected readonly CelGuideTab = CelGuideTab;

  // Time Range Filter (Dumb state)
  /** Two-way model binding managing the time range filter state. */
  readonly timeRangeFilter = model<TimeRangeFilter | null>(null);

  /** Default start timestamp in nanoseconds when no explicit range is set. */
  readonly defaultStartTime = input<bigint>(0n);

  /** Default end timestamp in nanoseconds when no explicit range is set. */
  readonly defaultEndTime = input<bigint>(0n);

  // Time Range Filter builder popover state
  protected readonly isTimeRangeBuilderOpen = signal<boolean>(false);
  protected readonly timeRangeChipLines = computed<TimeRangeChipLines>(() => {
    const range = this.timeRangeFilter();
    return range
      ? formatTimeRangeChipLines(
          range.startTime,
          range.endTime,
          this.timezoneShift(),
        )
      : { startLine: '', endLine: '' };
  });
  protected readonly timeRangeChipTooltip = computed(() => {
    const range = this.timeRangeFilter();
    return range
      ? formatTimeRangeTooltip(
          range.startTime,
          range.endTime,
          this.timezoneShift(),
        )
      : '';
  });

  protected toggleTimeRangeBuilder(): void {
    this.isTimeRangeBuilderOpen.set(!this.isTimeRangeBuilderOpen());
  }

  protected onTimeRangeConfirm(range: TimeRangeFilter): void {
    this.timeRangeFilter.set(range);
    this.isTimeRangeBuilderOpen.set(false);
  }

  protected clearTimeRangeFilter(event?: Event): void {
    event?.stopPropagation();
    this.timeRangeFilter.set(null);
    this.isTimeRangeBuilderOpen.set(false);
  }

  /**
   * Reference to the Log CEL input component for search focus management.
   */
  public readonly logCelInput = viewChild<CelInputComponent>('logCelInput');

  /**
   * Custom positions for the CDK overlay to align the popup top-center with the help button bottom-center.
   */
  protected readonly overlayPositions = [
    {
      originX: 'center' as const,
      originY: 'bottom' as const,
      overlayX: 'center' as const,
      overlayY: 'top' as const,
    },
  ];

  /**
   * Signal managing the open/closed state of the help guide popover.
   */
  protected readonly helpPopupOpen = signal(false);

  /**
   * Signal managing the active tab of the help guide popover.
   */
  protected readonly helpActiveTab = signal<CelGuideTab>(CelGuideTab.Overview);

  /**
   * Signal managing the open/closed state of the settings popover.
   */
  protected readonly settingsPopupOpen = signal(false);

  /**
   * Holds the two-way bound timezone shift in hours.
   */
  readonly timezoneShift = model(0);

  /**
   * Specifies whether a log or timeline is currently not selected.
   */
  readonly logOrTimelineNotSelected = input(true);

  /**
   * Holds the current active search scope.
   */
  readonly activeSearchScope = input<SearchScope>(SearchScope.Global);

  /**
   * Validation error message for the timeline include CEL filter.
   */
  readonly timelineIncludeCelError = input('');

  /**
   * Validation error message for the timeline exclude CEL filter.
   */
  readonly timelineExcludeCelError = input('');

  /**
   * Validation error message for the log CEL filter.
   */
  readonly logCelError = input('');

  /**
   * Holds the two-way bound option to hide timelines without matching logs.
   */
  readonly hideTimelinesWithoutMatchingLogs = model(false);

  /**
   * Holds the two-way bound timeline include CEL filter string.
   */
  readonly timelineIncludeCelFilter = model('');

  /**
   * Holds the two-way bound timeline exclude CEL filter string.
   */
  readonly timelineExcludeCelFilter = model('');

  /**
   * Holds the two-way bound log CEL filter string.
   */
  readonly logCelFilter = model('');

  /**
   * Emits an event to switch to standard mode.
   */
  readonly switchToStandard = output<void>();

  /**
   * Handles the change event for the timezone shift input.
   */
  public onTimezoneshiftCommit(event: Event): void {
    const value = +(event.target as HTMLInputElement).value;
    this.timezoneShift.set(isNaN(value) ? 0 : value);
  }

  /**
   * Switches the CEL guide popup active tab to Timeline CEL when focused if the guide popup is currently open.
   */
  public onTimelineCelFocus(): void {
    if (this.helpPopupOpen()) {
      this.helpActiveTab.set(CelGuideTab.TimelineCel);
    }
  }

  /**
   * Switches the CEL guide popup active tab to Log CEL when focused if the guide popup is currently open.
   */
  public onLogCelFocus(): void {
    if (this.helpPopupOpen()) {
      this.helpActiveTab.set(CelGuideTab.LogCel);
    }
  }

  /**
   * Intercepts Ctrl+F or Cmd+F to focus the Log CEL input when KHI log or diff search is not active.
   * @param event The keyboard event.
   */
  @HostListener('window:keydown', ['$event'])
  public onKeyDown(event: KeyboardEvent): void {
    if (isEventFromOverlay(event)) {
      return;
    }
    if (isSearchShortcut(event)) {
      if (event.defaultPrevented) {
        return;
      }
      if (this.activeSearchScope() === SearchScope.Global) {
        event.preventDefault();
        this.logCelInput()?.focus();
      }
    }
  }
}
