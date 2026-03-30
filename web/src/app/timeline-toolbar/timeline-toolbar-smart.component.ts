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
  template: `
    <khi-timeline-toolbar
      [showButtonLabel]="showButtonLabel() || false"
      [kinds]="kinds() || emptySet"
      [includedKinds]="includedKinds() || emptySet"
      [namespaces]="namespaces() || emptySet"
      [includedNamespaces]="includedNamespaces() || emptySet"
      [subresourceRelationships]="subresourceRelationships() || emptySet"
      [includedSubresourceRelationships]="
        includedSubresourceRelationships() || emptySet
      "
      [timezoneShift]="timezoneShift() || 0"
      [logOrTimelineNotSelected]="logOrTimelineNotSelected() || false"
      [hideSubresourcesWithoutMatchingLogs]="
        hideSubresourcesWithoutMatchingLogs() || false
      "
      [hideResourcesWithoutMatchingLogs]="
        hideResourcesWithoutMatchingLogs() || false
      "
      (timezoneShiftChange)="onTimezoneshiftCommit($event)"
      (includedKindsChange)="onKindFilterCommit($event)"
      (includedNamespacesChange)="onNamespaceFilterCommit($event)"
      (includedSubresourceRelationshipsChange)="
        onSubresourceRelationshipFilterCommit($event)
      "
      (nameFilterChange)="onNameFilterChange($event)"
      (logFilterChange)="onLogFilterChange($event)"
      (hideSubresourcesWithoutMatchingLogsChange)="
        onToggleHideSubresourcesWithoutMatchingLogs($event)
      "
      (hideResourcesWithoutMatchingLogsChange)="
        onToggleHideResourcesWithoutMatchingLogs($event)
      "
      (drawDiagram)="onDrawDiagram()"
    ></khi-timeline-toolbar>
  `,
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

  readonly emptySet = new Set<string>();

  readonly showButtonLabel = toSignal(
    this.breakpointObserver
      .observe(['(min-width: 1200px)'])
      .pipe(map((result) => result.matches)),
  );

  readonly kinds = toSignal(this.inspectionDataStore.availableKinds);
  readonly includedKinds = toSignal(this.timelineFilter.kindTimelineFilter);
  readonly namespaces = toSignal(this.inspectionDataStore.availableNamespaces);
  readonly includedNamespaces = toSignal(
    this.timelineFilter.namespaceTimelineFilter,
  );

  readonly subresourceRelationships = toSignal(
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

  readonly includedSubresourceRelationships = toSignal(
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

  readonly timezoneShift = toSignal(this.viewStateService.timezoneShift);

  readonly logOrTimelineNotSelected = toSignal(
    combineLatest([
      this.selectionManager.selectedLog,
      this.selectionManager.selectedTimeline,
    ]).pipe(map(([l, t]) => l == null || t == null)),
  );

  readonly hideSubresourcesWithoutMatchingLogs = toSignal(
    this.viewStateService.hideSubresourcesWithoutMatchingLogs,
  );
  readonly hideResourcesWithoutMatchingLogs = toSignal(
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

  onTimezoneshiftCommit(value: number) {
    this.viewStateService.setTimezoneShift(value);
  }

  onKindFilterCommit(kinds: Set<string>) {
    this.timelineFilter.setKindFilter(kinds);
  }

  onNamespaceFilterCommit(namespaces: Set<string>) {
    this.timelineFilter.setNamespaceFilter(namespaces);
  }

  onSubresourceRelationshipFilterCommit(
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

  onNameFilterChange(filter: string) {
    this.timelineFilter.setResourceNameRegexFilter(filter);
  }

  onLogFilterChange(filter: string) {
    this.logFilter$.next(filter);
  }

  onToggleHideSubresourcesWithoutMatchingLogs(value: boolean) {
    this.viewStateService.setHideSubresourcesWithoutMatchingLogs(value);
  }

  onToggleHideResourcesWithoutMatchingLogs(value: boolean) {
    this.viewStateService.setHideResourcesWithoutMatchingLogs(value);
  }

  onDrawDiagram() {
    window.open(window.location.pathname + '/graph', '_blank');
  }
}
