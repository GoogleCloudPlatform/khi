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

import { Component, OnDestroy, computed, inject, output } from '@angular/core';
import {
  CelTimelineFilter,
  CelLogFilter,
} from 'src/app/store/domain/filter/cel-filter';
import { ExcludeNoLogsFilter } from 'src/app/store/domain/filter/other-filter';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs';
import { ViewStateService } from 'src/app/services/view-state.service';
import { ToolbarAdvancedComponent } from 'src/app/timeline-toolbar-advanced/components/toolbar-advanced.component';
import {
  BehaviorSubject,
  debounceTime,
  distinctUntilChanged,
  Subject,
  takeUntil,
} from 'rxjs';
import { SelectionManagerV2 } from 'src/app/services/selection-manager-v2.service';

/**
 * Acts as a smart container logic layer for the advanced timeline toolbar component.
 */
@Component({
  selector: 'khi-timeline-toolbar-advanced-smart',
  templateUrl: './timeline-toolbar-advanced-smart.component.html',
  styleUrls: ['./timeline-toolbar-advanced-smart.component.scss'],
  imports: [ToolbarAdvancedComponent],
})
export class TimelineToolbarAdvancedSmartComponent implements OnDestroy {
  private readonly viewStateService = inject(ViewStateService);
  private readonly selectionManager = inject(SelectionManagerV2);
  private readonly celTimelineFilter = inject(CelTimelineFilter);
  private readonly celLogFilter = inject(CelLogFilter);
  private readonly excludeNoLogsFilter = inject(ExcludeNoLogsFilter);

  /**
   * Emits an event to switch to standard mode.
   */
  readonly switchToStandard = output<void>();

  /**
   * Emits the latest committed timeline CEL expression string.
   */
  readonly timelineCelFilterChange = output<string>();

  /**
   * Emits the latest committed log CEL expression string.
   */
  readonly logCelFilterChange = output<string>();

  /**
   * Signal holding the current timezone offset shift in hours.
   */
  protected readonly timezoneShift = toSignal(
    this.viewStateService.timezoneShift,
  );

  /**
   * Signal indicating if neither log nor timeline elements are actively selected.
   */
  protected readonly logOrTimelineNotSelected = computed(() => {
    const selectedTimeline = this.selectionManager.selectedTimeline();
    const selectedLog = this.selectionManager.selectedLog();
    return selectedTimeline == null || selectedLog == null;
  });

  /**
   * Signal holding the flag to hide timelines lacking matching logs.
   */
  protected readonly hideTimelinesWithoutMatchingLogs = toSignal(
    this.viewStateService.hideTimelinesWithoutMatchingLogs,
  );

  private readonly timelineCelFilter$ = new BehaviorSubject<string>('');
  private readonly logCelFilter$ = new BehaviorSubject<string>('');

  /**
   * Signal containing the validation error message for the timeline CEL filter.
   */
  protected readonly timelineCelError = toSignal(
    this.timelineCelFilter$.pipe(
      map((val) => {
        if (!val || val.trim() === '') return '';
        const checkRes = this.celTimelineFilter.validate(val);
        return checkRes.success
          ? ''
          : (checkRes.error?.message ?? 'Invalid CEL expression.');
      }),
    ),
  );

  /**
   * Signal containing the validation error message for the log CEL filter.
   */
  protected readonly logCelError = toSignal(
    this.logCelFilter$.pipe(
      map((val) => {
        if (!val || val.trim() === '') return '';
        const checkRes = this.celLogFilter.validate(val);
        return checkRes.success
          ? ''
          : (checkRes.error?.message ?? 'Invalid CEL expression.');
      }),
    ),
  );

  private readonly destroyed = new Subject<void>();

  constructor() {
    this.timelineCelFilter$
      .pipe(
        debounceTime(200),
        distinctUntilChanged(),
        takeUntil(this.destroyed),
      )
      .subscribe((filter) => {
        this.celTimelineFilter.updateFilter(filter);
        this.timelineCelFilterChange.emit(filter);
      });

    this.logCelFilter$
      .pipe(
        debounceTime(200),
        distinctUntilChanged(),
        takeUntil(this.destroyed),
      )
      .subscribe((filter) => {
        this.celLogFilter.updateFilter(filter);
        this.logCelFilterChange.emit(filter);
      });

    this.viewStateService.hideTimelinesWithoutMatchingLogs
      .pipe(takeUntil(this.destroyed))
      .subscribe((hide) => {
        this.excludeNoLogsFilter.enabled.set(hide);
      });
  }

  ngOnDestroy(): void {
    this.destroyed.next();
    this.destroyed.complete();
  }

  /**
   * Commits a newly modified timezone shift value.
   */
  protected onTimezoneshiftCommit(value: number): void {
    this.viewStateService.setTimezoneShift(value);
  }

  /**
   * Pushes a newly modified timeline CEL filter value to the internal pipeline.
   */
  protected onTimelineCelFilterChange(filter: string): void {
    this.timelineCelFilter$.next(filter);
  }

  /**
   * Pushes a newly modified log CEL filter value to the internal pipeline.
   */
  protected onLogCelFilterChange(filter: string): void {
    this.logCelFilter$.next(filter);
  }

  /**
   * Updates the state visibility for timelines missing log hits.
   */
  protected onToggleHideTimelinesWithoutMatchingLogs(value: boolean): void {
    this.viewStateService.setHideTimelinesWithoutMatchingLogs(value);
  }

  /**
   * Launches the external architecture diagram output window.
   */
  protected onDrawDiagram(): void {
    window.open(window.location.pathname + '/graph', '_blank');
  }
}
