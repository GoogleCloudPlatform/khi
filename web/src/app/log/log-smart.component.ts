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
import {
  LogContentComponent,
  LogContentViewModel,
} from './components/log-content.component';
import { LogListComponent } from './components/log-list.component';
import { toSignal } from '@angular/core/rxjs-interop';
import { ViewStateService } from '../services/view-state.service';
import { firstValueFrom, filter, of } from 'rxjs';
import jsyaml from 'js-yaml';
import {
  LogAnnotationTypeResourceRef,
  KHIFileTextReference,
  KHILogAnnotation,
} from '../common/schema/khi-file-types';
import { ToTextReferenceFromKHIFileBinary } from '../common/loader/reference-type';
import { resource } from '@angular/core';
import { ResourceTimeline } from '../store/timeline';

import { MatProgressBarModule } from '@angular/material/progress-bar';

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
    MatProgressBarModule,
  ],
})
export class LogSmartComponent {
  private readonly selectionManager = inject(SelectionManagerService);
  private readonly inspectionDataStore = inject(InspectionDataStoreService);
  private readonly viewState = inject(ViewStateService);

  /**
   * The timezone shift to apply to the timestamp.
   */
  public readonly timezoneShift = toSignal(this.viewState.timezoneShift, {
    initialValue: 0,
  });

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
   * Output of the currently selected timeline from the selection manager.
   */
  public readonly selectedTimeline = toSignal(
    this.selectionManager.selectedTimeline,
    { initialValue: null },
  );

  /**
   * A signal representing whether the log list should be filtered by the currently selected timeline(s).
   */
  protected readonly filterByTimeline = signal(true);

  /**
   * Signal tracking the currently selected timeline path to visually indicate selection state.
   */
  public readonly currentSelectedTimelinePath = computed(() => {
    const selected = this.selectedTimeline();
    return selected ? selected.resourcePath : '';
  });

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
   * Signal containing the current text reference resolver from the data store.
   */
  private readonly referenceResolver = toSignal(
    this.inspectionDataStore.referenceResolver.pipe(filter((tb) => !!tb)) ??
      of(null),
  );

  private readonly inspectionData = toSignal(
    this.inspectionDataStore.inspectionData,
    {
      initialValue: null,
    },
  );

  /**
   * Aggregates the selected log entry, its body, and its resource paths into a view model.
   */
  public readonly logContentViewModel = resource({
    params: () => ({
      log: this.selectedLog(),
      resolver: this.referenceResolver(),
      inspectionData: this.inspectionData(),
    }),
    loader: async ({ params }) => {
      const { log, resolver, inspectionData } = params;
      if (!log || !resolver || !inspectionData) {
        return null;
      }

      const logBodyText = await firstValueFrom(resolver.getText(log.body));
      let parsedLogBody: unknown = null;
      try {
        parsedLogBody = jsyaml.load(logBodyText);
      } catch (e) {
        console.warn('Failed to parse log body as YAML', e);
      }

      const textRefs = log.annotations
        .filter(
          (a: KHILogAnnotation) => a.type === LogAnnotationTypeResourceRef,
        )
        .map((a: KHILogAnnotation) => a['path'] as KHIFileTextReference);

      let paths: string[] = [];
      if (textRefs.length > 0) {
        const refs = await Promise.all(
          textRefs.map((ref: KHIFileTextReference) =>
            firstValueFrom(
              resolver.getText(ToTextReferenceFromKHIFileBinary(ref)),
            ),
          ),
        );

        const allPaths = refs.flatMap((ref: string) => [
          ref,
          ...(
            inspectionData.getAliasedTimelines(ref) as ResourceTimeline[]
          ).map((t: ResourceTimeline) => t.resourcePath),
        ]);
        paths = [...new Set(allPaths)] as string[];
      }

      return {
        logEntry: log,
        logBody: logBodyText as string,
        parsedLogBody,
        referencedResourcePaths: paths,
      } as LogContentViewModel;
    },
  });

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

  /**
   * Selects the resource at the resource path.
   */
  protected onResourceSelected(resourcePath: string) {
    this.selectionManager.onSelectTimeline(resourcePath);
  }

  /**
   * Highlights the resource at the resource path.
   */
  protected onResourceHighlighted(resourcePath: string) {
    this.selectionManager.onHighlightTimeline(resourcePath);
  }
}
