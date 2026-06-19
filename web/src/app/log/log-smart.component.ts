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
import { InspectionDataStoreV2 } from 'src/app/services/inspection-data-store-v2.service';
import { SelectionManagerV2 } from 'src/app/services/selection-manager-v2.service';
import { Log } from 'src/app/store/domain/log';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { CommonModule } from '@angular/common';
import { AngularSplitModule } from 'angular-split';
import {
  LogContentComponent,
  LogContentViewModel,
} from 'src/app/log/components/log-content.component';
import { ResourceRefAnnotationViewModel } from 'src/app/log/components/resource-reference-list.component';
import { LogListComponent } from 'src/app/log/components/log-list.component';
import { toSignal } from '@angular/core/rxjs-interop';
import {
  SearchScope,
  ViewStateService,
} from 'src/app/services/view-state.service';
import jsyaml from 'js-yaml';

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
  private readonly selectionManager = inject(SelectionManagerV2);
  private readonly inspectionDataStore = inject(InspectionDataStoreV2);
  private readonly viewState = inject(ViewStateService);

  /** Holds the active search scope. */
  public readonly activeSearchScope = this.viewState.activeSearchScope;

  /**
   * The timezone shift to apply to the timestamp.
   */
  public readonly timezoneShift = toSignal(this.viewState.timezoneShift, {
    initialValue: 0,
  });

  /**
   * The currently selected log entry.
   */
  public readonly selectedLog = this.selectionManager.selectedLog;

  /**
   * The list of logs that match the current filter criteria.
   */
  public readonly filteredLogs = computed<ReadonlyDomainElement<Log>[]>(() => {
    return this.inspectionDataStore.timelineView()?.filteredLogs() ?? [];
  });

  /**
   * The index of the currently selected log entry.
   * Defaults to -1 if no log is selected.
   */
  public readonly selectedLogIndex = this.selectionManager.selectedLogIndex;

  /**
   * A set of indices representing logs that are currently highlighted (e.g., on hover).
   */
  public readonly highlightLogIndices =
    this.selectionManager.highlightLogIndices;

  /**
   * The list of currently selected resource timelines, including their children if the
   * `includeTimelineChildren` option is enabled.
   */
  public readonly selectedTimelinesWithChildren =
    this.selectionManager.selectedTimelinesWithChildren;

  /**
   * Output of the currently selected timeline from the selection manager.
   */
  public readonly selectedTimeline = this.selectionManager.selectedTimeline;

  /**
   * A signal representing whether the log list should be filtered by the currently selected timeline(s).
   */
  protected readonly filterByTimeline = signal(true);

  /**
   * Signal tracking the currently selected timeline path to visually indicate selection state.
   */
  public readonly currentSelectedTimelinePath = computed(() => {
    const selected = this.selectedTimeline();
    return selected ? selected.path.map((n) => n.label).join('#') : '';
  });

  /**
   * A signal representing whether children of the selected timeline(s) should be included
   * in the timeline filter.
   */
  protected readonly includeTimelineChildren =
    this.selectionManager.timelineSelectionShouldIncludeChildren;

  /**
   * The total number of logs available, prior to any filtering.
   */
  public readonly allLogsCount = computed(() => {
    return this.inspectionDataStore.inspectionData()?.logStore.count ?? 0;
  });

  /**
   * Aggregates the selected log entry, its body, and its resource paths into a view model.
   */
  public readonly logContentViewModel = computed<LogContentViewModel | null>(
    () => {
      const log = this.selectedLog();
      if (!log) {
        return null;
      }

      const timelines =
        this.inspectionDataStore.timelineView()?.filteredTimelines() ?? [];
      const resourceRefs: ResourceRefAnnotationViewModel[] = [];
      for (const timeline of timelines) {
        if (
          timeline.lookupEventFromLog(log) !== null ||
          timeline.lookupRevisionFromLog(log) !== null
        ) {
          resourceRefs.push({
            label: timeline.debugPathText, // TODO: Replace with better readable path
            timelineId: timeline.id,
          });
        }
      }

      const logBodyText = log.body
        ? jsyaml.dump(log.body, { lineWidth: -1 })
        : '';
      return {
        logEntry: log,
        logBody: logBodyText,
        parsedLogBody: log.body,
        resourceRefs,
      };
    },
  );

  /**
   * Internal click handler invoked when a log is selected from the list.
   * Updates the global selection state via `SelectionManagerV2`.
   */
  protected onLogSelected(logEntry: ReadonlyDomainElement<Log>) {
    this.selectionManager.onSelectLog(logEntry);
  }

  /**
   * Internal hover handler invoked when a user hovers over a log in the list.
   * Updates the global highlight state via `SelectionManagerV2`.
   */
  protected onLogHovered(logEntry: ReadonlyDomainElement<Log>) {
    this.selectionManager.onHighlightLog(logEntry);
  }

  /**
   * Internal change handler invoked when the "include timeline children" toggle is toggled.
   * Updates the global setting in the `SelectionManagerV2`.
   */
  protected onIncludeTimelineChildrenChange(value: boolean) {
    this.selectionManager.timelineSelectionShouldIncludeChildren.set(value);
  }

  /**
   * Selects the timeline by its ID.
   */
  protected onTimelineSelected(timelineId: number) {
    const timeline = this.inspectionDataStore
      .inspectionData()
      ?.timelineStore.getTimeline(timelineId);
    if (timeline) {
      this.selectionManager.onSelectTimeline(timeline);
    }
  }

  /**
   * Highlights the timeline by its ID.
   */
  protected onTimelineHighlighted(timelineId: number) {
    const timeline = this.inspectionDataStore
      .inspectionData()
      ?.timelineStore.getTimeline(timelineId);
    if (timeline) {
      this.selectionManager.onHighlightTimeline(timeline);
    }
  }

  /**
   * Sets the active search scope in the ViewStateService based on whether Log Content is hovered or focused.
   */
  protected onScopeActiveChange(active: boolean): void {
    if (active) {
      this.viewState.activeSearchScope.set(SearchScope.Log);
    } else if (this.viewState.activeSearchScope() === SearchScope.Log) {
      this.viewState.activeSearchScope.set(SearchScope.Global);
    }
  }
}
