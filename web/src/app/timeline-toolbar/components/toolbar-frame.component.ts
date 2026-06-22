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

import { Component, input, model, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { TimelineFilterConfig } from 'src/app/timeline-toolbar/types/filter-config';
import { TimelineType } from 'src/app/store/domain/style';
import { SearchScope } from 'src/app/services/view-state.service';
import { ToolbarComponent } from './toolbar.component';
import { ToolbarAdvancedComponent } from './toolbar-advanced.component';

/**
 * ToolbarFrameComponent acts as a Dumb layout wrapper component.
 * It conditionally switches rendering between standard and advanced timeline toolbars based on `isAdvancedMode`.
 */
@Component({
  selector: 'khi-toolbar-frame',
  templateUrl: './toolbar-frame.component.html',
  styleUrls: ['./toolbar-frame.component.scss'],
  imports: [
    CommonModule,
    ToolbarComponent,
    ToolbarAdvancedComponent,
    MatProgressBarModule,
  ],
})
export class ToolbarFrameComponent {
  /** Two-way model binding managing the advanced display mode state. */
  readonly isAdvancedMode = model.required<boolean>();

  /** Holds the current active search scope. */
  readonly activeSearchScope = input.required<SearchScope>();

  // Shared parameters
  /** Flag indicating if the app is currently filtering timelines/logs. */
  readonly isFiltering = input.required<boolean>();

  /** Percentage completion of the current filtering task. */
  readonly progressPercent = input.required<number>();

  /** Input holding the current timezone shift in hours. */
  readonly timezoneShift = input.required<number>();

  /** Flag locking button triggers when selection context is missing. */
  readonly logOrTimelineNotSelected = input.required<boolean>();

  // Standard Toolbar properties
  /** Holds the currently selected severity state. */
  readonly selectedSeverity = model.required<string>();

  /** Holds the standard log search input query. */
  readonly logSearchQuery = model.required<string>();

  /** Configured standard timeline filters. */
  readonly timelineFilters = model.required<TimelineFilterConfig[]>();

  /** Selected timeline type used within interactive filter builders. */
  readonly selectedTimelineTypeForBuilder = model.required<string>();

  /** Registered timeline types within store. */
  readonly timelineTypes = input.required<TimelineType[]>();

  /** Current autocomplete suggestions. */
  readonly candidates = input.required<string[]>();

  /** Calculated type candidate counts. */
  readonly typeCandidateCounts = input.required<Record<string, number>>();

  /** Controls button label visibility based on screen width. */
  readonly showButtonLabel = input.required<boolean>();

  // Advanced Toolbar properties
  /** Holds the advanced timeline include CEL filter text. */
  readonly timelineIncludeCelFilter = input.required<string>();

  /** Holds the advanced timeline exclude CEL filter text. */
  readonly timelineExcludeCelFilter = input.required<string>();

  /** Holds the advanced log CEL filter text. */
  readonly logCelFilter = input.required<string>();

  /** Displays validation error for timeline include CEL field. */
  readonly timelineIncludeCelError = input.required<string>();

  /** Displays validation error for timeline exclude CEL field. */
  readonly timelineExcludeCelError = input.required<string>();

  /** Displays validation error for log CEL field. */
  readonly logCelError = input.required<string>();

  /** Options mapping to exclude resource lines missing matching log occurrences. */
  readonly hideTimelinesWithoutMatchingLogs = input.required<boolean>();

  // Output Events
  /** Commits timezone changes to the parent store. */
  readonly timezoneShiftChange = output<number>();

  /** Commits hide timelines without matching logs toggle changes. */
  readonly hideTimelinesWithoutMatchingLogsChange = output<boolean>();

  /** Pushes changes to the timeline include CEL string. */
  readonly timelineIncludeCelFilterChange = output<string>();

  /** Pushes changes to the timeline exclude CEL string. */
  readonly timelineExcludeCelFilterChange = output<string>();

  /** Pushes changes to the log CEL string. */
  readonly logCelFilterChange = output<string>();
}
