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

import { Component, OnDestroy, computed, inject, output } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { BreakpointObserver } from '@angular/cdk/layout';
import { map } from 'rxjs';
import { ViewStateService } from '../services/view-state.service';
import { ToolbarComponent } from './components/toolbar.component';
import { BehaviorSubject, Subject } from 'rxjs';
import { SelectionManagerV2 } from '../services/selection-manager-v2.service';

@Component({
  selector: 'khi-timeline-toolbar-smart',
  templateUrl: './timeline-toolbar-smart.component.html',
  imports: [ToolbarComponent],
})
export class TimelineToolbarSmartComponent implements OnDestroy {
  private readonly viewStateService = inject(ViewStateService);

  /**
   * Emits an event to switch to advanced mode.
   */
  readonly switchToAdvanced = output<void>();
  private readonly selectionManager = inject(SelectionManagerV2);
  private readonly breakpointObserver = inject(BreakpointObserver);

  /**
   * An empty set used as a fallback for template bindings.
   */
  protected readonly emptySet = new Set<string>();

  /**
   * Signal indicating whether to show button labels based on screen width.
   */
  protected readonly showButtonLabel = toSignal(
    this.breakpointObserver
      .observe(['(min-width: 1200px)'])
      .pipe(map((result) => result.matches)),
  );

  /**
   * Signal containing the current timezone shift in hours.
   */
  protected readonly timezoneShift = toSignal(
    this.viewStateService.timezoneShift,
  );

  /**
   * Signal indicating if no log or timeline is selected.
   */
  protected readonly logOrTimelineNotSelected = computed(() => {
    const selectedTimeline = this.selectionManager.selectedTimeline();
    const selectedLog = this.selectionManager.selectedLog();
    return selectedTimeline == null || selectedLog == null;
  });

  /**
   * Signal indicating whether to hide timelines without matching logs.
   */
  protected readonly hideTimelinesWithoutMatchingLogs = toSignal(
    this.viewStateService.hideTimelinesWithoutMatchingLogs,
  );

  private readonly logFilter$ = new BehaviorSubject<string>('');
  private readonly destroyed = new Subject<void>();

  constructor() {}

  ngOnDestroy() {
    this.destroyed.next();
    this.destroyed.complete();
  }

  /**
   * Handles the commit of a new timezone shift value.
   */
  protected onTimezoneshiftCommit(value: number) {
    this.viewStateService.setTimezoneShift(value);
  }

  /**
   * Handles the change of the log filter.
   */
  protected onLogFilterChange(filter: string) {
    this.logFilter$.next(filter);
  }

  /**
   * Toggles the visibility of timelines without matching logs.
   */
  protected onToggleHideTimelinesWithoutMatchingLogs(value: boolean) {
    this.viewStateService.setHideTimelinesWithoutMatchingLogs(value);
  }

  /**
   * Opens the graph page in a new tab.
   */
  protected onDrawDiagram() {
    window.open(window.location.pathname + '/graph', '_blank');
  }
}
