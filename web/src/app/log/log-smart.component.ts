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

import { Component, computed, inject, signal } from '@angular/core';
import { InspectionDataStoreService } from '../services/inspection-data-store.service';
import { SelectionManagerService } from '../services/selection-manager.service';
import { LogEntry } from '../store/log';
import { CommonModule } from '@angular/common';
import { AngularSplitModule } from 'angular-split';
import { LogContentComponent } from './components/log-content.component';
import { LogListComponent } from './components/log-list.component';
import { toSignal } from '@angular/core/rxjs-interop';

/**
 * `LogSmartComponent` is the main container for the log viewing interface.
 * It consists of a split view containing the `LogListComponent` for displaying the list of logs
 * and the `LogContentComponent` for showing the detailed content of a selected log.
 * It also manages the state synchronization between the UI and the underlying data stores.
 */
@Component({
  selector: 'khi-log-smart',
  templateUrl: './log-smart.component.html',
  styleUrls: ['./log-smart.component.scss'],
  imports: [
    CommonModule,
    LogListComponent,
    LogContentComponent,
    AngularSplitModule,
  ],
})
export class LogSmartComponent {
  private readonly selectionManager = inject(SelectionManagerService);
  private readonly inspectionDataStore = inject(InspectionDataStoreService);

  /**
   * The currently selected log entry.
   */
  public readonly selectedLog = toSignal(this.selectionManager.selectedLog, {
    initialValue: null,
  });

  /**
   * The list of logs that match the current filter criteria.
   */
  public readonly filteredLogs = toSignal(
    this.inspectionDataStore.filteredLogs,
    { initialValue: [] },
  );

  /**
   * The complete, unfiltered list of all logs.
   */
  public readonly allLogs = toSignal(this.inspectionDataStore.allLogs, {
    initialValue: [],
  });

  /**
   * The index of the currently selected log entry.
   * Defaults to -1 if no log is selected.
   */
  public readonly selectedLogIndex = toSignal(
    this.selectionManager.selectedLogIndex,
    { initialValue: -1 },
  );

  /**
   * A set of indices representing logs that are currently highlighted (e.g., on hover).
   */
  public readonly highlightLogIndices = toSignal(
    this.selectionManager.highlightLogIndices,
    { initialValue: new Set<number>() },
  );

  /**
   * The list of currently selected resource timelines, including their children if the
   * `includeTimelineChildren` option is enabled.
   */
  public readonly selectedTimelinesWithChildren = toSignal(
    this.selectionManager.selectedTimelinesWithChildren,
    { initialValue: [] },
  );

  /**
   * A signal representing whether the log list should be filtered by the currently selected timeline(s).
   */
  protected readonly filterByTimeline = signal(true);

  /**
   * A signal representing whether children of the selected timeline(s) should be included
   * in the timeline filter.
   */
  protected readonly includeTimelineChildren = toSignal(
    this.selectionManager.timelineSelectionShouldIncludeChildren,
    { initialValue: true },
  );

  /**
   * The total number of logs available, prior to any filtering.
   */
  public readonly allLogsCount = computed(() => this.allLogs().length);

  /**
   * Internal click handler invoked when a log is selected from the list.
   * Updates the global selection state via `SelectionManagerService`.
   */
  protected onLogSelected(logEntry: LogEntry) {
    this.selectionManager.changeSelectionByLog(logEntry);
  }

  /**
   * Internal hover handler invoked when a user hovers over a log in the list.
   * Updates the global highlight state via `SelectionManagerService`.
   */
  protected onLogHovered(logEntry: LogEntry) {
    this.selectionManager.onHighlightLog(logEntry);
  }

  /**
   * Internal change handler invoked when the "include timeline children" toggle is toggled.
   * Updates the global setting in the `SelectionManagerService`.
   */
  protected onIncludeTimelineChildrenChange(value: boolean) {
    this.selectionManager.timelineSelectionShouldIncludeChildren.next(value);
  }
}
