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

import { Component, OnDestroy, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { BreakpointObserver } from '@angular/cdk/layout';
import { map } from 'rxjs';
import { ViewStateService } from '../services/view-state.service';
import { SelectionManagerService } from '../services/selection-manager.service';
import {
  DEFAULT_TIMELINE_FILTER,
  TimelineFilter,
} from '../services/timeline-filter.service';
import { InspectionDataStoreService } from '../services/inspection-data-store.service';
import { ToolbarComponent } from './components/toolbar.component';
import * as generated from '../zzz-generated';
import { nonEmptyOrDefaultString } from '../utils/state-util';
import {
  BehaviorSubject,
  combineLatest,
  debounceTime,
  distinctUntilChanged,
  Subject,
  takeUntil,
} from 'rxjs';

@Component({
  selector: 'khi-timeline-toolbar-smart',
  templateUrl: './timeline-toolbar-smart.component.html',
  imports: [ToolbarComponent],
})
export class TimelineToolbarSmartComponent implements OnDestroy {
  private readonly viewStateService = inject(ViewStateService);
  private readonly selectionManager = inject(SelectionManagerService);
  private readonly timelineFilter = inject<TimelineFilter>(
    DEFAULT_TIMELINE_FILTER,
  );
  private readonly inspectionDataStore = inject(InspectionDataStoreService);
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
   * Signal containing all available resource kinds.
   */
  protected readonly kinds = toSignal(this.inspectionDataStore.availableKinds);

  /**
   * Signal containing the set of included resource kinds for filtering.
   */
  protected readonly includedKinds = toSignal(
    this.timelineFilter.kindTimelineFilter,
  );

  /**
   * Signal containing all available namespaces.
   */
  protected readonly namespaces = toSignal(
    this.inspectionDataStore.availableNamespaces,
  );

  /**
   * Signal containing the set of included namespaces for filtering.
   */
  protected readonly includedNamespaces = toSignal(
    this.timelineFilter.namespaceTimelineFilter,
  );

  /**
   * Signal containing all available subresource parent relationships as labels.
   */
  protected readonly subresourceRelationships = toSignal(
    this.inspectionDataStore.availableSubresourceParentRelationships.pipe(
      map((rels) => {
        const relationshipLabels = new Set<string>();
        for (const relationship of rels) {
          relationshipLabels.add(
            generated.ParentRelationshipToLabel(relationship),
          );
        }
        return relationshipLabels;
      }),
    ),
  );

  /**
   * Signal containing the set of included subresource parent relationships as labels for filtering.
   */
  protected readonly includedSubresourceRelationships = toSignal(
    this.timelineFilter.subresourceParentRelationshipFilter.pipe(
      map((rels) => {
        const relationshipLabels = new Set<string>();
        for (const relationship of rels) {
          relationshipLabels.add(
            generated.ParentRelationshipToLabel(relationship),
          );
        }
        return relationshipLabels;
      }),
    ),
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
  protected readonly logOrTimelineNotSelected = toSignal(
    combineLatest([
      this.selectionManager.selectedLog,
      this.selectionManager.selectedTimeline,
    ]).pipe(map(([l, t]) => l == null || t == null)),
  );

  /**
   * Signal indicating whether to hide subresources without matching logs.
   */
  protected readonly hideSubresourcesWithoutMatchingLogs = toSignal(
    this.viewStateService.hideSubresourcesWithoutMatchingLogs,
  );

  /**
   * Signal indicating whether to hide resources without matching logs.
   */
  protected readonly hideResourcesWithoutMatchingLogs = toSignal(
    this.viewStateService.hideResourcesWithoutMatchingLogs,
  );

  private readonly logFilter$ = new BehaviorSubject<string>('');
  private readonly destroyed = new Subject<void>();

  constructor() {
    this.logFilter$
      .pipe(
        map((a) => nonEmptyOrDefaultString(a, '.*')),
        debounceTime(200),
        distinctUntilChanged(),
        takeUntil(this.destroyed),
      )
      .subscribe((filter) => {
        this.inspectionDataStore.setLogRegexFilter(filter);
      });
  }

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
   * Handles the commit of a new set of included resource kinds.
   */
  protected onKindFilterCommit(kinds: Set<string>) {
    this.timelineFilter.setKindFilter(kinds);
  }

  /**
   * Handles the commit of a new set of included namespaces.
   */
  protected onNamespaceFilterCommit(namespaces: Set<string>) {
    this.timelineFilter.setNamespaceFilter(namespaces);
  }

  /**
   * Handles the commit of a new set of included subresource parent relationships.
   */
  protected onSubresourceRelationshipFilterCommit(
    subresourceRelationshipLabels: Set<string>,
  ) {
    const relationships = [];
    for (const relationshipLabel of subresourceRelationshipLabels) {
      relationships.push(
        generated.ParseParentRelationshipLabel(relationshipLabel),
      );
    }
    this.timelineFilter.setSubresourceParentRelationshipFilter(
      new Set(relationships),
    );
  }

  /**
   * Handles the change of the resource name filter.
   */
  protected onNameFilterChange(filter: string) {
    this.timelineFilter.setResourceNameRegexFilter(filter);
  }

  /**
   * Handles the change of the log filter.
   */
  protected onLogFilterChange(filter: string) {
    this.logFilter$.next(filter);
  }

  /**
   * Toggles the visibility of subresources without matching logs.
   */
  protected onToggleHideSubresourcesWithoutMatchingLogs(value: boolean) {
    this.viewStateService.setHideSubresourcesWithoutMatchingLogs(value);
  }

  /**
   * Toggles the visibility of resources without matching logs.
   */
  protected onToggleHideResourcesWithoutMatchingLogs(value: boolean) {
    this.viewStateService.setHideResourcesWithoutMatchingLogs(value);
  }

  /**
   * Opens the graph page in a new tab.
   */
  protected onDrawDiagram() {
    window.open(window.location.pathname + '/graph', '_blank');
  }
}
