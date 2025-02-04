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

import { Component, Inject, OnDestroy, OnInit, inject } from '@angular/core';
import { FormControl } from '@angular/forms';
import {
  BehaviorSubject,
  Subject,
  combineLatest,
  debounceTime,
  distinctUntilChanged,
  map,
  takeUntil,
} from 'rxjs';
import * as generated from '../generated';
import { InspectionDataStoreService } from '../services/inspection-data-store.service';
import { ViewStateService } from '../services/view-state.service';
import { nonEmptyOrDefaultString } from '../utils/state-util';
import { SelectionManagerService } from '../services/selection-manager.service';
import { BreakpointObserver } from '@angular/cdk/layout';
import {
  DEFAULT_TIMELINE_FILTER,
  TimelineFilter,
} from '../services/timeline-filter.service';
import { SetInputComponent } from './set-input.component';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { OverlayModule } from '@angular/cdk/overlay';
import { RegexInputComponent } from './regex-input.component';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatButtonModule } from '@angular/material/button';

/**
 * ToolbarPopupStatus represents which popup of the toolbar is open.
 */
type ToolbarPopupStatus =
  | 'NONE_OPEN'
  | 'KIND_FILTER_OPEN'
  | 'NAMESPACE_FILTER_OPEN'
  | 'SUBRESOURCE_FILTER_OPEN';

@Component({
  selector: 'khi-header-toolbar',
  templateUrl: './toolbar.component.html',
  styleUrls: ['./toolbar.component.sass'],
  imports: [
    CommonModule,
    SetInputComponent,
    MatIconModule,
    OverlayModule,
    RegexInputComponent,
    MatButtonModule,
    MatButtonToggleModule,
  ],
})
export class ToolbarComponent implements OnInit, OnDestroy {
  private destoroyed = new Subject<void>();

  private breakpointObserver = inject(BreakpointObserver);

  showButtonLabel = this.breakpointObserver
    .observe(['(min-width: 1200px)'])
    .pipe(map((result) => result.matches));

  LOG_FILTER_DEBOUNCE_TIME = 200;

  filterControl: FormControl = new FormControl('');

  timezoneShift$ = this.viewStateService.timezoneShift;

  kinds$ = this.inspectionDataStore.availableKinds;
  includedKinds$ = this.timelineFilter.kindTimelineFilter;
  namespaces$ = this.inspectionDataStore.availableNamespaces;
  includedNamespaces$ = this.timelineFilter.namespaceTimelineFilter;
  subresourceRelationships =
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
    );
  includedSubresourceRelationships =
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
    );
  logTypes = new Set(generated.logTypes);

  logFilter$ = new BehaviorSubject<string>('');

  logOrTimelineNotSelected = combineLatest([
    this.selectionManager.selectedLog,
    this.selectionManager.selectedTimeline,
  ]).pipe(map(([l, t]) => l == null || t == null));

  popupStatus: ToolbarPopupStatus = 'NONE_OPEN';

  constructor(
    private selectionManager: SelectionManagerService,
    public readonly viewStateService: ViewStateService,
    private inspectionDataStore: InspectionDataStoreService,
    @Inject(DEFAULT_TIMELINE_FILTER) private timelineFilter: TimelineFilter,
  ) {}
  ngOnDestroy(): void {
    this.destoroyed.next();
  }

  ngOnInit(): void {
    const $reductedFilterEvent = this.logFilter$.pipe(
      takeUntil(this.destoroyed),
      map((a) => nonEmptyOrDefaultString(a, '.*')),
      debounceTime(this.LOG_FILTER_DEBOUNCE_TIME),
      distinctUntilChanged(),
    );
    $reductedFilterEvent.subscribe((filter) => {
      this.inspectionDataStore.setLogRegexFilter(filter);
    });
  }

  onTimezoneshiftCommit(event: Event) {
    const value = +(event.target as HTMLInputElement).value;
    this.viewStateService.setTimezoneShift(value);
  }

  setPopupState(state: ToolbarPopupStatus) {
    // Set the state to NONE_OPEN when the toolbarPopup is already with the state.
    this.popupStatus = state === this.popupStatus ? 'NONE_OPEN' : state;
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

  onToggleHideSubresourcesWithoutMatchingLogs() {
    this.viewStateService.setHideSubresourcesWithoutMatchingLogs(
      !this.viewStateService.hideSubresourcesWithoutMatchingLogs.value,
    );
  }

  onToggleHideResourcesWithoutMatchingLogs() {
    this.viewStateService.setHideResourcesWithoutMatchingLogs(
      !this.viewStateService.hideResourcesWithoutMatchingLogs.value,
    );
  }

  onDrawDiagram() {
    window.open(window.location.pathname + '/graph', '_blank');
  }
}
